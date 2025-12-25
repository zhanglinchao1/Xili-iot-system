package models

import (
	"time"
)

// License 许可证模型
type License struct {
	LicenseID    string     `json:"license_id" db:"license_id"`
	CabinetID    string     `json:"cabinet_id" db:"cabinet_id"`
	MACAddress   string     `json:"mac_address" db:"mac_address"`
	IssuedAt     time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	Permissions  string     `json:"permissions" db:"permissions"`
	Status       string     `json:"status" db:"status"` // active, expired, revoked
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevokeReason *string    `json:"revoke_reason,omitempty" db:"revoke_reason"`
	MaxDevices   int        `json:"max_devices" db:"max_devices"`
	CreatedBy    string     `json:"created_by" db:"created_by"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateLicenseRequest 创建许可证请求
type CreateLicenseRequest struct {
	CabinetID   string   `json:"cabinet_id" binding:"required"`
	ValidDays   int      `json:"valid_days" binding:"required,min=1"`
	MaxDevices  int      `json:"max_devices" binding:"required,min=1"`
	Permissions []string `json:"permissions" binding:"required"`
}

// SyncLicensesRequest 同步历史许可证请求
type SyncLicensesRequest struct {
	CabinetIDs  []string `json:"cabinet_ids"`
	ValidDays   *int     `json:"valid_days"`
	MaxDevices  *int     `json:"max_devices"`
	Permissions []string `json:"permissions"`
}

// RenewLicenseRequest 续期许可证请求
type RenewLicenseRequest struct {
	ExtendDays int `json:"extend_days" binding:"required,min=1"`
}

// RevokeLicenseRequest 吊销许可证请求
type RevokeLicenseRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ValidateLicenseRequest 验证许可证请求（Edge端调用）
type ValidateLicenseRequest struct {
	CabinetID  string `json:"cabinet_id" binding:"required"`
	MACAddress string `json:"mac_address" binding:"required"`
}

// LicenseListFilter 许可证列表过滤参数
type LicenseListFilter struct {
	Status   *string `form:"status"`
	Page     int     `form:"page"`
	PageSize int     `form:"page_size"`
}

// ValidLicenseStatuses 有效的许可证状态列表
var ValidLicenseStatuses = []string{"active", "expired", "revoked"}

// IsValidLicenseStatus 检查许可证状态是否有效
func IsValidLicenseStatus(status string) bool {
	for _, valid := range ValidLicenseStatuses {
		if status == valid {
			return true
		}
	}
	return false
}
