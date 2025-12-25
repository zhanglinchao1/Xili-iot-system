/*
 * 云端同步服务
 * 负责将本地数据同步到云端监控平台
 */
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// CloudSync 云端同步服务
type CloudSync struct {
	logger        *zap.Logger
	db            *storage.SQLiteDB
	storage       *storage.SQLiteDB
	config        config.CloudConfig
	client        *http.Client
	syncInterval  time.Duration
	retryCount    int
	retryInterval time.Duration
	stopChan      chan struct{}
	running       bool
}

// NewCloudSync 创建云端同步服务
func NewCloudSync(cfg config.CloudConfig, db *storage.SQLiteDB, storageService *storage.SQLiteDB, logger *zap.Logger, syncInterval time.Duration) *CloudSync {
	// 如果未指定同步间隔，使用默认值3分钟（平衡及时性和资源消耗）
	if syncInterval <= 0 {
		syncInterval = 3 * time.Minute
		logger.Warn("未指定同步间隔，使用默认值3分钟")
	}

	// 尝试从数据库加载API凭证
	if storageService != nil {
		cred, err := storageService.GetFirstCloudCredentials()
		if err == nil && cred != nil {
			// 从数据库读取成功,覆盖配置文件中的值
			cfg.APIKey = cred.APIKey
			// 注意: Edge端CloudConfig没有APISecret字段,API Secret已废弃
			if cred.CloudEndpoint != "" {
				cfg.Endpoint = cred.CloudEndpoint
			}
			logger.Info("从数据库加载Cloud凭证",
				zap.String("cabinet_id", cred.CabinetID),
				zap.String("api_key", maskAPIKey(cred.APIKey)),
				zap.String("endpoint", cred.CloudEndpoint))
		} else if err != nil {
			logger.Warn("从数据库加载凭证失败,使用配置文件凭证", zap.Error(err))
		} else {
			logger.Warn("数据库中没有凭证,使用配置文件凭证")
		}
	}

	return &CloudSync{
		logger:        logger,
		db:            db,
		storage:       storageService,
		config:        cfg,
		client:        &http.Client{Timeout: cfg.Timeout},
		syncInterval:  syncInterval,
		retryCount:    cfg.RetryCount,
		retryInterval: cfg.RetryInterval,
		stopChan:      make(chan struct{}),
	}
}

// maskAPIKey 脱敏显示API Key
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// getAPIKey 获取当前有效的API Key（优先从数据库读取）
func (cs *CloudSync) getAPIKey() string {
	// 优先从数据库读取
	if cs.storage != nil {
		cred, err := cs.storage.GetFirstCloudCredentials()
		if err == nil && cred != nil && cred.APIKey != "" {
			return cred.APIKey
		}
	}
	// 回退到配置文件中的值
	return cs.config.APIKey
}

// getEndpoint 获取当前有效的Cloud端点（优先从数据库读取）
func (cs *CloudSync) getEndpoint() string {
	// 优先从数据库读取
	if cs.storage != nil {
		cred, err := cs.storage.GetFirstCloudCredentials()
		if err == nil && cred != nil && cred.CloudEndpoint != "" {
			return cred.CloudEndpoint
		}
	}
	// 回退到配置文件中的值
	return cs.config.Endpoint
}

// Start 启动云端同步服务
func (cs *CloudSync) Start(ctx context.Context) error {
	if !cs.config.Enabled {
		cs.logger.Info("云端同步服务已禁用")
		return nil
	}

	cs.running = true
	cs.logger.Info("云端同步服务启动")

	// 启动定时同步
	go cs.syncLoop(ctx)

	return nil
}

// Stop 停止云端同步服务
func (cs *CloudSync) Stop() {
	if !cs.running {
		return
	}

	cs.running = false
	close(cs.stopChan)
	cs.logger.Info("云端同步服务已停止")
}

