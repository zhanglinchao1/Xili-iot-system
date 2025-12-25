/*
 * MQTT 消息类型定义
 * 与网关端数据结构完全一致
 */
package mqtt

import (
	"time"
)

// SensorData 传感器数据（与网关端一致）
type SensorData struct {
	DeviceID   string    `json:"device_id"`
	SensorType string    `json:"sensor_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Quality    int       `json:"quality"` // 数据质量 0-100
	Timestamp  time.Time `json:"timestamp"`
}

// DeviceStatus 设备状态
type DeviceStatus struct {
	DeviceID  string    `json:"device_id"`
	Status    string    `json:"status"` // online, offline, error
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"` // 可选的状态信息
}

// Alert 告警信息
type Alert struct {
	DeviceID  string    `json:"device_id"`
	AlertType string    `json:"alert_type"`
	Severity  string    `json:"severity"` // low, medium, high, critical
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// Heartbeat 心跳信息
type Heartbeat struct {
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Topic 前缀常量
const (
	TopicSensors       = "sensors/#"           // 所有传感器数据
	TopicDevices       = "devices/+/status"    // 所有设备状态
	TopicAlerts        = "alerts/#"            // 所有告警
	TopicHeartbeat     = "devices/+/heartbeat" // 所有心跳
	TopicTraffic       = "traffic/#"           // 所有流量统计
	TopicTrafficPrefix = "traffic/"            // 流量统计发布前缀
)

// TrafficStat 流量统计信息
type TrafficStat struct {
	CabinetID         string    `json:"cabinet_id"`
	Timestamp         time.Time `json:"timestamp"`
	ThroughputKbps    float64   `json:"throughput_kbps"`
	LatencyMs         float64   `json:"latency_ms"`
	PacketLossRate    float64   `json:"packet_loss_rate"`
	MQTTSuccessRate   float64   `json:"mqtt_success_rate"`
	ReconnectionCount int       `json:"reconnection_count"`
	RiskLevel         string    `json:"risk_level"`
}
