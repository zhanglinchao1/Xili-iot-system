package models

import (
	"time"
)

// Cabinet 储能柜模型
type Cabinet struct {
	CabinetID         string     `json:"cabinet_id" db:"cabinet_id"`
	Name              string     `json:"name" db:"name"`
	Location          *string    `json:"location,omitempty" db:"location"`
	Latitude          *float64   `json:"latitude,omitempty" db:"latitude"`   // 纬度坐标
	Longitude         *float64   `json:"longitude,omitempty" db:"longitude"` // 经度坐标
	CapacityKwh       *float64   `json:"capacity_kwh,omitempty" db:"capacity_kwh"`
	MACAddress                 string     `json:"mac_address" db:"mac_address"`
	Status                     string     `json:"status" db:"status"` // pending, active, inactive, offline, maintenance
	LatestVulnerabilityScore   float64    `json:"latest_vulnerability_score" db:"latest_vulnerability_score"`
	LatestRiskLevel            string     `json:"latest_risk_level" db:"latest_risk_level"`
	VulnerabilityUpdatedAt     *time.Time `json:"vulnerability_updated_at,omitempty" db:"vulnerability_updated_at"`
	LastSyncAt                 *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	ActivationStatus  string     `json:"activation_status" db:"activation_status"` // pending, activated
	RegistrationToken *string    `json:"-" db:"registration_token"`                // 不返回给前端
	TokenExpiresAt    *time.Time `json:"token_expires_at,omitempty" db:"token_expires_at"`
	APIKey            *string    `json:"-" db:"api_key"`         // 不返回给前端
	APISecretHash     *string    `json:"-" db:"api_secret_hash"` // 不返回给前端
	ActivatedAt       *time.Time `json:"activated_at,omitempty" db:"activated_at"`
	IPAddress         *string    `json:"ip_address,omitempty" db:"ip_address"`
	DeviceModel       *string    `json:"device_model,omitempty" db:"device_model"`
	Notes             *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateCabinetInput 创建储能柜输入参数
type CreateCabinetInput struct {
	CabinetID   string   `json:"cabinet_id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Location    *string  `json:"location,omitempty"`
	CapacityKwh *float64 `json:"capacity_kwh,omitempty"`
	MACAddress  string   `json:"mac_address" binding:"required"`
}

// UpdateCabinetInput 更新储能柜输入参数
type UpdateCabinetInput struct {
	Name        *string  `json:"name,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`  // 纬度坐标
	Longitude   *float64 `json:"longitude,omitempty"` // 经度坐标
	CapacityKwh *float64 `json:"capacity_kwh,omitempty"`
	Status      *string  `json:"status,omitempty"`
}

// CabinetListFilter 储能柜列表过滤参数
type CabinetListFilter struct {
	Status   *string `form:"status"`
	Location *string `form:"location"`
	Page     int     `form:"page"`
	PageSize int     `form:"page_size"`
}

// PreRegisterCabinetInput 预注册储能柜输入参数
type PreRegisterCabinetInput struct {
	CabinetID        string   `json:"cabinet_id" binding:"required"`
	Name             string   `json:"name" binding:"required"`
	Location         *string  `json:"location,omitempty"`
	CapacityKwh      *float64 `json:"capacity_kwh,omitempty"`
	MACAddress       string   `json:"mac_address" binding:"required"`
	LicenseExpiresAt *string  `json:"license_expires_at,omitempty"` // ISO8601格式
	Permissions      []string `json:"permissions,omitempty"`
	IPAddress        *string  `json:"ip_address,omitempty"`
	DeviceModel      *string  `json:"device_model,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}

// PreRegisterResponse 预注册响应
type PreRegisterResponse struct {
	CabinetID         string    `json:"cabinet_id"`
	RegistrationToken string    `json:"registration_token"`
	TokenExpiresAt    time.Time `json:"token_expires_at"`
}

// ActivationInfoResponse 激活信息响应
type ActivationInfoResponse struct {
	CabinetID         string    `json:"cabinet_id"`
	Name              string    `json:"name"`
	MACAddress        string    `json:"mac_address"`
	RegistrationToken string    `json:"registration_token"`
	TokenExpiresAt    time.Time `json:"token_expires_at"`
	TokenExpired      bool      `json:"token_expired"`
	CloudAPIURL       string    `json:"cloud_api_url"`
}

// ActivateCabinetInput Edge端激活输入
type ActivateCabinetInput struct {
	RegistrationToken string `json:"registration_token" binding:"required"`
	MACAddress        string `json:"mac_address" binding:"required"`
}

// ActivateCabinetResponse Edge端激活响应
type ActivateCabinetResponse struct {
	CabinetID string `json:"cabinet_id"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// RegenerateTokenResponse 重新生成Token响应
type RegenerateTokenResponse struct {
	RegistrationToken string    `json:"registration_token"`
	TokenExpiresAt    time.Time `json:"token_expires_at"`
}

// RegisterCabinetInput Edge端直接注册输入（一步完成注册和激活）
// 只需提供cabinet_id即可注册，其他字段可选，可在注册后使用API Key更新
type RegisterCabinetInput struct {
	CabinetID   string   `json:"cabinet_id" binding:"required"`
	Name        *string  `json:"name,omitempty"`        // 可选，为空时使用cabinet_id作为默认值
	Location    *string  `json:"location,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`     // 纬度坐标
	Longitude   *float64 `json:"longitude,omitempty"`   // 经度坐标
	CapacityKwh *float64 `json:"capacity_kwh,omitempty"`
	DeviceModel *string  `json:"device_model,omitempty"`
	IPAddress   *string  `json:"ip_address,omitempty"`
	MACAddress  *string  `json:"mac_address,omitempty"` // 可选，可在注册后更新
}

// RegisterCabinetResponse Edge端直接注册响应
type RegisterCabinetResponse struct {
	CabinetID string `json:"cabinet_id"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// CabinetLocation 储能柜位置信息（用于地图展示）
type CabinetLocation struct {
	CabinetID string   `json:"cabinet_id"`
	Name      string   `json:"name"`
	Location  *string  `json:"location,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Status    string   `json:"status"` // 用于地图标记颜色
}

// CabinetStatistics 储能柜统计信息
type CabinetStatistics struct {
	TotalCabinets       int64 `json:"total_cabinets"`       // 储能柜总数
	ActiveCabinets      int64 `json:"active_cabinets"`      // 激活且同步中的设备数
	OfflineCabinets     int64 `json:"offline_cabinets"`     // 离线设备数
	InactiveCabinets    int64 `json:"inactive_cabinets"`    // 已停用储能柜数
	PendingCabinets     int64 `json:"pending_cabinets"`     // 待激活储能柜数
	MaintenanceCabinets int64 `json:"maintenance_cabinets"` // 维护中设备数
	ActivatedCabinets   int64 `json:"activated_cabinets"`   // 已激活储能柜数（activation_status='activated'）
}

// ValidStatuses 有效的储能柜状态列表
// 已废弃：请使用 ValidCabinetStatuses 和 IsValidCabinetStatus (定义在 cabinet_status.go)
// 保留此变量仅为向后兼容
var ValidStatuses = ValidCabinetStatuses

// IsValidStatus 检查状态是否有效
// 已废弃：请使用 IsValidCabinetStatus (定义在 cabinet_status.go)
// 保留此函数仅为向后兼容
func IsValidStatus(status string) bool {
	return IsValidCabinetStatus(status)
}