// syncLoop 同步循环
func (cs *CloudSync) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(cs.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopChan:
			return
		case <-ticker.C:
			// 同步传感器数据
			if err := cs.syncData(); err != nil {
				cs.logger.Error("数据同步失败", zap.Error(err))
			}
			// 同步告警
			if err := cs.SyncAlerts(); err != nil {
				cs.logger.Error("告警同步失败", zap.Error(err))
			}
			// 同步脆弱性评估数据（定期补漏）
			if err := cs.syncUnsyncedVulnerabilityAssessments(); err != nil {
				cs.logger.Error("脆弱性评估同步失败", zap.Error(err))
			}
		}
	}
}

// syncData 同步传感器数据到云端（不包括告警，告警通过SyncAlerts()单独同步）
func (cs *CloudSync) syncData() error {
	// 获取未同步的传感器数据
	sensorData, err := cs.getUnsyncedSensorData()
	if err != nil {
		return fmt.Errorf("获取未同步传感器数据失败: %w", err)
	}

	if len(sensorData) == 0 {
		cs.logger.Debug("没有需要同步的传感器数据")
		return nil
	}

	// 构建同步负载（不包含告警）
	payload := models.CloudSyncPayload{
		CabinetID:  cs.getCabinetID(),
		Timestamp:  time.Now(),
		SensorData: sensorData,
		Alerts:     []models.Alert{}, // 空告警列表
	}

	// 发送到云端
	if err := cs.sendToCloud(&payload); err != nil {
		return fmt.Errorf("发送数据到云端失败: %w", err)
	}

	// 标记数据为已同步（只标记传感器数据）
	if err := cs.markAsSynced(sensorData, []models.Alert{}); err != nil {
		cs.logger.Error("标记传感器数据为已同步失败", zap.Error(err))
		// 不返回错误，避免重复发送
	}

	cs.logger.Info("传感器数据同步成功",
		zap.Int("sensor_data_count", len(sensorData)))

	return nil
}

// getUnsyncedSensorData 获取未同步的传感器数据
func (cs *CloudSync) getUnsyncedSensorData() ([]models.SensorData, error) {
	query := `
		SELECT id, device_id, sensor_type, value, unit, timestamp, quality
		FROM sensor_data 
		WHERE synced = false 
		ORDER BY timestamp ASC 
		LIMIT 1000`

	rows, err := cs.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.SensorData
	for rows.Next() {
		var d models.SensorData
		err := rows.Scan(&d.ID, &d.DeviceID, &d.SensorType, &d.Value, &d.Unit, &d.Timestamp, &d.Quality)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	return data, nil
}

// getUnsyncedAlerts 获取未同步的告警数据
func (cs *CloudSync) getUnsyncedAlerts() ([]models.Alert, error) {
	query := `
		SELECT id, device_id, alert_type, severity, message, value, threshold, timestamp, resolved, resolved_at
		FROM alerts
		WHERE synced_at IS NULL
		   OR (resolved = 1 AND resolved_at > synced_at)
		ORDER BY timestamp ASC
		LIMIT 100`

	cs.logger.Debug("开始查询未同步告警", zap.String("query", query))

	rows, err := cs.db.Query(query)
	if err != nil {
		cs.logger.Error("查询未同步告警失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(&alert.ID, &alert.DeviceID, &alert.AlertType, &alert.Severity,
			&alert.Message, &alert.Value, &alert.Threshold, &alert.Timestamp,
			&alert.Resolved, &alert.ResolvedAt)
		if err != nil {
			cs.logger.Error("扫描告警数据失败", zap.Error(err))
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	cs.logger.Info("查询到未同步告警", zap.Int("count", len(alerts)))
	return alerts, nil
}

// sendToCloud 发送数据到云端
func (cs *CloudSync) sendToCloud(payload *models.CloudSyncPayload) error {
	// 序列化数据
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 构建请求
	url := fmt.Sprintf("%s/cabinets/%s/sync", cs.getEndpoint(), payload.CabinetID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cs.getAPIKey())
	req.Header.Set("User-Agent", "Edge-System/1.0")

	// 发送请求（带重试）
	var lastErr error
	for i := 0; i <= cs.retryCount; i++ {
		resp, err := cs.client.Do(req)
		if err != nil {
			lastErr = err
			if i < cs.retryCount {
				cs.logger.Warn("请求失败，准备重试",
					zap.Int("attempt", i+1),
					zap.Error(err))
				time.Sleep(cs.retryInterval)
				continue
			}
			break
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // 成功
		}

		lastErr = fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
		if i < cs.retryCount {
			cs.logger.Warn("HTTP请求失败，准备重试",
				zap.Int("attempt", i+1),
				zap.Int("status_code", resp.StatusCode))
			time.Sleep(cs.retryInterval)
		}
	}

	return fmt.Errorf("发送数据失败，已重试%d次: %w", cs.retryCount, lastErr)
}

// markAsSynced 标记数据为已同步
func (cs *CloudSync) markAsSynced(sensorData []models.SensorData, alerts []models.Alert) error {
	now := time.Now()

	// 标记传感器数据为已同步
	if len(sensorData) > 0 {
		var ids []interface{}
		for _, data := range sensorData {
			ids = append(ids, data.ID)
		}

		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
		}

		query := fmt.Sprintf("UPDATE sensor_data SET synced = true, synced_at = ? WHERE id IN (%s)",
			strings.Join(placeholders, ","))
		args := append([]interface{}{now}, ids...)

		_, err := cs.db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("标记传感器数据为已同步失败: %w", err)
		}
	}

	// 标记告警为已同步
	if len(alerts) > 0 {
		var ids []interface{}
		for _, alert := range alerts {
			ids = append(ids, alert.ID)
		}

		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
		}

		query := fmt.Sprintf("UPDATE alerts SET synced_at = ? WHERE id IN (%s)",
			strings.Join(placeholders, ","))
		args := append([]interface{}{now}, ids...)

		_, err := cs.db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("标记告警为已同步失败: %w", err)
		}
	}

	return nil
}

