// Package services 实现用户管理业务逻辑
package services

import (
	"context"
	"net/mail"
	"strings"

	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户管理服务接口
type UserService interface {
	// GetProfile 获取个人信息
	GetProfile(ctx context.Context, userID int) (*models.UserInfo, error)

	// UpdateProfile 更新个人信息（普通用户）
	UpdateProfile(ctx context.Context, userID int, req *models.UpdateProfileRequest) error

	// UpdatePassword 修改密码
	UpdatePassword(ctx context.Context, userID int, req *models.UpdatePasswordRequest) error

	// ListUsers 获取用户列表（管理员）
	ListUsers(ctx context.Context, filter *models.UserListFilter) ([]*models.UserInfo, int64, error)

	// GetUser 获取用户详情（管理员）
	GetUser(ctx context.Context, userID int) (*models.UserInfo, error)

	// CreateUser 创建用户（管理员）
	CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserInfo, error)

	// UpdateUser 更新用户（管理员）
	UpdateUser(ctx context.Context, userID int, req *models.UpdateUserRequest) error

	// DeleteUser 删除用户（管理员）
	DeleteUser(ctx context.Context, userID int) error

	// ResetUserPassword 重置用户密码（管理员）
	ResetUserPassword(ctx context.Context, userID int, newPassword string) error
}

// userService 实现UserService接口
type userService struct {
	userRepo repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// GetProfile 获取个人信息
func (s *userService) GetProfile(ctx context.Context, userID int) (*models.UserInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrNotFound, "获取用户信息失败")
	}

	return &models.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
	}, nil
}

// UpdateProfile 更新个人信息（普通用户）
func (s *userService) UpdateProfile(ctx context.Context, userID int, req *models.UpdateProfileRequest) error {
	updates := make(map[string]interface{})

	if req.Email != "" {
		email := strings.TrimSpace(strings.ToLower(req.Email))
		if _, err := mail.ParseAddress(email); err != nil {
			return errors.NewValidationError("邮箱格式不正确")
		}

		// 检查邮箱是否已被其他用户使用
		exists, err := s.userRepo.ExistsByEmail(ctx, email)
		if err != nil {
			return errors.Wrap(err, errors.ErrDatabaseQuery, "检查邮箱是否存在失败")
		}
		if exists {
			// 需要确认该邮箱是否属于当前用户
			user, err := s.userRepo.GetByID(ctx, userID)
			if err != nil {
				return errors.Wrap(err, errors.ErrNotFound, "获取用户信息失败")
			}
			if strings.ToLower(user.Email) != email {
				return errors.New(errors.ErrConflict, "邮箱已被其他用户使用")
			}
		}

		updates["email"] = email
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新用户信息失败")
	}

	utils.Info("User profile updated", zap.Int("user_id", userID))
	return nil
}

// UpdatePassword 修改密码
func (s *userService) UpdatePassword(ctx context.Context, userID int, req *models.UpdatePasswordRequest) error {
	// 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, errors.ErrNotFound, "获取用户信息失败")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errors.NewUnauthorizedError("原密码错误")
	}

	// 验证新密码强度
	if !passwordPattern.MatchString(req.NewPassword) {
		return errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}
	hasLower, hasUpper, hasDigit := false, false, false
	for _, c := range req.NewPassword {
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
		return errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}

	// 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, errors.ErrInternalServer, "加密密码失败")
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新密码失败")
	}

	utils.Info("User password updated", zap.Int("user_id", userID))
	return nil
}

// ListUsers 获取用户列表（管理员）
func (s *userService) ListUsers(ctx context.Context, filter *models.UserListFilter) ([]*models.UserInfo, int64, error) {
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "获取用户列表失败")
	}

	userInfos := make([]*models.UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = &models.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Status:   user.Status,
		}
	}

	return userInfos, total, nil
}

// GetUser 获取用户详情（管理员）
func (s *userService) GetUser(ctx context.Context, userID int) (*models.UserInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrNotFound, "获取用户信息失败")
	}

	return &models.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
	}, nil
}

// CreateUser 创建用户（管理员）
func (s *userService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserInfo, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	// 验证输入
	if !usernamePattern.MatchString(username) {
		return nil, errors.NewValidationError("用户名需为3-64位，仅支持字母、数字、下划线和连字符")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, errors.NewValidationError("邮箱格式不正确")
	}

	if !passwordPattern.MatchString(req.Password) {
		return nil, errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}
	hasLower, hasUpper, hasDigit := false, false, false
	for _, c := range req.Password {
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

	// 检查用户名是否存在
	exists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "检查用户名是否存在失败")
	}
	if exists {
		return nil, errors.New(errors.ErrConflict, "用户名已存在")
	}

	// 检查邮箱是否存在
	exists, err = s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "检查邮箱是否存在失败")
	}
	if exists {
		return nil, errors.New(errors.ErrConflict, "邮箱已被使用")
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternalServer, "加密密码失败")
	}

	// 创建用户
	user := &models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Email:        email,
		Role:         req.Role,
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "创建用户失败")
	}

	utils.Info("User created by admin", zap.String("username", user.Username), zap.Int("user_id", user.ID))

	return &models.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
	}, nil
}

// UpdateUser 更新用户（管理员）
func (s *userService) UpdateUser(ctx context.Context, userID int, req *models.UpdateUserRequest) error {
	updates := make(map[string]interface{})

	if req.Email != "" {
		email := strings.TrimSpace(strings.ToLower(req.Email))
		if _, err := mail.ParseAddress(email); err != nil {
			return errors.NewValidationError("邮箱格式不正确")
		}

		// 检查邮箱是否已被其他用户使用
		exists, err := s.userRepo.ExistsByEmail(ctx, email)
		if err != nil {
			return errors.Wrap(err, errors.ErrDatabaseQuery, "检查邮箱是否存在失败")
		}
		if exists {
			user, err := s.userRepo.GetByID(ctx, userID)
			if err != nil {
				return errors.Wrap(err, errors.ErrNotFound, "获取用户信息失败")
			}
			if strings.ToLower(user.Email) != email {
				return errors.New(errors.ErrConflict, "邮箱已被其他用户使用")
			}
		}

		updates["email"] = email
	}

	if req.Role != "" {
		updates["role"] = req.Role
	}

	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新用户信息失败")
	}

	utils.Info("User updated by admin", zap.Int("user_id", userID))
	return nil
}

// DeleteUser 删除用户（管理员）
func (s *userService) DeleteUser(ctx context.Context, userID int) error {
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "删除用户失败")
	}

	utils.Info("User deleted by admin", zap.Int("user_id", userID))
	return nil
}

// ResetUserPassword 重置用户密码（管理员）
func (s *userService) ResetUserPassword(ctx context.Context, userID int, newPassword string) error {
	// 验证新密码强度
	if !passwordPattern.MatchString(newPassword) {
		return errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}
	hasLower, hasUpper, hasDigit := false, false, false
	for _, c := range newPassword {
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
		return errors.NewValidationError("密码至少8位，且需包含大写字母、小写字母与数字")
	}

	// 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, errors.ErrInternalServer, "加密密码失败")
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "重置密码失败")
	}

	utils.Info("User password reset by admin", zap.Int("user_id", userID))
	return nil
}
