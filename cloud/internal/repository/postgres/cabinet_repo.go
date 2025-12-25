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

// CabinetRepo PostgreSQL储能柜仓库实现
type CabinetRepo struct {
	pool *pgxpool.Pool
}

// NewCabinetRepo 创建储能柜仓库实例
func NewCabinetRepo(pool *pgxpool.Pool) *CabinetRepo {
	return &CabinetRepo{
		pool: pool,
	}
}

// Create 创建储能柜（用于预注册）
func (r *CabinetRepo) Create(ctx context.Context, cabinet *models.Cabinet) error {
	query := `
		INSERT INTO cabinets (
			cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
			status, latest_vulnerability_score, latest_risk_level, activation_status,
			registration_token, token_expires_at, api_key, api_secret_hash, activated_at,
			ip_address, device_model, notes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	now := time.Now()
	cabinet.CreatedAt = now
	cabinet.UpdatedAt = now

	// 只有当status未设置时才设置默认值
	if cabinet.Status == "" {
		cabinet.Status = "pending" // 默认待激活状态
	}
	if cabinet.LatestRiskLevel == "" {
		cabinet.LatestRiskLevel = "unknown" // 默认未知风险等级
	}
	// 只有当activation_status未设置时才设置默认值
	if cabinet.ActivationStatus == "" {
		cabinet.ActivationStatus = "pending" // 默认待激活
	}

	_, err := r.pool.Exec(ctx, query,
		cabinet.CabinetID,
		cabinet.Name,
		cabinet.Location,
		cabinet.Latitude,
		cabinet.Longitude,
		cabinet.CapacityKwh,
		cabinet.MACAddress,
		cabinet.Status,
		cabinet.LatestVulnerabilityScore,
		cabinet.LatestRiskLevel,
		cabinet.ActivationStatus,
		cabinet.RegistrationToken,
		cabinet.TokenExpiresAt,
		cabinet.APIKey,
		cabinet.APISecretHash,
		cabinet.ActivatedAt,
		cabinet.IPAddress,
		cabinet.DeviceModel,
		cabinet.Notes,
		cabinet.CreatedAt,
		cabinet.UpdatedAt,
	)

	if err != nil {
		// 检查唯一约束冲突
		if err.Error() == "duplicate key value violates unique constraint" {
			return errors.New(errors.ErrRecordExists, "储能柜ID或MAC地址已存在")
		}
		return errors.Wrap(err, errors.ErrDatabaseQuery, "创建储能柜失败")
	}

	return nil
}

// GetByID 根据ID获取储能柜
func (r *CabinetRepo) GetByID(ctx context.Context, cabinetID string) (*models.Cabinet, error) {
	query := `
		SELECT cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
		       status, latest_vulnerability_score, latest_risk_level, vulnerability_updated_at, last_sync_at,
		       activation_status, registration_token, token_expires_at,
		       api_key, api_secret_hash, activated_at, ip_address, device_model, notes,
		       created_at, updated_at
		FROM cabinets
		WHERE cabinet_id = $1
	`

	cabinet := &models.Cabinet{}
	err := r.pool.QueryRow(ctx, query, cabinetID).Scan(
		&cabinet.CabinetID,
		&cabinet.Name,
		&cabinet.Location,
		&cabinet.Latitude,
		&cabinet.Longitude,
		&cabinet.CapacityKwh,
		&cabinet.MACAddress,
		&cabinet.Status,
		&cabinet.LatestVulnerabilityScore,
		&cabinet.LatestRiskLevel,
		&cabinet.VulnerabilityUpdatedAt,
		&cabinet.LastSyncAt,
		&cabinet.ActivationStatus,
		&cabinet.RegistrationToken,
		&cabinet.TokenExpiresAt,
		&cabinet.APIKey,
		&cabinet.APISecretHash,
		&cabinet.ActivatedAt,
		&cabinet.IPAddress,
		&cabinet.DeviceModel,
		&cabinet.Notes,
		&cabinet.CreatedAt,
		&cabinet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜失败")
	}

	return cabinet, nil
}

// GetByMACAddress 根据MAC地址获取储能柜
func (r *CabinetRepo) GetByMACAddress(ctx context.Context, macAddress string) (*models.Cabinet, error) {
	query := `
		SELECT cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
		       status, latest_vulnerability_score, latest_risk_level, vulnerability_updated_at, last_sync_at,
		       activation_status, registration_token, token_expires_at,
		       api_key, api_secret_hash, activated_at, ip_address, device_model, notes,
		       created_at, updated_at
		FROM cabinets
		WHERE mac_address = $1
	`

	cabinet := &models.Cabinet{}
	err := r.pool.QueryRow(ctx, query, macAddress).Scan(
		&cabinet.CabinetID,
		&cabinet.Name,
		&cabinet.Location,
		&cabinet.Latitude,
		&cabinet.Longitude,
		&cabinet.CapacityKwh,
		&cabinet.MACAddress,
		&cabinet.Status,
		&cabinet.LatestVulnerabilityScore,
		&cabinet.LatestRiskLevel,
		&cabinet.VulnerabilityUpdatedAt,
		&cabinet.LastSyncAt,
		&cabinet.ActivationStatus,
		&cabinet.RegistrationToken,
		&cabinet.TokenExpiresAt,
		&cabinet.APIKey,
		&cabinet.APISecretHash,
		&cabinet.ActivatedAt,
		&cabinet.IPAddress,
		&cabinet.DeviceModel,
		&cabinet.Notes,
		&cabinet.CreatedAt,
		&cabinet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜失败")
	}

	return cabinet, nil
}

// List 获取储能柜列表（支持过滤和分页）
func (r *CabinetRepo) List(ctx context.Context, filter *models.CabinetListFilter) ([]*models.Cabinet, int64, error) {
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.Status != nil && *filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Location != nil && *filter.Location != "" {
		whereClause += fmt.Sprintf(" AND location ILIKE $%d", argIndex)
		args = append(args, "%"+*filter.Location+"%")
		argIndex++
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cabinets %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜总数失败")
	}

	// 查询列表
	listQuery := fmt.Sprintf(`
		SELECT cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
		       status, latest_vulnerability_score, latest_risk_level, vulnerability_updated_at, last_sync_at,
		       activation_status, token_expires_at, activated_at,
		       ip_address, device_model, notes,
		       created_at, updated_at
		FROM cabinets
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜列表失败")
	}
	defer rows.Close()

	cabinets := []*models.Cabinet{}
	for rows.Next() {
		cabinet := &models.Cabinet{}
		err := rows.Scan(
			&cabinet.CabinetID,
			&cabinet.Name,
			&cabinet.Location,
			&cabinet.Latitude,
			&cabinet.Longitude,
			&cabinet.CapacityKwh,
			&cabinet.MACAddress,
			&cabinet.Status,
			&cabinet.LatestVulnerabilityScore,
			&cabinet.LatestRiskLevel,
			&cabinet.VulnerabilityUpdatedAt,
			&cabinet.LastSyncAt,
			&cabinet.ActivationStatus,
			&cabinet.TokenExpiresAt,
			&cabinet.ActivatedAt,
			&cabinet.IPAddress,
			&cabinet.DeviceModel,
			&cabinet.Notes,
			&cabinet.CreatedAt,
			&cabinet.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描储能柜数据失败")
		}
		cabinets = append(cabinets, cabinet)
	}

	return cabinets, total, nil
}

