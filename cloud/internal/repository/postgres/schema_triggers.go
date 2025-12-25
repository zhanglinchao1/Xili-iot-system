package postgres

import (
	"context"
	"fmt"
	"strings"

	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// createTriggers 创建触发器和触发器函数
// SQL来源: migrations/FULL_INIT.sql (Lines 471-503)
func (c *Client) createTriggers(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// 步骤1: 创建触发器函数
	if err := createTriggerFunction(ctx, conn); err != nil {
		return err
	}

	// 步骤2: 为需要的表创建触发器
	if err := createTableTriggers(ctx, conn); err != nil {
		return err
	}

	utils.Info("Triggers created successfully")
	return nil
}

// createTriggerFunction 创建自动更新updated_at的触发器函数
func createTriggerFunction(ctx context.Context, conn *pgxpool.Conn) error {
	triggerFunc := `
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

	if _, err := conn.Exec(ctx, triggerFunc); err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	utils.Info("Trigger function created", zap.String("function", "update_updated_at_column"))
	return nil
}

// createTableTriggers 为各表创建触发器
func createTableTriggers(ctx context.Context, conn *pgxpool.Conn) error {
	triggers := []struct {
		name  string
		table string
		sql   string
	}{
		{
			name:  "update_cabinets_updated_at",
			table: "cabinets",
			sql: `CREATE TRIGGER update_cabinets_updated_at
                  BEFORE UPDATE ON cabinets
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_users_updated_at",
			table: "users",
			sql: `CREATE TRIGGER update_users_updated_at
                  BEFORE UPDATE ON users
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_sensor_devices_updated_at",
			table: "sensor_devices",
			sql: `CREATE TRIGGER update_sensor_devices_updated_at
                  BEFORE UPDATE ON sensor_devices
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_alerts_updated_at",
			table: "alerts",
			sql: `CREATE TRIGGER update_alerts_updated_at
                  BEFORE UPDATE ON alerts
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_commands_updated_at",
			table: "commands",
			sql: `CREATE TRIGGER update_commands_updated_at
                  BEFORE UPDATE ON commands
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_licenses_updated_at",
			table: "licenses",
			sql: `CREATE TRIGGER update_licenses_updated_at
                  BEFORE UPDATE ON licenses
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
		{
			name:  "update_policies_updated_at",
			table: "access_policies",
			sql: `CREATE TRIGGER update_policies_updated_at
                  BEFORE UPDATE ON access_policies
                  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
		},
	}

	successCount := 0
	for _, trigger := range triggers {
		if _, err := conn.Exec(ctx, trigger.sql); err != nil {
			// 触发器已存在不算错误
			if strings.Contains(err.Error(), "already exists") {
				utils.Info("Trigger already exists",
					zap.String("trigger", trigger.name),
					zap.String("table", trigger.table))
				successCount++
				continue
			}
			return fmt.Errorf("failed to create trigger %s: %w", trigger.name, err)
		}

		utils.Info("Trigger created",
			zap.String("trigger", trigger.name),
			zap.String("table", trigger.table))
		successCount++
	}

	utils.Info("All triggers processed",
		zap.Int("total", len(triggers)),
		zap.Int("success", successCount))

	return nil
}
