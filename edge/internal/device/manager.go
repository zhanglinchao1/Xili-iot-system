/*
 * 设备管理器
 * 负责设备的注册、状态管理和心跳监测
 */
package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// LicenseService 许可证服务接口
type LicenseService interface {
	Check() error
	IsEnabled() bool
	GetMaxDevices() int
}

// Manager 设备管理器
type Manager struct {
	logger           *zap.Logger
	db               *storage.SQLiteDB
	license          LicenseService // 许可证服务（可选）
	devices          map[string]*models.Device
	sessions         map[string]*DeviceSession
	mu               sync.RWMutex
	heartbeatTTL     time.Duration
	offlineTTL       time.Duration
	maxDevices       int
	supportedSensors []string
	cabinetID        string // 储能柜ID（从配置获取）
	stopChan         chan struct{}
	running          bool
}

// DeviceSession 设备会话信息
type DeviceSession struct {
	Device        *models.Device
	LastHeartbeat time.Time
	IsOnline      bool
	FailCount     int
}

// NewManager 创建设备管理器
func NewManager(cfg config.DeviceConfig, db *storage.SQLiteDB, license LicenseService, logger *zap.Logger, cabinetID string) *Manager {
	if cabinetID == "" {
		cabinetID = "CABINET-001" // 默认值
	}
	return &Manager{
		logger:           logger,
		db:               db,
		license:          license,
		devices:          make(map[string]*models.Device),
		sessions:         make(map[string]*DeviceSession),
		heartbeatTTL:     cfg.HeartbeatInterval,
		offlineTTL:       cfg.OfflineTimeout,
		maxDevices:       cfg.MaxDevices,
		supportedSensors: cfg.SupportedSensors,
		cabinetID:        cabinetID,
		stopChan:         make(chan struct{}),
		running:          false,
	}
}

// Start 启动设备管理器
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("manager already running")
	}
	m.running = true
	m.mu.Unlock()

	// 加载设备列表
	if err := m.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	// 启动心跳检查
	go m.heartbeatChecker(ctx)

	m.logger.Info("Device manager started")
	return nil
}

// Stop 停止设备管理器
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopChan)
	m.logger.Info("Device manager stopped")
}

// RegisterDevice 注册设备
func (m *Manager) RegisterDevice(req *models.DeviceRegistration) (*models.Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 验证输入参数
	if err := m.validateDeviceRegistration(req); err != nil {
		return nil, fmt.Errorf("device registration validation failed: %w", err)
	}

	// 2. 检查设备是否已存在
	if _, exists := m.devices[req.DeviceID]; exists {
		return nil, fmt.Errorf("device already registered: %s", req.DeviceID)
	}

	// 3. 检查设备数量限制（优先检查许可证限制）
	// max_devices = -1 表示无限制
	// max_devices = 0 表示不检查（向后兼容）
	// max_devices > 0 表示具体限制数量
	if m.license != nil && m.license.IsEnabled() {
		licenseMaxDevices := m.license.GetMaxDevices()
		// 只有当 max_devices > 0 时才检查限制
		// -1 和 0 都表示无限制
		if licenseMaxDevices > 0 && len(m.devices) >= licenseMaxDevices {
			return nil, fmt.Errorf("许可证设备数量限制: %d/%d (许可证允许的最大设备数)", len(m.devices), licenseMaxDevices)
		}
	}
	// 检查配置文件中的设备数量限制
	if len(m.devices) >= m.maxDevices {
		return nil, fmt.Errorf("device limit reached: %d/%d", len(m.devices), m.maxDevices)
	}

	// 创建设备记录
	device := &models.Device{
		DeviceID:     req.DeviceID,
		DeviceType:   req.DeviceType,
		SensorType:   req.SensorType,
		CabinetID:    m.getCabinetID(req.CabinetID), // 如果请求中没有，使用默认值
		PublicKey:    req.PublicKey,
		Commitment:   req.Commitment,
		Status:       models.DeviceStatusOffline,
		Model:        req.Model,
		Manufacturer: req.Manufacturer,
		FirmwareVer:  req.FirmwareVer,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 保存到数据库
	if err := m.saveDevice(device); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// 添加到内存缓存
	m.devices[device.DeviceID] = device
	m.sessions[device.DeviceID] = &DeviceSession{
		Device:        device,
		LastHeartbeat: time.Now(),
		IsOnline:      false,
		FailCount:     0,
	}

	m.logger.Info("Device registered",
		zap.String("device_id", device.DeviceID),
		zap.String("sensor_type", string(device.SensorType)))

	return device, nil
}

// UnregisterDevice 注销设备
func (m *Manager) UnregisterDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// 从数据库删除
	if err := m.deleteDevice(deviceID); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	// 从内存删除
	delete(m.devices, deviceID)
	delete(m.sessions, deviceID)

	m.logger.Info("Device unregistered",
		zap.String("device_id", deviceID),
		zap.String("sensor_type", string(device.SensorType)))

	return nil
}

