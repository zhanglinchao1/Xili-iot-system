package repository

import (
	"context"

	"cloud-system/internal/models"
)

// AlertRepository 告警数据访问接口
type AlertRepository interface {
	// Create 创建告警
	Create(ctx context.Context, alert *models.Alert) error

	// CreateOrUpdate 创建或更新告警（用于同步）
	// 如果已存在相同cabinet_id、device_id、alert_type且未解决的告警，则更新；否则创建新告警
	CreateOrUpdate(ctx context.Context, alert *models.Alert) error

	// GetByID 根据ID获取告警
	GetByID(ctx context.Context, alertID string) (*models.Alert, error)

	// List 获取告警列表（支持过滤和分页）
	List(ctx context.Context, filter *models.AlertListFilter) ([]*models.Alert, int64, error)

	// Resolve 解决告警
	Resolve(ctx context.Context, alertID string, resolvedBy string) error

	// GetActiveByCabinet 获取储能柜的活跃告警
	GetActiveByCabinet(ctx context.Context, cabinetID string) ([]*models.Alert, error)
	// GetRecentByCabinet 获取储能柜最近的告警
	GetRecentByCabinet(ctx context.Context, cabinetID string, limit int) ([]*models.Alert, error)
}