// getCabinetID 获取储能柜ID
func (cs *CloudSync) getCabinetID() string {
	// 优先从配置读取
	if cs.config.CabinetID != "" {
		return cs.config.CabinetID
	}

	// 如果配置为空，记录警告并返回默认值
	cs.logger.Warn("储能柜ID未配置，使用默认值",
		zap.String("default_cabinet_id", "CABINET-001"),
		zap.String("config_hint", "请在config.yaml中设置cloud.cabinet_id"))

	return "CABINET-001"
}

// SyncNow 立即执行一次同步
func (cs *CloudSync) SyncNow() error {
	if !cs.config.Enabled {
		return fmt.Errorf("云端同步服务未启用")
	}

	return cs.syncData()
}

// GetSyncStatus 获取同步状态
func (cs *CloudSync) GetSyncStatus() map[string]interface{} {
	// 获取未同步数据统计
	unsyncedSensorCount := cs.getUnsyncedCount("sensor_data", "synced = false")
	unsyncedAlertCount := cs.getUnsyncedCount("alerts", "synced_at IS NULL")

	return map[string]interface{}{
		"enabled":               cs.config.Enabled,
		"running":               cs.running,
		"unsynced_sensor_data":  unsyncedSensorCount,
		"unsynced_alerts":       unsyncedAlertCount,
		"last_sync_time":        time.Now(), // TODO: 记录实际的最后同步时间
		"sync_interval_seconds": cs.syncInterval.Seconds(),
	}
}

// getUnsyncedCount 获取未同步数据数量
func (cs *CloudSync) getUnsyncedCount(table, condition string) int {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, condition)
	var count int
	err := cs.db.QueryRow(query).Scan(&count)
	if err != nil {
		cs.logger.Error("获取未同步数据数量失败", zap.Error(err))
		return 0
	}
	return count
}

