package postgres

import (
	"context"
	"fmt"
	"strings"

	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// initializeHypertables 初始化TimescaleDB Hypertables
// SQL来源: migrations/FULL_INIT.sql (Lines 352-368)
func (c *Client) initializeHypertables(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// 检查TimescaleDB是否可用
	if !isTimescaleDBAvailable(ctx, conn) {
		utils.Warn("TimescaleDB extension not installed, skipping hypertables initialization")
		return nil // 不算错误，允许在非TimescaleDB环境运行
	}

	// 创建3个Hypertable
	hypertables := []struct {
		table    string
		timeCol  string
		interval string
	}{
		{"sensor_data", "time", "1 day"},
		{"health_scores", "time", "7 days"},
		{"vulnerability_assessments", "timestamp", "7 days"},
	}

	successCount := 0
	for _, ht := range hypertables {
		if err := createHypertable(ctx, conn, ht.table, ht.timeCol, ht.interval); err != nil {
			// Hypertable创建失败记录警告但继续
			utils.Warn("Failed to create hypertable",
				zap.String("table", ht.table),
				zap.Error(err))
			continue
		}
		successCount++
	}

	utils.Info("Hypertables initialized",
		zap.Int("total", len(hypertables)),
		zap.Int("success", successCount))

	return nil
}

// isTimescaleDBAvailable 检查TimescaleDB扩展是否可用
func isTimescaleDBAvailable(ctx context.Context, conn *pgxpool.Conn) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')`

	err := conn.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		utils.Warn("Failed to check TimescaleDB availability", zap.Error(err))
		return false
	}

	return exists
}

// isHypertable 检查表是否已经是hypertable
func isHypertable(ctx context.Context, conn *pgxpool.Conn, tableName string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM timescaledb_information.hypertables
            WHERE hypertable_name = $1
        )
    `

	err := conn.QueryRow(ctx, query, tableName).Scan(&exists)
	if err != nil {
		// 如果查询失败可能是timescaledb_information schema不存在
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}

// createHypertable 创建单个hypertable
func createHypertable(ctx context.Context, conn *pgxpool.Conn, table, timeCol, interval string) error {
	// 检查表是否已经是hypertable
	isHT, err := isHypertable(ctx, conn, table)
	if err != nil {
		return fmt.Errorf("failed to check hypertable status: %w", err)
	}

	if isHT {
		utils.Info("Hypertable already exists", zap.String("table", table))
		return nil
	}

	// 创建hypertable
	createQuery := fmt.Sprintf(`
        SELECT create_hypertable('%s', '%s',
            chunk_time_interval => INTERVAL '%s',
            if_not_exists => TRUE
        )
    `, table, timeCol, interval)

	if _, err := conn.Exec(ctx, createQuery); err != nil {
		// 如果表已经是hypertable，不算错误
		if strings.Contains(err.Error(), "already a hypertable") {
			utils.Info("Table is already a hypertable", zap.String("table", table))
			return nil
		}
		return fmt.Errorf("failed to create hypertable: %w", err)
	}

	utils.Info("Hypertable created",
		zap.String("table", table),
		zap.String("time_column", timeCol),
		zap.String("chunk_interval", interval))

	return nil
}
