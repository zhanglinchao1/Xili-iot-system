package abac

import (
	"context"
)

// PolicyRepository 策略数据访问接口
type PolicyRepository interface {
	// 策略CRUD
	Create(ctx context.Context, policy *AccessPolicy) error
	GetByID(ctx context.Context, id string) (*AccessPolicy, error)
	List(ctx context.Context, filter *PolicyListFilter) ([]*AccessPolicy, int64, error)
	Update(ctx context.Context, id string, req *UpdatePolicyRequest) error
	Delete(ctx context.Context, id string) error
	ToggleEnabled(ctx context.Context, id string) error

	// 策略查询
	GetBySubjectType(ctx context.Context, subjectType string, enabledOnly bool) ([]*AccessPolicy, error)
	GetAllEnabled(ctx context.Context) ([]*AccessPolicy, error)

	// 访问日志
	LogAccess(ctx context.Context, log *AccessLog) error
	GetAccessLogs(ctx context.Context, filter *AccessLogFilter) ([]*AccessLog, int64, error)
	GetAccessStats(ctx context.Context, startTime, endTime *string) (*AccessStats, error)

	// 策略分发日志
	LogDistribution(ctx context.Context, log *DistributionLog) error
	GetDistributionLogs(ctx context.Context, filter *DistributionLogFilter) ([]*DistributionLog, int64, error)
	UpdateDistributionAck(ctx context.Context, policyID, cabinetID string) error
}