// SyncVulnerabilityAssessment 同步脆弱性评估到云端
func (cs *CloudSync) SyncVulnerabilityAssessment(report *models.EdgeVulnerabilityReport) error {
	if !cs.config.Enabled {
		cs.logger.Debug("云端同步服务未启用，跳过脆弱性评估同步")
		return nil
	}

	// 构建Cloud端期望的同步请求格式
	syncRequest := map[string]interface{}{
		"cabinet_id":               cs.getCabinetID(),
		"timestamp":                report.Timestamp,
		"license_compliance_score": report.LicenseComplianceScore,
		"communication_score":      report.CommunicationScore,
		"config_security_score":    report.ConfigSecurityScore,
		"data_anomaly_score":       report.DataAnomalyScore,
		"overall_score":            report.OverallScore,
		"risk_level":               report.RiskLevel,
	}

	// 添加可选的详细数据
	if report.TransmissionMetrics != nil {
		syncRequest["transmission_metrics"] = report.TransmissionMetrics
	}
	if report.TrafficFeatures != nil {
		syncRequest["traffic_features"] = report.TrafficFeatures
	}
	if report.ConfigChecks != nil {
		syncRequest["config_checks"] = report.ConfigChecks
	}

	// 转换漏洞事件为Cloud端格式
	if len(report.DetectedVulnerabilities) > 0 {
		var events []map[string]interface{}
		for _, vuln := range report.DetectedVulnerabilities {
			event := map[string]interface{}{
				"type":        vuln.Type,
				"category":    vuln.Category,
				"title":       vuln.Title,
				"severity":    vuln.Severity,
				"description": vuln.Description,
				"solution":    vuln.Solution,
				"detected_at": vuln.DetectedAt,
			}
			events = append(events, event)
		}
		syncRequest["detected_vulnerabilities"] = events
	}

	// 序列化数据
	jsonData, err := json.Marshal(syncRequest)
	if err != nil {
		return fmt.Errorf("序列化脆弱性评估数据失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/cabinets/%s/vulnerability/sync", cs.getEndpoint(), cs.getCabinetID())

	// 发送请求（带重试）
	var lastErr error
	for i := 0; i <= cs.retryCount; i++ {
		// 每次重试都需要重新创建请求,因为Body只能读取一次
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("创建脆弱性评估同步请求失败: %w", err)
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cs.getAPIKey())
		req.Header.Set("User-Agent", "Edge-System/1.0")

		resp, err := cs.client.Do(req)
		if err != nil {
			lastErr = err
			if i < cs.retryCount {
				cs.logger.Warn("脆弱性评估同步请求失败，准备重试",
					zap.Int("attempt", i+1),
					zap.Error(err))
				time.Sleep(cs.retryInterval)
				continue
			}
			break
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// 标记为已同步
			if err := cs.markVulnerabilityAsSynced(report.ID); err != nil {
				cs.logger.Error("标记脆弱性评估为已同步失败", zap.Error(err))
			}

			cs.logger.Info("脆弱性评估同步成功",
				zap.Int64("assessment_id", report.ID),
				zap.Float64("overall_score", report.OverallScore),
				zap.String("risk_level", report.RiskLevel))
			return nil
		}

		lastErr = fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
		if i < cs.retryCount {
			cs.logger.Warn("脆弱性评估同步HTTP请求失败，准备重试",
				zap.Int("attempt", i+1),
				zap.Int("status_code", resp.StatusCode))
			time.Sleep(cs.retryInterval)
		}
	}

	return fmt.Errorf("脆弱性评估同步失败，已重试%d次: %w", cs.retryCount, lastErr)
}

// markVulnerabilityAsSynced 标记脆弱性评估为已同步
func (cs *CloudSync) markVulnerabilityAsSynced(assessmentID int64) error {
	query := `UPDATE vulnerability_assessments SET synced = 1, synced_at = ? WHERE id = ?`
	_, err := cs.db.Exec(query, time.Now(), assessmentID)
	if err != nil {
		return fmt.Errorf("更新同步状态失败: %w", err)
	}
	return nil
}

