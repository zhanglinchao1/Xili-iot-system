package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"cloud-system/internal/models"
	"cloud-system/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertRepo PostgreSQL告警仓库实现
type AlertRepo struct {
	pool *pgxpool.Pool
}

// NewAlertRepo 创建告警仓库实例
func NewAlertRepo(pool *pgxpool.Pool) *AlertRepo {
	repo := &AlertRepo{
		pool: pool,
	}
	if err := repo.ensureSchema(context.Background()); err != nil {
		log.Printf("warn: ensure alert schema failed: %v", err)
	}
	return repo
}

// CreateOrUpdate 创建或更新告警（用于同步）
// 如果已存在相同cabinet_id、device_id、alert_type的告警，则更新；否则创建新告警
func (r *AlertRepo) CreateOrUpdate(ctx context.Context, alert *models.Alert) error {
	var existingAlertID int64
	var existingCreatedAt time.Time
	var existingResolved bool
	var err error

	// 查询最近的相同类型告警（不限resolved状态）
	deviceID := ""
	if alert.DeviceID != nil {
		deviceID = *alert.DeviceID
	}
	err = r.pool.QueryRow(ctx, `
		SELECT alert_id, created_at, resolved
		FROM alerts
		WHERE cabinet_id = $1
		  AND alert_type = $2
		  AND details->>'device_id' = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, alert.CabinetID, alert.AlertType, deviceID).Scan(&existingAlertID, &existingCreatedAt, &existingResolved)

	if err == nil {
		detailsJSON, serErr := serializeAlertDetails(alert)
		if serErr != nil {
			return serErr
		}

		updateQuery := `
			UPDATE alerts
			SET severity = $1,
			    message = $2,
			    details = $3,
			    resolved = $4,
			    resolved_at = $5,
			    resolved_by = $6,
			    edge_alert_id = $7
			WHERE alert_id = $8
		`

		_, err = r.pool.Exec(ctx, updateQuery,
			alert.Severity,
			alert.Message,
			detailsJSON,
			alert.Resolved,
			alert.ResolvedAt,
			alert.ResolvedBy,
			alert.EdgeAlertID,
			existingAlertID,
		)
		if err != nil {
			return errors.Wrap(err, errors.ErrDatabaseQuery, "更新告警失败")
		}

		alert.AlertID = fmt.Sprintf("%d", existingAlertID)
		alert.CreatedAt = existingCreatedAt
		return nil
	}

	if err != nil && err != pgx.ErrNoRows {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "查询告警失败")
	}

	return r.Create(ctx, alert)
}

// Create 创建告警
func (r *AlertRepo) Create(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			cabinet_id, edge_alert_id, alert_type, severity, message,
			details, resolved, resolved_at, resolved_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING alert_id
	`

	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}

	detailsJSON, err := serializeAlertDetails(alert)
	if err != nil {
		return err
	}

	var alertID int64
	err = r.pool.QueryRow(ctx, query,
		alert.CabinetID,
		alert.EdgeAlertID,
		alert.AlertType,
		alert.Severity,
		alert.Message,
		detailsJSON,
		alert.Resolved,
		alert.ResolvedAt,
		alert.ResolvedBy,
		alert.CreatedAt,
	).Scan(&alertID)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "创建告警失败")
	}

	alert.AlertID = fmt.Sprintf("%d", alertID)
	return nil
}

