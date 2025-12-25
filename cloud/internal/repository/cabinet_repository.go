package repository

import (
	"context"

	"cloud-system/internal/models"
)

// CabinetRepository 储能柜数据访问接口
type CabinetRepository interface {
	// Create 创建储能柜
	Create(ctx context.Context, cabinet *models.Cabinet) error

	// GetByID 根据ID获取储能柜
	GetByID(ctx context.Context, cabinetID string) (*models.Cabinet, error)

	// GetByMACAddress 根据MAC地址获取储能柜
	GetByMACAddress(ctx context.Context, macAddress string) (*models.Cabinet, error)

	// List 获取储能柜列表（支持过滤和分页）
	List(ctx context.Context, filter *models.CabinetListFilter) ([]*models.Cabinet, int64, error)

	// Update 更新储能柜信息
	Update(ctx context.Context, cabinetID string, input *models.UpdateCabinetInput) error

	// Delete 删除储能柜
	Delete(ctx context.Context, cabinetID string) error

	// UpdateVulnerabilityScore 更新脆弱性评分缓存
	UpdateVulnerabilityScore(ctx context.Context, cabinetID string, score float64, riskLevel string) error

	// UpdateLastSyncTime 更新最后同步时间
	UpdateLastSyncTime(ctx context.Context, cabinetID string) error

	// Exists 检查储能柜是否存在
	Exists(ctx context.Context, cabinetID string) (bool, error)

	// GetByRegistrationToken 根据注册Token获取储能柜
	GetByRegistrationToken(ctx context.Context, token string) (*models.Cabinet, error)

	// GetByAPIKey 根据API Key获取储能柜
	GetByAPIKey(ctx context.Context, apiKey string) (*models.Cabinet, error)

	// UpdateActivation 更新激活信息
	UpdateActivation(ctx context.Context, cabinetID string, apiKey, apiSecretHash string) error

	// UpdateRegistrationToken 更新注册Token
	UpdateRegistrationToken(ctx context.Context, cabinetID, token string, expiresAt interface{}) error

	// GetLocations 获取所有储能柜位置信息（用于地图展示）
	GetLocations(ctx context.Context) ([]*models.CabinetLocation, error)

	// GetStatistics 获取储能柜统计信息
	GetStatistics(ctx context.Context) (*models.CabinetStatistics, error)
}
