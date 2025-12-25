package repository

import (
	"context"

	"cloud-system/internal/models"
)

// LicenseRepository 许可证数据访问接口
type LicenseRepository interface {
	// Create 创建许可证
	Create(ctx context.Context, license *models.License) error

	// GetByCabinetID 根据储能柜ID获取许可证
	GetByCabinetID(ctx context.Context, cabinetID string) (*models.License, error)

	// List 获取许可证列表（支持过滤和分页）
	List(ctx context.Context, filter *models.LicenseListFilter) ([]*models.License, int64, error)

	// Update 更新许可证
	Update(ctx context.Context, cabinetID string, license *models.License) error

	// Revoke 吊销许可证
	Revoke(ctx context.Context, cabinetID string, reason string, revokedBy string) error

	// Renew 续期许可证
	Renew(ctx context.Context, cabinetID string, extendDays int) error

	// Delete 删除许可证
	Delete(ctx context.Context, cabinetID string) error

	// Validate 验证许可证（检查MAC地址和有效期）
	Validate(ctx context.Context, cabinetID string, macAddress string) (*models.License, error)
}