// GetByID 根据ID获取告警
func (r *AlertRepo) GetByID(ctx context.Context, alertID string) (*models.Alert, error) {
	// 注意：alerts表没有device_id, sensor_value, status, updated_at列
	// alert_id是BIGINT类型，需要转换为字符串
	// LEFT JOIN cabinets表获取位置信息
	// 查询告警信息,包括edge_alert_id用于命令下发
	query := `
		SELECT a.alert_id, a.cabinet_id, a.edge_alert_id, a.alert_type, a.severity, a.message,
		       a.details, a.resolved, a.resolved_at, a.resolved_by, a.created_at,
		       c.location
		FROM alerts a
		LEFT JOIN cabinets c ON a.cabinet_id = c.cabinet_id
		WHERE a.alert_id = $1
	`

	alert := &models.Alert{}
	var detailsJSON string
	var resolved bool
	var alertIDInt int64
	var edgeAlertIDInt *int64
	var location *string

	err := r.pool.QueryRow(ctx, query, alertID).Scan(
		&alertIDInt,
		&alert.CabinetID,
		&edgeAlertIDInt,
		&alert.AlertType,
		&alert.Severity,
		&alert.Message,
		&detailsJSON,
		&resolved,
		&alert.ResolvedAt,
		&alert.ResolvedBy,
		&alert.CreatedAt,
		&location,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrNotFound, "告警不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询告警失败")
	}

	// 将int64转换为字符串
	alert.AlertID = fmt.Sprintf("%d", alertIDInt)
	// 设置EdgeAlertID (如果数据库中有值)
	if edgeAlertIDInt != nil {
		alert.EdgeAlertID = edgeAlertIDInt
	}

	// 设置位置信息
	alert.Location = location

	// 解析details JSON
	alert.Resolved = resolved
	if detailsJSON != "" && detailsJSON != "{}" {
		_ = json.Unmarshal([]byte(detailsJSON), &alert.Details)
	}
	alert.PopulateCalculatedFields()

	return alert, nil
}

// List 获取告警列表（支持过滤和分页）
func (r *AlertRepo) List(ctx context.Context, filter *models.AlertListFilter) ([]*models.Alert, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.CabinetID != nil && *filter.CabinetID != "" {
		whereClause += fmt.Sprintf(" AND a.cabinet_id = $%d", argIndex)
		args = append(args, *filter.CabinetID)
		argIndex++
	}

	if filter.Severity != nil && *filter.Severity != "" {
		whereClause += fmt.Sprintf(" AND a.severity = $%d", argIndex)
		args = append(args, *filter.Severity)
		argIndex++
	}

	if filter.Status != nil && *filter.Status != "" {
		// 将status转换为resolved布尔值
		if *filter.Status == "resolved" {
			whereClause += " AND a.resolved = TRUE"
		} else if *filter.Status == "active" {
			whereClause += " AND a.resolved = FALSE"
		}
		// ignored状态暂不处理
	}

	// 查询总数（需要JOIN以保持一致性）
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM alerts a
		LEFT JOIN cabinets c ON a.cabinet_id = c.cabinet_id
		%s
	`, whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询告警总数失败")
	}

	// 查询列表（LEFT JOIN cabinets表获取位置信息）
	// 注意：alerts表没有device_id, sensor_value, status, updated_at列
	// edge_alert_id暂时不查询（表中可能不存在）
	listQuery := fmt.Sprintf(`
		SELECT a.alert_id, a.cabinet_id, a.edge_alert_id, a.alert_type, a.severity, a.message,
		       a.details, a.resolved, a.resolved_at, a.resolved_by, a.created_at,
		       c.location
		FROM alerts a
		LEFT JOIN cabinets c ON a.cabinet_id = c.cabinet_id
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询告警列表失败")
	}
	defer rows.Close()

	alerts := []*models.Alert{}
	for rows.Next() {
		alert := &models.Alert{}
		var detailsJSON string
		var resolved bool
		var alertIDInt int64
		var location *string
		var edgeAlertID sql.NullInt64

		err := rows.Scan(
			&alertIDInt,
			&alert.CabinetID,
			&edgeAlertID,
			&alert.AlertType,
			&alert.Severity,
			&alert.Message,
			&detailsJSON,
			&resolved,
			&alert.ResolvedAt,
			&alert.ResolvedBy,
			&alert.CreatedAt,
			&location,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描告警数据失败")
		}

		alert.AlertID = fmt.Sprintf("%d", alertIDInt)
		alert.EdgeAlertID = nullableInt64(edgeAlertID)
		alert.Location = location

		// 解析details JSON
		alert.Resolved = resolved
		if detailsJSON != "" && detailsJSON != "{}" {
			_ = json.Unmarshal([]byte(detailsJSON), &alert.Details)
		}
		alert.PopulateCalculatedFields()

		alerts = append(alerts, alert)
	}

	return alerts, total, nil
}