// GetDevice 获取设备信息
func (m *Manager) GetDevice(deviceID string) (*models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return device, nil
}

// GetAllDevices 获取所有设备
func (m *Manager) GetAllDevices() ([]*models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]*models.Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDevicesByCabinet 获取储能柜的所有设备
func (m *Manager) GetDevicesByCabinet(cabinetID string) ([]*models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]*models.Device, 0)
	for _, device := range m.devices {
		if device.CabinetID == cabinetID {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// UpdateDeviceStatus 更新设备状态
func (m *Manager) UpdateDeviceStatus(deviceID string, status models.DeviceStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	device.Status = status
	device.UpdatedAt = time.Now()

	// 更新数据库
	if err := m.updateDevice(device); err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	m.logger.Debug("Device status updated",
		zap.String("device_id", deviceID),
		zap.String("status", string(status)))

	return nil
}

// ProcessHeartbeat 处理心跳
func (m *Manager) ProcessHeartbeat(hb *models.Heartbeat) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[hb.DeviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", hb.DeviceID)
	}

	// 更新心跳时间
	session.LastHeartbeat = time.Now()

	// 每次心跳都更新LastSeenAt字段
	session.Device.LastSeenAt = &session.LastHeartbeat
	session.Device.UpdatedAt = time.Now()

	// 如果设备之前离线，更新为在线
	if !session.IsOnline {
		session.IsOnline = true
		session.FailCount = 0
		session.Device.Status = models.DeviceStatusOnline

		m.logger.Info("Device online",
			zap.String("device_id", hb.DeviceID))
	}

	// 更新数据库中的设备信息
	if err := m.updateDevice(session.Device); err != nil {
		m.logger.Error("Failed to update device heartbeat",
			zap.String("device_id", hb.DeviceID),
			zap.Error(err))
		return err
	}

	return nil
}

// heartbeatChecker 心跳检查器
func (m *Manager) heartbeatChecker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkHeartbeats()
		}
	}
}

// checkHeartbeats 检查所有设备心跳
func (m *Manager) checkHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for deviceID, session := range m.sessions {
		timeSinceLastHeartbeat := now.Sub(session.LastHeartbeat)

		// 检查是否超时
		if timeSinceLastHeartbeat > m.offlineTTL {
			if session.IsOnline {
				session.IsOnline = false
				session.Device.Status = models.DeviceStatusOffline

				if err := m.updateDevice(session.Device); err != nil {
					m.logger.Error("Failed to update device status",
						zap.String("device_id", deviceID),
						zap.Error(err))
				}

				m.logger.Warn("Device offline",
					zap.String("device_id", deviceID),
					zap.Duration("timeout", timeSinceLastHeartbeat))

				// 触发离线告警
				m.triggerOfflineAlert(deviceID)
			}
		} else if timeSinceLastHeartbeat > m.heartbeatTTL*2 {
			// 心跳延迟警告
			session.FailCount++
			if session.FailCount >= 3 {
				m.logger.Warn("Device heartbeat delayed",
					zap.String("device_id", deviceID),
					zap.Duration("delay", timeSinceLastHeartbeat))
			}
		}
	}
}

// triggerOfflineAlert 触发离线告警
func (m *Manager) triggerOfflineAlert(deviceID string) {
	// TODO: 实现告警逻辑
	m.logger.Error("Device offline alert triggered",
		zap.String("device_id", deviceID))
}

// loadDevices 从数据库加载设备
func (m *Manager) loadDevices() error {
	query := `
		SELECT device_id, device_type, sensor_type, 
			   public_key, commitment, status, model, manufacturer, 
			   firmware_ver, created_at, updated_at, last_seen_at
		FROM devices
	`

	rows, err := m.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var device models.Device
		err := rows.Scan(
			&device.DeviceID, &device.DeviceType, &device.SensorType,
			&device.PublicKey, &device.Commitment,
			&device.Status, &device.Model, &device.Manufacturer,
			&device.FirmwareVer, &device.CreatedAt, &device.UpdatedAt,
			&device.LastSeenAt,
		)
		if err != nil {
			m.logger.Error("Failed to scan device", zap.Error(err))
			continue
		}

		// 设置默认cabinet_id（数据库中没有此字段）
		device.CabinetID = m.cabinetID

		// 使用最后活跃时间或当前时间作为心跳时间
		lastHeartbeat := time.Now()
		if device.LastSeenAt != nil {
			lastHeartbeat = *device.LastSeenAt
		}

		m.devices[device.DeviceID] = &device
		m.sessions[device.DeviceID] = &DeviceSession{
			Device:        &device,
			LastHeartbeat: lastHeartbeat,
			IsOnline:      device.Status == models.DeviceStatusOnline,
			FailCount:     0,
		}
	}

	m.logger.Info("Devices loaded", zap.Int("count", len(m.devices)))
	return nil
}

