package timescaledb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cloud-system/internal/models"
	"cloud-system/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SensorDataRepo TimescaleDB传感器数据仓库实现
type SensorDataRepo struct {
	pool *pgxpool.Pool
}

// NewSensorDataRepo 创建传感器数据仓库实例
func NewSensorDataRepo(pool *pgxpool.Pool) *SensorDataRepo {
	return &SensorDataRepo{
		pool: pool,
	}
}

// Insert 插入单条传感器数据（MQTT实时数据）
func (r *SensorDataRepo) Insert(ctx context.Context, data *models.SensorData, cabinetID, sensorType string) error {
	// 注意：sensor_data表使用time列（TimescaleDB要求），且没有status列
	// status信息可以通过quality值计算得出，这里不存储
	// TimescaleDB hypertable可能没有唯一约束，直接插入即可
	query := `
		INSERT INTO sensor_data (device_id, time, value, quality, cabinet_id, sensor_type)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		data.DeviceID,
		data.Timestamp,
		data.Value,
		data.Quality,
		cabinetID,
		sensorType,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "插入传感器数据失败")
	}

	return nil
}

// BatchInsert 批量插入传感器数据
// 注意：此方法已废弃，因为需要cabinet_id和sensor_type，但接口无法提供
// 请使用Insert方法逐个插入，或修改接口签名
func (r *SensorDataRepo) BatchInsert(ctx context.Context, data []models.SensorData) error {
	// 此方法已废弃，保留仅为兼容性
	// 实际应该使用Insert方法，因为需要cabinet_id和sensor_type
	return fmt.Errorf("BatchInsert已废弃，请使用Insert方法")
}

// GetLatestByCabinetID 获取储能柜的所有传感器最新数据
func (r *SensorDataRepo) GetLatestByCabinetID(ctx context.Context, cabinetID string) ([]*models.LatestSensorData, error) {
	// 注意：sensor_data表使用time列，且没有status列
	// status根据quality值计算：quality >= 80为normal，50-80为warning，<50为error
	// 使用LEFT JOIN LATERAL，即使没有数据也返回所有传感器设备
	// 修复：使用device_name列（sensor_devices表的实际列名）
	query := `
		SELECT 
			s.device_id,
			s.sensor_type,
			s.device_name,
			s.unit,
			COALESCE(sd.value, 0) AS value,
			COALESCE(sd.quality, 0) AS quality,
			CASE 
				WHEN sd.quality IS NULL THEN 'offline'
				WHEN sd.quality >= 80 THEN 'normal'
				WHEN sd.quality >= 50 THEN 'warning'
				ELSE 'error'
			END AS status,
			COALESCE(sd.time, NOW()) AS timestamp
		FROM sensor_devices s
		LEFT JOIN LATERAL (
			SELECT device_id, value, quality, time
			FROM sensor_data
			WHERE device_id = s.device_id
			ORDER BY time DESC
			LIMIT 1
		) sd ON s.device_id = sd.device_id
		WHERE s.cabinet_id = $1
		ORDER BY s.sensor_type
	`

	rows, err := r.pool.Query(ctx, query, cabinetID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询最新传感器数据失败")
	}
	defer rows.Close()

	result := []*models.LatestSensorData{}
	for rows.Next() {
		data := &models.LatestSensorData{}
		var value sql.NullFloat64
		var quality sql.NullFloat64 // 改为Float64，因为数据库是DECIMAL(5,2)
		var timestamp sql.NullTime

		err := rows.Scan(
			&data.DeviceID,
			&data.SensorType,
			&data.Name,
			&data.Unit,
			&value,
			&quality,
			&data.Status,
			&timestamp,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描传感器数据失败")
		}

		// 处理NULL值
		if value.Valid {
			data.Value = value.Float64
		} else {
			data.Value = 0 // 没有数据时显示0
		}

		if quality.Valid {
			data.Quality = int(quality.Float64) // 从Float64转换为int
		} else {
			data.Quality = 0 // 没有数据时质量为0
		}

		if timestamp.Valid {
			data.Timestamp = timestamp.Time
		} else {
			data.Timestamp = time.Now() // 没有数据时使用当前时间
		}

		result = append(result, data)
	}

	return result, nil
}

// GetHistoricalData 获取历史数据（原始数据）
func (r *SensorDataRepo) GetHistoricalData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.SensorData, int64, error) {
	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM sensor_data
		WHERE device_id = $1 
		  AND time >= $2 
		  AND time <= $3
	`

	var total int64
	err := r.pool.QueryRow(ctx, countQuery, query.DeviceID, query.StartTime, query.EndTime).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询数据总数失败")
	}

	// 查询数据
	// 注意：sensor_data表使用time列，且没有status列
	dataQuery := `
		SELECT device_id, time AS timestamp, value, quality,
		       CASE 
		           WHEN quality >= 80 THEN 'normal'
		           WHEN quality >= 50 THEN 'warning'
		           ELSE 'error'
		       END AS status
		FROM sensor_data
		WHERE device_id = $1 
		  AND time >= $2 
		  AND time <= $3
		ORDER BY time DESC
		LIMIT $4 OFFSET $5
	`

	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	rows, err := r.pool.Query(ctx, dataQuery,
		query.DeviceID,
		query.StartTime,
		query.EndTime,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "查询历史数据失败")
	}
	defer rows.Close()

	data := []*models.SensorData{}
	for rows.Next() {
		d := &models.SensorData{}
		err := rows.Scan(
			&d.DeviceID,
			&d.Timestamp,
			&d.Value,
			&d.Quality,
			&d.Status,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描历史数据失败")
		}
		data = append(data, d)
	}

	return data, total, nil
}

// GetAggregatedData 获取聚合数据
func (r *SensorDataRepo) GetAggregatedData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.AggregatedData, error) {
	// 根据聚合方式确定时间桶
	var timeBucket string
	switch query.Aggregation {
	case "1m":
		timeBucket = "1 minute"
	case "5m":
		timeBucket = "5 minutes"
	case "1h":
		timeBucket = "1 hour"
	case "1d":
		timeBucket = "1 day"
	default:
		timeBucket = "5 minutes" // 默认5分钟
	}

	// 使用TimescaleDB的time_bucket函数进行时间聚合
	dataQuery := fmt.Sprintf(`
		SELECT 
			time_bucket('%s', time) AS bucket,
			AVG(value) AS avg_value,
			MIN(value) AS min_value,
			MAX(value) AS max_value,
			COUNT(*) AS count
		FROM sensor_data
		WHERE device_id = $1 
		  AND time >= $2 
		  AND time <= $3
		GROUP BY bucket
		ORDER BY bucket DESC
	`, timeBucket)

	rows, err := r.pool.Query(ctx, dataQuery,
		query.DeviceID,
		query.StartTime,
		query.EndTime,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "查询聚合数据失败")
	}
	defer rows.Close()

	data := []*models.AggregatedData{}
	for rows.Next() {
		d := &models.AggregatedData{}
		err := rows.Scan(
			&d.Timestamp,
			&d.AvgValue,
			&d.MinValue,
			&d.MaxValue,
			&d.Count,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "扫描聚合数据失败")
		}
		data = append(data, d)
	}

	return data, nil
}

// DeleteOldData 删除旧数据（数据清理）
func (r *SensorDataRepo) DeleteOldData(ctx context.Context, beforeTime time.Time) error {
	query := `
		DELETE FROM sensor_data
		WHERE time < $1
	`

	result, err := r.pool.Exec(ctx, query, beforeTime)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "删除旧数据失败")
	}

	_ = result.RowsAffected() // 记录删除的行数用于日志

	return nil
}
