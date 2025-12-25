/*
 * 认证服务
 * 负责设备的零知识认证和会话管理
 */
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/internal/zkp"
	"github.com/edge/storage-cabinet/pkg/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LicenseService 许可证服务接口
type LicenseService interface {
	Check() error
	IsEnabled() bool
	GetMaxDevices() int
}

// Service 认证服务
type Service struct {
	logger       *zap.Logger
	db           *storage.SQLiteDB
	verifier     zkp.ZKPVerifier
	license      LicenseService // 许可证服务（可选）
	jwtSecret    []byte
	challengeTTL time.Duration
	sessionTTL   time.Duration
	maxRetry     int
	retryCount   map[string]int // 设备重试次数记录
}

// NewService 创建认证服务
func NewService(cfg config.AuthConfig, db *storage.SQLiteDB, verifier zkp.ZKPVerifier, license LicenseService, logger *zap.Logger) *Service {
	// 生成或加载JWT密钥
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// 生成随机密钥
		secret := make([]byte, 32)
		rand.Read(secret)
		jwtSecret = hex.EncodeToString(secret)
		logger.Warn("JWT_SECRET not set, using random secret")
	}

	return &Service{
		logger:       logger,
		db:           db,
		verifier:     verifier,
		license:      license,
		jwtSecret:    []byte(jwtSecret),
		challengeTTL: cfg.ChallengeTTL,
		sessionTTL:   cfg.SessionTTL,
		maxRetry:     cfg.MaxRetry,
		retryCount:   make(map[string]int),
	}
}

// GenerateChallenge 生成认证挑战
func (s *Service) GenerateChallenge(deviceID string) (*models.Challenge, error) {
	// 【SPA单包授权】许可证校验（单点门控）
	if s.license != nil && s.license.IsEnabled() {
		if err := s.license.Check(); err != nil {
			s.logger.Error("许可证校验失败",
				zap.String("device_id", deviceID),
				zap.Error(err))
			return nil, fmt.Errorf("LICENSE_001: 许可证校验失败 - %w", err)
		}
	}

	// 检查设备是否存在
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// 检查重试次数
	if count := s.retryCount[deviceID]; count >= s.maxRetry {
		s.logger.Warn("Too many auth attempts",
			zap.String("device_id", deviceID),
			zap.Int("attempts", count))
		return nil, fmt.Errorf("too many authentication attempts")
	}

	// 生成挑战
	nonce, err := s.verifier.GenerateChallenge()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// 创建挑战记录
	challenge := &models.Challenge{
		ChallengeID: uuid.New().String(),
		DeviceID:    device.DeviceID,
		Nonce:       nonce,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(s.challengeTTL),
		Used:        false,
	}

	// 保存到数据库
	if err := s.saveChallenge(challenge); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	s.logger.Info("Challenge created",
		zap.String("device_id", deviceID),
		zap.String("challenge_id", challenge.ChallengeID))

	return challenge, nil
}

