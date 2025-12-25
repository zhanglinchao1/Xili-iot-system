/*
 * SQLite数据库存储模块
 * 负责本地数据的持久化存储
 */
package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// SQLiteDB SQLite数据库管理器
type SQLiteDB struct {
	logger *zap.Logger
	db     *sql.DB
	path   string
}

// NewSQLiteDB 创建SQLite数据库管理器
func NewSQLiteDB(cfg config.DatabaseConfig, logger *zap.Logger) (*SQLiteDB, error) {
	db, err := sql.Open(cfg.Driver, cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(time.Hour)

	storage := &SQLiteDB{
		logger: logger,
		db:     db,
		path:   cfg.Path,
	}

	// 初始化数据库表
	if err := storage.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	logger.Info("SQLite storage initialized", zap.String("path", cfg.Path))
	return storage, nil
}

// Close 关闭数据库连接
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// GetDB 获取数据库连接
func (s *SQLiteDB) GetDB() *sql.DB {
	return s.db
}

// initTables 初始化数据库表
func (s *SQLiteDB) initTables() error {
	schemas := []string{
		// 设备表
		`CREATE TABLE IF NOT EXISTS devices (
			device_id VARCHAR(64) PRIMARY KEY,
			device_type VARCHAR(32),
			sensor_type VARCHAR(32),
			public_key TEXT,
			commitment TEXT,
			status VARCHAR(16),
			model VARCHAR(64),
			manufacturer VARCHAR(64),
			firmware_ver VARCHAR(32),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP
		)`,

		// 设备索引
		`CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_sensor_type ON devices(sensor_type)`,

		// 认证挑战表
		`CREATE TABLE IF NOT EXISTS challenges (
			challenge_id VARCHAR(64) PRIMARY KEY,
			device_id VARCHAR(64),
			nonce VARCHAR(128),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			used BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		)`,

		// 挑战索引
		`CREATE INDEX IF NOT EXISTS idx_challenges_device ON challenges(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_challenges_expires ON challenges(expires_at)`,

		// 会话表
		`CREATE TABLE IF NOT EXISTS sessions (
			session_id VARCHAR(64) PRIMARY KEY,
			device_id VARCHAR(64),
			token TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ip_address VARCHAR(45),
			user_agent TEXT,
			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		)`,

		// 会话索引
		`CREATE INDEX IF NOT EXISTS idx_sessions_device ON sessions(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,

		// 传感器数据表
		`CREATE TABLE IF NOT EXISTS sensor_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id VARCHAR(64),
			sensor_type VARCHAR(32),
			value REAL,
			unit VARCHAR(16),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			quality INTEGER DEFAULT 100,
			synced BOOLEAN DEFAULT FALSE,
			synced_at TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		)`,

		// 数据索引
		`CREATE INDEX IF NOT EXISTS idx_sensor_data_device ON sensor_data(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sensor_data_timestamp ON sensor_data(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_sensor_data_synced ON sensor_data(synced)`,
		`CREATE INDEX IF NOT EXISTS idx_sensor_data_sensor_type ON sensor_data(sensor_type)`,

		// 告警表
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id VARCHAR(64),
			alert_type VARCHAR(32),
			severity VARCHAR(16),
			message TEXT,
			value REAL,
			threshold REAL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			resolved BOOLEAN DEFAULT FALSE,
			resolved_at TIMESTAMP,
			synced_at TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		)`,

		// 告警索引
		`CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_timestamp ON alerts(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity)`,

		// 系统日志表
		`CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level VARCHAR(16),
			module VARCHAR(32),
			message TEXT,
			details TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// 日志索引
		`CREATE INDEX IF NOT EXISTS idx_system_logs_timestamp ON system_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level)`,

		// 数据统计表（用于缓存统计结果）
		`CREATE TABLE IF NOT EXISTS data_statistics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id VARCHAR(64),
			sensor_type VARCHAR(32),
			period VARCHAR(16),
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			count INTEGER,
			min_value REAL,
			max_value REAL,
			avg_value REAL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		)`,

		// 统计索引
		`CREATE INDEX IF NOT EXISTS idx_statistics_device ON data_statistics(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_statistics_period ON data_statistics(period, start_time)`,

		// 脆弱性评估结果表
		`CREATE TABLE IF NOT EXISTS vulnerability_assessments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cabinet_id VARCHAR(64),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			license_compliance_score REAL DEFAULT 100,
			communication_score REAL,
			config_security_score REAL,
			data_anomaly_score REAL,
			overall_score REAL,
			risk_level VARCHAR(16),
			transmission_metrics TEXT,
			traffic_features TEXT,
			config_checks TEXT,
			detected_vulnerabilities TEXT,
			synced BOOLEAN DEFAULT FALSE,
			synced_at TIMESTAMP
		)`,

		// 脆弱性评估索引
		`CREATE INDEX IF NOT EXISTS idx_va_cabinet ON vulnerability_assessments(cabinet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_va_timestamp ON vulnerability_assessments(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_va_synced ON vulnerability_assessments(synced)`,

		// 传输指标历史表
		`CREATE TABLE IF NOT EXISTS transmission_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cabinet_id VARCHAR(64),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			latency_avg REAL,
			packet_loss_rate REAL,
			throughput REAL,
			mqtt_success_rate REAL,
			reconnection_count INTEGER
		)`,

		// 传输指标索引
		`CREATE INDEX IF NOT EXISTS idx_tm_cabinet ON transmission_metrics(cabinet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tm_timestamp ON transmission_metrics(timestamp)`,

		// 已消除的漏洞表
		`CREATE TABLE IF NOT EXISTS dismissed_vulnerabilities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vulnerability_type VARCHAR(64),
			reason TEXT,
			dismissed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			dismissed_by VARCHAR(64) DEFAULT 'system',
			expires_at TIMESTAMP
		)`,

		// 已消除漏洞索引
		`CREATE INDEX IF NOT EXISTS idx_dv_type ON dismissed_vulnerabilities(vulnerability_type)`,
		`CREATE INDEX IF NOT EXISTS idx_dv_expires ON dismissed_vulnerabilities(expires_at)`,

		// Cloud凭证表（存储API Key等敏感信息）
		`CREATE TABLE IF NOT EXISTS cloud_credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cabinet_id VARCHAR(64) UNIQUE NOT NULL,
			api_key VARCHAR(256) NOT NULL,
			api_secret VARCHAR(256) DEFAULT '',
			cloud_endpoint VARCHAR(512) DEFAULT '',
			enabled INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Cloud凭证索引
		`CREATE INDEX IF NOT EXISTS idx_cc_cabinet_id ON cloud_credentials(cabinet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cc_enabled ON cloud_credentials(enabled)`,
	}

	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 执行所有建表语句
	for _, schema := range schemas {
		if _, err := tx.Exec(schema); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return err
	}

	s.logger.Info("Database tables initialized")

	// 执行数据库迁移
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations 执行数据库迁移
func (s *SQLiteDB) runMigrations() error {
	// 检查 vulnerability_assessments 表是否有 license_compliance_score 字段
	query := `PRAGMA table_info(vulnerability_assessments)`
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasLicenseColumn := false
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dfltValue interface{}
		var pk int

		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return err
		}

		if name == "license_compliance_score" {
			hasLicenseColumn = true
			break
		}
	}

	// 如果没有该字段,则添加
	if !hasLicenseColumn {
		s.logger.Info("Adding license_compliance_score column to vulnerability_assessments")
		_, err = s.db.Exec(`ALTER TABLE vulnerability_assessments ADD COLUMN license_compliance_score REAL DEFAULT 100`)
		if err != nil {
			return fmt.Errorf("failed to add license_compliance_score column: %w", err)
		}
		s.logger.Info("Migration completed: license_compliance_score column added")
	}

	return nil
}

