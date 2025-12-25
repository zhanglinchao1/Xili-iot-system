package models

import (
	"time"
)

// Alert 告警模型
type Alert struct {
	AlertID      string                 `json:"alert_id" db:"alert_id"`
	CabinetID    string                 `json:"cabinet_id" db:"cabinet_id"`
	EdgeAlertID  *int64                 `json:"edge_alert_id,omitempty" db:"edge_alert_id"`
	AlertType    string                 `json:"alert_type" db:"alert_type"` // sensor_abnormal, device_offline, threshold_exceeded
	Severity     string                 `json:"severity" db:"severity"`     // info, warning, error, critical
	Message      string                 `json:"message" db:"message"`
	Details      map[string]interface{} `json:"details,omitempty" db:"details"` // 存储device_id, sensor_value等额外信息
	Resolved     bool                   `json:"resolved" db:"resolved"`         // 是否已解决
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy   *string                `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`

	// 计算字段，从details中提取
	DeviceID    *string  `json:"device_id,omitempty" db:"-"`
	SensorValue *float64 `json:"sensor_value,omitempty" db:"-"`
	Status      string   `json:"status" db:"-"` // 从resolved字段计算：active/resolved
	Location    *string  `json:"location,omitempty" db:"-"` // 从cabinets表关联获取
}

// AlertListFilter 告警列表过滤参数
type AlertListFilter struct {
	CabinetID *string `form:"cabinet_id"`
	Severity  *string `form:"severity"`
	Status    *string `form:"status"`
	Page      int     `form:"page"`
	PageSize  int     `form:"page_size"`
}

// ValidAlertTypes 有效的告警类型列表
var ValidAlertTypes = []string{"sensor_abnormal", "device_offline", "threshold_exceeded"}

// ValidAlertSeverities 有效的告警严重程度列表
var ValidAlertSeverities = []string{"info", "warning", "error", "critical"}

// ValidAlertStatuses 有效的告警状态列表
var ValidAlertStatuses = []string{"active", "resolved", "ignored"}

// PopulateCalculatedFields 从Details字段填充计算字段(DeviceID, SensorValue)
func (a *Alert) PopulateCalculatedFields() {
	if a.Details == nil {
		a.Details = make(map[string]interface{})
	}

	// 提取device_id
	if deviceID, ok := a.Details["device_id"].(string); ok && deviceID != "" {
		a.DeviceID = &deviceID
	}

	// 提取sensor_value (可能是float64或string)
	if val, ok := a.Details["sensor_value"]; ok {
		switch v := val.(type) {
		case float64:
			a.SensorValue = &v
		case int:
			fv := float64(v)
			a.SensorValue = &fv
		}
	}

	a.Status = "active"
	if a.Resolved {
		a.Status = "resolved"
	}
}

// PrepareDetailsForDB 准备Details字段用于存储(将DeviceID和SensorValue放入Details)
func (a *Alert) PrepareDetailsForDB() {
	if a.Details == nil {
		a.Details = make(map[string]interface{})
	}

	if a.DeviceID != nil {
		a.Details["device_id"] = *a.DeviceID
	}
	if a.SensorValue != nil {
		a.Details["sensor_value"] = *a.SensorValue
	}
}

// AlertSyncRequest Edge端同步告警请求
type AlertSyncRequest struct {
	CabinetID string          `json:"cabinet_id" binding:"required"`
	Timestamp time.Time       `json:"timestamp"`
	Alerts    []AlertSyncData `json:"alerts" binding:"required"`
}

// AlertSyncData 单个告警同步数据
type AlertSyncData struct {
	AlertID    *int64    `json:"alert_id,omitempty"`
	DeviceID   string    `json:"device_id"`
	AlertType  string    `json:"alert_type" binding:"required"`
	Severity   string    `json:"severity" binding:"required"`
	Message    string    `json:"message" binding:"required"`
	Value      *float64  `json:"value,omitempty"`
	Threshold  *float64  `json:"threshold,omitempty"`
	Timestamp  time.Time `json:"timestamp" binding:"required"`
	Resolved   bool      `json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// BatchResolveRequest 批量解决告警请求
type BatchResolveRequest struct {
	AlertIDs []string `json:"alert_ids" binding:"required"`
}