// syncUnsyncedVulnerabilityAssessments 定期同步未同步的脆弱性评估数据
// 功能说明：
// - 作为脆弱性评估同步的兜底机制
// - 同步所有synced=0的评估数据到Cloud端
// - 每次最多同步10条，避免请求过大
func (cs *CloudSync) syncUnsyncedVulnerabilityAssessments() error {
	if !cs.config.Enabled {
		return nil
	}

	// 查询未同步的脆弱性评估（限制数量，按时间倒序取最新的）
	query := `
		SELECT id, cabinet_id, timestamp, license_compliance_score, communication_score, 
		       config_security_score, data_anomaly_score, overall_score, risk_level,
		       transmission_metrics, traffic_features, config_checks, detected_vulnerabilities
		FROM vulnerability_assessments 
		WHERE synced = 0 
		ORDER BY timestamp DESC 
		LIMIT 10`

	rows, err := cs.db.Query(query)
	if err != nil {
		return fmt.Errorf("查询未同步脆弱性评估失败: %w", err)
	}
	defer rows.Close()

	var syncCount int
	for rows.Next() {
		var id int64
		var cabinetID, riskLevel string
		var timestamp time.Time
		var licenseScore, commScore, configScore, dataScore, overallScore float64
		var metricsJSON, featuresJSON, checksJSON, vulnJSON string

		err := rows.Scan(&id, &cabinetID, &timestamp, &licenseScore, &commScore,
			&configScore, &dataScore, &overallScore, &riskLevel,
			&metricsJSON, &featuresJSON, &checksJSON, &vulnJSON)
		if err != nil {
			cs.logger.Warn("扫描脆弱性评估记录失败", zap.Error(err))
			continue
		}

		// 构建同步请求
		report := &models.EdgeVulnerabilityReport{
			ID:                     id,
			CabinetID:              cabinetID,
			Timestamp:              timestamp,
			LicenseComplianceScore: licenseScore,
			CommunicationScore:     commScore,
			ConfigSecurityScore:    configScore,
			DataAnomalyScore:       dataScore,
			OverallScore:           overallScore,
			RiskLevel:              riskLevel,
		}

		// 解析JSON字段
		if metricsJSON != "" {
			json.Unmarshal([]byte(metricsJSON), &report.TransmissionMetrics)
		}
		if featuresJSON != "" {
			json.Unmarshal([]byte(featuresJSON), &report.TrafficFeatures)
		}
		if checksJSON != "" {
			json.Unmarshal([]byte(checksJSON), &report.ConfigChecks)
		}
		if vulnJSON != "" {
			json.Unmarshal([]byte(vulnJSON), &report.DetectedVulnerabilities)
		}

		// 同步到Cloud
		if err := cs.SyncVulnerabilityAssessment(report); err != nil {
			cs.logger.Warn("脆弱性评估同步失败",
				zap.Int64("assessment_id", id),
				zap.Error(err))
			// 继续尝试同步下一条
			continue
		}

		syncCount++
	}

	if syncCount > 0 {
		cs.logger.Info("定期脆弱性评估同步完成", zap.Int("synced_count", syncCount))
	}

	return nil
}

