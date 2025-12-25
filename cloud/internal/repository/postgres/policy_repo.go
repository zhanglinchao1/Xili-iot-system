package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud-system/internal/abac"

	"github.com/jackc/pgx/v5/pgxpool"
)

type policyRepo struct {
	pool *pgxpool.Pool
}

// NewPolicyRepo 创建策略Repository
func NewPolicyRepo(pool *pgxpool.Pool) abac.PolicyRepository {
	return &policyRepo{pool: pool}
}

// Create 创建策略
func (r *policyRepo) Create(ctx context.Context, policy *abac.AccessPolicy) error {
	conditionsJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	permissionsJSON, err := json.Marshal(policy.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	query := `
		INSERT INTO access_policies (id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	_, err = r.pool.Exec(ctx, query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.SubjectType,
		conditionsJSON,
		permissionsJSON,
		policy.Priority,
		policy.Enabled,
		now,
		now,
	)

	return err
}

// GetByID 根据ID获取策略
func (r *policyRepo) GetByID(ctx context.Context, id string) (*abac.AccessPolicy, error) {
	query := `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM access_policies
		WHERE id = $1
	`

	var policy abac.AccessPolicy
	var conditionsJSON, permissionsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.Name,
		&policy.Description,
		&policy.SubjectType,
		&conditionsJSON,
		&permissionsJSON,
		&policy.Priority,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
		return nil, fmt.Errorf("unmarshal conditions: %w", err)
	}

	if err := json.Unmarshal(permissionsJSON, &policy.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}

	return &policy, nil
}

// List 列出策略
func (r *policyRepo) List(ctx context.Context, filter *abac.PolicyListFilter) ([]*abac.AccessPolicy, int64, error) {
	// 构建WHERE条件
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.SubjectType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("subject_type = $%d", argPos))
		args = append(args, *filter.SubjectType)
		argPos++
	}

	if filter.Enabled != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *filter.Enabled)
		argPos++
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+filter.Search+"%")
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 计数查询
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM access_policies %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 数据查询
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM access_policies %s
		ORDER BY priority DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	policies := []*abac.AccessPolicy{}
	for rows.Next() {
		var policy abac.AccessPolicy
		var conditionsJSON, permissionsJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Description,
			&policy.SubjectType,
			&conditionsJSON,
			&permissionsJSON,
			&policy.Priority,
			&policy.Enabled,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
			return nil, 0, fmt.Errorf("unmarshal conditions: %w", err)
		}

		if err := json.Unmarshal(permissionsJSON, &policy.Permissions); err != nil {
			return nil, 0, fmt.Errorf("unmarshal permissions: %w", err)
		}

		policies = append(policies, &policy)
	}

	return policies, total, nil
}

// Update 更新策略
func (r *policyRepo) Update(ctx context.Context, id string, req *abac.UpdatePolicyRequest) error {
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}

	if req.Conditions != nil {
		conditionsJSON, err := json.Marshal(*req.Conditions)
		if err != nil {
			return fmt.Errorf("marshal conditions: %w", err)
		}
		updates = append(updates, fmt.Sprintf("conditions = $%d", argPos))
		args = append(args, conditionsJSON)
		argPos++
	}

	if req.Permissions != nil {
		permissionsJSON, err := json.Marshal(*req.Permissions)
		if err != nil {
			return fmt.Errorf("marshal permissions: %w", err)
		}
		updates = append(updates, fmt.Sprintf("permissions = $%d", argPos))
		args = append(args, permissionsJSON)
		argPos++
	}

	if req.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argPos))
		args = append(args, *req.Priority)
		argPos++
	}

	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *req.Enabled)
		argPos++
	}

	if len(updates) == 0 {
		return nil // 没有更新
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE access_policies
		SET %s
		WHERE id = $%d
	`, strings.Join(updates, ", "), argPos)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// Delete 删除策略
func (r *policyRepo) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM access_policies WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ToggleEnabled 切换策略启用状态
func (r *policyRepo) ToggleEnabled(ctx context.Context, id string) error {
	query := `
		UPDATE access_policies
		SET enabled = NOT enabled, updated_at = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, time.Now(), id)
	return err
}

// GetBySubjectType 根据主体类型获取策略
func (r *policyRepo) GetBySubjectType(ctx context.Context, subjectType string, enabledOnly bool) ([]*abac.AccessPolicy, error) {
	query := `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM access_policies
		WHERE subject_type = $1
	`

	if enabledOnly {
		query += " AND enabled = true"
	}

	query += " ORDER BY priority DESC"

	rows, err := r.pool.Query(ctx, query, subjectType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPolicies(rows)
}

// GetAllEnabled 获取所有启用的策略
func (r *policyRepo) GetAllEnabled(ctx context.Context) ([]*abac.AccessPolicy, error) {
	query := `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM access_policies
		WHERE enabled = true
		ORDER BY priority DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPolicies(rows)
}

// LogAccess 记录访问日志
func (r *policyRepo) LogAccess(ctx context.Context, log *abac.AccessLog) error {
	query := `
		INSERT INTO access_logs (subject_type, subject_id, resource, action, allowed, policy_id, trust_score, ip_address, timestamp, attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		log.SubjectType,
		log.SubjectID,
		log.Resource,
		log.Action,
		log.Allowed,
		log.PolicyID,
		log.TrustScore,
		log.IPAddress,
		log.Timestamp,
		log.Attributes,
	)

	return err
}

// GetAccessLogs 获取访问日志
func (r *policyRepo) GetAccessLogs(ctx context.Context, filter *abac.AccessLogFilter) ([]*abac.AccessLog, int64, error) {
	// 构建WHERE条件
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.SubjectType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("subject_type = $%d", argPos))
		args = append(args, *filter.SubjectType)
		argPos++
	}

	if filter.SubjectID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("subject_id = $%d", argPos))
		args = append(args, *filter.SubjectID)
		argPos++
	}

	if filter.Resource != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("resource ILIKE $%d", argPos))
		args = append(args, "%"+*filter.Resource+"%")
		argPos++
	}

	if filter.Allowed != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("allowed = $%d", argPos))
		args = append(args, *filter.Allowed)
		argPos++
	}

	if filter.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argPos))
		args = append(args, *filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argPos))
		args = append(args, *filter.EndTime)
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 计数查询
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM access_logs %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 数据查询
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, subject_type, subject_id, resource, action, allowed, policy_id, trust_score, ip_address, timestamp, attributes
		FROM access_logs %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []*abac.AccessLog{}
	for rows.Next() {
		var log abac.AccessLog

		err := rows.Scan(
			&log.ID,
			&log.SubjectType,
			&log.SubjectID,
			&log.Resource,
			&log.Action,
			&log.Allowed,
			&log.PolicyID,
			&log.TrustScore,
			&log.IPAddress,
			&log.Timestamp,
			&log.Attributes,
		)
		if err != nil {
			return nil, 0, err
		}

		logs = append(logs, &log)
	}

	return logs, total, nil
}

// GetAccessStats 获取访问统计
func (r *policyRepo) GetAccessStats(ctx context.Context, startTime, endTime *string) (*abac.AccessStats, error) {
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if startTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argPos))
		args = append(args, *startTime)
		argPos++
	}

	if endTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argPos))
		args = append(args, *endTime)
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	stats := &abac.AccessStats{}

	// 总请求数和通过/拒绝数
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE allowed = true) as allowed,
			COUNT(*) FILTER (WHERE allowed = false) as denied
		FROM access_logs %s
	`, whereClause)

	err := r.pool.QueryRow(ctx, query, args...).Scan(&stats.TotalRequests, &stats.AllowedRequests, &stats.DeniedRequests)
	if err != nil {
		return nil, err
	}

	if stats.TotalRequests > 0 {
		stats.AllowRate = float64(stats.AllowedRequests) / float64(stats.TotalRequests) * 100
		stats.DenyRate = float64(stats.DeniedRequests) / float64(stats.TotalRequests) * 100
	}

	// 信任度分布
	trustQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE trust_score >= 0 AND trust_score < 30) as range_0_30,
			COUNT(*) FILTER (WHERE trust_score >= 30 AND trust_score < 60) as range_30_60,
			COUNT(*) FILTER (WHERE trust_score >= 60 AND trust_score < 80) as range_60_80,
			COUNT(*) FILTER (WHERE trust_score >= 80 AND trust_score <= 100) as range_80_100
		FROM access_logs %s
	`, whereClause)

	err = r.pool.QueryRow(ctx, trustQuery, args...).Scan(
		&stats.TrustScoreDist.Range0_30,
		&stats.TrustScoreDist.Range30_60,
		&stats.TrustScoreDist.Range60_80,
		&stats.TrustScoreDist.Range80_100,
	)
	if err != nil {
		return nil, err
	}

	// 热点资源TOP5
	resourceQuery := fmt.Sprintf(`
		SELECT resource, COUNT(*) as count
		FROM access_logs %s
		GROUP BY resource
		ORDER BY count DESC
		LIMIT 5
	`, whereClause)

	resourceRows, err := r.pool.Query(ctx, resourceQuery, args...)
	if err != nil {
		return nil, err
	}
	defer resourceRows.Close()

	stats.TopResources = []abac.ResourceStat{}
	for resourceRows.Next() {
		var rs abac.ResourceStat
		if err := resourceRows.Scan(&rs.Resource, &rs.Count); err != nil {
			return nil, err
		}
		stats.TopResources = append(stats.TopResources, rs)
	}

	// 拒绝原因统计 (简化版，基于policy_id)
	denyWhereClause := "WHERE allowed = false"
	if len(whereClauses) > 0 {
		denyWhereClause = whereClause + " AND allowed = false"
	}
	denyQuery := fmt.Sprintf(`
		SELECT
			CASE
				WHEN policy_id IS NULL THEN '无匹配策略'
				WHEN trust_score < 30 THEN '信任度过低'
				ELSE '权限不足'
			END as reason,
			COUNT(*) as count
		FROM access_logs
		%s
		GROUP BY reason
		ORDER BY count DESC
	`, denyWhereClause)

	denyRows, err := r.pool.Query(ctx, denyQuery, args...)
	if err != nil {
		return nil, err
	}
	defer denyRows.Close()

	stats.DenyReasons = []abac.DenyReasonStat{}
	for denyRows.Next() {
		var dr abac.DenyReasonStat
		if err := denyRows.Scan(&dr.Reason, &dr.Count); err != nil {
			return nil, err
		}
		stats.DenyReasons = append(stats.DenyReasons, dr)
	}

	return stats, nil
}

// LogDistribution 记录策略分发日志
func (r *policyRepo) LogDistribution(ctx context.Context, log *abac.DistributionLog) error {
	query := `
		INSERT INTO policy_distribution_logs (policy_id, cabinet_id, operation_type, status, operator_id, operator_name, error_message, distributed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := r.pool.QueryRow(ctx, query,
		log.PolicyID,
		log.CabinetID,
		log.OperationType,
		log.Status,
		log.OperatorID,
		log.OperatorName,
		log.ErrorMessage,
		log.DistributedAt,
	).Scan(&log.ID)

	return err
}

