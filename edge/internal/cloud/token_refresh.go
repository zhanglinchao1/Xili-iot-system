/*
 * Token续签服务
 * 负责自动续签Cloud端admin_token，确保token在过期前自动更新
 */
package cloud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"go.uber.org/zap"
)

// CloudCredentialsStore 数据库凭证存储接口
type CloudCredentialsStore interface {
	GetFirstCloudCredentials() (*storage.CloudCredential, error)
	GetCloudCredentials(cabinetID string) (*storage.CloudCredential, error)
}

// TokenRefreshService Token续签服务
type TokenRefreshService struct {
	logger     *zap.Logger
	cfg        *config.Config
	storage    CloudCredentialsStore
	httpClient *http.Client
	stopChan   chan struct{}
	running    bool
}

// NewTokenRefreshService 创建Token续签服务
func NewTokenRefreshService(cfg *config.Config, storage CloudCredentialsStore, logger *zap.Logger) *TokenRefreshService {
	return &TokenRefreshService{
		logger:  logger,
		cfg:     cfg,
		storage: storage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// getEndpoint 获取Cloud端点（优先从数据库读取）
func (s *TokenRefreshService) getEndpoint() string {
	if s.storage != nil {
		cabinetID := s.cfg.Cloud.CabinetID
		if cred, err := s.storage.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
			return cred.CloudEndpoint
		}
		if cred, err := s.storage.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
			return cred.CloudEndpoint
		}
	}
	return s.cfg.Cloud.Endpoint
}

// Start 启动Token续签服务
func (s *TokenRefreshService) Start() error {
	if s.running {
		return fmt.Errorf("token refresh service already running")
	}

	s.running = true
	s.logger.Info("Token续签服务已启动")

	// 立即检查一次
	go s.checkAndRefreshToken()

	// 定期检查（每24小时检查一次）
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.checkAndRefreshToken()
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Stop 停止Token续签服务
func (s *TokenRefreshService) Stop() {
	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
	s.logger.Info("Token续签服务已停止")
}

// checkAndRefreshToken 检查并续签token
func (s *TokenRefreshService) checkAndRefreshToken() {
	if !s.cfg.Cloud.Enabled || s.cfg.Cloud.AdminToken == "" {
		s.logger.Debug("Cloud未启用或admin_token未配置，跳过token续签")
		return
	}

	// 解析token获取过期时间
	expiresAt, err := s.parseTokenExpiry(s.cfg.Cloud.AdminToken)
	if err != nil {
		s.logger.Warn("无法解析token过期时间，尝试续签", zap.Error(err))
		// 如果无法解析，尝试续签
		s.refreshToken()
		return
	}

	// 检查是否需要在7天内续签
	now := time.Now()
	daysUntilExpiry := expiresAt.Sub(now).Hours() / 24

	if daysUntilExpiry <= 7 {
		s.logger.Info("Token即将过期，开始续签",
			zap.Float64("days_until_expiry", daysUntilExpiry),
			zap.Time("expires_at", expiresAt))
		s.refreshToken()
	} else {
		s.logger.Debug("Token尚未到期，无需续签",
			zap.Float64("days_until_expiry", daysUntilExpiry),
			zap.Time("expires_at", expiresAt))
	}
}

// parseTokenExpiry 解析JWT token获取过期时间
func (s *TokenRefreshService) parseTokenExpiry(token string) (time.Time, error) {
	// JWT格式: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}

	// 解码payload（base64url）
	payload := parts[1]
	// 添加padding（如果需要）
	for len(payload)%4 != 0 {
		payload += "="
	}

	// Base64URL解码
	decoded, err := base64URLDecode(payload)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// 解析JSON
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// 获取exp字段（Unix时间戳）
	exp, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("token missing exp claim")
	}

	return time.Unix(int64(exp), 0), nil
}

// refreshToken 续签token
func (s *TokenRefreshService) refreshToken() {
	s.logger.Info("开始续签admin_token")

	// 从Cloud端登录获取新token（优先从数据库获取endpoint）
	endpoint := s.getEndpoint()
	loginURL := strings.TrimSuffix(endpoint, "/api/v1") + "/api/v1/auth/login"
	
	// 使用默认管理员账号（可以从环境变量或配置文件读取）
	username := os.Getenv("CLOUD_ADMIN_USERNAME")
	password := os.Getenv("CLOUD_ADMIN_PASSWORD")
	
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin"
	}

	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}

	payloadBytes, err := json.Marshal(loginPayload)
	if err != nil {
		s.logger.Error("无法序列化登录请求", zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		s.logger.Error("创建登录请求失败", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("登录请求失败", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("登录失败", 
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
		return
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		s.logger.Error("解析登录响应失败", zap.Error(err))
		return
	}

	if !loginResp.Success || loginResp.Data.Token == "" {
		s.logger.Error("登录响应无效", zap.String("message", loginResp.Message))
		return
	}

	// 更新配置文件
	if err := s.updateConfigToken(loginResp.Data.Token); err != nil {
		s.logger.Error("更新配置文件失败", zap.Error(err))
		return
	}

	// 更新内存中的配置
	s.cfg.Cloud.AdminToken = loginResp.Data.Token

	s.logger.Info("Token续签成功",
		zap.String("expires_at", loginResp.Data.ExpiresAt))
}

// updateConfigToken 更新配置文件中的token
func (s *TokenRefreshService) updateConfigToken(newToken string) error {
	configPath := "configs/config.yaml"
	if s.cfg != nil {
		// 尝试从配置中获取路径（如果可用）
		// 这里简化处理，直接使用相对路径
	}

	// 读取配置文件
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 简单的字符串替换（更安全的方法是使用yaml库）
	lines := strings.Split(string(content), "\n")
	updated := false

	for i, line := range lines {
		if strings.Contains(line, "admin_token:") {
			// 保留注释
			comment := ""
			if idx := strings.Index(line, "#"); idx > 0 {
				comment = line[idx:]
			}
			lines[i] = fmt.Sprintf(`  admin_token: "%s"  %s`, newToken, comment)
			updated = true
			break
		}
	}

	if !updated {
		// 如果找不到admin_token行，在cloud配置块中添加
		for i, line := range lines {
			if strings.Contains(line, "cabinet_id:") {
				// 在cabinet_id之前插入admin_token
				lines = append(lines[:i], append([]string{fmt.Sprintf(`  admin_token: "%s"  # Cloud端管理员JWT token（自动续签）`, newToken)}, lines[i:]...)...)
				updated = true
				break
			}
		}
	}

	if !updated {
		return fmt.Errorf("could not find admin_token or cabinet_id in config file")
	}

	// 写回文件
	if err := os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// base64URLDecode Base64URL解码
func base64URLDecode(s string) ([]byte, error) {
	// Base64URL使用-和_而不是+和/
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// 标准base64解码
	decoded := make([]byte, len(s)*3/4)
	n, err := base64.StdEncoding.Decode(decoded, []byte(s))
	if err != nil {
		return nil, err
	}

	return decoded[:n], nil
}