// Resolve 解决告警
func (r *AlertRepo) Resolve(ctx context.Context, alertID string, resolvedBy string) error {
	// 注意：alerts表使用resolved布尔值，不是status列
	// alert_id是BIGSERIAL类型，需要将字符串转换为int64
	alertIDInt, err := strconv.ParseInt(alertID, 10, 64)
	if err != nil {
		return errors.Wrap(err, errors.ErrValidation, "无效的告警ID格式")
	}

	query := `
		UPDATE alerts
		SET resolved = TRUE, resolved_at = $1, resolved_by = $2
		WHERE alert_id = $3 AND resolved = FALSE
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, now, resolvedBy, alertIDInt)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "解决告警失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "告警不存在或已被解决")
	}

	return nil
}

// GetActiveByCabinet 获取储能柜的活跃告警
func (r *AlertRepo) GetActiveByCabinet(ctx context.Context, cabinetID string) ([]*models.Alert, error) {
	// 查询活跃告警（未解决）
	// 注意：alerts表没有device_id列，只有cabinet_id
	// edge_alert_id暂时不查询
	query := `
		SELECT alert_id, cabinet_id, edge_alert_id, alert_type, severity, message,
		       details, resolved, resolved_at, resolved_by, created_at
		FROM alerts
		WHERE cabinet_id = $1 AND resolved = FALSE
		ORDER BY severity DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, cabinetID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询活跃告警失败")
	}
	defer rows.Close()

	alerts := []*models.Alert{}
	for rows.Next() {
		alert := &models.Alert{}
		var detailsJSON string
		var resolved bool
		var alertIDInt int64
		var edgeAlertID sql.NullInt64

		err := rows.Scan(
			&alertIDInt,
			&alert.CabinetID,
			&edgeAlertID,
			&alert.AlertType,
			&alert.Severity,
			&alert.Message,
			&detailsJSON,
			&resolved,
			&alert.ResolvedAt,
			&alert.ResolvedBy,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描告警数据失败")
		}

		alert.AlertID = fmt.Sprintf("%d", alertIDInt)
		alert.EdgeAlertID = nullableInt64(edgeAlertID)
		alert.Resolved = resolved
		if detailsJSON != "" && detailsJSON != "{}" {
			_ = json.Unmarshal([]byte(detailsJSON), &alert.Details)
		}
		alert.PopulateCalculatedFields()

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetRecentByCabinet 获取储能柜的最近告警（包含已解决）
func (r *AlertRepo) GetRecentByCabinet(ctx context.Context, cabinetID string, limit int) ([]*models.Alert, error) {
	if limit <= 0 {
		limit = 5
	}

	// edge_alert_id暂时不查询
	query := `
		SELECT alert_id, cabinet_id, edge_alert_id, alert_type, severity, message,
		       details, resolved, resolved_at, resolved_by, created_at
		FROM alerts
		WHERE cabinet_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, cabinetID, limit)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询告警失败")
	}
	defer rows.Close()

	alerts := []*models.Alert{}
	for rows.Next() {
		alert := &models.Alert{}
		var detailsJSON string
		var resolved bool
		var alertIDInt int64
		var edgeAlertID sql.NullInt64

		err := rows.Scan(
			&alertIDInt,
			&alert.CabinetID,
			&edgeAlertID,
			&alert.AlertType,
			&alert.Severity,
			&alert.Message,
			&detailsJSON,
			&resolved,
			&alert.ResolvedAt,
			&alert.ResolvedBy,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描告警数据失败")
		}

		alert.AlertID = fmt.Sprintf("%d", alertIDInt)
		alert.EdgeAlertID = nullableInt64(edgeAlertID)
		alert.Resolved = resolved
		if detailsJSON != "" && detailsJSON != "{}" {
			_ = json.Unmarshal([]byte(detailsJSON), &alert.Details)
		}
		alert.PopulateCalculatedFields()

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *AlertRepo) ensureSchema(ctx context.Context) error {
	migrations := []string{
		`ALTER TABLE alerts ADD COLUMN IF NOT EXISTS edge_alert_id BIGINT`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_edge_id ON alerts(edge_alert_id)`,
	}

	for _, stmt := range migrations {
		if _, err := r.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func serializeAlertDetails(alert *models.Alert) (string, error) {
	details := map[string]interface{}{}
	if alert.DeviceID != nil {
		details["device_id"] = *alert.DeviceID
	}
	if alert.SensorValue != nil {
		details["sensor_value"] = *alert.SensorValue
	}
	if len(details) == 0 {
		return "{}", nil
	}
	bytes, err := json.Marshal(details)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrValidation, "序列化details失败")
	}
	return string(bytes), nil
}

func nullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}
