/*
 * 数据采集服务
 * 负责传感器数据的采集、存储和处理
 */
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/device"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// Service 数据采集服务
type Service struct {
	logger          *zap.Logger
	db              *storage.SQLiteDB
	deviceManager   *device.Manager
	rs485Collector  *RS485Collector
	dataChan        chan *models.SensorData
	alertChan       chan *models.Alert
	bufferSize      int
	batchSize       int
	collectInterval time.Duration
	syncInterval    time.Duration
	retentionDays   int
	thresholds      map[models.SensorType]*models.SensorThreshold
	mu              sync.RWMutex
	running         bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	cloudSync       CloudSyncInterface      // 云端同步接口（用于即时告警上报）
	alertPublisher  AlertPublisherInterface // MQTT告警发布器（用于实时推送）
}

// CloudSyncInterface 定义云端同步接口（避免循环依赖）
type CloudSyncInterface interface {
	ReportAlertImmediately(alert *models.Alert) error
}

// AlertPublisherInterface 定义告警MQTT发布接口
type AlertPublisherInterface interface {
	PublishAlert(alert *models.Alert) error
	IsEnabled() bool
}

// NewService 创建数据采集服务
func NewService(cfg config.DataConfig, alertCfg config.AlertConfig, db *storage.SQLiteDB, deviceManager *device.Manager, logger *zap.Logger) *Service {
	return &Service{
		logger:          logger,
		db:              db,
		deviceManager:   deviceManager,
		dataChan:        make(chan *models.SensorData, cfg.BufferSize),
		alertChan:       make(chan *models.Alert, 1000),
		bufferSize:      cfg.BufferSize,
		batchSize:       cfg.BatchSize,
		collectInterval: cfg.CollectInterval,
		syncInterval:    cfg.SyncInterval,
		retentionDays:   cfg.RetentionDays,
		thresholds:      initThresholdsFromConfig(alertCfg),
		stopChan:        make(chan struct{}),
	}
}

// SetCloudSync 设置云端同步接口（用于即时告警上报）
func (s *Service) SetCloudSync(cloudSync CloudSyncInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cloudSync = cloudSync
	s.logger.Info("Cloud sync interface set for immediate alert reporting")
}

// SetAlertPublisher 设置告警MQTT发布器（用于实时推送）
func (s *Service) SetAlertPublisher(publisher AlertPublisherInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertPublisher = publisher
	s.logger.Info("Alert MQTT publisher set for real-time alert notification")
}

// initThresholdsFromConfig 从配置文件初始化传感器阈值
func initThresholdsFromConfig(alertCfg config.AlertConfig) map[models.SensorType]*models.SensorThreshold {
	if !alertCfg.Enabled {
		return map[models.SensorType]*models.SensorThreshold{}
	}

	t := alertCfg.Thresholds
	return map[models.SensorType]*models.SensorThreshold{
		models.SensorCO2: {
			SensorType: models.SensorCO2,
			MinValue:   0,
			MaxValue:   t.CO2Max,
			Unit:       "ppm",
		},
		models.SensorCO: {
			SensorType: models.SensorCO,
			MinValue:   0,
			MaxValue:   t.COMax,
			Unit:       "ppm",
		},
		models.SensorSmoke: {
			SensorType: models.SensorSmoke,
			MinValue:   0,
			MaxValue:   t.SmokeMax,
			Unit:       "ppm",
		},
		models.SensorLiquidLevel: {
			SensorType: models.SensorLiquidLevel,
			MinValue:   t.LiquidLevelMin,
			MaxValue:   t.LiquidLevelMax,
			Unit:       "mm",
		},
		models.SensorConductivity: {
			SensorType: models.SensorConductivity,
			MinValue:   t.ConductivityMin,
			MaxValue:   t.ConductivityMax,
			Unit:       "mS/cm",
		},
		models.SensorTemperature: {
			SensorType: models.SensorTemperature,
			MinValue:   t.TemperatureMin,
			MaxValue:   t.TemperatureMax,
			Unit:       "°C",
		},
		models.SensorFlow: {
			SensorType: models.SensorFlow,
			MinValue:   t.FlowMin,
			MaxValue:   t.FlowMax,
			Unit:       "L/min",
		},
	}
}