// Update 更新储能柜信息
func (r *CabinetRepo) Update(ctx context.Context, cabinetID string, input *models.UpdateCabinetInput) error {
	// 动态构建更新语句
	updateFields := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argIndex := 2

	if input.Name != nil {
		updateFields = append(updateFields, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *input.Name)
		argIndex++
	}

	if input.Location != nil {
		updateFields = append(updateFields, fmt.Sprintf("location = $%d", argIndex))
		args = append(args, *input.Location)
		argIndex++
	}

	if input.Latitude != nil {
		updateFields = append(updateFields, fmt.Sprintf("latitude = $%d", argIndex))
		args = append(args, *input.Latitude)
		argIndex++
	}

	if input.Longitude != nil {
		updateFields = append(updateFields, fmt.Sprintf("longitude = $%d", argIndex))
		args = append(args, *input.Longitude)
		argIndex++
	}

	if input.CapacityKwh != nil {
		updateFields = append(updateFields, fmt.Sprintf("capacity_kwh = $%d", argIndex))
		args = append(args, *input.CapacityKwh)
		argIndex++
	}

	if input.Status != nil {
		updateFields = append(updateFields, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *input.Status)
		argIndex++
	}

	args = append(args, cabinetID)

	query := fmt.Sprintf(`
		UPDATE cabinets
		SET %s
		WHERE cabinet_id = $%d
	`, joinStrings(updateFields, ", "), argIndex)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新储能柜失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// Delete 删除储能柜
func (r *CabinetRepo) Delete(ctx context.Context, cabinetID string) error {
	query := "DELETE FROM cabinets WHERE cabinet_id = $1"

	result, err := r.pool.Exec(ctx, query, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "删除储能柜失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// UpdateVulnerabilityScore 更新脆弱性评分缓存(由vulnerability service调用)
func (r *CabinetRepo) UpdateVulnerabilityScore(ctx context.Context, cabinetID string, score float64, riskLevel string) error {
	query := `
		UPDATE cabinets
		SET latest_vulnerability_score = $1, latest_risk_level = $2, vulnerability_updated_at = $3, updated_at = $4
		WHERE cabinet_id = $5
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, score, riskLevel, now, now, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新脆弱性评分失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// UpdateLastSyncTime 更新最后同步时间
func (r *CabinetRepo) UpdateLastSyncTime(ctx context.Context, cabinetID string) error {
	query := `
		UPDATE cabinets
		SET last_sync_at = $1, updated_at = $2, status = 'active'
		WHERE cabinet_id = $3
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, now, now, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新同步时间失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// Exists 检查储能柜是否存在
func (r *CabinetRepo) Exists(ctx context.Context, cabinetID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM cabinets WHERE cabinet_id = $1)"

	var exists bool
	err := r.pool.QueryRow(ctx, query, cabinetID).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, errors.ErrDatabaseQuery, "检查储能柜存在性失败")
	}

	return exists, nil
}

// GetByRegistrationToken 根据注册Token获取储能柜
func (r *CabinetRepo) GetByRegistrationToken(ctx context.Context, token string) (*models.Cabinet, error) {
	query := `
		SELECT cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
		       status, latest_vulnerability_score, latest_risk_level, vulnerability_updated_at, last_sync_at,
		       activation_status, registration_token, token_expires_at,
		       api_key, api_secret_hash, activated_at, ip_address, device_model, notes,
		       created_at, updated_at
		FROM cabinets
		WHERE registration_token = $1
	`

	cabinet := &models.Cabinet{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&cabinet.CabinetID,
		&cabinet.Name,
		&cabinet.Location,
		&cabinet.Latitude,
		&cabinet.Longitude,
		&cabinet.CapacityKwh,
		&cabinet.MACAddress,
		&cabinet.Status,
		&cabinet.LatestVulnerabilityScore,
		&cabinet.LatestRiskLevel,
		&cabinet.VulnerabilityUpdatedAt,
		&cabinet.LastSyncAt,
		&cabinet.ActivationStatus,
		&cabinet.RegistrationToken,
		&cabinet.TokenExpiresAt,
		&cabinet.APIKey,
		&cabinet.APISecretHash,
		&cabinet.ActivatedAt,
		&cabinet.IPAddress,
		&cabinet.DeviceModel,
		&cabinet.Notes,
		&cabinet.CreatedAt,
		&cabinet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在或Token无效")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜失败")
	}

	return cabinet, nil
}

