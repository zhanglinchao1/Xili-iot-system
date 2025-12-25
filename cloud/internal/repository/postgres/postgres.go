package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Client PostgreSQL客户端
type Client struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewClient 创建PostgreSQL客户端
func NewClient(cfg *config.Config) (*Client, error) {
	ctx := context.Background()

	// 配置连接池
	poolConfig, err := pgxpool.ParseConfig(cfg.GetPostgresConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// 设置连接池参数
	poolConfig.MaxConns = int32(cfg.Database.Postgres.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.Postgres.MaxIdleConnections)

	// 解析连接生命周期
	if lifetime, err := time.ParseDuration(cfg.Database.Postgres.ConnectionMaxLifetime); err == nil {
		poolConfig.MaxConnLifetime = lifetime
	}

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	utils.Info("PostgreSQL connection established",
		zap.String("host", cfg.Database.Postgres.Host),
		zap.Int("port", cfg.Database.Postgres.Port),
		zap.String("database", cfg.Database.Postgres.DBName),
	)

	return &Client{
		pool: pool,
		cfg:  cfg,
	}, nil
}

// GetPool 获取连接池
func (c *Client) GetPool() *pgxpool.Pool {
	return c.pool
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.pool != nil {
		c.pool.Close()
		utils.Info("PostgreSQL connection closed")
	}
	return nil
}

// Ping 检查连接
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// RunMigrations 运行数据库迁移
func (c *Client) RunMigrations(ctx context.Context, migrationsPath string) error {
	logger := utils.GetLogger()
	logger.Info("Running database migrations", zap.String("path", migrationsPath))

	// 步骤1: 初始化核心Schema（14张表+索引+触发器+Hypertables+初始数据）
	// 使用InitSchema创建完整数据库结构（如果表已存在则跳过）
	if err := InitSchema(ctx, c.pool); err != nil {
		// Schema初始化失败记录警告但不中断（允许使用现有数据库）
		logger.Warn("Failed to initialize core schema, database may already exist",
			zap.Error(err))
		// 继续执行，可能数据库已经存在
	} else {
		logger.Info("Core schema initialization completed successfully")
	}

	// 步骤2: 执行兼容性迁移：添加activation_status等字段
	// 保留此步骤以兼容从旧版本升级的数据库
	if err := c.migrateActivationFields(ctx); err != nil {
		logger.Warn("Failed to migrate activation fields", zap.Error(err))
		// 不返回错误，允许应用继续启动
	}

	return nil
}

// migrateActivationFields 迁移激活相关字段
func (c *Client) migrateActivationFields(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// 检查并添加字段的SQL（使用DO块避免权限问题）
	migrations := []string{
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'activation_status') THEN
				ALTER TABLE cabinets ADD COLUMN activation_status VARCHAR(20) DEFAULT 'pending';
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'registration_token') THEN
				ALTER TABLE cabinets ADD COLUMN registration_token VARCHAR(500);
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'token_expires_at') THEN
				ALTER TABLE cabinets ADD COLUMN token_expires_at TIMESTAMP WITH TIME ZONE;
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'api_key') THEN
				ALTER TABLE cabinets ADD COLUMN api_key VARCHAR(100);
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'api_secret_hash') THEN
				ALTER TABLE cabinets ADD COLUMN api_secret_hash VARCHAR(255);
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'activated_at') THEN
				ALTER TABLE cabinets ADD COLUMN activated_at TIMESTAMP WITH TIME ZONE;
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'ip_address') THEN
				ALTER TABLE cabinets ADD COLUMN ip_address VARCHAR(45);
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'device_model') THEN
				ALTER TABLE cabinets ADD COLUMN device_model VARCHAR(100);
			END IF;
		END $$;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'cabinets' AND column_name = 'notes') THEN
				ALTER TABLE cabinets ADD COLUMN notes TEXT;
			END IF;
		END $$;`,
	}

	for _, migration := range migrations {
		if _, err := conn.Exec(ctx, migration); err != nil {
			// 如果是权限错误，记录警告但继续
			errStr := err.Error()
			if strings.Contains(errStr, "must be owner") || strings.Contains(errStr, "permission denied") {
				utils.Warn("Migration requires database owner privileges", zap.Error(err))
				continue
			}
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	utils.Info("Activation fields migration completed")
	return nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	return nil
}

// Stats 获取连接池统计信息
func (c *Client) Stats() map[string]interface{} {
	stats := c.pool.Stat()
	return map[string]interface{}{
		"total_connections":    stats.TotalConns(),
		"idle_connections":     stats.IdleConns(),
		"acquired_connections": stats.AcquiredConns(),
		"max_connections":      stats.MaxConns(),
	}
}