// SetRS485Collector 设置RS485采集器
func (s *Service) SetRS485Collector(rs485 *RS485Collector) {
	s.rs485Collector = rs485
}

// Start 启动数据采集
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("collector already running")
	}
	s.running = true
	s.mu.Unlock()

	// 启动RS485采集器
	if s.rs485Collector != nil {
		if err := s.rs485Collector.Start(); err != nil {
			return fmt.Errorf("failed to start RS485 collector: %w", err)
		}

		// 启动RS485数据接收协程
		s.wg.Add(1)
		go s.receiveRS485Data()
	}

	// 启动数据处理协程
	s.wg.Add(1)
	go s.processData()

	// 启动批量存储协程
	s.wg.Add(1)
	go s.batchStore()

	// 启动告警处理协程
	s.wg.Add(1)
	go s.processAlerts()

	// 启动数据清理协程
	s.wg.Add(1)
	go s.cleanupOldData()

	s.logger.Info("Data collector started")
	return nil
}

// Stop 停止数据采集
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	// 停止RS485采集器
	if s.rs485Collector != nil {
		s.rs485Collector.Stop()
	}

	close(s.stopChan)
	s.wg.Wait()

	s.logger.Info("Data collector stopped")
}

// CollectData 接收传感器数据
func (s *Service) CollectData(req *models.DataCollectRequest) error {
	// 创建传感器数据记录
	data := &models.SensorData{
		DeviceID:   req.DeviceID,
		SensorType: req.SensorType,
		Value:      req.Value,
		Unit:       req.Unit,
		Timestamp:  req.Timestamp,
		Quality:    req.Quality,
		Synced:     false,
	}

	// 数据质量检查
	if data.Quality == 0 {
		data.Quality = 100 // 默认质量为100
	}

	// 时间戳检查
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// 检查数据是否超出阈值
	if err := s.checkThreshold(data); err != nil {
		s.logger.Warn("Data threshold exceeded",
			zap.String("device_id", data.DeviceID),
			zap.String("sensor_type", string(data.SensorType)),
			zap.Float64("value", data.Value),
			zap.Error(err))
	}

	// 发送到数据通道
	select {
	case s.dataChan <- data:
		s.logger.Debug("Data collected",
			zap.String("device_id", data.DeviceID),
			zap.String("sensor_type", string(data.SensorType)),
			zap.Float64("value", data.Value))
	default:
		s.logger.Warn("Data channel full, dropping data",
			zap.String("device_id", data.DeviceID))
		return fmt.Errorf("data buffer full")
	}

	return nil
}

// receiveRS485Data 接收RS485数据
func (s *Service) receiveRS485Data() {
	defer s.wg.Done()

	if s.rs485Collector == nil {
		return
	}

	dataChan := s.rs485Collector.GetDataChannel()

	for {
		select {
		case <-s.stopChan:
			return
		case frame := <-dataChan:
			if frame == nil {
				continue
			}

			// 转换为传感器数据
			data := &models.SensorData{
				DeviceID:   frame.DeviceID,
				SensorType: frame.SensorType,
				Value:      frame.Value,
				Unit:       frame.Unit,
				Timestamp:  frame.Timestamp,
				Quality:    frame.Quality,
				Synced:     false,
			}

			// 检查阈值并发送
			if err := s.checkThreshold(data); err != nil {
				s.logger.Warn("RS485 data threshold exceeded",
					zap.String("device_id", data.DeviceID),
					zap.Error(err))
			}

			select {
			case s.dataChan <- data:
			default:
				s.logger.Warn("Data channel full from RS485")
			}
		}
	}
}

