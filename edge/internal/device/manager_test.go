/*
 * 设备管理器单元测试
 * 测试设备注册、查询、更新、心跳等功能
 */
package device

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// 创建测试用的设备管理器
func createTestManager(t *testing.T) (*Manager, func()) {
	logger, _ := zap.NewDevelopment()
	
	// 创建临时数据库
	dbPath := "/tmp/test_device_" + time.Now().Format("20060102150405") + ".db"
	dbCfg := config.DatabaseConfig{
		Driver:             "sqlite3",
		Path:               dbPath,
		MaxConnections:     5,
		MaxIdleConnections: 2,
	}
	
	db, err := storage.NewSQLiteDB(dbCfg, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	
	// 创建设备配置
	deviceCfg := config.DeviceConfig{
		HeartbeatInterval: 10 * time.Second,
		OfflineTimeout:    30 * time.Second,
		MaxDevices:        10,
		SupportedSensors:  []string{"co2", "co", "smoke", "liquid_level", "conductivity", "temperature", "flow"},
	}
	
	manager := NewManager(deviceCfg, db, nil, logger) // nil license for test
	
	// 清理函数
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}
	
	return manager, cleanup
}

// 测试设备注册
func TestRegisterDevice(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 创建设备注册请求
	req := &models.DeviceRegistration{
		DeviceID:     "TEST_SENSOR_001",
		DeviceType:   "sensor",
		SensorType:   "co2",
		CabinetID:    "CABINET_A1",
		PublicKey:    "04a1b2c3d4e5f6",
		Commitment:   "0x1a2b3c4d5e6f",
		Model:        "CO2-SEN-V2",
		Manufacturer: "TestSensorTech",
		FirmwareVer:  "1.0.0",
	}
	
	// 注册设备
	device, err := manager.RegisterDevice(req)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}
	
	// 验证返回的设备信息
	if device.DeviceID != req.DeviceID {
		t.Errorf("Expected device ID %s, got %s", req.DeviceID, device.DeviceID)
	}
	if device.SensorType != req.SensorType {
		t.Errorf("Expected sensor type %s, got %s", req.SensorType, device.SensorType)
	}
	if device.Status != models.DeviceStatusOffline {
		t.Errorf("Expected status offline, got %s", device.Status)
	}
	
	t.Logf("Device registered successfully: %s", device.DeviceID)
}

// 测试设备注册验证
func TestRegisterDeviceValidation(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	tests := []struct {
		name    string
		req     *models.DeviceRegistration
		wantErr bool
		errMsg  string
	}{
		{
			name: "空设备ID",
			req: &models.DeviceRegistration{
				DeviceID:   "",
				DeviceType: "sensor",
				SensorType: "co2",
				CabinetID:  "CABINET_A1",
				PublicKey:  "test",
				Commitment: "test",
			},
			wantErr: true,
			errMsg:  "device ID cannot be empty",
		},
		{
			name: "设备ID过长",
			req: &models.DeviceRegistration{
				DeviceID:   "VERY_LONG_DEVICE_ID_THAT_EXCEEDS_THE_MAXIMUM_LENGTH_LIMIT_OF_64_CHARACTERS",
				DeviceType: "sensor",
				SensorType: "co2",
				CabinetID:  "CABINET_A1",
				PublicKey:  "test",
				Commitment: "test",
			},
			wantErr: true,
			errMsg:  "device ID too long",
		},
		{
			name: "不支持的传感器类型",
			req: &models.DeviceRegistration{
				DeviceID:   "TEST_001",
				DeviceType: "sensor",
				SensorType: "invalid_sensor",
				CabinetID:  "CABINET_A1",
				PublicKey:  "test",
				Commitment: "test",
			},
			wantErr: true,
			errMsg:  "unsupported sensor type",
		},
		{
			name: "空公钥",
			req: &models.DeviceRegistration{
				DeviceID:   "TEST_001",
				DeviceType: "sensor",
				SensorType: "co2",
				CabinetID:  "CABINET_A1",
				PublicKey:  "",
				Commitment: "test",
			},
			wantErr: true,
			errMsg:  "public key cannot be empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.RegisterDevice(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// 测试获取设备
func TestGetDevice(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 先注册一个设备
	req := &models.DeviceRegistration{
		DeviceID:   "TEST_002",
		DeviceType: "sensor",
		SensorType: "temperature",
		CabinetID:  "CABINET_A1",
		PublicKey:  "test_key",
		Commitment: "test_commitment",
	}
	
	_, err := manager.RegisterDevice(req)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}
	
	// 获取设备
	device, err := manager.GetDevice("TEST_002")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}
	
	if device.DeviceID != "TEST_002" {
		t.Errorf("Expected device ID TEST_002, got %s", device.DeviceID)
	}
	
	// 获取不存在的设备
	_, err = manager.GetDevice("NON_EXISTENT")
	if err == nil {
		t.Error("Expected error for non-existent device, got nil")
	}
}

// 测试设备列表查询
func TestListDevices(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 注册多个设备
	sensors := []models.SensorType{"co2", "co", "temperature"}
	for i, sensor := range sensors {
		req := &models.DeviceRegistration{
			DeviceID:   fmt.Sprintf("TEST_%03d", i+1),
			DeviceType: "sensor",
			SensorType: sensor,
			CabinetID:  "CABINET_A1",
			PublicKey:  "test_key",
			Commitment: "test_commitment",
		}
		_, err := manager.RegisterDevice(req)
		if err != nil {
			t.Fatalf("Failed to register device: %v", err)
		}
	}
	
	// 查询所有设备
	devices, total, err := manager.ListDevices(1, 10, "", "")
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}
	
	if total != 3 {
		t.Errorf("Expected 3 devices, got %d", total)
	}
	if len(devices) != 3 {
		t.Errorf("Expected 3 devices in result, got %d", len(devices))
	}
	
	// 按传感器类型过滤
	devices, total, err = manager.ListDevices(1, 10, "", "co2")
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}
	
	if total != 1 {
		t.Errorf("Expected 1 CO2 device, got %d", total)
	}
}

