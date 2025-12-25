package postgres

import (
	"context"
	"testing"
	"time"

	"cloud-system/internal/repository/postgres/testutils"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitSchema_AllTablesCreated 测试所有14张表都被创建
func TestInitSchema_AllTablesCreated(t *testing.T) {
	ctx := context.Background()

	// 启动PostgreSQL测试容器（带TimescaleDB）
	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err, "Failed to start test container")
	defer container.Close(ctx)

	// 创建连接池
	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err, "Failed to create connection pool")
	defer pool.Close()

	// 执行schema初始化
	err = InitSchema(ctx, pool)
	require.NoError(t, err, "InitSchema should succeed")

	// 验证14张表都存在
	expectedTables := []string{
		"cabinets",
		"users",
		"sensor_devices",
		"sensor_data",
		"alerts",
		"commands",
		"licenses",
		"audit_logs",
		"health_scores",
		"vulnerability_assessments",
		"vulnerability_events",
		"access_policies",
		"access_logs",
		"policy_distribution_logs",
	}

	for _, tableName := range expectedTables {
		exists := tableExists(ctx, t, pool, tableName)
		assert.True(t, exists, "Table %s should exist", tableName)
	}
}

// TestInitSchema_CabinetsTableStructure 测试cabinets表结构的正确性
func TestInitSchema_CabinetsTableStructure(t *testing.T) {
	ctx := context.Background()

	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	err = InitSchema(ctx, pool)
	require.NoError(t, err)

	// 验证cabinets表的关键字段
	expectedColumns := []string{
		"cabinet_id",
		"name",
		"mac_address",
		"status",
		"activation_status",
		"registration_token",
		"api_key",
		"latest_vulnerability_score",
		"created_at",
		"updated_at",
	}

	for _, colName := range expectedColumns {
		exists := columnExists(ctx, t, pool, "cabinets", colName)
		assert.True(t, exists, "Column cabinets.%s should exist", colName)
	}

	// 验证主键
	hasPrimaryKey := checkPrimaryKey(ctx, t, pool, "cabinets", "cabinet_id")
	assert.True(t, hasPrimaryKey, "cabinets should have primary key on cabinet_id")
}

// TestInitSchema_IndexesCreated 测试索引的创建
func TestInitSchema_IndexesCreated(t *testing.T) {
	ctx := context.Background()

	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	err = InitSchema(ctx, pool)
	require.NoError(t, err)

	// 验证部分关键索引存在
	keyIndexes := []string{
		"idx_cabinets_status",
		"idx_cabinets_mac_address",
		"idx_users_username",
		"idx_alerts_cabinet",
		"idx_sensor_data_cabinet_time",
		"idx_va_cabinet_timestamp",
	}

	for _, indexName := range keyIndexes {
		exists := indexExists(ctx, t, pool, indexName)
		assert.True(t, exists, "Index %s should exist", indexName)
	}

	// 统计总索引数量（应该有60+个）
	totalIndexes := countIndexes(ctx, t, pool)
	assert.GreaterOrEqual(t, totalIndexes, 60, "Should have at least 60 indexes")
}

// TestInitSchema_TriggersWorking 测试触发器的功能
func TestInitSchema_TriggersWorking(t *testing.T) {
	ctx := context.Background()

	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	err = InitSchema(ctx, pool)
	require.NoError(t, err)

	// 插入一条cabinet记录
	_, err = pool.Exec(ctx, `
		INSERT INTO cabinets (cabinet_id, name, mac_address, status)
		VALUES ('test-001', 'Test Cabinet', 'AA:BB:CC:DD:EE:FF', 'active')
	`)
	require.NoError(t, err)

	// 获取初始created_at和updated_at
	var createdAt, updatedAtBefore time.Time
	err = pool.QueryRow(ctx, `
		SELECT created_at, updated_at FROM cabinets WHERE cabinet_id = 'test-001'
	`).Scan(&createdAt, &updatedAtBefore)
	require.NoError(t, err)

	// 等待1秒确保时间戳不同
	time.Sleep(1 * time.Second)

	// 更新记录
	_, err = pool.Exec(ctx, `
		UPDATE cabinets SET name = 'Updated Cabinet' WHERE cabinet_id = 'test-001'
	`)
	require.NoError(t, err)

	// 获取更新后的updated_at
	var updatedAtAfter time.Time
	err = pool.QueryRow(ctx, `
		SELECT updated_at FROM cabinets WHERE cabinet_id = 'test-001'
	`).Scan(&updatedAtAfter)
	require.NoError(t, err)

	// 验证触发器自动更新了updated_at
	assert.True(t, updatedAtAfter.After(updatedAtBefore),
		"Trigger should auto-update updated_at timestamp")
}

// TestInitSchema_HypertablesCreated 测试Hypertable的创建
func TestInitSchema_HypertablesCreated(t *testing.T) {
	ctx := context.Background()

	// 使用TimescaleDB容器
	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	err = InitSchema(ctx, pool)
	require.NoError(t, err)

	// 验证成功转换的2个Hypertable
	// 注意：vulnerability_assessments由于主键约束无法转换为Hypertable，保持为普通表
	expectedHypertables := []string{
		"sensor_data",
		"health_scores",
	}

	for _, tableName := range expectedHypertables {
		isHypertable := checkIsHypertable(ctx, t, pool, tableName)
		assert.True(t, isHypertable, "Table %s should be a hypertable", tableName)
	}

	// 验证vulnerability_assessments仍然存在（作为普通表）
	exists := tableExists(ctx, t, pool, "vulnerability_assessments")
	assert.True(t, exists, "vulnerability_assessments should exist as regular table")
}