// processData 处理数据
func (s *Service) processData() {
	defer s.wg.Done()

	batch := make([]*models.SensorData, 0, s.batchSize)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			// 保存剩余数据
			if len(batch) > 0 {
				s.storeBatch(batch)
			}
			return

		case data := <-s.dataChan:
			batch = append(batch, data)

			// 批量存储
			if len(batch) >= s.batchSize {
				s.storeBatch(batch)
				batch = make([]*models.SensorData, 0, s.batchSize)
			}

		case <-ticker.C:
			// 定时存储
			if len(batch) > 0 {
				s.storeBatch(batch)
				batch = make([]*models.SensorData, 0, s.batchSize)
			}
		}
	}
}

// batchStore 批量存储协程
func (s *Service) batchStore() {
	defer s.wg.Done()
	// 这个协程可以用于额外的批量操作或统计
}

// storeBatch 批量存储数据
func (s *Service) storeBatch(batch []*models.SensorData) {
	if len(batch) == 0 {
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return
	}
	defer tx.Rollback()

	query := `
		INSERT INTO sensor_data (device_id, sensor_type, value, unit, timestamp, quality, synced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := tx.Prepare(query)
	if err != nil {
		s.logger.Error("Failed to prepare statement", zap.Error(err))
		return
	}
	defer stmt.Close()

	for _, data := range batch {
		_, err := stmt.Exec(
			data.DeviceID, data.SensorType, data.Value,
			data.Unit, data.Timestamp, data.Quality, data.Synced,
		)
		if err != nil {
			s.logger.Error("Failed to insert data",
				zap.String("device_id", data.DeviceID),
				zap.Error(err))
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return
	}

	s.logger.Debug("Batch stored", zap.Int("count", len(batch)))
}

// checkThreshold 检查数据阈值
func (s *Service) checkThreshold(data *models.SensorData) error {
	threshold, exists := s.thresholds[data.SensorType]
	if !exists {
		return nil
	}

	var alertType models.AlertType
	var severity models.Severity
	var message string

	// 检查上下限
	if data.Value < threshold.MinValue {
		switch data.SensorType {
		case models.SensorLiquidLevel:
			alertType = models.AlertLiquidLevelLow
			message = fmt.Sprintf("液位过低: %.2f%s (下限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MinValue, threshold.Unit)
		case models.SensorTemperature:
			alertType = models.AlertTemperatureLow
			message = fmt.Sprintf("温度过低: %.2f%s (下限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MinValue, threshold.Unit)
		case models.SensorFlow:
			alertType = models.AlertFlowAbnormal
			message = fmt.Sprintf("流速过低: %.2f%s (下限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MinValue, threshold.Unit)
		case models.SensorConductivity: // 电导率
			alertType = models.AlertConductivityAbnormal
			message = fmt.Sprintf("电导率过低: %.2f%s (下限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MinValue, threshold.Unit)
		default:
			return nil
		}
		severity = models.SeverityMedium
	} else if data.Value > threshold.MaxValue {
		switch data.SensorType {
		case models.SensorCO2:
			alertType = models.AlertCO2High
			message = fmt.Sprintf("CO2浓度超标: %.2fppm (上限: %.2fppm)",
				data.Value, threshold.MaxValue)
			severity = models.SeverityHigh
		case models.SensorCO:
			alertType = models.AlertCOHigh
			message = fmt.Sprintf("CO浓度超标: %.2fppm (上限: %.2fppm)",
				data.Value, threshold.MaxValue)
			severity = models.SeverityCritical // CO更危险
		case models.SensorSmoke:
			alertType = models.AlertSmokeDetected
			message = fmt.Sprintf("检测到烟雾: %.2fppm (上限: %.2fppm)",
				data.Value, threshold.MaxValue)
			severity = models.SeverityCritical
		case models.SensorLiquidLevel:
			alertType = models.AlertLiquidLevelHigh
			message = fmt.Sprintf("液位过高: %.2f%s (上限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MaxValue, threshold.Unit)
			severity = models.SeverityMedium
		case models.SensorTemperature:
			alertType = models.AlertTemperatureHigh
			message = fmt.Sprintf("温度过高: %.2f%s (上限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MaxValue, threshold.Unit)
			severity = models.SeverityHigh
		case models.SensorFlow:
			alertType = models.AlertFlowAbnormal
			message = fmt.Sprintf("流速过高: %.2f%s (上限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MaxValue, threshold.Unit)
			severity = models.SeverityMedium
		case models.SensorConductivity: // 电导率
			alertType = models.AlertConductivityAbnormal
			message = fmt.Sprintf("电导率过高: %.2f%s (上限: %.2f%s)",
				data.Value, threshold.Unit, threshold.MaxValue, threshold.Unit)
			severity = models.SeverityMedium
		default:
			return nil
		}
	} else {
		return nil // 数据正常
	}

	// 创建告警
	valuePtr := data.Value
	thresholdPtr := threshold.MaxValue
	alert := &models.Alert{
		DeviceID:  data.DeviceID,
		AlertType: string(alertType),
		Severity:  string(severity),
		Message:   message,
		Value:     &valuePtr,
		Threshold: &thresholdPtr,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	// 发送告警
	select {
	case s.alertChan <- alert:
		s.logger.Warn("Alert triggered",
			zap.String("device_id", data.DeviceID),
			zap.String("alert_type", string(alertType)),
			zap.String("severity", string(severity)),
			zap.String("message", message))
	default:
		s.logger.Error("Alert channel full")
	}

	return fmt.Errorf("%s", message)
}

// processAlerts 处理告警
func (s *Service) processAlerts() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case alert := <-s.alertChan:
			if alert == nil {
				continue
			}

			// 保存告警到数据库
			if err := s.saveAlert(alert); err != nil {
				s.logger.Error("Failed to save alert", zap.Error(err))
			}

			// TODO: 发送告警通知（邮件、短信、推送等）
		}
	}
}

// saveAlert 保存告警（带去重：同一设备同一类型的未解决告警不重复创建）
func (s *Service) saveAlert(alert *models.Alert) error {
	s.logger.Debug("saveAlert called",
		zap.String("device_id", alert.DeviceID),
		zap.String("alert_type", alert.AlertType),
		zap.String("severity", alert.Severity))

	// 先检查是否存在相同的未解决告警
	checkQuery := `
		SELECT id, timestamp FROM alerts
		WHERE device_id = ? AND alert_type = ? AND resolved = 0
		ORDER BY timestamp DESC LIMIT 1
	`
	var existingID int64
	var existingTimestamp time.Time
	err := s.db.QueryRow(checkQuery, alert.DeviceID, alert.AlertType).Scan(&existingID, &existingTimestamp)

	s.logger.Debug("Check existing alert result",
		zap.Error(err),
		zap.Int64("existing_id", existingID))

	if err == nil {
		// 存在未解决的告警，更新时间戳和数值（表示告警仍在持续）
		// 同时重置synced_at为NULL，确保更新后的告警会被同步到Cloud端
		updateQuery := `
			UPDATE alerts
			SET timestamp = ?, message = ?, value = ?, threshold = ?, severity = ?, synced_at = NULL
			WHERE id = ?
		`
		_, err = s.db.Exec(updateQuery,
			alert.Timestamp, alert.Message, alert.Value, alert.Threshold, alert.Severity, existingID)
		if err == nil {
			s.logger.Debug("Updated existing alert",
				zap.String("device_id", alert.DeviceID),
				zap.String("alert_type", alert.AlertType),
				zap.Int64("alert_id", existingID))

			// 更新告警ID，用于上报
			alert.ID = existingID

			// 优先通过MQTT实时推送告警（异步执行，不阻塞告警更新）
			s.mu.RLock()
			alertPublisher := s.alertPublisher
			cloudSync := s.cloudSync
			s.mu.RUnlock()

			go func() {
				mqttSuccess := false

				// 尝试MQTT发布
				if alertPublisher != nil && alertPublisher.IsEnabled() {
					if err := alertPublisher.PublishAlert(alert); err == nil {
						mqttSuccess = true
						// MQTT发布成功，标记为已同步
						s.db.Exec("UPDATE alerts SET synced_at = ? WHERE id = ?", time.Now(), existingID)
						s.logger.Debug("Alert published via MQTT and marked as synced")
					} else {
						s.logger.Debug("MQTT alert publish failed, will use HTTP fallback", zap.Error(err))
					}
				}

				// MQTT失败时，使用HTTP作为兜底
				if !mqttSuccess && cloudSync != nil {
					if err := cloudSync.ReportAlertImmediately(alert); err != nil {
						s.logger.Debug("HTTP alert report also failed (will retry in batch sync)", zap.Error(err))
					}
				}
			}()
		}
		return err
	}

	// 不存在未解决告警，创建新告警
	insertQuery := `
		INSERT INTO alerts (device_id, alert_type, severity, message, value, threshold, timestamp, resolved)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(insertQuery,
		alert.DeviceID, alert.AlertType, alert.Severity, alert.Message,
		alert.Value, alert.Threshold, alert.Timestamp, alert.Resolved,
	)
	if err == nil {
		// 获取新创建的告警ID
		newAlertID, _ := result.LastInsertId()
		alert.ID = newAlertID

		s.logger.Info("Created new alert",
			zap.String("device_id", alert.DeviceID),
			zap.String("alert_type", alert.AlertType),
			zap.String("severity", alert.Severity),
			zap.Int64("alert_id", newAlertID))

		// 优先通过MQTT实时推送告警（异步执行，不阻塞告警创建）
		s.mu.RLock()
		alertPublisher := s.alertPublisher
		cloudSync := s.cloudSync
		s.mu.RUnlock()

		go func() {
			mqttSuccess := false

			// 尝试MQTT发布
			if alertPublisher != nil && alertPublisher.IsEnabled() {
				if err := alertPublisher.PublishAlert(alert); err == nil {
					mqttSuccess = true
					// MQTT发布成功，标记为已同步
					s.db.Exec("UPDATE alerts SET synced_at = ? WHERE id = ?", time.Now(), newAlertID)
					s.logger.Debug("Alert published via MQTT and marked as synced",
						zap.Int64("alert_id", newAlertID))
				} else {
					s.logger.Debug("MQTT alert publish failed, will use HTTP fallback", zap.Error(err))
				}
			}

			// MQTT失败时，使用HTTP作为兜底
			if !mqttSuccess && cloudSync != nil {
				if err := cloudSync.ReportAlertImmediately(alert); err != nil {
					s.logger.Debug("HTTP alert report also failed (will retry in batch sync)", zap.Error(err))
				}
			}
		}()
	}
	return err
}

