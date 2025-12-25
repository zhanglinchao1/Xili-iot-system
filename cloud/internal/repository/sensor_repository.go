package repository

import (
	"context"
	"time"

	"cloud-system/internal/models"
)

// SensorDeviceRepository 传感器设备数据访问接口
type SensorDeviceRepository interface {
	// Create 创建传感器设备
	Create(ctx context.Context, device *models.SensorDevice) error

	// GetByID 根据ID获取传感器设备
	GetByID(ctx context.Context, deviceID string) (*models.SensorDevice, error)

	// ListByCabinetID 获取储能柜的所有传感器设备
	ListByCabinetID(ctx context.Context, cabinetID string) ([]*models.SensorDevice, error)

	// Update 更新传感器设备
	Update(ctx context.Context, deviceID string, device *models.SensorDevice) error

	// Delete 删除传感器设备
	Delete(ctx context.Context, deviceID string) error

	// Exists 检查设备是否存在
	Exists(ctx context.Context, deviceID string) (bool, error)
}

// SensorDataRepository 传感器数据访问接口（TimescaleDB）
type SensorDataRepository interface {
	// Insert 插入单条传感器数据（MQTT实时数据）
	// cabinetID和sensorType需要从设备信息中获取
	Insert(ctx context.Context, data *models.SensorData, cabinetID, sensorType string) error
	
	// BatchInsert 批量插入传感器数据
	BatchInsert(ctx context.Context, data []models.SensorData) error

	// GetLatestByCabinetID 获取储能柜的所有传感器最新数据
	GetLatestByCabinetID(ctx context.Context, cabinetID string) ([]*models.LatestSensorData, error)

	// GetHistoricalData 获取历史数据（原始数据）
	GetHistoricalData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.SensorData, int64, error)

	// GetAggregatedData 获取聚合数据
	GetAggregatedData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.AggregatedData, error)

	// DeleteOldData 删除旧数据（数据清理）
	DeleteOldData(ctx context.Context, beforeTime time.Time) error
}
