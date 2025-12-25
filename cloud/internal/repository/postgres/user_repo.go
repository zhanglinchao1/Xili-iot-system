// Package postgres 实现基于PostgreSQL的用户数据访问
package postgres

import (
	"context"
	"fmt"

	"cloud-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo 用户仓储实现
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo 创建用户仓储
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// GetByUsername 根据用户名查询用户
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, email, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE LOWER(username) = LOWER($1) AND status = 'active'
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("用户不存在或已禁用")
		}
		return nil, err
	}

	return &user, nil
}

// ExistsByUsername 判断用户名是否存在
func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE LOWER(username) = LOWER($1)
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, username).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// ExistsByEmail 判断邮箱是否存在
func (r *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE LOWER(email) = LOWER($1)
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID int) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// Create 创建用户
func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, password_hash, email, role, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.Role,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetByID 根据ID查询用户
func (r *UserRepo) GetByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, email, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}

	return &user, nil
}

// List 获取用户列表（支持过滤和分页）
func (r *UserRepo) List(ctx context.Context, filter *models.UserListFilter) ([]*models.User, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.Role != nil && *filter.Role != "" {
		whereClause += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, *filter.Role)
		argIndex++
	}

	if filter.Status != nil && *filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	// 查询总数
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM users %s`, whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	listQuery := fmt.Sprintf(`
		SELECT id, username, password_hash, email, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.PasswordHash,
			&user.Email,
			&user.Role,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

// Update 更新用户信息
func (r *UserRepo) Update(ctx context.Context, userID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setClause := ""
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		if argIndex > 1 {
			setClause += ", "
		}
		setClause += fmt.Sprintf("%s = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	// 添加 updated_at
	setClause += ", updated_at = NOW()"

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d`, setClause, argIndex)
	args = append(args, userID)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}

// UpdatePassword 更新用户密码
func (r *UserRepo) UpdatePassword(ctx context.Context, userID int, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.pool.Exec(ctx, query, newPasswordHash, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}

// Delete 删除用户（物理删除）
func (r *UserRepo) Delete(ctx context.Context, userID int) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}