// CleanOldData 清理过期数据
func (s *SQLiteDB) CleanOldData(retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 清理传感器数据
	query := `DELETE FROM sensor_data WHERE timestamp < ? AND synced = TRUE`
	result, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to clean sensor data: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info("Cleaned old sensor data",
		zap.Int64("rows_deleted", rowsAffected),
		zap.Time("cutoff_time", cutoffTime))

	// 清理已解决的告警
	query = `DELETE FROM alerts WHERE timestamp < ? AND resolved = TRUE`
	result, err = s.db.Exec(query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to clean alerts: %w", err)
	}

	rowsAffected, _ = result.RowsAffected()
	s.logger.Info("Cleaned old alerts",
		zap.Int64("rows_deleted", rowsAffected))

	// 清理过期会话
	query = `DELETE FROM sessions WHERE expires_at < ?`
	result, err = s.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to clean sessions: %w", err)
	}

	rowsAffected, _ = result.RowsAffected()
	s.logger.Info("Cleaned expired sessions",
		zap.Int64("rows_deleted", rowsAffected))

	// 清理过期挑战
	query = `DELETE FROM challenges WHERE expires_at < ?`
	result, err = s.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to clean challenges: %w", err)
	}

	rowsAffected, _ = result.RowsAffected()
	s.logger.Info("Cleaned expired challenges",
		zap.Int64("rows_deleted", rowsAffected))

	// 执行VACUUM优化数据库
	if _, err := s.db.Exec("VACUUM"); err != nil {
		s.logger.Warn("Failed to vacuum database", zap.Error(err))
	}

	return nil
}

