/*
 * PostgreSQL数据库Schema初始化模块
 * 负责创建所有数据库表、扩展、索引和触发器
 * SQL来源: migrations/FULL_INIT.sql
 */
package postgres

import (
	"context"
	"fmt"
	"strings"

	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// createExtensions 创建PostgreSQL扩展
// 来源: FULL_INIT.sql 行7-14
func createExtensions(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	// 创建TimescaleDB扩展
	_, err := conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE")
	if err != nil {
		// TimescaleDB可能未安装，记录警告但继续执行
		logger.Warn("Failed to create TimescaleDB extension, hypertable features will be disabled",
			zap.Error(err))
	} else {
		logger.Info("TimescaleDB extension created")
	}

	// 创建uuid-ossp扩展
	_, err = conn.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	// 创建pg_trgm扩展(用于文本搜索)
	_, err = conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_trgm")
	if err != nil {
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	logger.Info("PostgreSQL extensions created successfully")
	return nil
}

// createTables 在事务中创建所有表
func createTables(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 按顺序创建所有表
	tables := []struct {
		name string
		sql  string
	}{
		{"cabinets", createCabinetsTable()},
		{"users", createUsersTable()},
		{"sensor_devices", createSensorDevicesTable()},
		{"sensor_data", createSensorDataTable()},
		{"alerts", createAlertsTable()},
		{"commands", createCommandsTable()},
		{"licenses", createLicensesTable()},
		{"audit_logs", createAuditLogsTable()},
		{"health_scores", createHealthScoresTable()},
		{"vulnerability_assessments", createVulnerabilityAssessmentsTable()},
		{"vulnerability_events", createVulnerabilityEventsTable()},
		{"access_policies", createAccessPoliciesTable()},
		{"access_logs", createAccessLogsTable()},
		{"policy_distribution_logs", createPolicyDistributionLogsTable()},
	}

	for _, table := range tables {
		logger.Info("Creating table", zap.String("table", table.name))
		if _, err := tx.Exec(ctx, table.sql); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("All tables created successfully", zap.Int("table_count", len(tables)))
	return nil
}

// createCabinetsTable 创建储能柜表
// 来源: FULL_INIT.sql 行20-71
func createCabinetsTable() string {
	return `
CREATE TABLE IF NOT EXISTS cabinets (
    cabinet_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    location VARCHAR(500),
    latitude DECIMAL(10, 6),
    longitude DECIMAL(10, 6),
    address VARCHAR(500),
    capacity_kwh DECIMAL(10, 2),
    mac_address VARCHAR(17) UNIQUE NOT NULL,

    -- 状态字段
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    activation_status VARCHAR(20) DEFAULT 'pending',

    -- 激活相关字段
    registration_token VARCHAR(500),
    token_expires_at TIMESTAMP WITH TIME ZONE,
    api_key VARCHAR(100),
    api_secret_hash VARCHAR(255),
    activated BOOLEAN DEFAULT FALSE,
    activated_at TIMESTAMP WITH TIME ZONE,

    -- 设备信息
    ip_address VARCHAR(45),
    device_model VARCHAR(100),
    notes TEXT,

    -- 脆弱性评分字段(替代health_score)
    latest_vulnerability_score FLOAT DEFAULT 0,
    latest_risk_level TEXT DEFAULT 'unknown',
    vulnerability_updated_at TIMESTAMP,

    -- 时间戳
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT valid_cabinet_status CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance')),
    CONSTRAINT valid_activation_status CHECK (activation_status IN ('pending', 'activated')),
    CONSTRAINT valid_latitude CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT valid_longitude CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

COMMENT ON TABLE cabinets IS '储能柜基本信息表';
COMMENT ON COLUMN cabinets.capacity_kwh IS '储能容量(kWh)';
COMMENT ON COLUMN cabinets.status IS '状态: pending(待激活), active(在线), inactive(未激活), offline(离线), maintenance(维护中)';
COMMENT ON COLUMN cabinets.activation_status IS '激活状态: pending-待激活, activated-已激活';
COMMENT ON COLUMN cabinets.registration_token IS '注册Token,用于首次激活,24小时有效';
COMMENT ON COLUMN cabinets.api_key IS 'Edge端API密钥(激活后生成)';
COMMENT ON COLUMN cabinets.latest_vulnerability_score IS '最新脆弱性评分(0-100)';
COMMENT ON COLUMN cabinets.latest_risk_level IS '最新风险等级';
`
}

// createUsersTable 创建用户表
// 来源: FULL_INIT.sql 行72-91
func createUsersTable() string {
	return `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    email VARCHAR(128),
    role VARCHAR(32) DEFAULT 'user',
    status VARCHAR(16) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_login_at TIMESTAMPTZ,

    CONSTRAINT valid_user_role CHECK (role IN ('admin', 'user', 'viewer')),
    CONSTRAINT valid_user_status CHECK (status IN ('active', 'disabled'))
);

COMMENT ON TABLE users IS '系统用户表';
COMMENT ON COLUMN users.password_hash IS 'bcrypt哈希后的密码';
COMMENT ON COLUMN users.role IS '用户角色: admin(管理员)/user(普通用户)/viewer(只读)';
COMMENT ON COLUMN users.status IS '用户状态: active(激活)/disabled(禁用)';
`
}

// createSensorDevicesTable 创建传感器设备表
// 来源: FULL_INIT.sql 行93-114
func createSensorDevicesTable() string {
	return `
CREATE TABLE IF NOT EXISTS sensor_devices (
    device_id VARCHAR(50) PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    device_name VARCHAR(200),
    sensor_type VARCHAR(50) NOT NULL,
    location VARCHAR(200),
    unit VARCHAR(20),
    min_value DECIMAL(15, 6),
    max_value DECIMAL(15, 6),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_reading_at TIMESTAMP WITH TIME ZONE,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_device_status CHECK (status IN ('online', 'offline', 'active', 'inactive', 'error', 'maintenance'))
);

COMMENT ON TABLE sensor_devices IS '传感器设备表';
COMMENT ON COLUMN sensor_devices.sensor_type IS '传感器类型: co2, co, smoke, temperature等';
`
}

// createSensorDataTable 创建传感器数据表(时序数据)
// 来源: FULL_INIT.sql 行116-131
func createSensorDataTable() string {
	return `
CREATE TABLE IF NOT EXISTS sensor_data (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    cabinet_id VARCHAR(50) NOT NULL,
    device_id VARCHAR(50) NOT NULL,
    sensor_type VARCHAR(50) NOT NULL,
    value DECIMAL(15, 6) NOT NULL,
    unit VARCHAR(20),
    quality DECIMAL(5, 2) DEFAULT 100.00,
    raw_value JSONB,

    CONSTRAINT valid_quality CHECK (quality >= 0 AND quality <= 100)
);

COMMENT ON TABLE sensor_data IS '传感器时序数据表';
COMMENT ON COLUMN sensor_data.quality IS '数据质量指标(0-100)';
`
}

// createAlertsTable 创建告警表
// 来源: FULL_INIT.sql 行133-157
func createAlertsTable() string {
	return `
CREATE TABLE IF NOT EXISTS alerts (
    alert_id BIGSERIAL PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    device_id VARCHAR(50),
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    edge_alert_id BIGINT,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_alert_severity CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    CONSTRAINT valid_alert_status CHECK (status IN ('active', 'resolved', 'acknowledged'))
);

COMMENT ON TABLE alerts IS '告警记录表';
COMMENT ON COLUMN alerts.severity IS '告警严重度: info, warning, error, critical';
COMMENT ON COLUMN alerts.edge_alert_id IS 'Edge端告警ID,用于命令下发时定位Edge端数据';
`
}

// createCommandsTable 创建命令表
// 来源: FULL_INIT.sql 行159-181
func createCommandsTable() string {
	return `
CREATE TABLE IF NOT EXISTS commands (
    command_id VARCHAR(50) PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    command_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    response JSONB,
    result TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_by VARCHAR(100) NOT NULL DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_command_status CHECK (status IN ('pending', 'sent', 'completed', 'failed', 'timeout'))
);

COMMENT ON TABLE commands IS '命令下发记录表';
COMMENT ON COLUMN commands.created_by IS '命令创建者(用户ID或系统标识)';
COMMENT ON COLUMN commands.result IS 'Edge端返回的命令执行结果';
`
}

// createLicensesTable 创建许可证表
// 来源: FULL_INIT.sql 行183-205
func createLicensesTable() string {
	return `
CREATE TABLE IF NOT EXISTS licenses (
    license_id VARCHAR(100) PRIMARY KEY,
    cabinet_id VARCHAR(50) UNIQUE NOT NULL,
    mac_address VARCHAR(17) NOT NULL,
    max_devices INTEGER NOT NULL DEFAULT 0,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    permissions JSONB DEFAULT '[]',
    created_by VARCHAR(100) NOT NULL DEFAULT 'system',
    revoked_by VARCHAR(100),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoke_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_license_status CHECK (status IN ('active', 'expired', 'revoked'))
);

COMMENT ON TABLE licenses IS '许可证管理表';
COMMENT ON COLUMN licenses.mac_address IS 'MAC地址绑定,防止设备克隆';
`
}

// createAuditLogsTable 创建审计日志表
// 来源: FULL_INIT.sql 行207-224
func createAuditLogsTable() string {
	return `
CREATE TABLE IF NOT EXISTS audit_logs (
    log_id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    result VARCHAR(20) NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_result CHECK (result IN ('success', 'failure'))
);

COMMENT ON TABLE audit_logs IS '审计日志表';
COMMENT ON COLUMN audit_logs.result IS '操作结果: success, failure';
`
}

// createHealthScoresTable 创建健康评分历史表(时序数据)
// 来源: FULL_INIT.sql 行226-240
func createHealthScoresTable() string {
	return `
CREATE TABLE IF NOT EXISTS health_scores (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    cabinet_id VARCHAR(50) NOT NULL,
    score DECIMAL(5, 2) NOT NULL,
    online_rate DECIMAL(5, 2),
    data_quality DECIMAL(5, 2),
    alert_severity_score DECIMAL(5, 2),
    sensor_normalcy DECIMAL(5, 2),
    details JSONB,

    CONSTRAINT valid_score CHECK (score >= 0 AND score <= 100)
);

COMMENT ON TABLE health_scores IS '健康评分历史表';
`
}

// createVulnerabilityAssessmentsTable 创建脆弱性评估表
// 来源: FULL_INIT.sql 行242-272
func createVulnerabilityAssessmentsTable() string {
	return `
CREATE TABLE IF NOT EXISTS vulnerability_assessments (
    id BIGSERIAL PRIMARY KEY,
    cabinet_id VARCHAR(64) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 四维度评分(0-100)
    license_compliance_score DOUBLE PRECISION NOT NULL DEFAULT 100,
    communication_score DOUBLE PRECISION NOT NULL,
    config_security_score DOUBLE PRECISION NOT NULL,
    data_anomaly_score DOUBLE PRECISION NOT NULL,

    -- 综合评分
    overall_score DOUBLE PRECISION NOT NULL,
    risk_level VARCHAR(16) NOT NULL,

    -- 详细指标
    transmission_metrics JSONB,
    traffic_features JSONB,
    config_checks JSONB,
    detected_vulnerabilities JSONB,

    -- 同步状态
    synced_from_edge BOOLEAN DEFAULT TRUE,
    received_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT fk_cabinet FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE
);

COMMENT ON TABLE vulnerability_assessments IS '储能柜脆弱性评估结果';
COMMENT ON COLUMN vulnerability_assessments.risk_level IS '风险等级: healthy/low/medium/high/critical';
`
}

// createVulnerabilityEventsTable 创建脆弱性事件表
// 来源: FULL_INIT.sql 行274-296
func createVulnerabilityEventsTable() string {
	return `
CREATE TABLE IF NOT EXISTS vulnerability_events (
    id BIGSERIAL PRIMARY KEY,
    assessment_id BIGINT NOT NULL,
    cabinet_id VARCHAR(64) NOT NULL,

    -- 漏洞信息
    event_type VARCHAR(64) NOT NULL,
    category VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    severity VARCHAR(16) NOT NULL,
    description TEXT NOT NULL,
    solution TEXT,

    detected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT fk_assessment FOREIGN KEY (assessment_id) REFERENCES vulnerability_assessments(id) ON DELETE CASCADE,
    CONSTRAINT fk_cabinet_event FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE
);

COMMENT ON TABLE vulnerability_events IS '脆弱性事件详细记录';
COMMENT ON COLUMN vulnerability_events.category IS '漏洞分类: network/config/data/license';
`
}

// createAccessPoliciesTable 创建ABAC访问策略表
// 来源: FULL_INIT.sql 行298-313
func createAccessPoliciesTable() string {
	return `
CREATE TABLE IF NOT EXISTS access_policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    subject_type TEXT NOT NULL,
    conditions JSONB NOT NULL,
    permissions JSONB NOT NULL,
    priority INTEGER DEFAULT 50,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE access_policies IS 'ABAC访问策略表';
COMMENT ON COLUMN access_policies.subject_type IS '主体类型: user/cabinet/device';
`
}

// createAccessLogsTable 创建ABAC访问日志表
// 来源: FULL_INIT.sql 行315-330
func createAccessLogsTable() string {
	return `
CREATE TABLE IF NOT EXISTS access_logs (
    id SERIAL PRIMARY KEY,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    allowed BOOLEAN NOT NULL,
    policy_id TEXT,
    trust_score FLOAT,
    ip_address TEXT,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    attributes JSONB
);

COMMENT ON TABLE access_logs IS 'ABAC访问日志表,用于审计';
`
}

// createPolicyDistributionLogsTable 创建策略分发日志表
// 来源: FULL_INIT.sql 行332-346
func createPolicyDistributionLogsTable() string {
	return `
CREATE TABLE IF NOT EXISTS policy_distribution_logs (
    id SERIAL PRIMARY KEY,
    policy_id TEXT NOT NULL,
    cabinet_id TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    operator_id INTEGER,
    operator_name TEXT,
    error_message TEXT,
    distributed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMPTZ
);

COMMENT ON TABLE policy_distribution_logs IS '策略分发日志表,记录每次策略分发操作';
`
}

// createHypertables 将时序表转换为TimescaleDB Hypertable
// 来源: FULL_INIT.sql 行348-368
func createHypertables(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	hypertables := []struct {
		table    string
		timeCol  string
		interval string
	}{
		{"sensor_data", "time", "1 day"},
		{"health_scores", "time", "7 days"},
		{"vulnerability_assessments", "timestamp", "7 days"},
	}

	for _, ht := range hypertables {
		sql := fmt.Sprintf(
			"SELECT create_hypertable('%s', '%s', chunk_time_interval => INTERVAL '%s', if_not_exists => TRUE)",
			ht.table, ht.timeCol, ht.interval,
		)

		_, err := conn.Exec(ctx, sql)
		if err != nil {
			errStr := err.Error()

			// 如果TimescaleDB未安装,记录警告但不返回错误
			if strings.Contains(errStr, "function create_hypertable") {
				logger.Warn("TimescaleDB not available, skipping hypertable creation",
					zap.String("table", ht.table))
				continue
			}

			// 如果是主键/唯一索引冲突（表的主键不包含分区列），记录警告但继续
			if strings.Contains(errStr, "cannot create a unique index") ||
				strings.Contains(errStr, "must include all partitioning columns") {
				logger.Warn("Table primary key does not include partitioning column, using as regular table",
					zap.String("table", ht.table),
					zap.String("partition_column", ht.timeCol),
					zap.Error(err))
				continue
			}

			return fmt.Errorf("failed to create hypertable %s: %w", ht.table, err)
		}

		logger.Info("Hypertable created", zap.String("table", ht.table))
	}

	return nil
}

// createIndexes 创建所有索引
// 来源: FULL_INIT.sql 行370-468
func createIndexes(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	// 所有索引的SQL语句
	indexes := []string{
		// cabinets表索引 (行374-386)
		"CREATE INDEX IF NOT EXISTS idx_cabinets_status ON cabinets(status)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_mac_address ON cabinets(mac_address)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_last_sync_at ON cabinets(last_sync_at)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_vulnerability_score ON cabinets(latest_vulnerability_score DESC)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_risk_level ON cabinets(latest_risk_level)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_activation_status ON cabinets(activation_status)",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_latitude ON cabinets(latitude) WHERE latitude IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_longitude ON cabinets(longitude) WHERE longitude IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_location ON cabinets(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_cabinets_api_key ON cabinets(api_key) WHERE api_key IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_registration_token ON cabinets(registration_token) WHERE registration_token IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_cabinets_location_text ON cabinets USING GIN(to_tsvector('simple', location))",

		// users表索引 (行388-390)
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)",

		// sensor_devices表索引 (行392-395)
		"CREATE INDEX IF NOT EXISTS idx_sensor_devices_cabinet ON sensor_devices(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_sensor_devices_type ON sensor_devices(sensor_type)",
		"CREATE INDEX IF NOT EXISTS idx_sensor_devices_status ON sensor_devices(status)",

		// sensor_data表索引 (行397-400)
		"CREATE INDEX IF NOT EXISTS idx_sensor_data_cabinet_time ON sensor_data(cabinet_id, time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_sensor_data_device_time ON sensor_data(device_id, time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_sensor_data_type_time ON sensor_data(sensor_type, time DESC)",

		// alerts表索引 (行402-410)
		"CREATE INDEX IF NOT EXISTS idx_alerts_cabinet ON alerts(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_cabinet_created ON alerts(cabinet_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_edge_alert_id ON alerts(edge_alert_id) WHERE edge_alert_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_alerts_details ON alerts USING GIN(details)",

		// commands表索引 (行412-418)
		"CREATE INDEX IF NOT EXISTS idx_commands_cabinet_id ON commands(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_cabinet_status ON commands(cabinet_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_created_at ON commands(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_commands_payload ON commands USING GIN(payload)",
		"CREATE INDEX IF NOT EXISTS idx_commands_response ON commands USING GIN(response)",

		// licenses表索引 (行420-425)
		"CREATE INDEX IF NOT EXISTS idx_licenses_cabinet_id ON licenses(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_licenses_mac_address ON licenses(mac_address)",
		"CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status)",
		"CREATE INDEX IF NOT EXISTS idx_licenses_expires_at ON licenses(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_licenses_permissions ON licenses USING GIN(permissions)",

		// audit_logs表索引 (行427-434)
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action_result ON audit_logs(action, result)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_details ON audit_logs USING GIN(details)",

		// health_scores表索引 (行436-437)
		"CREATE INDEX IF NOT EXISTS idx_health_scores_cabinet_time ON health_scores(cabinet_id, time DESC)",

		// vulnerability_assessments表索引 (行439-443)
		"CREATE INDEX IF NOT EXISTS idx_va_cabinet_id ON vulnerability_assessments(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_va_timestamp ON vulnerability_assessments(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_va_risk_level ON vulnerability_assessments(risk_level)",
		"CREATE INDEX IF NOT EXISTS idx_va_cabinet_timestamp ON vulnerability_assessments(cabinet_id, timestamp DESC)",

		// vulnerability_events表索引 (行445-450)
		"CREATE INDEX IF NOT EXISTS idx_ve_assessment_id ON vulnerability_events(assessment_id)",
		"CREATE INDEX IF NOT EXISTS idx_ve_cabinet_id ON vulnerability_events(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_ve_severity ON vulnerability_events(severity)",
		"CREATE INDEX IF NOT EXISTS idx_ve_category ON vulnerability_events(category)",
		"CREATE INDEX IF NOT EXISTS idx_ve_detected_at ON vulnerability_events(detected_at DESC)",

		// access_policies表索引 (行452-455)
		"CREATE INDEX IF NOT EXISTS idx_policies_subject_type ON access_policies(subject_type)",
		"CREATE INDEX IF NOT EXISTS idx_policies_enabled ON access_policies(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_policies_priority ON access_policies(priority DESC)",

		// access_logs表索引 (行457-462)
		"CREATE INDEX IF NOT EXISTS idx_logs_subject ON access_logs(subject_type, subject_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON access_logs(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_logs_resource ON access_logs(resource)",
		"CREATE INDEX IF NOT EXISTS idx_logs_allowed ON access_logs(allowed)",
		"CREATE INDEX IF NOT EXISTS idx_logs_policy_id ON access_logs(policy_id)",

		// policy_distribution_logs表索引 (行464-468)
		"CREATE INDEX IF NOT EXISTS idx_distribution_policy ON policy_distribution_logs(policy_id)",
		"CREATE INDEX IF NOT EXISTS idx_distribution_cabinet ON policy_distribution_logs(cabinet_id)",
		"CREATE INDEX IF NOT EXISTS idx_distribution_status ON policy_distribution_logs(status)",
		"CREATE INDEX IF NOT EXISTS idx_distribution_time ON policy_distribution_logs(distributed_at DESC)",
	}

	// 执行所有索引创建
	for _, indexSQL := range indexes {
		if _, err := conn.Exec(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w\nSQL: %s", err, indexSQL)
		}
	}

	logger.Info("All indexes created successfully", zap.Int("index_count", len(indexes)))
	return nil
}

// createTriggers 创建触发器
// 来源: FULL_INIT.sql 行470-503
func createTriggers(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	// 创建自动更新updated_at的函数 (行474-481)
	functionSQL := `
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

	if _, err := conn.Exec(ctx, functionSQL); err != nil {
		return fmt.Errorf("failed to create update_updated_at_column function: %w", err)
	}

	// 为各表添加触发器 (行483-503)
	triggers := []struct {
		name  string
		table string
	}{
		{"update_cabinets_updated_at", "cabinets"},
		{"update_users_updated_at", "users"},
		{"update_sensor_devices_updated_at", "sensor_devices"},
		{"update_alerts_updated_at", "alerts"},
		{"update_commands_updated_at", "commands"},
		{"update_licenses_updated_at", "licenses"},
		{"update_policies_updated_at", "access_policies"},
	}

	for _, trigger := range triggers {
		triggerSQL := fmt.Sprintf(
			"CREATE TRIGGER %s BEFORE UPDATE ON %s FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()",
			trigger.name, trigger.table,
		)

		// 先删除旧触发器(如果存在)
		dropSQL := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", trigger.name, trigger.table)
		if _, err := conn.Exec(ctx, dropSQL); err != nil {
			return fmt.Errorf("failed to drop trigger %s: %w", trigger.name, err)
		}

		// 创建新触发器
		if _, err := conn.Exec(ctx, triggerSQL); err != nil {
			return fmt.Errorf("failed to create trigger %s: %w", trigger.name, err)
		}
	}

	logger.Info("All triggers created successfully", zap.Int("trigger_count", len(triggers)))
	return nil
}

// insertDefaultData 插入初始数据
// 来源: FULL_INIT.sql 行505-588
func insertDefaultData(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()

	// 创建默认管理员用户 (行509-512)
	// 密码: admin
	adminSQL := `
INSERT INTO users (username, password_hash, email, role, status)
VALUES ('admin', '$2a$10$Bz2EG1pvt1eLSacMeLTk2euuNqXuiNFI2Ec2aMlK7vp67WlqKEzr2', 'admin@example.com', 'admin', 'active')
ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
`

	if _, err := conn.Exec(ctx, adminSQL); err != nil {
		return fmt.Errorf("failed to insert default admin user: %w", err)
	}

	// 插入预定义ABAC策略 (行514-588)
	policiesSQL := `
INSERT INTO access_policies (id, name, description, subject_type, conditions, permissions, priority, enabled)
VALUES
    -- 策略1: 管理员完全访问
    (
        'policy_admin_full',
        '管理员完全访问',
        '管理员拥有所有权限',
        'user',
        '[
            {"attribute": "role", "operator": "eq", "value": "admin"},
            {"attribute": "status", "operator": "eq", "value": "active"}
        ]'::jsonb,
        '["*"]'::jsonb,
        100,
        true
    ),
    -- 策略2: 普通用户只读访问
    (
        'policy_user_readonly',
        '普通用户只读访问',
        '普通用户只能读取数据',
        'user',
        '[
            {"attribute": "role", "operator": "eq", "value": "user"},
            {"attribute": "status", "operator": "eq", "value": "active"}
        ]'::jsonb,
        '["read:cabinets", "read:sensors", "read:alerts", "read:devices", "read:licenses"]'::jsonb,
        50,
        true
    ),
    -- 策略3: 已激活储能柜数据同步
    (
        'policy_cabinet_sync',
        '已激活储能柜数据同步',
        '健康且已激活的储能柜可以同步数据',
        'cabinet',
        '[
            {"attribute": "activation_status", "operator": "eq", "value": "activated"},
            {"attribute": "status", "operator": "in", "value": ["active", "maintenance"]},
            {"attribute": "trust_score", "operator": "gte", "value": 30}
        ]'::jsonb,
        '["write:sensor_data", "write:alerts", "write:vulnerability", "write:sync", "read:commands", "write:heartbeat", "*"]'::jsonb,
        80,
        true
    ),
    -- 策略4: 低信任度储能柜受限访问
    (
        'policy_cabinet_limited',
        '低信任度储能柜受限访问',
        '信任度较低的储能柜仅可上传传感器数据',
        'cabinet',
        '[
            {"attribute": "activation_status", "operator": "eq", "value": "activated"},
            {"attribute": "trust_score", "operator": "lt", "value": 30}
        ]'::jsonb,
        '["write:sensor_data"]'::jsonb,
        60,
        true
    ),
    -- 策略5: 高质量传感器完全数据上传
    (
        'policy_device_high_quality',
        '高质量传感器完全数据上传',
        '数据质量高的传感器可以上传所有数据',
        'device',
        '[
            {"attribute": "status", "operator": "eq", "value": "active"},
            {"attribute": "quality", "operator": "gte", "value": 80}
        ]'::jsonb,
        '["write:sensor_data", "trigger:alert"]'::jsonb,
        70,
        true
    )
