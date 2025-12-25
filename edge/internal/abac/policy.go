package abac

import (
	"encoding/json"
	"time"
)

// AccessPolicy 访问策略
type AccessPolicy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SubjectType string            `json:"subject_type"` // device
	Conditions  []PolicyCondition `json:"conditions"`
	Permissions []string          `json:"permissions"`
	Priority    int               `json:"priority"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PolicyCondition 策略条件
type PolicyCondition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"` // eq, ne, gt, lt, gte, lte, in, contains
	Value     interface{} `json:"value"`
}

// AccessLog 访问日志
type AccessLog struct {
	ID          int64           `json:"id"`
	SubjectType string          `json:"subject_type"`
	SubjectID   string          `json:"subject_id"`
	Resource    string          `json:"resource"`
	Action      string          `json:"action"`
	Allowed     bool            `json:"allowed"`
	PolicyID    *string         `json:"policy_id,omitempty"`
	TrustScore  *float64        `json:"trust_score,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	Synced      bool            `json:"synced"` // 是否已同步到Cloud
}

// PolicySyncMessage Cloud下发的策略同步消息
type PolicySyncMessage struct {
	Action    string          `json:"action"` // sync, delete
	Policies  []*AccessPolicy `json:"policies,omitempty"`
	PolicyIDs []string        `json:"policy_ids,omitempty"` // 用于删除
	Timestamp time.Time       `json:"timestamp"`
}
