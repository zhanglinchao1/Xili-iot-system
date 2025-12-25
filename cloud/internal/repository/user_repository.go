// Package repository 定义用户数据访问接口
package repository

import (
	"context"

	"cloud-system/internal/models"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	// GetByUsername 根据用户名查询用户
	GetByUsername(ctx context.Context, username string) (*models.User, error)

	// GetByID 根据ID查询用户
	GetByID(ctx context.Context, userID int) (*models.User, error)

	// List 获取用户列表（支持过滤和分页）
	List(ctx context.Context, filter *models.UserListFilter) ([]*models.User, int64, error)

	// ExistsByUsername 判断用户名是否存在
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail 判断邮箱是否存在
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// UpdateLastLogin 更新最后登录时间
	UpdateLastLogin(ctx context.Context, userID int) error

	// Create 创建用户
	Create(ctx context.Context, user *models.User) error

	// Update 更新用户信息
	Update(ctx context.Context, userID int, updates map[string]interface{}) error

	// UpdatePassword 更新用户密码
	UpdatePassword(ctx context.Context, userID int, newPasswordHash string) error

	// Delete 删除用户（软删除，设置status为inactive）
	Delete(ctx context.Context, userID int) error
}
