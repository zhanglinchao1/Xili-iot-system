package repository

import (
	"context"

	"cloud-system/internal/models"
)

// CommandRepository 命令数据访问接口
type CommandRepository interface {
	// Create 创建命令
	Create(ctx context.Context, command *models.Command) error

	// GetByID 根据ID获取命令
	GetByID(ctx context.Context, commandID string) (*models.Command, error)

	// List 获取命令列表（支持过滤和分页）
	List(ctx context.Context, filter *models.CommandListFilter) ([]*models.Command, int64, error)

	// UpdateStatus 更新命令状态
	UpdateStatus(ctx context.Context, commandID string, status string, result *string) error

	// MarkAsSent 标记命令为已发送
	MarkAsSent(ctx context.Context, commandID string) error

	// MarkAsCompleted 标记命令为已完成
	MarkAsCompleted(ctx context.Context, commandID string, status string, result string) error
}
