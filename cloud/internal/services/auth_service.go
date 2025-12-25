// Package services 实现认证业务逻辑
package services

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务接口
type AuthService interface {
	// Login 用户登录
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)

	// Register 用户注册
	Register(ctx context.Context, req *models.RegisterRequest) (*models.RegisterResponse, error)
}

// authService 实现AuthService接口
type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	// Go的regexp不支持lookahead,所以需要在代码中手动检查
	// 密码要求: 8-128字符,包含大小写字母和数字
	passwordPattern = regexp.MustCompile(`^.{8,128}$`)
)

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, req *models.RegisterRequest) (*models.RegisterResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	password := req.Password

	if !usernamePattern.MatchString(username) {
		return nil, errors.NewValidationError("用户名需为3-64位，仅支持字母、数字、下划线和连字符")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, errors.NewValidationError("邮箱格式不正确")
	}
	email = strings.ToLower(email)

	if !passwordPattern.MatchString(password) {
		return nil, errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}
	// 手动检查密码复杂度(因为Go regexp不支持lookahead)
	hasLower := false
	hasUpper := false
	hasDigit := false
	for _, c := range password {
		switch {
		case 'a' <= c && c <= 'z':
			hasLower = true
		case 'A' <= c && c <= 'Z':
			hasUpper = true
		case '0' <= c && c <= '9':
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		return nil, errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}

	exists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "检查用户名是否存在失败")
	}
	if exists {
		return nil, errors.New(errors.ErrConflict, "用户名已存在")
	}

	exists, err = s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "检查邮箱是否存在失败")
	}
	if exists {
		return nil, errors.New(errors.ErrConflict, "邮箱已被使用")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternalServer, "加密密码失败")
	}

	user := &models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Email:        email,
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "创建用户失败")
	}

	utils.Info("User registered successfully",
		zap.String("username", user.Username),
		zap.Int("user_id", user.ID))

	return &models.RegisterResponse{
		UserID: user.ID,
	}, nil
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// 1. 查询用户
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		utils.Warn("Login failed: user not found", zap.String("username", req.Username))
		return nil, errors.NewUnauthorizedError("用户名或密码错误")
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.Warn("Login failed: invalid password", zap.String("username", req.Username))
		return nil, errors.NewUnauthorizedError("用户名或密码错误")
	}

	// 3. 生成JWT Token
	expiryDuration, _ := time.ParseDuration(s.cfg.JWT.Expiry)
	expiresAt := time.Now().Add(expiryDuration)

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		utils.Error("Failed to generate JWT token", zap.Error(err))
		return nil, errors.NewInternalServerError("生成Token失败")
	}

	// 4. 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		utils.Warn("Failed to update last login time", zap.Error(err))
		// 不影响登录流程
	}

	utils.Info("User login successful", zap.String("username", user.Username), zap.Int("user_id", user.ID))

	return &models.LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: &models.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Status:   user.Status,
		},
	}, nil
}

// VerifyToken 验证JWT Token（已在middleware中使用）
func VerifyToken(tokenString string, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

