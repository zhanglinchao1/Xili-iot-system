package models

import (
	"time"
)

// SensorDevice 传感器设备模型
// 注意: 数据库表 sensor_devices 使用 device_name 列
type SensorDevice struct {
	DeviceID      string     `json:"device_id" db:"device_id"`
	CabinetID     string     `json:"cabinet_id" db:"cabinet_id"`
	SensorType    string     `json:"sensor_type" db:"sensor_type"` // co2, co, smoke, liquid_level, conductivity, temperature, flow
	Name          string     `json:"name" db:"device_name"`        // 数据库列名为 device_name
	Unit          string     `json:"unit" db:"unit"`
	Status        string     `json:"status" db:"status"`           // active, inactive, error, online, offline, maintenance
	Location      *string    `json:"location,omitempty" db:"location"`
	MinValue      *float64   `json:"min_value,omitempty" db:"min_value"`
	MaxValue      *float64   `json:"max_value,omitempty" db:"max_value"`
	LastValue     *float64   `json:"last_value,omitempty" db:"-"`
	LastReadingAt *time.Time `json:"last_reading_at,omitempty" db:"last_reading_at"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	// 兼容旧代码的 Description 字段（已废弃，数据库中不存在此列）
	Description   *string    `json:"description,omitempty" db:"-"`
}

// SensorData 传感器数据模型（TimescaleDB）
type SensorData struct {
	DeviceID  string    `json:"device_id" db:"device_id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Value     float64   `json:"value" db:"value"`
	Quality   int       `json:"quality" db:"quality"` // 0-100，数据质量指标
	Status    string    `json:"status" db:"status"`   // normal, warning, error
}

// SyncDataRequest 数据同步请求（Edge端同步数据格式）
type SyncDataRequest struct {
	CabinetID  string             `json:"cabinet_id" binding:"required"`
	Timestamp  time.Time          `json:"timestamp" binding:"required"`
	SensorData []SensorDataPoint  `json:"sensor_data" binding:"required"`
	Alerts     []AlertDataPoint   `json:"alerts,omitempty"`
	Statistics *DataStatistics    `json:"statistics,omitempty"`
}

// SensorDataPoint 单个传感器数据点（与Edge端格式匹配）
type SensorDataPoint struct {
	ID         int64     `json:"id,omitempty"`          // Edge端的数据库ID
	DeviceID   string    `json:"device_id" binding:"required"`
	SensorType string    `json:"sensor_type" binding:"required"`
	Value      float64   `json:"value" binding:"required"`
	Unit       string    `json:"unit" binding:"required"`
	Timestamp  time.Time `json:"timestamp" binding:"required"`
	Quality    int       `json:"quality" binding:"required,min=0,max=100"`
	Synced     bool      `json:"synced,omitempty"`      // Edge端同步标记
	SyncedAt   *time.Time `json:"synced_at,omitempty"`  // Edge端同步时间
}

// AlertDataPoint 告警数据点（与Edge端格式匹配）
type AlertDataPoint struct {
	ID         int64      `json:"id,omitempty"`
	DeviceID   string     `json:"device_id" binding:"required"`
	AlertType  string     `json:"alert_type" binding:"required"`
	Severity   string     `json:"severity" binding:"required"`
	Message    string     `json:"message" binding:"required"`
	Value      float64    `json:"value"`
	Threshold  float64    `json:"threshold"`
	Timestamp  time.Time  `json:"timestamp" binding:"required"`
	Resolved   bool       `json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	SyncedAt   *time.Time `json:"synced_at,omitempty"`
}

// DataStatistics 数据统计（与Edge端格式匹配）
type DataStatistics struct {
	DeviceID   string    `json:"device_id"`
	SensorType string    `json:"sensor_type"`
	Count      int64     `json:"count"`
	MinValue   float64   `json:"min_value"`
	MaxValue   float64   `json:"max_value"`
	AvgValue   float64   `json:"avg_value"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// LatestSensorData 最新传感器数据（包含设备信息）
type LatestSensorData struct {
	DeviceID   string    `json:"device_id"`
	SensorType string    `json:"sensor_type"`
	Name       string    `json:"name"`
	Unit       string    `json:"unit"`
	Value      float64   `json:"value"`
	Quality    int       `json:"quality"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

// HistoricalDataQuery 历史数据查询参数
type HistoricalDataQuery struct {
	DeviceID    string    `form:"device_id" binding:"required"`
	StartTime   time.Time `form:"start_time" binding:"required"`
	EndTime     time.Time `form:"end_time" binding:"required"`
	Aggregation string    `form:"aggregation"` // raw, 1m, 5m, 1h, 1d
	Page        int       `form:"page"`
	PageSize    int       `form:"page_size"`
}

// AggregatedData 聚合数据
type AggregatedData struct {
	Timestamp time.Time `json:"timestamp"`
	AvgValue  float64   `json:"avg_value"`
	MinValue  float64   `json:"min_value"`
	MaxValue  float64   `json:"max_value"`
	Count     int       `json:"count"`
}

// ValidSensorTypes 有效的传感器类型列表
var ValidSensorTypes = []string{
	"co2",
	"co",
	"smoke",
	"liquid_level",
	"conductivity",
	"temperature",
	"flow",
}

// ValidSensorStatuses 有效的传感器状态列表
var ValidSensorStatuses = []string{"normal", "warning", "error"}

// ValidDeviceStatuses 有效的设备状态列表
var ValidDeviceStatuses = []string{"active", "inactive", "error"}

// ValidAggregations 有效的聚合方式列表
var ValidAggregations = []string{"raw", "1m", "5m", "1h", "1d"}

// IsValidSensorType 检查传感器类型是否有效
func IsValidSensorType(sensorType string) bool {
	for _, valid := range ValidSensorTypes {
		if sensorType == valid {
			return true
		}
	}
	return false
}

// IsValidSensorStatus 检查传感器状态是否有效
func IsValidSensorStatus(status string) bool {
	for _, valid := range ValidSensorStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// IsValidDeviceStatus 检查设备状态是否有效
func IsValidDeviceStatus(status string) bool {
	for _, valid := range ValidDeviceStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// IsValidAggregation 检查聚合方式是否有效
func IsValidAggregation(aggregation string) bool {
	if aggregation == "" {
		return true // 空值表示raw
	}
	for _, valid := range ValidAggregations {
		if aggregation == valid {
			return true
		}
	}
	return false
}