// GetDistributionLogs 获取策略分发日志
func (r *policyRepo) GetDistributionLogs(ctx context.Context, filter *abac.DistributionLogFilter) ([]*abac.DistributionLog, int64, error) {
	// 构建WHERE条件
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.PolicyID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("policy_id = $%d", argPos))
		args = append(args, *filter.PolicyID)
		argPos++
	}

	if filter.CabinetID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("cabinet_id = $%d", argPos))
		args = append(args, *filter.CabinetID)
		argPos++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filter.Status)
		argPos++
	}

	if filter.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("distributed_at >= $%d", argPos))
		args = append(args, *filter.StartTime)
		argPos++
	}

	if filter.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("distributed_at <= $%d", argPos))
		args = append(args, *filter.EndTime)
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 计数查询
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM policy_distribution_logs %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 数据查询
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, policy_id, cabinet_id, operation_type, status, operator_id, operator_name, error_message, distributed_at, acknowledged_at
		FROM policy_distribution_logs %s
		ORDER BY distributed_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []*abac.DistributionLog{}
	for rows.Next() {
		var log abac.DistributionLog

		err := rows.Scan(
			&log.ID,
			&log.PolicyID,
			&log.CabinetID,
			&log.OperationType,
			&log.Status,
			&log.OperatorID,
			&log.OperatorName,
			&log.ErrorMessage,
			&log.DistributedAt,
			&log.AcknowledgedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		logs = append(logs, &log)
	}

	return logs, total, nil
}

// UpdateDistributionAck 更新策略分发确认
func (r *policyRepo) UpdateDistributionAck(ctx context.Context, policyID, cabinetID string) error {
	query := `
		UPDATE policy_distribution_logs
		SET status = 'success', acknowledged_at = $1
		WHERE policy_id = $2 AND cabinet_id = $3 AND status = 'pending'
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), policyID, cabinetID)
	return err
}

// scanPolicies 扫描策略行
func (r *policyRepo) scanPolicies(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]*abac.AccessPolicy, error) {
	policies := []*abac.AccessPolicy{}

	for rows.Next() {
		var policy abac.AccessPolicy
		var conditionsJSON, permissionsJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Description,
			&policy.SubjectType,
			&conditionsJSON,
			&permissionsJSON,
			&policy.Priority,
			&policy.Enabled,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}

		if err := json.Unmarshal(permissionsJSON, &policy.Permissions); err != nil {
			return nil, fmt.Errorf("unmarshal permissions: %w", err)
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}
