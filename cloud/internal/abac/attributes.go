package abac

import (
	"time"
)

// UserAttributes 用户主体属性
type UserAttributes struct {
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`          // admin, user
	Status      string    `json:"status"`        // active, disabled
	LastLoginIP string    `json:"last_login_ip"` // 最后登录IP
	LastLoginAt time.Time `json:"last_login_at"` // 最后登录时间
	TrustScore  float64   `json:"trust_score"`   // 信任分数 (0-100)
}

// CabinetAttributes 储能柜主体属性
type CabinetAttributes struct {
	CabinetID           string    `json:"cabinet_id"`
	MACAddress          string    `json:"mac_address"`
	IPAddress           string    `json:"ip_address"`
	Status              string    `json:"status"`              // online, offline, maintenance, error
	ActivationStatus    string    `json:"activation_status"`   // activated, pending
	VulnerabilityScore  float64   `json:"vulnerability_score"` // 脆弱性评分 (0-100, 由Edge端评估)
	RiskLevel           string    `json:"risk_level"`          // low, medium, high, critical
	LastSyncAt          time.Time `json:"last_sync_at"`        // 最后同步时间
	HasValidLicense     bool      `json:"has_valid_license"`   // 是否有效许可证
	TrustScore          float64   `json:"trust_score"`         // 信任分数 (0-100)
}

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
	SubjectTypeUser    SubjectType = "user"
	SubjectTypeCabinet SubjectType = "cabinet"
	SubjectTypeDevice  SubjectType = "device"
)

// Attributes 通用属性接口
type Attributes interface {
	GetType() SubjectType
	GetID() string
	GetTrustScore() float64
}

// GetType 实现Attributes接口 - UserAttributes
func (u *UserAttributes) GetType() SubjectType {
	return SubjectTypeUser
}

// GetID 实现Attributes接口 - UserAttributes
func (u *UserAttributes) GetID() string {
	return u.Username
}

// GetTrustScore 实现Attributes接口 - UserAttributes
func (u *UserAttributes) GetTrustScore() float64 {
	return u.TrustScore
}

// GetType 实现Attributes接口 - CabinetAttributes
func (c *CabinetAttributes) GetType() SubjectType {
	return SubjectTypeCabinet
}

// GetID 实现Attributes接口 - CabinetAttributes
func (c *CabinetAttributes) GetID() string {
	return c.CabinetID
}

// GetTrustScore 实现Attributes接口 - CabinetAttributes
func (c *CabinetAttributes) GetTrustScore() float64 {
	return c.TrustScore
}

// GetType 实现Attributes接口 - DeviceAttributes
func (d *DeviceAttributes) GetType() SubjectType {
	return SubjectTypeDevice
}

// GetID 实现Attributes接口 - DeviceAttributes
func (d *DeviceAttributes) GetID() string {
	return d.DeviceID
}

// GetTrustScore 实现Attributes接口 - DeviceAttributes
func (d *DeviceAttributes) GetTrustScore() float64 {
	return d.TrustScore
}