// GetDatabaseStats 获取数据库统计信息
func (s *SQLiteDB) GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取各表记录数
	tables := []string{"devices", "sensor_data", "alerts", "sessions"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := s.db.QueryRow(query).Scan(&count); err != nil {
			s.logger.Error("Failed to get table count",
				zap.String("table", table),
				zap.Error(err))
			continue
		}
		stats[table+"_count"] = count
	}

	// 获取数据库文件大小
	var pageCount, pageSize int
	s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	stats["database_size"] = pageCount * pageSize

	// 获取未同步数据数量
	var unsyncedCount int
	s.db.QueryRow("SELECT COUNT(*) FROM sensor_data WHERE synced = FALSE").Scan(&unsyncedCount)
	stats["unsynced_data"] = unsyncedCount

	// 获取未解决告警数量
	var unresolvedAlerts int
	s.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE resolved = FALSE").Scan(&unresolvedAlerts)
	stats["unresolved_alerts"] = unresolvedAlerts

	return stats, nil
}

// BeginTransaction 开始事务
func (s *SQLiteDB) BeginTransaction() (*sql.Tx, error) {
	return s.db.Begin()
}

// Backup 备份数据库
func (s *SQLiteDB) Backup(backupPath string) error {
	// 验证备份路径，防止SQL注入
	if backupPath == "" || strings.Contains(backupPath, "'") || strings.Contains(backupPath, ";") {
		return fmt.Errorf("invalid backup path")
	}

	// 使用参数化查询或文件操作进行备份
	// 方法1: 使用文件复制（更安全）
	sourceFile, err := os.Open(s.path)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	// 确保数据库连接关闭以释放文件锁
	s.db.Close()
	defer func() {
		// 重新打开数据库连接
		db, err := sql.Open("sqlite3", s.path)
		if err == nil {
			s.db = db
		}
	}()

	// 复制文件
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	s.logger.Info("Database backed up", zap.String("backup_path", backupPath))
	return nil
}

// 数据库操作代理方法，使SQLiteDB可以像sql.DB一样使用

// Query 执行查询
func (s *SQLiteDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// QueryRow 执行单行查询
func (s *SQLiteDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// Exec 执行SQL语句
func (s *SQLiteDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// Begin 开始事务
func (s *SQLiteDB) Begin() (*sql.Tx, error) {
	return s.db.Begin()
}

// Prepare 预编译SQL语句
func (s *SQLiteDB) Prepare(query string) (*sql.Stmt, error) {
	return s.db.Prepare(query)
}

// Ping 测试数据库连接
func (s *SQLiteDB) Ping() error {
	return s.db.Ping()
}

// Migrate 执行数据库迁移
func (s *SQLiteDB) Migrate() error {
	return s.initTables()
}