// VerifyProof 验证零知识证明
func (s *Service) VerifyProof(req *models.AuthRequest) (*models.Session, error) {
	// 获取挑战
	challenge, err := s.getChallenge(req.ChallengeID)
	if err != nil {
		s.retryCount[req.DeviceID]++
		return nil, fmt.Errorf("invalid challenge: %w", err)
	}

	// 检查挑战是否过期
	if time.Now().After(challenge.ExpiresAt) {
		s.retryCount[req.DeviceID]++
		return nil, fmt.Errorf("challenge expired")
	}

	// 检查挑战是否已使用
	if challenge.Used {
		s.retryCount[req.DeviceID]++
		return nil, fmt.Errorf("challenge already used")
	}

	// 获取设备信息
	device, err := s.getDevice(req.DeviceID)
	if err != nil {
		s.retryCount[req.DeviceID]++
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// 从PublicWitness对象中提取参数（新格式：对象而非数组）
	pw := req.Proof.PublicWitness
	if pw.DeviceID == "" || pw.Challenge == "" || pw.Commitment == "" || pw.Response == "" {
		return nil, fmt.Errorf("invalid public witness: missing required fields")
	}

	// 验证公开见证的一致性
	// ✅ 修复：将DeviceID转换为域元素hex后再比较
	deviceIDBytes := []byte(device.DeviceID)
	deviceIDBig := new(big.Int).SetBytes(deviceIDBytes)
	deviceIDFieldBytes := make([]byte, 32)
	deviceIDBig.FillBytes(deviceIDFieldBytes)
	expectedDeviceIDHex := hex.EncodeToString(deviceIDFieldBytes)

	if pw.DeviceID != expectedDeviceIDHex {
		return nil, fmt.Errorf("device ID mismatch in witness")
	}
	if pw.Challenge != challenge.Nonce {
		return nil, fmt.Errorf("challenge mismatch in witness")
	}
	if pw.Commitment != device.Commitment {
		return nil, fmt.Errorf("commitment mismatch in witness")
	}

	// 解码Base64 proof数据（新格式：Base64字符串而非字节数组）
	proofBytes, err := base64.StdEncoding.DecodeString(req.Proof.Proof)
	if err != nil {
		s.logger.Error("Failed to decode proof", zap.Error(err))
		return nil, fmt.Errorf("failed to decode proof: %w", err)
	}

	// 验证零知识证明
	valid, err := s.verifier.VerifyProof(
		device.DeviceID,
		challenge.Nonce,
		device.Commitment,
		pw.Response,
		proofBytes,
	)
	if err != nil {
		s.logger.Error("Failed to verify proof", zap.Error(err))
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	if !valid {
		s.retryCount[req.DeviceID]++
		s.logger.Warn("Invalid proof",
			zap.String("device_id", req.DeviceID),
			zap.String("challenge_id", req.ChallengeID))
		return nil, fmt.Errorf("invalid proof")
	}

	// 标记挑战已使用
	if err := s.markChallengeUsed(req.ChallengeID); err != nil {
		s.logger.Error("Failed to mark challenge as used", zap.Error(err))
	}

	// 创建会话
	session, err := s.createSession(device.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 生成JWT令牌
	token, err := s.GenerateToken(device.DeviceID, session.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 重置重试计数
	delete(s.retryCount, req.DeviceID)

	s.logger.Info("Authentication successful",
		zap.String("device_id", req.DeviceID),
		zap.String("session_id", session.SessionID))

	session.Token = token
	return session, nil
}

// ValidateToken 验证令牌
func (s *Service) ValidateToken(tokenString string) (*models.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.TokenClaims); ok && token.Valid {
		// 检查会话是否有效
		session, err := s.getSession(claims.SessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found")
		}

		if time.Now().After(session.ExpiresAt) {
			return nil, fmt.Errorf("session expired")
		}

		// 更新最后使用时间
		s.updateSessionLastUsed(claims.SessionID)

		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshSession 刷新会话
func (s *Service) RefreshSession(tokenString string) (*models.Session, error) {
	// 验证当前token并获取session信息
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	sessionID := claims.SessionID
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// 检查会话是否过期
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// 延长会话时间
	session.ExpiresAt = time.Now().Add(s.sessionTTL)
	if err := s.updateSession(session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// 生成新令牌
	token, err := s.GenerateToken(session.DeviceID, session.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session.Token = token
	return session, nil
}

// RevokeSession 撤销会话
func (s *Service) RevokeSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = ?`
	_, err := s.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	s.logger.Info("Session revoked", zap.String("session_id", sessionID))
	return nil
}

// createSession 创建会话
func (s *Service) createSession(deviceID string) (*models.Session, error) {
	session := &models.Session{
		SessionID:  uuid.New().String(),
		DeviceID:   deviceID,
		Token:      "", // 将在生成JWT后更新
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(s.sessionTTL),
		LastUsedAt: time.Now(),
	}

	query := `
		INSERT INTO sessions (session_id, device_id, token, created_at, expires_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		session.SessionID, session.DeviceID, session.Token,
		session.CreatedAt, session.ExpiresAt, session.LastUsedAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GenerateToken 生成JWT令牌
func (s *Service) GenerateToken(deviceID, sessionID string) (string, error) {
	now := time.Now()
	claims := &models.TokenClaims{
		DeviceID:  deviceID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.sessionTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   deviceID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// Database helper methods

func (s *Service) getDevice(deviceID string) (*models.Device, error) {
	var device models.Device
	query := `
		SELECT device_id, device_type, sensor_type,
		       public_key, commitment, status
		FROM devices WHERE device_id = ?
	`
	err := s.db.QueryRow(query, deviceID).Scan(
		&device.DeviceID, &device.DeviceType, &device.SensorType,
		&device.PublicKey, &device.Commitment,
		&device.Status,
	)
	if err != nil {
		return nil, err
	}
	// 从配置获取cabinet_id（数据库中没有此字段）
	device.CabinetID = s.getCabinetID()
	return &device, nil
}

func (s *Service) saveChallenge(challenge *models.Challenge) error {
	query := `
		INSERT INTO challenges (challenge_id, device_id, nonce, created_at, expires_at, used)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		challenge.ChallengeID, challenge.DeviceID, challenge.Nonce,
		challenge.CreatedAt, challenge.ExpiresAt, challenge.Used,
	)
	return err
}

func (s *Service) getChallenge(challengeID string) (*models.Challenge, error) {
	var challenge models.Challenge
	query := `
		SELECT challenge_id, device_id, nonce, created_at, expires_at, used
		FROM challenges WHERE challenge_id = ?
	`
	err := s.db.QueryRow(query, challengeID).Scan(
		&challenge.ChallengeID, &challenge.DeviceID, &challenge.Nonce,
		&challenge.CreatedAt, &challenge.ExpiresAt, &challenge.Used,
	)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *Service) markChallengeUsed(challengeID string) error {
	query := `UPDATE challenges SET used = TRUE WHERE challenge_id = ?`
	_, err := s.db.Exec(query, challengeID)
	return err
}

func (s *Service) getSession(sessionID string) (*models.Session, error) {
	var session models.Session
	query := `
		SELECT session_id, device_id, token, created_at, expires_at, last_used_at
		FROM sessions WHERE session_id = ?
	`
	err := s.db.QueryRow(query, sessionID).Scan(
		&session.SessionID, &session.DeviceID, &session.Token,
		&session.CreatedAt, &session.ExpiresAt, &session.LastUsedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Service) updateSession(session *models.Session) error {
	query := `
		UPDATE sessions SET expires_at = ?, last_used_at = ?
		WHERE session_id = ?
	`
	_, err := s.db.Exec(query, session.ExpiresAt, time.Now(), session.SessionID)
	return err
}

func (s *Service) updateSessionLastUsed(sessionID string) error {
	query := `UPDATE sessions SET last_used_at = ? WHERE session_id = ?`
	_, err := s.db.Exec(query, time.Now(), sessionID)
	return err
}

// getCabinetID 获取储能柜ID（从环境变量或使用默认值）
func (s *Service) getCabinetID() string {
	cabinetID := os.Getenv("CABINET_ID")
	if cabinetID == "" {
		cabinetID = "CABINET-001" // 默认值
	}
	return cabinetID
}