// SyncAlerts 同步告警到Cloud端（兜底机制）
// 功能说明：
// - 主要作为MQTT告警推送的兜底机制
// - 只同步synced_at为NULL的告警（MQTT发布失败或HTTP上报失败的告警）
// - 正常情况下，告警通过MQTT实时推送后已标记为已同步，这里应该很少有数据
func (cs *CloudSync) SyncAlerts() error {
	cs.logger.Debug("触发告警兜底同步检查")

	if !cs.config.Enabled {
		cs.logger.Warn("云端同步服务未启用，跳过告警同步")
		return nil
	}

	// 获取未同步的告警（synced_at IS NULL）
	cs.logger.Debug("正在查询未同步告警...")
	alerts, err := cs.getUnsyncedAlerts()
	if err != nil {
		cs.logger.Error("获取未同步告警失败", zap.Error(err))
		return fmt.Errorf("获取未同步告警失败: %w", err)
	}

	if len(alerts) == 0 {
		// 没有未同步的告警，说明MQTT推送工作正常
		cs.logger.Debug("没有需要兜底同步的告警（MQTT推送工作正常）")
		return nil
	}

	// 发现未同步告警，执行兜底同步
	cs.logger.Warn("发现未同步告警，执行兜底同步（可能MQTT推送失败）",
		zap.Int("alert_count", len(alerts)))

	// 构建Cloud端期望的同步请求格式
	syncRequest := map[string]interface{}{
		"cabinet_id": cs.getCabinetID(),
		"timestamp":  time.Now(),
		"alerts":     []map[string]interface{}{},
	}

	// 转换告警数据为Cloud端格式
	alertsData := []map[string]interface{}{}
	for _, alert := range alerts {
		alertData := map[string]interface{}{
			"alert_id":   alert.ID,
			"device_id":  alert.DeviceID,
			"alert_type": alert.AlertType,
			"severity":   mapSeverityToCloud(alert.Severity), // 映射severity
			"message":    alert.Message,
			"timestamp":  alert.Timestamp,
			"resolved":   alert.Resolved,
		}

		// 添加可选字段
		if alert.Value != nil {
			alertData["value"] = *alert.Value
		}
		if alert.Threshold != nil {
			alertData["threshold"] = *alert.Threshold
		}
		if alert.ResolvedAt != nil {
			alertData["resolved_at"] = alert.ResolvedAt
		}

		alertsData = append(alertsData, alertData)
	}
	syncRequest["alerts"] = alertsData

	// 序列化数据
	jsonData, err := json.Marshal(syncRequest)
	if err != nil {
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/cabinets/%s/alerts/sync", cs.getEndpoint(), cs.getCabinetID())

	// 发送请求（带重试）
	var lastErr error
	for i := 0; i <= cs.retryCount; i++ {
		// 每次重试都需要重新创建请求,因为Body只能读取一次
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("创建告警同步请求失败: %w", err)
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cs.getAPIKey())
		req.Header.Set("User-Agent", "Edge-System/1.0")

		resp, err := cs.client.Do(req)
		if err != nil {
			lastErr = err
			if i < cs.retryCount {
				cs.logger.Warn("告警同步请求失败，准备重试",
					zap.Int("attempt", i+1),
					zap.Error(err))
				time.Sleep(cs.retryInterval)
				continue
			}
			break
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// 标记告警为已同步
			if err := cs.markAlertAsSynced(alerts); err != nil {
				cs.logger.Error("标记告警为已同步失败", zap.Error(err))
			}

			cs.logger.Info("告警同步成功",
				zap.Int("alert_count", len(alerts)))
			return nil
		}

		lastErr = fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
		if i < cs.retryCount {
			cs.logger.Warn("告警同步HTTP请求失败，准备重试",
				zap.Int("attempt", i+1),
				zap.Int("status_code", resp.StatusCode))
			time.Sleep(cs.retryInterval)
		}
	}

	return fmt.Errorf("告警同步失败，已重试%d次: %w", cs.retryCount, lastErr)
}

// mapSeverityToCloud 将Edge端severity映射到Cloud端格式
// Edge: low, medium, high, critical
// Cloud: info, warning, error, critical
func mapSeverityToCloud(edgeSeverity string) string {
	switch edgeSeverity {
	case "low":
		return "info"
	case "medium":
		return "warning"
	case "high":
		return "error"
	case "critical":
		return "critical"
	default:
		return "warning" // 默认返回warning
	}
}

