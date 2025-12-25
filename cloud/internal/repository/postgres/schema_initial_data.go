package postgres

import (
	"context"
	"fmt"

	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// insertInitialData 插入初始数据
// SQL来源: migrations/FULL_INIT.sql (Lines 509-588)
func (c *Client) insertInitialData(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// 步骤1: 插入默认admin用户
	if err := insertDefaultUser(ctx, conn); err != nil {
		return err
	}

	// 步骤2: 插入ABAC策略
	if err := insertABACPolicies(ctx, conn); err != nil {
		return err
	}

	utils.Info("Initial data inserted successfully")
	return nil
}

// insertDefaultUser 插入默认管理员用户
// 用户名: admin, 密码: admin
func insertDefaultUser(ctx context.Context, conn *pgxpool.Conn) error {
	query := `
        INSERT INTO users (username, password_hash, email, role, status)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (username) DO UPDATE
        SET password_hash = EXCLUDED.password_hash
    `

	// bcrypt hash of "admin"
	passwordHash := "$2a$10$Bz2EG1pvt1eLSacMeLTk2euuNqXuiNFI2Ec2aMlK7vp67WlqKEzr2"

	_, err := conn.Exec(ctx, query,
		"admin",
		passwordHash,
		"admin@example.com",
		"admin",
		"active",
	)

	if err != nil {
		return fmt.Errorf("failed to insert default user: %w", err)
	}

	utils.Info("Default admin user created", zap.String("username", "admin"))
	return nil
}

// insertABACPolicies 插入预定义ABAC策略
func insertABACPolicies(ctx context.Context, conn *pgxpool.Conn) error {
	policies := []struct {
		id          string
		name        string
		description string
		subjectType string
		conditions  string
		permissions string
		priority    int
	}{
		// 策略1: 管理员完全访问
		{
			id:          "policy_admin_full",
			name:        "管理员完全访问",
			description: "管理员拥有所有权限",
			subjectType: "user",
			conditions: `[
                {"attribute": "role", "operator": "eq", "value": "admin"},
                {"attribute": "status", "operator": "eq", "value": "active"}
            ]`,
			permissions: `["*"]`,
			priority:    100,
		},
		// 策略2: 普通用户只读访问
		{
			id:          "policy_user_readonly",
			name:        "普通用户只读访问",
			description: "普通用户只能读取数据",
			subjectType: "user",
			conditions: `[
                {"attribute": "role", "operator": "eq", "value": "user"},
                {"attribute": "status", "operator": "eq", "value": "active"}
            ]`,
			permissions: `["read:cabinets", "read:sensors", "read:alerts", "read:devices", "read:licenses"]`,
			priority:    50,
		},
		// 策略3: 已激活储能柜数据同步
		{
			id:          "policy_cabinet_sync",
			name:        "已激活储能柜数据同步",
			description: "健康且已激活的储能柜可以同步数据",
			subjectType: "cabinet",
			conditions: `[
                {"attribute": "activation_status", "operator": "eq", "value": "activated"},
                {"attribute": "status", "operator": "in", "value": ["active", "maintenance"]},
                {"attribute": "trust_score", "operator": "gte", "value": 30}
            ]`,
			permissions: `["write:sensor_data", "write:alerts", "write:vulnerability", "write:sync", "read:commands", "write:heartbeat", "*"]`,
			priority:    80,
		},
		// 策略4: 低信任度储能柜受限访问
		{
			id:          "policy_cabinet_limited",
			name:        "低信任度储能柜受限访问",
			description: "信任度较低的储能柜仅可上传传感器数据",
			subjectType: "cabinet",
			conditions: `[
                {"attribute": "activation_status", "operator": "eq", "value": "activated"},
                {"attribute": "trust_score", "operator": "lt", "value": 30}
            ]`,
			permissions: `["write:sensor_data"]`,
			priority:    60,
		},
		// 策略5: 高质量传感器完全数据上传
		{
			id:          "policy_device_high_quality",
			name:        "高质量传感器完全数据上传",
			description: "数据质量高的传感器可以上传所有数据",
			subjectType: "device",
			conditions: `[
                {"attribute": "status", "operator": "eq", "value": "active"},
                {"attribute": "quality", "operator": "gte", "value": 80}
            ]`,
			permissions: `["write:sensor_data", "trigger:alert"]`,
			priority:    70,
		},
	}

	query := `
        INSERT INTO access_policies (id, name, description, subject_type, conditions, permissions, priority, enabled)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, true)
        ON CONFLICT (id) DO NOTHING
    `

	successCount := 0
	for _, p := range policies {
		_, err := conn.Exec(ctx, query,
			p.id, p.name, p.description, p.subjectType,
			p.conditions, p.permissions, p.priority,
		)
		if err != nil {
			return fmt.Errorf("failed to insert policy %s: %w", p.id, err)
		}
		successCount++
	}

	utils.Info("ABAC policies created",
		zap.Int("total", len(policies)),
		zap.Int("success", successCount))

	return nil
}
