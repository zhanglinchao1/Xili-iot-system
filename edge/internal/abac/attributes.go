package abac

import (
	"time"
)

// DeviceAttributes 传感器设备主体属性
type DeviceAttributes struct {
	DeviceID      string    `json:"device_id"`
	CabinetID     string    `json:"cabinet_id"`
	SensorType    string    `json:"sensor_type"` // co2, co, smoke, etc.
	Status        string    `json:"status"`      // active, inactive, error
	Quality       int       `json:"quality"`     // 数据质量 (0-100)
	LastReadingAt time.Time `json:"last_reading_at"`
	TrustScore    float64   `json:"trust_score"` // 信任分数 (0-100)
}

// SubjectType 主体类型
type SubjectType string

const (
	SubjectTypeDevice SubjectType = "device"
)

// Attributes 通用属性接口
type Attributes interface {
	GetType() SubjectType
	GetID() string
	GetTrustScore() float64
}

// GetType 实现Attributes接口
func (d *DeviceAttributes) GetType() SubjectType {
	return SubjectTypeDevice
}

// GetID 实现Attributes接口
func (d *DeviceAttributes) GetID() string {
	return d.DeviceID
}

// GetTrustScore 实现Attributes接口
func (d *DeviceAttributes) GetTrustScore() float64 {
	return d.TrustScore
}