// GetRecentData 获取最近的数据
func (s *Service) GetRecentData(deviceID string, limit int) ([]*models.SensorData, error) {
	query := `
		SELECT id, device_id, sensor_type, value, unit, timestamp, quality, synced
		FROM sensor_data
		WHERE device_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*models.SensorData
	for rows.Next() {
		d := &models.SensorData{}
		err := rows.Scan(
			&d.ID, &d.DeviceID, &d.SensorType, &d.Value,
			&d.Unit, &d.Timestamp, &d.Quality, &d.Synced,
		)
		if err != nil {
			continue
		}
		data = append(data, d)
	}

	return data, nil
}

// GetStatistics 获取数据统计
func (s *Service) GetStatistics(deviceID string, sensorType models.SensorType, startTime, endTime time.Time) (*models.DataStatistics, error) {
	// 构建动态查询条件
	query := `
		SELECT
			COUNT(*) as count,
			COALESCE(MIN(value), 0) as min_value,
			COALESCE(MAX(value), 0) as max_value,
			COALESCE(AVG(value), 0) as avg_value
		FROM sensor_data
		WHERE timestamp BETWEEN ? AND ?
	`

	args := []interface{}{startTime, endTime}

	// 可选条件：设备ID
	if deviceID != "" {
		query += " AND device_id = ?"
		args = append(args, deviceID)
	}

	// 可选条件：传感器类型
	if sensorType != "" {
		query += " AND sensor_type = ?"
		args = append(args, sensorType)
	}

	stats := &models.DataStatistics{
		DeviceID:   deviceID,
		SensorType: sensorType,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	err := s.db.QueryRow(query, args...).Scan(
		&stats.Count, &stats.MinValue, &stats.MaxValue, &stats.AvgValue,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// QueryData 查询历史数据
func (s *Service) QueryData(deviceID, sensorType, startTime, endTime string, page, limit int) ([]*models.SensorData, int64, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}

	if deviceID != "" {
		conditions = append(conditions, "device_id = ?")
		args = append(args, deviceID)
	}

	if sensorType != "" {
		conditions = append(conditions, "sensor_type = ?")
		args = append(args, sensorType)
	}

	if startTime != "" {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, startTime)
	}

	if endTime != "" {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, endTime)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sensor_data %s", whereClause)
	var total int64
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count data: %w", err)
	}

	// 查询数据
	offset := (page - 1) * limit
	dataQuery := fmt.Sprintf(`
		SELECT id, device_id, sensor_type, value, unit, timestamp, quality, synced 
		FROM sensor_data %s 
		ORDER BY timestamp DESC 
		LIMIT ? OFFSET ?`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	var data []*models.SensorData
	for rows.Next() {
		var d models.SensorData
		err := rows.Scan(&d.ID, &d.DeviceID, &d.SensorType, &d.Value, &d.Unit, &d.Timestamp, &d.Quality, &d.Synced)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan data: %w", err)
		}
		data = append(data, &d)
	}

	return data, total, nil
}

// GetStatisticsByPeriod 根据时间段获取统计数据
func (s *Service) GetStatisticsByPeriod(deviceID, sensorType, period string) (*models.DataStatistics, error) {
	// 解析时间段
	var startTime time.Time
	switch period {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	return s.GetStatistics(deviceID, models.SensorType(sensorType), startTime, time.Now())
}

// ListAlerts 获取告警列表
func (s *Service) ListAlerts(page, limit int, severity, resolved string) ([]*models.Alert, int64, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}

	if severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}

	if resolved != "" {
		resolvedBool, _ := strconv.ParseBool(resolved)
		conditions = append(conditions, "resolved = ?")
		args = append(args, resolvedBool)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alerts %s", whereClause)
	var total int64
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	// 查询告警
	offset := (page - 1) * limit
	alertQuery := fmt.Sprintf(`
		SELECT id, device_id, alert_type, severity, message, value, threshold, timestamp, resolved, resolved_at 
		FROM alerts %s 
		ORDER BY timestamp DESC 
		LIMIT ? OFFSET ?`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.db.Query(alertQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(&alert.ID, &alert.DeviceID, &alert.AlertType, &alert.Severity,
			&alert.Message, &alert.Value, &alert.Threshold, &alert.Timestamp,
			&alert.Resolved, &alert.ResolvedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, &alert)
	}

	return alerts, total, nil
}

// CreateAlert 创建告警
func (s *Service) CreateAlert(alert *models.Alert) error {
	query := `
		INSERT INTO alerts (device_id, alert_type, severity, message, value, threshold, timestamp, resolved)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, alert.DeviceID, alert.AlertType, alert.Severity,
		alert.Message, alert.Value, alert.Threshold, alert.Timestamp, alert.Resolved)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	s.logger.Info("Alert created",
		zap.String("device_id", alert.DeviceID),
		zap.String("type", alert.AlertType),
		zap.String("severity", alert.Severity))

	return nil
}