// GetByAPIKey 根据API Key获取储能柜
func (r *CabinetRepo) GetByAPIKey(ctx context.Context, apiKey string) (*models.Cabinet, error) {
	query := `
		SELECT cabinet_id, name, location, latitude, longitude, capacity_kwh, mac_address,
		       status, latest_vulnerability_score, latest_risk_level, vulnerability_updated_at, last_sync_at,
		       activation_status, registration_token, token_expires_at,
		       api_key, api_secret_hash, activated_at, ip_address, device_model, notes,
		       created_at, updated_at
		FROM cabinets
		WHERE api_key = $1
	`

	cabinet := &models.Cabinet{}
	err := r.pool.QueryRow(ctx, query, apiKey).Scan(
		&cabinet.CabinetID,
		&cabinet.Name,
		&cabinet.Location,
		&cabinet.Latitude,
		&cabinet.Longitude,
		&cabinet.CapacityKwh,
		&cabinet.MACAddress,
		&cabinet.Status,
		&cabinet.LatestVulnerabilityScore,
		&cabinet.LatestRiskLevel,
		&cabinet.VulnerabilityUpdatedAt,
		&cabinet.LastSyncAt,
		&cabinet.ActivationStatus,
		&cabinet.RegistrationToken,
		&cabinet.TokenExpiresAt,
		&cabinet.APIKey,
		&cabinet.APISecretHash,
		&cabinet.ActivatedAt,
		&cabinet.IPAddress,
		&cabinet.DeviceModel,
		&cabinet.Notes,
		&cabinet.CreatedAt,
		&cabinet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在或API Key无效")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜失败")
	}

	return cabinet, nil
}

