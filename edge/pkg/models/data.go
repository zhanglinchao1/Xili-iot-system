/*
 * 传感器数据模型
 * 定义传感器数据采集和存储的数据结构
 */
package models

import (
	"time"
)

// SensorData 传感器数据
type SensorData struct {
	ID         int64      `json:"id" db:"id"`
	DeviceID   string     `json:"device_id" db:"device_id"`
	SensorType SensorType `json:"sensor_type" db:"sensor_type"`
	Value      float64    `json:"value" db:"value"`
	Unit       string     `json:"unit" db:"unit"`
	Timestamp  time.Time  `json:"timestamp" db:"timestamp"`
	Quality    int        `json:"quality" db:"quality"`       // 数据质量 0-100
	Synced     bool       `json:"synced" db:"synced"`         // 是否已同步到云端
	SyncedAt   *time.Time `json:"synced_at" db:"synced_at"`   // 同步时间
}

// DataCollectRequest 数据采集请求
type DataCollectRequest struct {
	DeviceID   string     `json:"device_id" binding:"required"`
	SensorType SensorType `json:"sensor_type" binding:"required"`
	Value      float64    `json:"value" binding:"required"`
	Unit       string     `json:"unit" binding:"required"`
	Timestamp  time.Time  `json:"timestamp"`
	Quality    int        `json:"quality"`
}

// BatchDataRequest 批量数据请求
type BatchDataRequest struct {
	CabinetID string               `json:"cabinet_id" binding:"required"`
	Data      []DataCollectRequest `json:"data" binding:"required"`
}

// Alert 告警信息
type Alert struct {
	ID         int64      `json:"id" db:"id"`
	DeviceID   string     `json:"device_id" db:"device_id"`
	AlertType  string     `json:"alert_type" db:"alert_type"`   // 告警类型
	Severity   string     `json:"severity" db:"severity"`       // 严重程度: low, medium, high, critical
	Message    string     `json:"message" db:"message"`
	Value      *float64   `json:"value,omitempty" db:"value"`             // 触发告警的数值（可选）
	Threshold  *float64   `json:"threshold,omitempty" db:"threshold"`     // 阈值（可选）
	Timestamp  time.Time  `json:"timestamp" db:"timestamp"`
	Resolved   bool       `json:"resolved" db:"resolved"`       // 是否已解决
	ResolvedAt *time.Time `json:"resolved_at" db:"resolved_at"` // 解决时间
	SyncedAt   *time.Time `json:"synced_at" db:"synced_at"`     // 同步到云端时间
}

// AlertType 告警类型
type AlertType string

const (
	AlertCO2High         AlertType = "co2_high"          // CO2浓度过高
	AlertCOHigh          AlertType = "co_high"           // CO浓度过高
	AlertSmokeDetected   AlertType = "smoke_detected"    // 检测到烟雾
	AlertLiquidLevelLow  AlertType = "liquid_level_low"  // 液位过低
	AlertLiquidLevelHigh AlertType = "liquid_level_high" // 液位过高
	AlertConductivityAbnormal AlertType = "conductivity_abnormal" // 电导率异常
	AlertTemperatureHigh AlertType = "temperature_high"  // 温度过高
	AlertTemperatureLow  AlertType = "temperature_low"   // 温度过低
	AlertFlowAbnormal    AlertType = "flow_abnormal"     // 流速异常
	AlertDeviceOffline   AlertType = "device_offline"    // 设备离线
	AlertAuthFailed      AlertType = "auth_failed"       // 认证失败
	AlertDataAbnormal    AlertType = "data_abnormal"     // 数据异常
)

// Severity 严重程度
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// SensorUnit 传感器单位映射
var SensorUnit = map[SensorType]string{
	SensorCO2:         "ppm",     // 二氧化碳浓度
	SensorCO:          "ppm",     // 一氧化碳浓度
	SensorSmoke:       "ppm",     // 烟雾浓度
	SensorLiquidLevel: "mm",      // 液位高度
	SensorConductivity: "mS/cm",   // 电导率
	SensorTemperature: "°C",      // 温度
	SensorFlow:        "L/min",   // 流速
}

// SensorThreshold 传感器阈值
type SensorThreshold struct {
	SensorType SensorType `json:"sensor_type"`
	MinValue   float64    `json:"min_value"`
	MaxValue   float64    `json:"max_value"`
	Unit       string     `json:"unit"`
}

// DataStatistics 数据统计
type DataStatistics struct {
	DeviceID   string     `json:"device_id"`
	SensorType SensorType `json:"sensor_type"`
	Count      int64      `json:"count"`
	MinValue   float64    `json:"min_value"`
	MaxValue   float64    `json:"max_value"`
	AvgValue   float64    `json:"avg_value"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    time.Time  `json:"end_time"`
}

// CloudSyncPayload 云端同步数据负载
type CloudSyncPayload struct {
	CabinetID  string        `json:"cabinet_id"`
	Timestamp  time.Time     `json:"timestamp"`
	SensorData []SensorData  `json:"sensor_data"`
	Alerts     []Alert       `json:"alerts,omitempty"`
	Statistics *DataStatistics `json:"statistics,omitempty"`
}
