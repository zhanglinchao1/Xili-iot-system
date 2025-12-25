package postgres

import (
	"context"
	"time"

	"cloud-system/internal/models"
	"cloud-system/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SensorDeviceRepo PostgreSQL传感器设备仓库实现
type SensorDeviceRepo struct {
	pool *pgxpool.Pool
}

// NewSensorDeviceRepo 创建传感器设备仓库实例
func NewSensorDeviceRepo(pool *pgxpool.Pool) *SensorDeviceRepo {
	return &SensorDeviceRepo{
		pool: pool,
	}
}

// Create 创建传感器设备
func (r *SensorDeviceRepo) Create(ctx context.Context, device *models.SensorDevice) error {
	// 注意：数据库表使用 device_name 列而不是 name
	query := `
		INSERT INTO sensor_devices (
			device_id, cabinet_id, sensor_type, device_name, unit, 
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	device.Status = "active" // 默认激活状态

	_, err := r.pool.Exec(ctx, query,
		device.DeviceID,
		device.CabinetID,
		device.SensorType,
		device.Name,
		device.Unit,
		device.Status,
		device.CreatedAt,
		device.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "创建传感器设备失败")
	}

	return nil
}

// GetByID 根据ID获取传感器设备
func (r *SensorDeviceRepo) GetByID(ctx context.Context, deviceID string) (*models.SensorDevice, error) {
	// 注意：数据库表使用 device_name 列而不是 name
	query := `
		SELECT device_id, cabinet_id, sensor_type, device_name, unit, 
		       status, created_at, updated_at
		FROM sensor_devices
		WHERE device_id = $1
	`

	device := &models.SensorDevice{}
	var description *string // 占位符，数据库中没有此列
	err := r.pool.QueryRow(ctx, query, deviceID).Scan(
		&device.DeviceID,
		&device.CabinetID,
		&device.SensorType,
		&device.Name,
		&device.Unit,
		&device.Status,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	// 设置Description为nil，因为数据库中没有此列
	device.Description = nil
	_ = description // 避免未使用变量警告

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrNotFound, "传感器设备不存在")
		}
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询传感器设备失败")
	}

	return device, nil
}

// ListByCabinetID 获取储能柜的所有传感器设备
func (r *SensorDeviceRepo) ListByCabinetID(ctx context.Context, cabinetID string) ([]*models.SensorDevice, error) {
	query := `
		WITH latest_data AS (
			SELECT DISTINCT ON (device_id) 
				device_id, value, time
			FROM sensor_data
			WHERE cabinet_id = $1
			ORDER BY device_id, time DESC
		)
		SELECT 
			d.device_id, d.cabinet_id, d.sensor_type, d.device_name, d.unit,
			d.status, d.created_at, d.updated_at,
			ld.value AS last_value,
			ld.time AS last_reading_at
		FROM sensor_devices d
		LEFT JOIN latest_data ld ON d.device_id = ld.device_id
		WHERE d.cabinet_id = $1
		ORDER BY d.sensor_type, d.created_at
	`

	rows, err := r.pool.Query(ctx, query, cabinetID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询传感器设备列表失败")
	}
	defer rows.Close()

	devices := []*models.SensorDevice{}
	for rows.Next() {
		device := &models.SensorDevice{}
		var lastValue *float64
		var lastReadingAt *time.Time
		err := rows.Scan(
			&device.DeviceID,
			&device.CabinetID,
			&device.SensorType,
			&device.Name,
			&device.Unit,
			&device.Status,
			&device.CreatedAt,
			&device.UpdatedAt,
			&lastValue,
			&lastReadingAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描传感器设备数据失败")
		}
		device.Description = nil
		if lastValue != nil {
			device.LastValue = lastValue
		}
		if lastReadingAt != nil {
			device.LastReadingAt = lastReadingAt
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// Update 更新传感器设备
func (r *SensorDeviceRepo) Update(ctx context.Context, deviceID string, device *models.SensorDevice) error {
	// 注意：数据库表使用 device_name 列而不是 name
	query := `
		UPDATE sensor_devices
		SET device_name = $1, unit = $2, status = $3, updated_at = $4
		WHERE device_id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		device.Name,
		device.Unit,
		device.Status,
		time.Now(),
		deviceID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "更新传感器设备失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "传感器设备不存在")
	}

	return nil
}

// Delete 删除传感器设备
func (r *SensorDeviceRepo) Delete(ctx context.Context, deviceID string) error {
	query := "DELETE FROM sensor_devices WHERE device_id = $1"

	result, err := r.pool.Exec(ctx, query, deviceID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "删除传感器设备失败")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrNotFound, "传感器设备不存在")
	}

	return nil
}

// Exists 检查设备是否存在
func (r *SensorDeviceRepo) Exists(ctx context.Context, deviceID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM sensor_devices WHERE device_id = $1)"

	var exists bool
	err := r.pool.QueryRow(ctx, query, deviceID).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, errors.ErrDatabaseQuery, "检查传感器设备存在性失败")
	}

	return exists, nil
}