// saveDevice 保存设备到数据库
func (m *Manager) saveDevice(device *models.Device) error {
	query := `
		INSERT INTO devices (
			device_id, device_type, sensor_type,
			public_key, commitment, status, model, manufacturer,
			firmware_ver, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := m.db.Exec(query,
		device.DeviceID, device.DeviceType, device.SensorType,
		device.PublicKey, device.Commitment,
		device.Status, device.Model, device.Manufacturer,
		device.FirmwareVer, device.CreatedAt, device.UpdatedAt,
	)

	return err
}

// updateDevice 更新设备信息
func (m *Manager) updateDevice(device *models.Device) error {
	query := `
		UPDATE devices SET
			status = ?, model = ?, manufacturer = ?,
			firmware_ver = ?, updated_at = ?, last_seen_at = ?
		WHERE device_id = ?
	`

	_, err := m.db.Exec(query,
		device.Status, device.Model, device.Manufacturer,
		device.FirmwareVer, device.UpdatedAt, device.LastSeenAt,
		device.DeviceID,
	)

	return err
}

// deleteDevice 删除设备
func (m *Manager) deleteDevice(deviceID string) error {
	query := `DELETE FROM devices WHERE device_id = ?`
	_, err := m.db.Exec(query, deviceID)
	return err
}

// GetStatistics 获取设备统计信息
func (m *Manager) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// 统计各状态设备数量
	statusCount := make(map[models.DeviceStatus]int)
	sensorTypeCount := make(map[models.SensorType]int)

	for _, device := range m.devices {
		statusCount[device.Status]++
		sensorTypeCount[device.SensorType]++
	}

	stats["total"] = len(m.devices)
	stats["status"] = statusCount
	stats["sensor_types"] = sensorTypeCount
	stats["online"] = statusCount[models.DeviceStatusOnline]
	stats["offline"] = statusCount[models.DeviceStatusOffline]

	return stats
}

// ListDevices 获取设备列表（分页）
func (m *Manager) ListDevices(page, limit int, status, sensorType string) ([]*models.Device, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 过滤设备
	var filteredDevices []*models.Device
	for _, device := range m.devices {
		// 状态过滤
		if status != "" && string(device.Status) != status {
			continue
		}
		// 传感器类型过滤
		if sensorType != "" && string(device.SensorType) != sensorType {
			continue
		}
		filteredDevices = append(filteredDevices, device)
	}

	total := int64(len(filteredDevices))

	// 分页处理
	start := (page - 1) * limit
	end := start + limit

	if start >= len(filteredDevices) {
		return []*models.Device{}, total, nil
	}

	if end > len(filteredDevices) {
		end = len(filteredDevices)
	}

	return filteredDevices[start:end], total, nil
}

// UpdateDevice 更新设备信息
func (m *Manager) UpdateDevice(deviceID string, updates map[string]interface{}) (*models.Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// 创建设备副本进行更新
	updatedDevice := *device
	updatedDevice.UpdatedAt = time.Now()

	// 应用更新
	for key, value := range updates {
		switch key {
		case "status":
			if statusStr, ok := value.(string); ok {
				updatedDevice.Status = models.DeviceStatus(statusStr)
			}
		case "model":
			if model, ok := value.(string); ok {
				updatedDevice.Model = model
			}
		case "manufacturer":
			if manufacturer, ok := value.(string); ok {
				updatedDevice.Manufacturer = manufacturer
			}
		case "firmware_ver":
			if firmwareVer, ok := value.(string); ok {
				updatedDevice.FirmwareVer = firmwareVer
			}
		}
	}

	// 保存到数据库
	if err := m.updateDevice(&updatedDevice); err != nil {
		return nil, fmt.Errorf("failed to update device in database: %w", err)
	}

	// 更新内存中的设备信息
	m.devices[deviceID] = &updatedDevice

	m.logger.Info("Device updated",
		zap.String("device_id", deviceID),
		zap.Any("updates", updates))

	return &updatedDevice, nil
}

// IsSensorSupported 检查传感器类型是否支持
func (m *Manager) IsSensorSupported(sensorType string) bool {
	for _, supported := range m.supportedSensors {
		if supported == sensorType {
			return true
		}
	}
	return false
}

// GetDeviceCount 获取设备总数
func (m *Manager) GetDeviceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.devices)
}

// IsDeviceOnline 检查设备是否在线
func (m *Manager) IsDeviceOnline(deviceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.sessions[deviceID]; exists {
		return session.IsOnline
	}
	return false
}

// validateDeviceRegistration 验证设备注册请求
func (m *Manager) validateDeviceRegistration(req *models.DeviceRegistration) error {
	// 验证设备ID
	if req.DeviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	if len(req.DeviceID) > 64 {
		return fmt.Errorf("device ID too long: %d > 64", len(req.DeviceID))
	}

	// 验证设备类型
	if req.DeviceType == "" {
		return fmt.Errorf("device type cannot be empty")
	}
	// 支持的设备类型：sensor, sensor_node
	validDeviceTypes := []string{"sensor", "sensor_node"}
	validType := false
	for _, validDeviceType := range validDeviceTypes {
		if req.DeviceType == validDeviceType {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("unsupported device type: %s", req.DeviceType)
	}

	// 验证传感器类型
	if !m.IsSensorSupported(string(req.SensorType)) {
		return fmt.Errorf("unsupported sensor type: %s", req.SensorType)
	}

	// 验证储能柜ID（可选，如果为空则使用默认值）
	if req.CabinetID != "" && len(req.CabinetID) > 64 {
		return fmt.Errorf("cabinet ID too long: %d > 64", len(req.CabinetID))
	}

	// 验证公钥和承诺值
	if req.PublicKey == "" {
		return fmt.Errorf("public key cannot be empty")
	}
	if req.Commitment == "" {
		return fmt.Errorf("commitment cannot be empty")
	}

	// 验证可选字段长度
	if len(req.Model) > 64 {
		return fmt.Errorf("model name too long: %d > 64", len(req.Model))
	}
	if len(req.Manufacturer) > 64 {
		return fmt.Errorf("manufacturer name too long: %d > 64", len(req.Manufacturer))
	}
	if len(req.FirmwareVer) > 32 {
		return fmt.Errorf("firmware version too long: %d > 32", len(req.FirmwareVer))
	}

	return nil
}

// UpdateDeviceStatusString 更新设备状态（MQTT 调用，接受字符串状态）
func (m *Manager) UpdateDeviceStatusString(deviceID, status string) error {
	// 验证状态值并转换为 models.DeviceStatus
	validStatuses := map[string]models.DeviceStatus{
		"online":  models.DeviceStatusOnline,
		"offline": models.DeviceStatusOffline,
		"error":   models.DeviceStatusFault,
		"fault":   models.DeviceStatusFault,
	}

	deviceStatus, valid := validStatuses[status]
	if !valid {
		return fmt.Errorf("invalid status: %s", status)
	}

	// 调用现有的 UpdateDeviceStatus 方法
	if err := m.UpdateDeviceStatus(deviceID, deviceStatus); err != nil {
		return err
	}

	// 更新会话状态（MQTT 特有逻辑）
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[deviceID]; ok {
		session.IsOnline = (status == "online")
		session.LastHeartbeat = time.Now()
	}

	m.logger.Info("Device status updated via MQTT",
		zap.String("device_id", deviceID),
		zap.String("status", status))

	return nil
}

// UpdateLastSeen 更新设备最后活跃时间（MQTT 调用）
func (m *Manager) UpdateLastSeen(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查设备是否存在
	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// 更新会话心跳时间
	if session, ok := m.sessions[deviceID]; ok {
		session.LastHeartbeat = time.Now()

		// 如果设备之前是离线的，现在标记为在线
		if !session.IsOnline {
			session.IsOnline = true
			device.Status = models.DeviceStatusOnline
			device.UpdatedAt = time.Now()

			// 更新数据库
			if err := m.updateDevice(device); err != nil {
				m.logger.Error("Failed to update device status in database",
					zap.String("device_id", deviceID),
					zap.Error(err))
			}

			m.logger.Info("Device came online via heartbeat",
				zap.String("device_id", deviceID))
		}
	} else {
		// 创建新会话
		m.sessions[deviceID] = &DeviceSession{
			Device:        device,
			LastHeartbeat: time.Now(),
			IsOnline:      true,
			FailCount:     0,
		}

		device.Status = models.DeviceStatusOnline
		device.UpdatedAt = time.Now()

		if err := m.updateDevice(device); err != nil {
			m.logger.Error("Failed to update device status in database",
				zap.String("device_id", deviceID),
				zap.Error(err))
		}

		m.logger.Info("New device session created via heartbeat",
			zap.String("device_id", deviceID))
	}

	return nil
}

// getCabinetID 获取储能柜ID，如果提供的为空则使用默认值
func (m *Manager) getCabinetID(providedID string) string {
	if providedID != "" {
		return providedID
	}
	return m.cabinetID
}
