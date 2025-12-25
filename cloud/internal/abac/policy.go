package abac

import (
	"encoding/json"
	"time"
)

// AccessPolicy 访问策略
type AccessPolicy struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	SubjectType string            `json:"subject_type" db:"subject_type"` // user, cabinet, device
	Conditions  []PolicyCondition `json:"conditions" db:"conditions"`      // 条件列表
	Permissions []string          `json:"permissions" db:"permissions"`    // 权限列表
	Priority    int               `json:"priority" db:"priority"`          // 优先级
	Enabled     bool              `json:"enabled" db:"enabled"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// PolicyCondition 策略条件
type PolicyCondition struct {
	Attribute string      `json:"attribute"` // 属性名
	Operator  string      `json:"operator"`  // 操作符: eq, ne, gt, lt, gte, lte, in, contains
	Value     interface{} `json:"value"`     // 比较值
}

// AccessLog 访问日志
type AccessLog struct {
	ID          int64           `json:"id" db:"id"`
	SubjectType string          `json:"subject_type" db:"subject_type"`
	SubjectID   string          `json:"subject_id" db:"subject_id"`
	Resource    string          `json:"resource" db:"resource"`
	Action      string          `json:"action" db:"action"`
	Allowed     bool            `json:"allowed" db:"allowed"`
	PolicyID    *string         `json:"policy_id,omitempty" db:"policy_id"`
	TrustScore  *float64        `json:"trust_score,omitempty" db:"trust_score"`
	IPAddress   *string         `json:"ip_address,omitempty" db:"ip_address"`
	Timestamp   time.Time       `json:"timestamp" db:"timestamp"`
	Attributes  json.RawMessage `json:"attributes,omitempty" db:"attributes"` // JSONB
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	ID          string            `json:"id" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	SubjectType string            `json:"subject_type" binding:"required,oneof=user cabinet device"`
	Conditions  []PolicyCondition `json:"conditions" binding:"required"`
	Permissions []string          `json:"permissions" binding:"required"`
	Priority    int               `json:"priority" binding:"required,min=0,max=1000"`
}

// UpdatePolicyRequest 更新策略请求
type UpdatePolicyRequest struct {
	Name        *string            `json:"name"`
	Description *string            `json:"description"`
	Conditions  *[]PolicyCondition `json:"conditions"`
	Permissions *[]string          `json:"permissions"`
	Priority    *int               `json:"priority" binding:"omitempty,min=0,max=1000"`
	Enabled     *bool              `json:"enabled"`
}

// PolicyListFilter 策略列表筛选
type PolicyListFilter struct {
	SubjectType *string `form:"subject_type" binding:"omitempty,oneof=user cabinet device"`
	Enabled     *bool   `form:"enabled"`
	Search      string  `form:"search"`
	Page        int     `form:"page" binding:"omitempty,min=1"`
	PageSize    int     `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// AccessLogFilter 访问日志筛选
type AccessLogFilter struct {
	SubjectType *string    `form:"subject_type" binding:"omitempty,oneof=user cabinet device"`
	SubjectID   *string    `form:"subject_id"`
	Resource    *string    `form:"resource"`
	Allowed     *bool      `form:"allowed"`
	StartTime   *time.Time `form:"start_time"`
	EndTime     *time.Time `form:"end_time"`
	Page        int        `form:"page" binding:"omitempty,min=1"`
	PageSize    int        `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// AccessStats 访问统计
type AccessStats struct {
	TotalRequests   int64            `json:"total_requests"`
	AllowedRequests int64            `json:"allowed_requests"`
	DeniedRequests  int64            `json:"denied_requests"`
	AllowRate       float64          `json:"allow_rate"`
	DenyRate        float64          `json:"deny_rate"`
	TrustScoreDist  TrustScoreDistribution `json:"trust_score_distribution"`
	TopResources    []ResourceStat   `json:"top_resources"`
	DenyReasons     []DenyReasonStat `json:"deny_reasons"`
}

// TrustScoreDistribution 信任度分布
type TrustScoreDistribution struct {
	Range0_30   int64 `json:"range_0_30"`
	Range30_60  int64 `json:"range_30_60"`
	Range60_80  int64 `json:"range_60_80"`
	Range80_100 int64 `json:"range_80_100"`
}

// ResourceStat 资源统计
type ResourceStat struct {
	Resource string `json:"resource"`
	Count    int64  `json:"count"`
}

// DenyReasonStat 拒绝原因统计
type DenyReasonStat struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// EvaluationRequest 策略评估测试请求
type EvaluationRequest struct {
	SubjectType string                 `json:"subject_type" binding:"required,oneof=user cabinet device"`
	Attributes  map[string]interface{} `json:"attributes" binding:"required"`
	Resource    string                 `json:"resource" binding:"required"`
	Action      string                 `json:"action" binding:"required"`
}

// EvaluationResult 策略评估结果
type EvaluationResult struct {
	Allowed       bool              `json:"allowed"`
	MatchedPolicy *AccessPolicy     `json:"matched_policy,omitempty"`
	TrustScore    float64           `json:"trust_score"`
	Permissions   []string          `json:"permissions"`
	Reason        string            `json:"reason"`
}

// DistributionLog 策略分发日志
type DistributionLog struct {
	ID              int64      `json:"id" db:"id"`
	PolicyID        string     `json:"policy_id" db:"policy_id"`
	CabinetID       string     `json:"cabinet_id" db:"cabinet_id"`
	OperationType   string     `json:"operation_type" db:"operation_type"` // distribute, broadcast, sync
	Status          string     `json:"status" db:"status"`                 // pending, success, failed
	OperatorID      *int       `json:"operator_id,omitempty" db:"operator_id"`
	OperatorName    *string    `json:"operator_name,omitempty" db:"operator_name"`
	ErrorMessage    *string    `json:"error_message,omitempty" db:"error_message"`
	DistributedAt   time.Time  `json:"distributed_at" db:"distributed_at"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
}

// DistributionLogFilter 分发日志筛选
type DistributionLogFilter struct {
	PolicyID    *string    `form:"policy_id"`
	CabinetID   *string    `form:"cabinet_id"`
	Status      *string    `form:"status" binding:"omitempty,oneof=pending success failed"`
	StartTime   *time.Time `form:"start_time"`
	EndTime     *time.Time `form:"end_time"`
	Page        int        `form:"page" binding:"omitempty,min=1"`
	PageSize    int        `form:"page_size" binding:"omitempty,min=1,max=100"`
}
