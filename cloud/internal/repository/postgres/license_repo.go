package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud-system/internal/models"
	"cloud-system/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LicenseRepo PostgreSQL许可证仓库实现
type LicenseRepo struct {
	pool *pgxpool.Pool
}

// NewLicenseRepo 创建许可证仓库实例
func NewLicenseRepo(pool *pgxpool.Pool) *LicenseRepo {
	repo := &LicenseRepo{
		pool: pool,
	}
	if err := repo.ensureSchema(context.Background()); err != nil {
		log.Printf("warn: ensure license schema failed: %v", err)
	}
	return repo
}

// Create 创建许可证
func (r *LicenseRepo) Create(ctx context.Context, license *models.License) error {
	query := `
		INSERT INTO licenses (
			license_id, cabinet_id, mac_address, issued_at, expires_at, permissions,
			status, created_by, created_at, updated_at, max_devices
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now()
	license.CreatedAt = now
	license.UpdatedAt = now
	license.IssuedAt = now
	license.Status = "active"

	_, err := r.pool.Exec(ctx, query,
		license.LicenseID,
		license.CabinetID,
		license.MACAddress,
		license.IssuedAt,
		license.ExpiresAt,
		license.Permissions,
		license.Status,
		license.CreatedBy,
		license.CreatedAt,
		license.UpdatedAt,
		license.MaxDevices,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "创建许可证失败")
	}

	return nil
}

// GetByCabinetID 根据储能柜ID获取许可证
func (r *LicenseRepo) GetByCabinetID(ctx context.Context, cabinetID string) (*models.License, error) {
	query := `
		SELECT license_id, cabinet_id, mac_address, issued_at, expires_at, permissions,
		       status, revoked_at, revoke_reason, created_by, created_at, updated_at, max_devices
		FROM licenses
		WHERE cabinet_id = $1
	`

	license := &models.License{}
	err := r.pool.QueryRow(ctx, query, cabinetID).Scan(
		&license.LicenseID,
		&license.CabinetID,
		&license.MACAddress,
		&license.IssuedAt,
		&license.ExpiresAt,
		&license.Permissions,
		&license.Status,
		&license.RevokedAt,
		&license.RevokeReason,
		&license.CreatedBy,
		&license.CreatedAt,
		&license.UpdatedAt,
		&license.MaxDevices,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrLicenseNotFound, "许可证不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询许可证失败")
	}

	return license, nil
}

// List 获取许可证列表（支持过滤和分页）
func (r *LicenseRepo) List(ctx context.Context, filter *models.LicenseListFilter) ([]*models.License, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.Status != nil && *filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM licenses %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询许可证总数失败")
	}

	// 查询列表
	listQuery := fmt.Sprintf(`
		SELECT license_id, cabinet_id, mac_address, issued_at, expires_at, permissions,
		       status, revoked_at, revoke_reason, created_by, created_at, updated_at, max_devices
		FROM licenses
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询许可证列表失败")
	}
	defer rows.Close()

	licenses := []*models.License{}
	for rows.Next() {
		license := &models.License{}
		err := rows.Scan(
			&license.LicenseID,
			&license.CabinetID,
			&license.MACAddress,
			&license.IssuedAt,
			&license.ExpiresAt,
			&license.Permissions,
			&license.Status,
			&license.RevokedAt,
			&license.RevokeReason,
			&license.CreatedBy,
			&license.CreatedAt,
			&license.UpdatedAt,
			&license.MaxDevices,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描许可证数据失败")
		}
		licenses = append(licenses, license)
	}

	return licenses, total, nil
}

// Update 更新许可证
func (r *LicenseRepo) Update(ctx context.Context, cabinetID string, license *models.License) error {
	query := `
		UPDATE licenses
		SET permissions = $1, expires_at = $2, max_devices = $3, updated_at = $4
		WHERE cabinet_id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		license.Permissions,
		license.ExpiresAt,
		license.MaxDevices,
		time.Now(),
		cabinetID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新许可证失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrLicenseNotFound, "许可证不存在")
	}

	return nil
}

// Revoke 吊销许可证
func (r *LicenseRepo) Revoke(ctx context.Context, cabinetID string, reason string, revokedBy string) error {
	query := `
		UPDATE licenses
		SET status = 'revoked', revoked_at = $1, revoke_reason = $2, updated_at = $3
		WHERE cabinet_id = $4 AND status != 'revoked'
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, now, reason, now, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "吊销许可证失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrLicenseNotFound, "许可证不存在或已被吊销")
	}

	return nil
}

// Renew 续期许可证
func (r *LicenseRepo) Renew(ctx context.Context, cabinetID string, extendDays int) error {
	query := `
		UPDATE licenses
		SET expires_at = expires_at + INTERVAL '1 day' * $1, updated_at = $2
		WHERE cabinet_id = $3 AND status = 'active'
	`

	result, err := r.pool.Exec(ctx, query, extendDays, time.Now(), cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "续期许可证失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrLicenseNotFound, "许可证不存在或状态不是active")
	}

	return nil
}

// Validate 验证许可证（检查MAC地址和有效期）
func (r *LicenseRepo) Validate(ctx context.Context, cabinetID string, macAddress string) (*models.License, error) {
	query := `
		SELECT license_id, cabinet_id, mac_address, issued_at, expires_at, permissions,
		       status, revoked_at, revoke_reason, created_by, created_at, updated_at, max_devices
		FROM licenses
		WHERE cabinet_id = $1 AND mac_address = $2
	`

	license := &models.License{}
	err := r.pool.QueryRow(ctx, query, cabinetID, macAddress).Scan(
		&license.LicenseID,
		&license.CabinetID,
		&license.MACAddress,
		&license.IssuedAt,
		&license.ExpiresAt,
		&license.Permissions,
		&license.Status,
		&license.RevokedAt,
		&license.RevokeReason,
		&license.CreatedBy,
		&license.CreatedAt,
		&license.UpdatedAt,
		&license.MaxDevices,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrLicenseNotFound, "许可证不存在或MAC地址不匹配")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "验证许可证失败")
	}

	return license, nil
}

// Delete 删除许可证
func (r *LicenseRepo) Delete(ctx context.Context, cabinetID string) error {
	query := `DELETE FROM licenses WHERE cabinet_id = $1`
	result, err := r.pool.Exec(ctx, query, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "删除许可证失败")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrLicenseNotFound, "许可证不存在")
	}
	return nil
}

func (r *LicenseRepo) ensureSchema(ctx context.Context) error {
	migrations := []string{
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS license_id VARCHAR(50)`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS max_devices INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS created_by VARCHAR(100) NOT NULL DEFAULT 'system'`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS revoke_reason TEXT`,
		`UPDATE licenses SET license_id = cabinet_id WHERE (license_id IS NULL OR license_id = '')`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_licenses_license_id ON licenses(license_id)`,
	}

	for _, stmt := range migrations {
		if _, err := r.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
