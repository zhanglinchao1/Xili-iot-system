package repository

import (
	"context"
	"time"

	"cloud-system/internal/models"
)

// TrafficRepository 定义持久化流量数据的访问接口
type TrafficRepository interface {
	// ListLatestStats 返回每个储能柜最近一次的流量统计
	ListLatestStats(ctx context.Context) ([]*models.TrafficStat, error)

	// GetLatestStat 返回指定储能柜最近一次流量统计
	GetLatestStat(ctx context.Context, cabinetID string) (*models.TrafficStat, error)

	// GetHistory 返回指定储能柜在时间范围内的流量样本
	GetHistory(ctx context.Context, cabinetID string, start time.Time, limit int) ([]models.TrafficSample, error)
}
