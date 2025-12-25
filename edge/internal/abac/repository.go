package abac

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository 策略和日志存储接口
type Repository interface {
	// 策略管理
	SavePolicy(ctx context.Context, policy *AccessPolicy) error
	GetPolicy(ctx context.Context, id string) (*AccessPolicy, error)
	GetAllPolicies(ctx context.Context) ([]*AccessPolicy, error)
	GetEnabledPolicies(ctx context.Context) ([]*AccessPolicy, error)
	DeletePolicy(ctx context.Context, id string) error
	ClearPolicies(ctx context.Context) error

	// 访问日志
	LogAccess(ctx context.Context, log *AccessLog) error
	GetUnsyncedLogs(ctx context.Context, limit int) ([]*AccessLog, error)
	MarkLogsSynced(ctx context.Context, ids []int64) error
}

// SQLiteRepository SQLite实现
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository 创建SQLite仓库
func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	repo := &SQLiteRepository{db: db}
	if err := repo.initTables(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) initTables() error {
	// 创建策略表
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS device_policies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			subject_type TEXT NOT NULL,
			conditions TEXT NOT NULL,
			permissions TEXT NOT NULL,
			priority INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("create device_policies table: %w", err)
	}

	// 创建访问日志表
	_, err = r.db.Exec(`
		CREATE TABLE IF NOT EXISTS device_access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			subject_type TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			allowed INTEGER NOT NULL,
			policy_id TEXT,
			trust_score REAL,
			reason TEXT,
			timestamp DATETIME NOT NULL,
			attributes TEXT,
			synced INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return fmt.Errorf("create device_access_logs table: %w", err)
	}

	// 创建索引
	r.db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_synced ON device_access_logs(synced)`)
	r.db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON device_access_logs(timestamp)`)

	return nil
}

// SavePolicy 保存策略
func (r *SQLiteRepository) SavePolicy(ctx context.Context, policy *AccessPolicy) error {
	conditionsJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return err
	}
	permissionsJSON, err := json.Marshal(policy.Permissions)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO device_policies
		(id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, policy.ID, policy.Name, policy.Description, policy.SubjectType,
		string(conditionsJSON), string(permissionsJSON),
		policy.Priority, policy.Enabled, policy.CreatedAt, policy.UpdatedAt)

	return err
}

// GetPolicy 获取单个策略
func (r *SQLiteRepository) GetPolicy(ctx context.Context, id string) (*AccessPolicy, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM device_policies WHERE id = ?
	`, id)

	return r.scanPolicy(row)
}

// GetAllPolicies 获取所有策略
func (r *SQLiteRepository) GetAllPolicies(ctx context.Context) ([]*AccessPolicy, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM device_policies ORDER BY priority DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPolicies(rows)
}

// GetEnabledPolicies 获取所有启用的策略
func (r *SQLiteRepository) GetEnabledPolicies(ctx context.Context) ([]*AccessPolicy, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, subject_type, conditions, permissions, priority, enabled, created_at, updated_at
		FROM device_policies WHERE enabled = 1 ORDER BY priority DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPolicies(rows)
}

// DeletePolicy 删除策略
func (r *SQLiteRepository) DeletePolicy(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_policies WHERE id = ?`, id)
	return err
}

// ClearPolicies 清空所有策略
func (r *SQLiteRepository) ClearPolicies(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_policies`)
	return err
}

// LogAccess 记录访问日志
func (r *SQLiteRepository) LogAccess(ctx context.Context, log *AccessLog) error {
	var attrsJSON string
	if log.Attributes != nil {
		attrsJSON = string(log.Attributes)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO device_access_logs
		(subject_type, subject_id, resource, action, allowed, policy_id, trust_score, reason, timestamp, attributes, synced)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.SubjectType, log.SubjectID, log.Resource, log.Action, log.Allowed,
		log.PolicyID, log.TrustScore, log.Reason, log.Timestamp, attrsJSON, 0)

	return err
}

// GetUnsyncedLogs 获取未同步的日志
func (r *SQLiteRepository) GetUnsyncedLogs(ctx context.Context, limit int) ([]*AccessLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subject_type, subject_id, resource, action, allowed, policy_id, trust_score, reason, timestamp, attributes, synced
		FROM device_access_logs WHERE synced = 0 ORDER BY timestamp ASC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AccessLog
	for rows.Next() {
		var log AccessLog
		var policyID, attrsJSON sql.NullString
		var trustScore sql.NullFloat64

		err := rows.Scan(&log.ID, &log.SubjectType, &log.SubjectID, &log.Resource, &log.Action,
			&log.Allowed, &policyID, &trustScore, &log.Reason, &log.Timestamp, &attrsJSON, &log.Synced)
		if err != nil {
			return nil, err
		}

		if policyID.Valid {
			log.PolicyID = &policyID.String
		}
		if trustScore.Valid {
			log.TrustScore = &trustScore.Float64
		}
		if attrsJSON.Valid && attrsJSON.String != "" {
			log.Attributes = json.RawMessage(attrsJSON.String)
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// MarkLogsSynced 标记日志已同步
func (r *SQLiteRepository) MarkLogsSynced(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := "UPDATE device_access_logs SET synced = 1 WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *SQLiteRepository) scanPolicy(row *sql.Row) (*AccessPolicy, error) {
	var policy AccessPolicy
	var conditionsJSON, permissionsJSON string

	err := row.Scan(&policy.ID, &policy.Name, &policy.Description, &policy.SubjectType,
		&conditionsJSON, &permissionsJSON, &policy.Priority, &policy.Enabled,
		&policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(conditionsJSON), &policy.Conditions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(permissionsJSON), &policy.Permissions); err != nil {
		return nil, err
	}

	return &policy, nil
}

func (r *SQLiteRepository) scanPolicies(rows *sql.Rows) ([]*AccessPolicy, error) {
	var policies []*AccessPolicy

	for rows.Next() {
		var policy AccessPolicy
		var conditionsJSON, permissionsJSON string

		err := rows.Scan(&policy.ID, &policy.Name, &policy.Description, &policy.SubjectType,
			&conditionsJSON, &permissionsJSON, &policy.Priority, &policy.Enabled,
			&policy.CreatedAt, &policy.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(conditionsJSON), &policy.Conditions); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(permissionsJSON), &policy.Permissions); err != nil {
			return nil, err
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}

// CleanOldLogs 清理旧日志 (保留最近7天)
func (r *SQLiteRepository) CleanOldLogs(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -7)
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM device_access_logs WHERE synced = 1 AND timestamp < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
