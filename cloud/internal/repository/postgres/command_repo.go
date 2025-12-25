package postgres

import (
	"context"
	"fmt"
	"time"

	"cloud-system/internal/models"
	"cloud-system/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommandRepo PostgreSQL命令仓库实现
type CommandRepo struct {
	pool *pgxpool.Pool
}

// NewCommandRepo 创建命令仓库实例
func NewCommandRepo(pool *pgxpool.Pool) *CommandRepo {
	return &CommandRepo{
		pool: pool,
	}
}

// Create 创建命令
func (r *CommandRepo) Create(ctx context.Context, command *models.Command) error {
	query := `
		INSERT INTO commands (
			command_id, cabinet_id, command_type, payload, status,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	command.CreatedAt = now
	command.UpdatedAt = now
	command.Status = "pending" // 默认待发送状态

	_, err := r.pool.Exec(ctx, query,
		command.CommandID,
		command.CabinetID,
		command.CommandType,
		command.Payload,
		command.Status,
		command.CreatedBy,
		command.CreatedAt,
		command.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "创建命令失败")
	}

	return nil
}

// GetByID 根据ID获取命令
func (r *CommandRepo) GetByID(ctx context.Context, commandID string) (*models.Command, error) {
	query := `
		SELECT command_id, cabinet_id, command_type, payload, status,
		       result, sent_at, completed_at, created_by, created_at, updated_at
		FROM commands
		WHERE command_id = $1
	`

	command := &models.Command{}
	err := r.pool.QueryRow(ctx, query, commandID).Scan(
		&command.CommandID,
		&command.CabinetID,
		&command.CommandType,
		&command.Payload,
		&command.Status,
		&command.Result,
		&command.SentAt,
		&command.CompletedAt,
		&command.CreatedBy,
		&command.CreatedAt,
		&command.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrNotFound, "命令不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询命令失败")
	}

	return command, nil
}

// List 获取命令列表（支持过滤和分页）
func (r *CommandRepo) List(ctx context.Context, filter *models.CommandListFilter) ([]*models.Command, int64, error) {
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.CabinetID != nil && *filter.CabinetID != "" {
		whereClause += fmt.Sprintf(" AND cabinet_id = $%d", argIndex)
		args = append(args, *filter.CabinetID)
		argIndex++
	}

	if filter.CommandType != nil && *filter.CommandType != "" {
		whereClause += fmt.Sprintf(" AND command_type = $%d", argIndex)
		args = append(args, *filter.CommandType)
		argIndex++
	}

	if filter.Status != nil && *filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM commands %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询命令总数失败")
	}

	// 查询列表
	listQuery := fmt.Sprintf(`
		SELECT command_id, cabinet_id, command_type, payload, status,
		       result, sent_at, completed_at, created_by, created_at, updated_at
		FROM commands
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询命令列表失败")
	}
	defer rows.Close()

	commands := []*models.Command{}
	for rows.Next() {
		command := &models.Command{}
		err := rows.Scan(
			&command.CommandID,
			&command.CabinetID,
			&command.CommandType,
			&command.Payload,
			&command.Status,
			&command.Result,
			&command.SentAt,
			&command.CompletedAt,
			&command.CreatedBy,
			&command.CreatedAt,
			&command.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描命令数据失败")
		}
		commands = append(commands, command)
	}

	return commands, total, nil
}

// UpdateStatus 更新命令状态
func (r *CommandRepo) UpdateStatus(ctx context.Context, commandID string, status string, result *string) error {
	query := `
		UPDATE commands
		SET status = $1, result = $2, updated_at = $3
		WHERE command_id = $4
	`

	commandResult, err := r.pool.Exec(ctx, query, status, result, time.Now(), commandID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新命令状态失败")
	}

	if commandResult.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "命令不存在")
	}

	return nil
}

// MarkAsSent 标记命令为已发送
func (r *CommandRepo) MarkAsSent(ctx context.Context, commandID string) error {
	query := `
		UPDATE commands
		SET status = 'sent', sent_at = $1, updated_at = $2
		WHERE command_id = $3
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, now, now, commandID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "标记命令为已发送失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "命令不存在")
	}

	return nil
}

// MarkAsCompleted 标记命令为已完成
func (r *CommandRepo) MarkAsCompleted(ctx context.Context, commandID string, status string, result string) error {
	query := `
		UPDATE commands
		SET status = $1, result = $2, completed_at = $3, updated_at = $4
		WHERE command_id = $5
	`

	now := time.Now()
	cmdResult, err := r.pool.Exec(ctx, query, status, result, now, now, commandID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "标记命令为已完成失败")
	}

	if cmdResult.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "命令不存在")
	}

	return nil
}

