/*
 * 设备数据模型
 * 定义储能柜传感器设备的数据结构
 */
package models

import (
	"time"
)

// SensorType 传感器类型
type SensorType string

const (
	SensorCO2          SensorType = "co2"          // 二氧化碳传感器
	SensorCO           SensorType = "co"           // 一氧化碳传感器
	SensorSmoke        SensorType = "smoke"        // 烟雾传感器
	SensorLiquidLevel  SensorType = "liquid_level" // 液位传感器
	SensorConductivity SensorType = "conductivity" // 电导率传感器
	SensorTemperature  SensorType = "temperature"  // 温度传感器
	SensorFlow         SensorType = "flow"         // 流速传感器
)

// DeviceStatus 设备状态
type DeviceStatus string

const (
	DeviceStatusOnline   DeviceStatus = "online"   // 在线
	DeviceStatusOffline  DeviceStatus = "offline"  // 离线
	DeviceStatusDisabled DeviceStatus = "disabled" // 禁用
	DeviceStatusFault    DeviceStatus = "fault"    // 故障
)

// Device 设备信息
type Device struct {
	DeviceID     string       `json:"device_id" db:"device_id"`       // 设备唯一标识
	DeviceType   string       `json:"device_type" db:"device_type"`   // 设备类型
	SensorType   SensorType   `json:"sensor_type" db:"sensor_type"`   // 传感器类型
	CabinetID    string       `json:"cabinet_id" db:"cabinet_id"`     // 所属储能柜ID
	PublicKey    string       `json:"public_key" db:"public_key"`     // 设备公钥（用于ZKP）
	Commitment   string       `json:"commitment" db:"commitment"`     // 设备承诺值（用于ZKP）
	Status       DeviceStatus `json:"status" db:"status"`             // 设备状态
	Model        string       `json:"model" db:"model"`               // 设备型号
	Manufacturer string       `json:"manufacturer" db:"manufacturer"` // 制造商
	FirmwareVer  string       `json:"firmware_ver" db:"firmware_ver"` // 固件版本
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`     // 创建时间
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`     // 更新时间
	LastSeenAt   *time.Time   `json:"last_seen_at" db:"last_seen_at"` // 最后在线时间
}

// DeviceRegistration 设备注册请求
type DeviceRegistration struct {
	DeviceID     string     `json:"device_id" binding:"required"`
	DeviceType   string     `json:"device_type" binding:"required"`
	SensorType   SensorType `json:"sensor_type" binding:"required"`
	CabinetID    string     `json:"cabinet_id"`                    // 所属储能柜ID（可选）
	PublicKey    string     `json:"public_key" binding:"required"` // ZKP公钥
	Commitment   string     `json:"commitment" binding:"required"` // ZKP承诺
	Model        string     `json:"model"`
	Manufacturer string     `json:"manufacturer"`
	FirmwareVer  string     `json:"firmware_ver"`
}

// Heartbeat 心跳数据
type Heartbeat struct {
	DeviceID  string                 `json:"device_id" binding:"required"`
	Timestamp time.Time              `json:"timestamp"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