// ReportAlertImmediately 立即上报单个告警到Cloud端（用于告警产生时的即时同步）
func (cs *CloudSync) ReportAlertImmediately(alert *models.Alert) error {
	if !cs.config.Enabled {
		cs.logger.Debug("云端同步服务未启用，跳过即时告警上报")
		return nil
	}

	cs.logger.Info("立即上报告警到Cloud端",
		zap.String("device_id", alert.DeviceID),
		zap.String("alert_type", alert.AlertType),
		zap.String("severity", alert.Severity))

	// 构建Cloud端期望的同步请求格式（需要映射severity）
	alertData := map[string]interface{}{
		"alert_id":   alert.ID,
		"device_id":  alert.DeviceID,
		"alert_type": alert.AlertType,
		"severity":   mapSeverityToCloud(alert.Severity), // 映射severity
		"message":    alert.Message,
		"timestamp":  alert.Timestamp,
		"resolved":   alert.Resolved,
	}

	// 添加可选字段
	if alert.Value != nil {
		alertData["value"] = *alert.Value
	}
	if alert.Threshold != nil {
		alertData["threshold"] = *alert.Threshold
	}
	if alert.ResolvedAt != nil {
		alertData["resolved_at"] = alert.ResolvedAt
	}

	syncRequest := map[string]interface{}{
		"cabinet_id": cs.getCabinetID(),
		"timestamp":  time.Now(),
		"alerts":     []map[string]interface{}{alertData},
	}

	// 序列化数据
	jsonData, err := json.Marshal(syncRequest)
	if err != nil {
		cs.logger.Error("序列化告警数据失败", zap.Error(err))
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/cabinets/%s/alerts/sync", cs.getEndpoint(), cs.getCabinetID())

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		cs.logger.Error("创建告警上报请求失败", zap.Error(err))
		return fmt.Errorf("创建告警上报请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cs.getAPIKey())
	req.Header.Set("User-Agent", "Edge-System/1.0")

	// 发送请求（不重试，失败后由定时同步补偿）
	resp, err := cs.client.Do(req)
	if err != nil {
		cs.logger.Warn("即时告警上报失败（将由定时同步补偿）", zap.Error(err))
		return nil // 返回nil，不影响告警创建流程
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// 标记告警为已同步
		if err := cs.markAlertAsSynced([]models.Alert{*alert}); err != nil {
			cs.logger.Error("标记告警为已同步失败", zap.Error(err))
		}

		cs.logger.Info("告警即时上报成功",
			zap.String("device_id", alert.DeviceID),
			zap.String("alert_type", alert.AlertType))
		return nil
	}

	cs.logger.Warn("即时告警上报HTTP失败（将由定时同步补偿）",
		zap.Int("status_code", resp.StatusCode))
	return nil // 返回nil，不影响告警创建流程
}

// markAlertAsSynced 标记告警为已同步
func (cs *CloudSync) markAlertAsSynced(alerts []models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	var ids []interface{}
	for _, alert := range alerts {
		ids = append(ids, alert.ID)
	}

	placeholders := make([]string, len(ids))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("UPDATE alerts SET synced_at = ? WHERE id IN (%s)",
		strings.Join(placeholders, ","))
	args := append([]interface{}{time.Now()}, ids...)

	_, err := cs.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("标记告警为已同步失败: %w", err)
	}

	return nil
}

// SyncCabinetInfo 同步储能柜信息到Cloud端
// 用于Edge前端保存储能柜信息时，通过Edge后端API同步到Cloud
func (cs *CloudSync) SyncCabinetInfo(cabinetID, name, location string, latitude, longitude, capacityKWh *float64) error {
	if !cs.config.Enabled {
		return fmt.Errorf("Cloud端未启用")
	}

	if cabinetID == "" {
		return fmt.Errorf("储能柜ID不能为空")
	}

	// 从数据库实时读取API Key（优先使用数据库中的凭证）
	apiKey := cs.getAPIKey()
	endpoint := cs.getEndpoint()

	// 检查API Key是否有效
	if apiKey == "" {
		return fmt.Errorf("API Key未配置，请先注册到Cloud端获取API Key")
	}

	// 使用配置文件中的cabinet_id作为Cloud端识别的ID
	// 这样即使用户修改了cabinet_id，也能正确更新Cloud端的记录
	cloudCabinetID := cs.config.CabinetID
	if cloudCabinetID == "" {
		// 如果配置中没有，使用传入的ID（首次同步的情况）
		cloudCabinetID = cabinetID
	}

	// 构建同步数据
	syncData := map[string]interface{}{
		"name": name,
	}

	if location != "" {
		syncData["location"] = location
	}
	if latitude != nil {
		syncData["latitude"] = *latitude
	}
	if longitude != nil {
		syncData["longitude"] = *longitude
	}
	if capacityKWh != nil {
		syncData["capacity_kwh"] = *capacityKWh
	}

	// 构建请求 - 使用Cloud端认可的cabinet_id
	url := fmt.Sprintf("%s/cabinets/%s/sync", endpoint, cloudCabinetID)

	payload, err := json.Marshal(syncData)
	if err != nil {
		return fmt.Errorf("序列化储能柜信息失败: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，包含API Key（从数据库读取）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "Edge-System/1.0")

	// 发送请求
	resp, err := cs.client.Do(req)
	if err != nil {
		return fmt.Errorf("同步储能柜信息请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("同步储能柜信息失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	cs.logger.Info("储能柜信息同步成功",
		zap.String("cabinet_id", cabinetID),
		zap.String("name", name),
	)

	return nil
}