// 测试设备更新
func TestUpdateDevice(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 注册设备
	req := &models.DeviceRegistration{
		DeviceID:   "TEST_UPDATE",
		DeviceType: "sensor",
		SensorType: "co2",
		CabinetID:  "CABINET_A1",
		PublicKey:  "test_key",
		Commitment: "test_commitment",
		Model:      "OLD_MODEL",
	}
	
	_, err := manager.RegisterDevice(req)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}
	
	// 更新设备
	updates := map[string]interface{}{
		"model":        "NEW_MODEL",
		"firmware_ver": "2.0.0",
	}
	
	device, err := manager.UpdateDevice("TEST_UPDATE", updates)
	if err != nil {
		t.Fatalf("Failed to update device: %v", err)
	}
	
	if device.Model != "NEW_MODEL" {
		t.Errorf("Expected model NEW_MODEL, got %s", device.Model)
	}
	if device.FirmwareVer != "2.0.0" {
		t.Errorf("Expected firmware version 2.0.0, got %s", device.FirmwareVer)
	}
}

// 测试设备心跳
func TestProcessHeartbeat(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 启动管理器
	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()
	
	// 注册设备
	req := &models.DeviceRegistration{
		DeviceID:   "TEST_HEARTBEAT",
		DeviceType: "sensor",
		SensorType: "co2",
		CabinetID:  "CABINET_A1",
		PublicKey:  "test_key",
		Commitment: "test_commitment",
	}
	
	_, err = manager.RegisterDevice(req)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}
	
	// 发送心跳
	hb := &models.Heartbeat{
		DeviceID:  "TEST_HEARTBEAT",
		Timestamp: time.Now(),
		Status:    "online",
	}
	
	err = manager.ProcessHeartbeat(hb)
	if err != nil {
		t.Fatalf("Failed to process heartbeat: %v", err)
	}
	
	// 检查设备是否在线
	if !manager.IsDeviceOnline("TEST_HEARTBEAT") {
		t.Error("Expected device to be online after heartbeat")
	}
	
	// 获取设备并检查状态
	device, err := manager.GetDevice("TEST_HEARTBEAT")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}
	
	if device.Status != models.DeviceStatusOnline {
		t.Errorf("Expected device status online, got %s", device.Status)
	}
}

// 测试设备注销
func TestUnregisterDevice(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 注册设备
	req := &models.DeviceRegistration{
		DeviceID:   "TEST_DELETE",
		DeviceType: "sensor",
		SensorType: "co2",
		CabinetID:  "CABINET_A1",
		PublicKey:  "test_key",
		Commitment: "test_commitment",
	}
	
	_, err := manager.RegisterDevice(req)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}
	
	// 注销设备
	err = manager.UnregisterDevice("TEST_DELETE")
	if err != nil {
		t.Fatalf("Failed to unregister device: %v", err)
	}
	
	// 验证设备已删除
	_, err = manager.GetDevice("TEST_DELETE")
	if err == nil {
		t.Error("Expected error for deleted device, got nil")
	}
}

// 测试设备数量限制
func TestMaxDevicesLimit(t *testing.T) {
	manager, cleanup := createTestManager(t)
	defer cleanup()
	
	// 注册到达上限的设备
	for i := 0; i < manager.maxDevices; i++ {
		req := &models.DeviceRegistration{
			DeviceID:   fmt.Sprintf("DEVICE_%03d", i),
			DeviceType: "sensor",
			SensorType: "co2",
			CabinetID:  "CABINET_A1",
			PublicKey:  "test_key",
			Commitment: "test_commitment",
		}
		_, err := manager.RegisterDevice(req)
		if err != nil {
			t.Fatalf("Failed to register device %d: %v", i, err)
		}
	}
	
	// 尝试注册超过上限的设备
	req := &models.DeviceRegistration{
		DeviceID:   "DEVICE_OVERFLOW",
		DeviceType: "sensor",
		SensorType: "co2",
		CabinetID:  "CABINET_A1",
		PublicKey:  "test_key",
		Commitment: "test_commitment",
	}
	
	_, err := manager.RegisterDevice(req)
	if err == nil {
		t.Error("Expected error for exceeding max devices, got nil")
	}
	if !contains(err.Error(), "device limit reached") {
		t.Errorf("Expected 'device limit reached' error, got: %v", err)
	}
}

// 辅助函数：检查字符串包含
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