ON CONFLICT (id) DO NOTHING
`

	if _, err := conn.Exec(ctx, policiesSQL); err != nil {
		return fmt.Errorf("failed to insert default ABAC policies: %w", err)
	}

	logger.Info("Default data inserted successfully")
	return nil
}

// InitSchema 初始化完整的数据库Schema
// 这是主入口函数,按顺序执行所有初始化步骤
func InitSchema(ctx context.Context, conn *pgxpool.Pool) error {
	logger := utils.GetLogger()
	logger.Info("Starting database schema initialization")

	// 1. 创建扩展
	if err := createExtensions(ctx, conn); err != nil {
		return fmt.Errorf("extension creation failed: %w", err)
	}

	// 2. 创建表
	if err := createTables(ctx, conn); err != nil {
		return fmt.Errorf("table creation failed: %w", err)
	}

	// 3. 创建Hypertable(如果TimescaleDB可用)
	if err := createHypertables(ctx, conn); err != nil {
		return fmt.Errorf("hypertable creation failed: %w", err)
	}

	// 4. 创建索引
	if err := createIndexes(ctx, conn); err != nil {
		return fmt.Errorf("index creation failed: %w", err)
	}

	// 5. 创建触发器
	if err := createTriggers(ctx, conn); err != nil {
		return fmt.Errorf("trigger creation failed: %w", err)
	}

	// 6. 插入初始数据
	if err := insertDefaultData(ctx, conn); err != nil {
		return fmt.Errorf("default data insertion failed: %w", err)
	}

	logger.Info("Database schema initialization completed successfully",
		zap.Int("tables", 14),
		zap.Int("hypertables", 3),
		zap.String("default_user", "admin"),
		zap.Int("default_policies", 5),
	)

	return nil
}