// ResolveAlert 解决告警（可被Cloud命令下发或本地API调用）
func (s *Service) ResolveAlert(alertID int64) error {
	now := time.Now()
	// 重置synced_at为NULL,触发重新同步到Cloud端
	query := `UPDATE alerts SET resolved = ?, resolved_at = ?, synced_at = NULL WHERE id = ?`

	result, err := s.db.Exec(query, true, now, alertID)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found: %d", alertID)
	}

	s.logger.Info("✅ 告警已解决", zap.Int64("alert_id", alertID))
	return nil
}

// SaveSensorData 保存单个传感器数据（MQTT 调用 - 立即写入优化版本）
func (s *Service) SaveSensorData(mqttData interface{}) error {
	// 定义MQTT数据结构（匹配mqtt.SensorData）
	type mqttSensorData struct {
		DeviceID   string    `json:"device_id"`
		SensorType string    `json:"sensor_type"`
		Value      float64   `json:"value"`
		Unit       string    `json:"unit"`
		Quality    int       `json:"quality"`
		Timestamp  time.Time `json:"timestamp"`
	}

	// 先转换为通用结构
	jsonBytes, err := json.Marshal(mqttData)
	if err != nil {
		return fmt.Errorf("failed to marshal mqtt data: %w", err)
	}

	var mqttD mqttSensorData
	if err := json.Unmarshal(jsonBytes, &mqttD); err != nil {
		return fmt.Errorf("failed to unmarshal mqtt data: %w", err)
	}

	// 转换为 models.SensorData
	data := &models.SensorData{
		DeviceID:   mqttD.DeviceID,
		SensorType: models.SensorType(mqttD.SensorType),
		Value:      mqttD.Value,
		Unit:       mqttD.Unit,
		Timestamp:  mqttD.Timestamp,
		Quality:    mqttD.Quality,
		Synced:     false,
	}

	// 检查阈值
	if err := s.checkThreshold(data); err != nil {
		s.logger.Warn("MQTT data threshold exceeded",
			zap.String("device_id", data.DeviceID),
			zap.Error(err))
	}

	// MQTT数据立即写入数据库（不走批量通道）
	// 优势：0延迟，前端可立即查询到最新数据
	if err := s.saveSensorDataImmediate(data); err != nil {
		s.logger.Error("Failed to save MQTT sensor data immediately",
			zap.String("device_id", data.DeviceID),
			zap.Error(err))
		return fmt.Errorf("failed to save immediately: %w", err)
	}

	// 更新设备最后活跃时间（标记为在线）
	// 只要设备发送传感器数据，就认为设备在线
	if err := s.deviceManager.UpdateLastSeen(data.DeviceID); err != nil {
		s.logger.Warn("Failed to update device last seen time",
			zap.String("device_id", data.DeviceID),
			zap.Error(err))
		// 不返回错误，因为数据已经保存成功
	}

	s.logger.Info("✅ 传感器数据已保存",
		zap.String("device_id", data.DeviceID),
		zap.String("sensor_type", string(data.SensorType)),
		zap.Float64("value", data.Value),
		zap.String("unit", data.Unit))

	return nil
}

