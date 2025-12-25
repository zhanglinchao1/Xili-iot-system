package postgres

import (
	"context"
	"fmt"
	"strings"

	"cloud-system/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// createIndexes 创建所有数据库索引
// 索引定义来源: /home/zhang/XiLi/Cloud/migrations/FULL_INIT.sql (Lines 374-469)
func (c *Client) createIndexes(ctx context.Context) error {
	logger := utils.GetLogger()
	logger.Info("开始创建数据库索引")

	// 索引按表分组，便于维护
	indexGroups := []struct {
		tableName string
		indexes   []string
	}{
		{
			tableName: "cabinets",
			indexes: []string{
				// Line 375
				"CREATE INDEX IF NOT EXISTS idx_cabinets_status ON cabinets(status)",
				// Line 376
				"CREATE INDEX IF NOT EXISTS idx_cabinets_mac_address ON cabinets(mac_address)",
				// Line 377
				"CREATE INDEX IF NOT EXISTS idx_cabinets_last_sync_at ON cabinets(last_sync_at)",
				// Line 378
				"CREATE INDEX IF NOT EXISTS idx_cabinets_vulnerability_score ON cabinets(latest_vulnerability_score DESC)",
				// Line 379
				"CREATE INDEX IF NOT EXISTS idx_cabinets_risk_level ON cabinets(latest_risk_level)",
				// Line 380
				"CREATE INDEX IF NOT EXISTS idx_cabinets_activation_status ON cabinets(activation_status)",
				// Line 381 - 部分索引
				"CREATE INDEX IF NOT EXISTS idx_cabinets_latitude ON cabinets(latitude) WHERE latitude IS NOT NULL",
				// Line 382 - 部分索引
				"CREATE INDEX IF NOT EXISTS idx_cabinets_longitude ON cabinets(longitude) WHERE longitude IS NOT NULL",
				// Line 383 - 复合索引 + 部分索引
				"CREATE INDEX IF NOT EXISTS idx_cabinets_location ON cabinets(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL",
				// Line 384 - 唯一索引 + 部分索引
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_cabinets_api_key ON cabinets(api_key) WHERE api_key IS NOT NULL",
				// Line 385 - 部分索引
				"CREATE INDEX IF NOT EXISTS idx_cabinets_registration_token ON cabinets(registration_token) WHERE registration_token IS NOT NULL",
				// Line 386 - GIN全文索引
				"CREATE INDEX IF NOT EXISTS idx_cabinets_location_text ON cabinets USING GIN(to_tsvector('simple', location))",
			},
		},
		{
			tableName: "users",
			indexes: []string{
				// Line 389
				"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
				// Line 390
				"CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)",
			},
		},
		{
			tableName: "sensor_devices",
			indexes: []string{
				// Line 393
				"CREATE INDEX IF NOT EXISTS idx_sensor_devices_cabinet ON sensor_devices(cabinet_id)",
				// Line 394
				"CREATE INDEX IF NOT EXISTS idx_sensor_devices_type ON sensor_devices(sensor_type)",
				// Line 395
				"CREATE INDEX IF NOT EXISTS idx_sensor_devices_status ON sensor_devices(status)",
			},
		},
		{
			tableName: "sensor_data",
			indexes: []string{
				// Line 398 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_sensor_data_cabinet_time ON sensor_data(cabinet_id, time DESC)",
				// Line 399 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_sensor_data_device_time ON sensor_data(device_id, time DESC)",
				// Line 400 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_sensor_data_type_time ON sensor_data(sensor_type, time DESC)",
			},
		},
		{
			tableName: "alerts",
			indexes: []string{
				// Line 403
				"CREATE INDEX IF NOT EXISTS idx_alerts_cabinet ON alerts(cabinet_id)",
				// Line 404
				"CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status)",
				// Line 405
				"CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity)",
				// Line 406
				"CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC)",
				// Line 407
				"CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved)",
				// Line 408 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_alerts_cabinet_created ON alerts(cabinet_id, created_at DESC)",
				// Line 409 - 部分索引
				"CREATE INDEX IF NOT EXISTS idx_alerts_edge_alert_id ON alerts(edge_alert_id) WHERE edge_alert_id IS NOT NULL",
				// Line 410 - GIN JSONB索引
				"CREATE INDEX IF NOT EXISTS idx_alerts_details ON alerts USING GIN(details)",
			},
		},
		{
			tableName: "commands",
			indexes: []string{
				// Line 413
				"CREATE INDEX IF NOT EXISTS idx_commands_cabinet_id ON commands(cabinet_id)",
				// Line 414
				"CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status)",
				// Line 415 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_commands_cabinet_status ON commands(cabinet_id, status)",
				// Line 416
				"CREATE INDEX IF NOT EXISTS idx_commands_created_at ON commands(created_at DESC)",
				// Line 417 - GIN JSONB索引
				"CREATE INDEX IF NOT EXISTS idx_commands_payload ON commands USING GIN(payload)",
				// Line 418 - GIN JSONB索引
				"CREATE INDEX IF NOT EXISTS idx_commands_response ON commands USING GIN(response)",
			},
		},
		{
			tableName: "licenses",
			indexes: []string{
				// Line 421
				"CREATE INDEX IF NOT EXISTS idx_licenses_cabinet_id ON licenses(cabinet_id)",
				// Line 422
				"CREATE INDEX IF NOT EXISTS idx_licenses_mac_address ON licenses(mac_address)",
				// Line 423
				"CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status)",
				// Line 424
				"CREATE INDEX IF NOT EXISTS idx_licenses_expires_at ON licenses(expires_at)",
				// Line 425 - GIN JSONB索引
				"CREATE INDEX IF NOT EXISTS idx_licenses_permissions ON licenses USING GIN(permissions)",
			},
		},
		{
			tableName: "audit_logs",
			indexes: []string{
				// Line 428
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)",
				// Line 429
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
				// Line 430
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC)",
				// Line 431
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type)",
				// Line 432
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id)",
				// Line 433 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_action_result ON audit_logs(action, result)",
				// Line 434 - GIN JSONB索引
				"CREATE INDEX IF NOT EXISTS idx_audit_logs_details ON audit_logs USING GIN(details)",
			},
		},
		{
			tableName: "health_scores",
			indexes: []string{
				// Line 437 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_health_scores_cabinet_time ON health_scores(cabinet_id, time DESC)",
			},
		},
		{
			tableName: "vulnerability_assessments",
			indexes: []string{
				// Line 440
				"CREATE INDEX IF NOT EXISTS idx_va_cabinet_id ON vulnerability_assessments(cabinet_id)",
				// Line 441
				"CREATE INDEX IF NOT EXISTS idx_va_timestamp ON vulnerability_assessments(timestamp DESC)",
				// Line 442
				"CREATE INDEX IF NOT EXISTS idx_va_risk_level ON vulnerability_assessments(risk_level)",
				// Line 443 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_va_cabinet_timestamp ON vulnerability_assessments(cabinet_id, timestamp DESC)",
			},
		},
		{
			tableName: "vulnerability_events",
			indexes: []string{
				// Line 446
				"CREATE INDEX IF NOT EXISTS idx_ve_assessment_id ON vulnerability_events(assessment_id)",
				// Line 447
				"CREATE INDEX IF NOT EXISTS idx_ve_cabinet_id ON vulnerability_events(cabinet_id)",
				// Line 448
				"CREATE INDEX IF NOT EXISTS idx_ve_severity ON vulnerability_events(severity)",
				// Line 449
				"CREATE INDEX IF NOT EXISTS idx_ve_category ON vulnerability_events(category)",
				// Line 450
				"CREATE INDEX IF NOT EXISTS idx_ve_detected_at ON vulnerability_events(detected_at DESC)",
			},
		},
		{
			tableName: "access_policies",
			indexes: []string{
				// Line 453
				"CREATE INDEX IF NOT EXISTS idx_policies_subject_type ON access_policies(subject_type)",
				// Line 454
				"CREATE INDEX IF NOT EXISTS idx_policies_enabled ON access_policies(enabled)",
				// Line 455
				"CREATE INDEX IF NOT EXISTS idx_policies_priority ON access_policies(priority DESC)",
			},
		},
		{
			tableName: "access_logs",
			indexes: []string{
				// Line 458 - 复合索引
				"CREATE INDEX IF NOT EXISTS idx_logs_subject ON access_logs(subject_type, subject_id)",
				// Line 459
				"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON access_logs(timestamp DESC)",
				// Line 460
				"CREATE INDEX IF NOT EXISTS idx_logs_resource ON access_logs(resource)",
				// Line 461
				"CREATE INDEX IF NOT EXISTS idx_logs_allowed ON access_logs(allowed)",
				// Line 462
				"CREATE INDEX IF NOT EXISTS idx_logs_policy_id ON access_logs(policy_id)",
			},
		},
		{
			tableName: "policy_distribution_logs",
			indexes: []string{
				// Line 465
				"CREATE INDEX IF NOT EXISTS idx_distribution_policy ON policy_distribution_logs(policy_id)",
				// Line 466
				"CREATE INDEX IF NOT EXISTS idx_distribution_cabinet ON policy_distribution_logs(cabinet_id)",
				// Line 467
				"CREATE INDEX IF NOT EXISTS idx_distribution_status ON policy_distribution_logs(status)",
				// Line 468
				"CREATE INDEX IF NOT EXISTS idx_distribution_time ON policy_distribution_logs(distributed_at DESC)",
			},
		},
	}

	// 统计索引创建情况
	var totalIndexes, successCount, failedCount int

	// 执行索引创建
	for _, group := range indexGroups {
		logger.Info("创建表索引",
			zap.String("table", group.tableName),
			zap.Int("count", len(group.indexes)))

		for _, indexSQL := range group.indexes {
			totalIndexes++
			if err := c.createSingleIndex(ctx, c.pool, indexSQL); err != nil {
				// 索引创建失败记录警告但不中断（允许部分失败）
				logger.Warn("索引创建失败",
					zap.String("table", group.tableName),
					zap.String("sql", indexSQL),
					zap.Error(err))
				failedCount++
			} else {
				successCount++
			}
		}
	}

	logger.Info("数据库索引创建完成",
		zap.Int("total", totalIndexes),
		zap.Int("success", successCount),
		zap.Int("failed", failedCount))

	// 只要有成功的索引就认为成功（允许部分失败）
	if successCount > 0 {
		return nil
	}

	// 如果所有索引都失败了，返回错误
	if failedCount == totalIndexes {
		return fmt.Errorf("所有索引创建均失败: %d/%d", failedCount, totalIndexes)
	}

	return nil
}

// createSingleIndex 创建单个索引
func (c *Client) createSingleIndex(ctx context.Context, pool *pgxpool.Pool, indexSQL string) error {
	logger := utils.GetLogger()

	_, err := pool.Exec(ctx, indexSQL)
	if err != nil {
		// 检查是否是索引已存在的错误（尽管使用了IF NOT EXISTS，某些情况下仍可能报错）
		if strings.Contains(err.Error(), "already exists") {
			logger.Debug("索引已存在（跳过）", zap.String("sql", indexSQL))
			return nil
		}
		return fmt.Errorf("执行索引创建失败: %w", err)
	}

	return nil
}
