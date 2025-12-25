// Package models 定义系统用户数据模型
package models

import "time"

// User 系统用户
type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"` // 不序列化到JSON
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=64"`
	Password string `json:"password" validate:"required,min=4,max=128"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *UserInfo `json:"user"`
}

// UserInfo 用户信息（不包含敏感数据）
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// RegisterResponse 用户注册响应
type RegisterResponse struct {
	UserID int `json:"user_id"`
}

// UpdateProfileRequest 更新个人信息请求（普通用户）
type UpdateProfileRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
}

// UpdatePasswordRequest 修改密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=4,max=128"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// CreateUserRequest 创建用户请求（管理员）
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Role     string `json:"role" binding:"required,oneof=user admin"`
}

// UpdateUserRequest 更新用户请求（管理员）
type UpdateUserRequest struct {
	Email  string `json:"email" binding:"omitempty,email"`
	Role   string `json:"role" binding:"omitempty,oneof=user admin"`
	Status string `json:"status" binding:"omitempty,oneof=active disabled"`
}

// ResetPasswordRequest 重置用户密码请求（管理员）
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// UserListFilter 用户列表过滤参数
type UserListFilter struct {
	Role   *string `json:"role,omitempty"`
	Status *string `json:"status,omitempty"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}