// TestInitSchema_DefaultDataInserted 测试初始数据的插入
func TestInitSchema_DefaultDataInserted(t *testing.T) {
	ctx := context.Background()

	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	err = InitSchema(ctx, pool)
	require.NoError(t, err)

	// 验证admin用户存在
	var userCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE username = 'admin' AND role = 'admin'
	`).Scan(&userCount)
	require.NoError(t, err)
	assert.Equal(t, 1, userCount, "Should have exactly one admin user")

	// 验证5条ABAC策略存在
	var policyCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM access_policies
	`).Scan(&policyCount)
	require.NoError(t, err)
	assert.Equal(t, 5, policyCount, "Should have exactly 5 ABAC policies")

	// 验证具体策略存在
	expectedPolicies := []string{
		"policy_admin_full",
		"policy_user_readonly",
		"policy_cabinet_sync",
		"policy_cabinet_limited",
		"policy_device_high_quality",
	}

	for _, policyID := range expectedPolicies {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM access_policies WHERE id = $1)
		`, policyID).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Policy %s should exist", policyID)
	}
}

// TestInitSchema_Idempotency 测试幂等性（多次运行不出错）
func TestInitSchema_Idempotency(t *testing.T) {
	ctx := context.Background()

	container, err := testutils.NewPostgresTestContainer(ctx, true)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	// 第一次运行
	err = InitSchema(ctx, pool)
	require.NoError(t, err, "First InitSchema should succeed")

	// 第二次运行（应该不出错）
	err = InitSchema(ctx, pool)
	require.NoError(t, err, "Second InitSchema should succeed (idempotent)")

	// 第三次运行（应该不出错）
	err = InitSchema(ctx, pool)
	require.NoError(t, err, "Third InitSchema should succeed (idempotent)")

	// 验证数据没有重复插入
	var userCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE username = 'admin'`).Scan(&userCount)
	require.NoError(t, err)
	assert.Equal(t, 1, userCount, "Admin user should not be duplicated")

	var policyCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM access_policies`).Scan(&policyCount)
	require.NoError(t, err)
	assert.Equal(t, 5, policyCount, "Policies should not be duplicated")
}

// TestInitSchema_WithoutTimescaleDB 测试在没有TimescaleDB的情况下的降级行为
func TestInitSchema_WithoutTimescaleDB(t *testing.T) {
	ctx := context.Background()

	// 使用普通PostgreSQL容器（不带TimescaleDB）
	container, err := testutils.NewPostgresTestContainer(ctx, false)
	require.NoError(t, err)
	defer container.Close(ctx)

	pool, err := pgxpool.New(ctx, container.GetConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	// InitSchema应该成功（即使没有TimescaleDB）
	err = InitSchema(ctx, pool)
	require.NoError(t, err, "InitSchema should succeed without TimescaleDB")

	// 验证表仍然被创建（作为普通表）
	exists := tableExists(ctx, t, pool, "sensor_data")
	assert.True(t, exists, "sensor_data should exist as regular table")

	exists = tableExists(ctx, t, pool, "health_scores")
	assert.True(t, exists, "health_scores should exist as regular table")
}

// ========== 辅助函数 ==========

// tableExists 检查表是否存在
func tableExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tableName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`
	err := pool.QueryRow(ctx, query, tableName).Scan(&exists)
	require.NoError(t, err, "Failed to check table existence")
	return exists
}

// columnExists 检查列是否存在
func columnExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tableName, columnName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
		)
	`
	err := pool.QueryRow(ctx, query, tableName, columnName).Scan(&exists)
	require.NoError(t, err, "Failed to check column existence")
	return exists
}

// indexExists 检查索引是否存在
func indexExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, indexName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = 'public' AND indexname = $1
		)
	`
	err := pool.QueryRow(ctx, query, indexName).Scan(&exists)
	require.NoError(t, err, "Failed to check index existence")
	return exists
}

// countIndexes 统计索引总数
func countIndexes(ctx context.Context, t *testing.T, pool *pgxpool.Pool) int {
	var count int
	query := `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public'`
	err := pool.QueryRow(ctx, query).Scan(&count)
	require.NoError(t, err, "Failed to count indexes")
	return count
}

// checkPrimaryKey 检查主键是否存在
func checkPrimaryKey(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tableName, columnName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
			WHERE tc.table_schema = 'public'
				AND tc.table_name = $1
				AND kcu.column_name = $2
				AND tc.constraint_type = 'PRIMARY KEY'
		)
	`
	err := pool.QueryRow(ctx, query, tableName, columnName).Scan(&exists)
	require.NoError(t, err, "Failed to check primary key")
	return exists
}

// checkIsHypertable 检查表是否是Hypertable
func checkIsHypertable(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tableName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables
			WHERE hypertable_name = $1
		)
	`
	err := pool.QueryRow(ctx, query, tableName).Scan(&exists)
	if err != nil {
		// TimescaleDB可能未安装，返回false
		return false
	}
	return exists
}