// UpdateActivation 更新激活信息
func (r *CabinetRepo) UpdateActivation(ctx context.Context, cabinetID string, apiKey, apiSecretHash string) error {
	query := `
		UPDATE cabinets
		SET activation_status = 'activated',
		    api_key = $1,
		    api_secret_hash = $2,
		    activated_at = $3,
		    status = 'active',
		    registration_token = NULL,
		    token_expires_at = NULL,
		    updated_at = $4
		WHERE cabinet_id = $5
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, apiKey, apiSecretHash, now, now, cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新激活信息失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// UpdateRegistrationToken 更新注册Token
func (r *CabinetRepo) UpdateRegistrationToken(ctx context.Context, cabinetID, token string, expiresAt interface{}) error {
	query := `
		UPDATE cabinets
		SET registration_token = $1,
		    token_expires_at = $2,
		    updated_at = $3
		WHERE cabinet_id = $4
	`

	result, err := r.pool.Exec(ctx, query, token, expiresAt, time.Now(), cabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新注册Token失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	return nil
}

// GetLocations 获取所有储能柜位置信息（用于地图展示）
func (r *CabinetRepo) GetLocations(ctx context.Context) ([]*models.CabinetLocation, error) {
	query := `
		SELECT cabinet_id, name, location, latitude, longitude, status
		FROM cabinets
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜位置列表失败")
	}
	defer rows.Close()

	locations := []*models.CabinetLocation{}
	for rows.Next() {
		location := &models.CabinetLocation{}
		err := rows.Scan(
			&location.CabinetID,
			&location.Name,
			&location.Location,
			&location.Latitude,
			&location.Longitude,
			&location.Status,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描储能柜位置数据失败")
		}
		locations = append(locations, location)
	}

	return locations, nil
}

// GetStatistics 获取储能柜统计信息
func (r *CabinetRepo) GetStatistics(ctx context.Context) (*models.CabinetStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_cabinets,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_cabinets,
			COUNT(CASE WHEN status = 'offline' THEN 1 END) as offline_cabinets,
			COUNT(CASE WHEN status = 'inactive' THEN 1 END) as inactive_cabinets,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_cabinets,
			COUNT(CASE WHEN status = 'maintenance' THEN 1 END) as maintenance_cabinets,
			COUNT(CASE WHEN activation_status = 'activated' THEN 1 END) as activated_cabinets
		FROM cabinets
	`

	stats := &models.CabinetStatistics{}
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalCabinets,
		&stats.ActiveCabinets,
		&stats.OfflineCabinets,
		&stats.InactiveCabinets,
		&stats.PendingCabinets,
		&stats.MaintenanceCabinets,
		&stats.ActivatedCabinets,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询储能柜统计信息失败")
	}

	return stats, nil
}

// joinStrings 连接字符串数组（辅助函数）
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