// saveSensorDataImmediate 立即写入单条传感器数据到数据库
func (s *Service) saveSensorDataImmediate(data *models.SensorData) error {
	// 直接写入数据库，不使用批量通道
	query := `
		INSERT INTO sensor_data (device_id, sensor_type, value, unit, timestamp, quality, synced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		data.DeviceID,
		data.SensorType,
		data.Value,
		data.Unit,
		data.Timestamp,
		data.Quality,
		data.Synced,
	)

	if err != nil {
		return fmt.Errorf("database insert failed: %w", err)
	}

	return nil
}

// SaveAlert 保存告警信息（MQTT 调用）
func (s *Service) SaveAlert(mqttAlert interface{}) error {
	// 类型转换
	type mqttAlertData struct {
		DeviceID  string
		AlertType string
		Severity  string
		Message   string
		Value     float64
		Threshold float64
		Timestamp time.Time
	}

	jsonBytes, err := json.Marshal(mqttAlert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	var mqttA mqttAlertData
	if err := json.Unmarshal(jsonBytes, &mqttA); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	// 转换为 models.Alert
	valuePtr := mqttA.Value
	thresholdPtr := mqttA.Threshold
	alert := &models.Alert{
		DeviceID:  mqttA.DeviceID,
		AlertType: mqttA.AlertType,
		Severity:  mqttA.Severity,
		Message:   mqttA.Message,
		Value:     &valuePtr,
		Threshold: &thresholdPtr,
		Timestamp: mqttA.Timestamp,
		Resolved:  false,
	}

	// 发送到告警通道
	select {
	case s.alertChan <- alert:
		s.logger.Info("MQTT alert queued",
			zap.String("device_id", alert.DeviceID),
			zap.String("alert_type", alert.AlertType),
			zap.String("severity", alert.Severity))
		return nil
	default:
		return fmt.Errorf("alert channel full")
	}
}

// cleanupOldData 定时清理旧数据（防止磁盘撑爆）
func (s *Service) cleanupOldData() {
	defer s.wg.Done()

	// 每天凌晨2点清理一次（24小时间隔）
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 启动时立即执行一次清理
	s.performCleanup()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("数据清理协程已停止")
			return
		case <-ticker.C:
			s.performCleanup()
		}
	}
}

// performCleanup 执行清理操作
func (s *Service) performCleanup() {
	// 计算截止时间（保留最近N天的数据）
	cutoffTime := time.Now().AddDate(0, 0, -s.retentionDays)

	s.logger.Info("🧹 开始清理旧传感器数据",
		zap.Time("cutoff_time", cutoffTime),
		zap.Int("retention_days", s.retentionDays))

	// 删除超过保留期的数据
	query := `DELETE FROM sensor_data WHERE timestamp < ?`
	result, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		s.logger.Error("❌ 清理旧数据失败", zap.Error(err))
		return
	}

	rowsAffected, _ := result.RowsAffected()

	if rowsAffected > 0 {
		s.logger.Info("✅ 旧传感器数据清理完成",
			zap.Int64("deleted_rows", rowsAffected),
			zap.String("freed_space_estimate", fmt.Sprintf("~%.2f MB", float64(rowsAffected)*100/1024/1024)))

		// 执行VACUUM整理数据库，回收磁盘空间
		s.logger.Info("🔧 开始执行数据库VACUUM（回收磁盘空间）...")
		if _, err := s.db.Exec("VACUUM"); err != nil {
			s.logger.Error("❌ 数据库VACUUM失败", zap.Error(err))
		} else {
			s.logger.Info("✅ 数据库VACUUM完成，磁盘空间已回收")
		}
	} else {
		s.logger.Info("ℹ️  无需清理，没有超过保留期的数据")
	}

	// 同时清理旧的告警记录
	alertQuery := `DELETE FROM alerts WHERE timestamp < ? AND resolved = 1`
	alertResult, err := s.db.Exec(alertQuery, cutoffTime)
	if err != nil {
		s.logger.Error("❌ 清理旧告警失败", zap.Error(err))
	} else {
		alertRows, _ := alertResult.RowsAffected()
		if alertRows > 0 {
			s.logger.Info("✅ 旧告警记录清理完成",
				zap.Int64("deleted_alerts", alertRows))
		}
	}
}
