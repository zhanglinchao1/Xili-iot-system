/*
 * MQTT订阅服务
 * 订阅Edge端传感器数据，实时接收并存储到数据库
 */
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// MQTTSubscriberService MQTT订阅服务
type MQTTSubscriberService struct {
	mqttClient       mqtt.Client
	sensorService    SensorService
	sensorDeviceRepo repository.SensorDeviceRepository // 用于获取设备信息
	trafficService   *TrafficService
	alertService     AlertService // 告警服务（用于处理MQTT告警）
	cfg              *config.Config
	ctx              context.Context
	cancel           context.CancelFunc
	wsHub            WebSocketHub   // WebSocket Hub用于实时推送
	abacLogHandler   ABACLogHandler // ABAC日志处理器（可选）
}

// ABACLogHandler ABAC日志处理接口
type ABACLogHandler interface {
	HandleAccessLogs(ctx context.Context, cabinetID string, logs []map[string]interface{}) error
	HandlePolicyAck(ctx context.Context, cabinetID, policyID string) error
}

// WebSocketHub WebSocket Hub接口
type WebSocketHub interface {
	BroadcastSensorData(data interface{})                         // 广播单个传感器数据
	BroadcastLatestSensorData(cabinetID string, data interface{}) // 广播最新传感器数据（兼容旧格式）
}

// MQTTSensorMessage MQTT传感器消息格式（来自Edge端）
type MQTTSensorMessage struct {
	DeviceID   string    `json:"device_id"`
	SensorType string    `json:"sensor_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Quality    int       `json:"quality"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewMQTTSubscriberService 创建MQTT订阅服务实例
func NewMQTTSubscriberService(
	mqttClient mqtt.Client,
	sensorService SensorService,
	sensorDeviceRepo repository.SensorDeviceRepository,
	trafficService *TrafficService,
	cfg *config.Config,
	wsHub WebSocketHub,
) *MQTTSubscriberService {
	ctx, cancel := context.WithCancel(context.Background())

	return &MQTTSubscriberService{
		mqttClient:       mqttClient,
		sensorService:    sensorService,
		sensorDeviceRepo: sensorDeviceRepo,
		trafficService:   trafficService,
		cfg:              cfg,
		ctx:              ctx,
		cancel:           cancel,
		wsHub:            wsHub,
	}
}

// SetABACLogHandler 设置ABAC日志处理器
func (s *MQTTSubscriberService) SetABACLogHandler(handler ABACLogHandler) {
	s.abacLogHandler = handler
}

// SetAlertService 设置告警服务（用于处理MQTT告警）
func (s *MQTTSubscriberService) SetAlertService(alertService AlertService) {
	s.alertService = alertService
}

// Start 启动MQTT订阅服务
func (s *MQTTSubscriberService) Start() error {
	utils.Info("Starting MQTT subscriber service...")

	topics := map[string]mqtt.MessageHandler{
		"sensors/#":                 s.handleSensorMessage,
		"traffic/#":                 s.handleTrafficMessage,
		"edge/cabinet/+/abac/logs":  s.handleABACLogMessage,   // ABAC设备访问日志
		"edge/cabinet/+/policy/ack": s.handlePolicyAckMessage, // 策略分发ACK确认
		"edge/cabinet/+/alerts":     s.handleAlertMessage,     // Edge端实时告警推送
	}

	for topic, handler := range topics {
		token := s.mqttClient.Subscribe(topic, s.cfg.EdgeMQTT.QoS, handler)
		if token.Wait() && token.Error() != nil {
			utils.Error("Failed to subscribe MQTT topic",
				zap.String("topic", topic),
				zap.Error(token.Error()),
			)
			return token.Error()
		}
		utils.Info("MQTT subscriber service subscribed",
			zap.String("topic", topic),
			zap.Int("qos", int(s.cfg.EdgeMQTT.QoS)))
	}

	return nil
}

// Stop 停止MQTT订阅服务
func (s *MQTTSubscriberService) Stop() error {
	utils.Info("Stopping MQTT subscriber service...")

	// 取消上下文
	s.cancel()

	// 取消订阅
	s.mqttClient.Unsubscribe("sensors/#", "traffic/#", "edge/cabinet/+/abac/logs", "edge/cabinet/+/alerts")

	utils.Info("MQTT subscriber service stopped")
	return nil
}

// handleSensorMessage 处理传感器消息
func (s *MQTTSubscriberService) handleSensorMessage(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	utils.Info("Received MQTT sensor message",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)),
	)

	// 解析消息
	var sensorMsg MQTTSensorMessage
	if err := json.Unmarshal(payload, &sensorMsg); err != nil {
		utils.Error("Failed to unmarshal MQTT sensor message",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	// 验证消息数据
	if err := s.validateSensorMessage(&sensorMsg); err != nil {
		utils.Warn("Invalid MQTT sensor message",
			zap.String("topic", topic),
			zap.String("device_id", sensorMsg.DeviceID),
			zap.Error(err),
		)
		return
	}

	// 从主题中提取device_id和sensor_type进行验证
	// 主题格式: sensors/{device_id}/{sensor_type}
	topicParts := strings.Split(topic, "/")
	if len(topicParts) >= 3 {
		topicDeviceID := topicParts[1]
		topicSensorType := topicParts[2]

		// 验证主题中的信息与消息体一致
		if topicSensorType != sensorMsg.SensorType {
			utils.Warn("Sensor type mismatch between topic and message",
				zap.String("topic_sensor_type", topicSensorType),
				zap.String("message_sensor_type", sensorMsg.SensorType),
			)
		}
		if topicDeviceID != sensorMsg.DeviceID {
			utils.Warn("Device ID mismatch between topic and message",
				zap.String("topic_device_id", topicDeviceID),
				zap.String("message_device_id", sensorMsg.DeviceID),
			)
		}
	}

	// 调用传感器服务保存数据
	if err := s.sensorService.SaveSensorDataFromMQTT(s.ctx, &sensorMsg); err != nil {
		utils.Error("Failed to save sensor data from MQTT",
			zap.String("device_id", sensorMsg.DeviceID),
			zap.String("sensor_type", sensorMsg.SensorType),
			zap.Error(err),
		)
		return
	}

	utils.Debug("Sensor data saved successfully",
		zap.String("device_id", sensorMsg.DeviceID),
		zap.String("sensor_type", sensorMsg.SensorType),
		zap.Float64("value", sensorMsg.Value),
		zap.String("unit", sensorMsg.Unit),
	)

	// 通过WebSocket直接广播MQTT消息中的传感器数据（无需查询数据库）
	if s.wsHub != nil {
		// 异步广播，避免阻塞MQTT消息处理
		go func() {
			// 获取设备信息以获取cabinet_id和设备名称
			device, err := s.sensorDeviceRepo.GetByID(s.ctx, sensorMsg.DeviceID)
			if err != nil || device == nil {
				utils.Debug("Failed to get device for WebSocket broadcast",
					zap.String("device_id", sensorMsg.DeviceID),
					zap.Error(err),
				)
				return
			}

			// 构建传感器数据（直接使用MQTT消息中的数据）
			// 注意：timestamp需要转换为字符串格式，确保前端能正确解析
			sensorData := map[string]interface{}{
				"cabinet_id":  device.CabinetID,
				"device_id":   sensorMsg.DeviceID,
				"sensor_type": sensorMsg.SensorType,
				"name":        device.Name,
				"unit":        sensorMsg.Unit,
				"value":       sensorMsg.Value,
				"quality":     sensorMsg.Quality,
				"status": func() string {
					if sensorMsg.Quality < 50 {
						return "error"
					} else if sensorMsg.Quality < 80 {
						return "warning"
					}
					return "normal"
				}(),
				"timestamp": sensorMsg.Timestamp.Format(time.RFC3339), // 转换为RFC3339格式字符串
			}

			// 直接广播单个传感器数据更新
			s.wsHub.BroadcastSensorData(sensorData)
			utils.Info("Broadcasted sensor data via WebSocket",
				zap.String("cabinet_id", device.CabinetID),
				zap.String("device_id", sensorMsg.DeviceID),
				zap.String("sensor_type", sensorMsg.SensorType),
				zap.Float64("value", sensorMsg.Value),
			)
		}()
	}
}

func (s *MQTTSubscriberService) handleTrafficMessage(client mqtt.Client, msg mqtt.Message) {
	if s.trafficService == nil {
		return
	}

	var trafficMsg models.TrafficMQTTMessage
	if err := json.Unmarshal(msg.Payload(), &trafficMsg); err != nil {
		utils.Warn("Failed to unmarshal traffic message",
			zap.String("topic", msg.Topic()),
			zap.Error(err))
		return
	}

	s.trafficService.Update(&trafficMsg)
}

// validateSensorMessage 验证传感器消息
func (s *MQTTSubscriberService) validateSensorMessage(msg *MQTTSensorMessage) error {
	// 验证必填字段
	if msg.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if msg.SensorType == "" {
		return fmt.Errorf("sensor_type is required")
	}
	if msg.Unit == "" {
		return fmt.Errorf("unit is required")
	}

	// 验证传感器类型是否有效
	if !models.IsValidSensorType(msg.SensorType) {
		return fmt.Errorf("invalid sensor_type: %s", msg.SensorType)
	}

	// 验证质量值范围
	if msg.Quality < 0 || msg.Quality > 100 {
		return fmt.Errorf("quality must be between 0 and 100")
	}

	// 验证时间戳
	if msg.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}

// handleABACLogMessage 处理ABAC设备访问日志消息
func (s *MQTTSubscriberService) handleABACLogMessage(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	// 从topic提取cabinet_id: edge/cabinet/{cabinet_id}/abac/logs
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 5 {
		utils.Warn("Invalid ABAC log topic format", zap.String("topic", topic))
		return
	}
	cabinetID := topicParts[2]

	utils.Info("Received ABAC access log message",
		zap.String("topic", topic),
		zap.String("cabinet_id", cabinetID),
		zap.Int("payload_size", len(payload)),
	)

	if s.abacLogHandler == nil {
		utils.Warn("ABAC log handler not configured, ignoring message")
		return
	}

	// 解析日志数组
	var logs []map[string]interface{}
	if err := json.Unmarshal(payload, &logs); err != nil {
		utils.Error("Failed to unmarshal ABAC log message",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	// 调用处理器保存日志
	if err := s.abacLogHandler.HandleAccessLogs(s.ctx, cabinetID, logs); err != nil {
		utils.Error("Failed to handle ABAC access logs",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return
	}

	utils.Info("ABAC access logs processed",
		zap.String("cabinet_id", cabinetID),
		zap.Int("log_count", len(logs)),
	)
}

// handlePolicyAckMessage 处理策略分发ACK消息
func (s *MQTTSubscriberService) handlePolicyAckMessage(client mqtt.Client, msg mqtt.Message) {
	if s.abacLogHandler == nil {
		return // ABAC功能未启用
	}

	// Topic格式: edge/cabinet/{cabinet_id}/policy/ack
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) != 5 {
		utils.Warn("Invalid policy ACK topic format", zap.String("topic", msg.Topic()))
		return
	}
	cabinetID := parts[2]

	// 解析ACK消息 {"policy_id": "xxx", "status": "success"}
	var ackMsg struct {
		PolicyID string `json:"policy_id"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(msg.Payload(), &ackMsg); err != nil {
		utils.Error("Failed to parse policy ACK message",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return
	}

	// 只处理成功的ACK
	if ackMsg.Status != "success" {
		utils.Warn("Received non-success policy ACK",
			zap.String("cabinet_id", cabinetID),
			zap.String("policy_id", ackMsg.PolicyID),
			zap.String("status", ackMsg.Status),
		)
		return
	}

	// 调用处理器更新分发状态
	if err := s.abacLogHandler.HandlePolicyAck(s.ctx, cabinetID, ackMsg.PolicyID); err != nil {
		utils.Error("Failed to handle policy ACK",
			zap.String("cabinet_id", cabinetID),
			zap.String("policy_id", ackMsg.PolicyID),
			zap.Error(err),
		)
		return
	}

	utils.Info("Policy ACK processed",
		zap.String("cabinet_id", cabinetID),
		zap.String("policy_id", ackMsg.PolicyID),
	)
}

// handleAlertMessage 处理Edge端实时推送的告警消息
func (s *MQTTSubscriberService) handleAlertMessage(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	utils.Info("Received MQTT alert message",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)),
	)

	// 解析Topic获取cabinet_id: edge/cabinet/{cabinet_id}/alerts
	topicParts := strings.Split(topic, "/")
	if len(topicParts) != 4 {
		utils.Warn("Invalid alert topic format",
			zap.String("topic", topic),
			zap.Int("parts", len(topicParts)),
		)
		return
	}
	cabinetID := topicParts[2]

	// 检查告警服务是否已设置
	if s.alertService == nil {
		utils.Warn("Alert service not configured, ignoring MQTT alert message")
		return
	}

	// 解析告警消息为AlertSyncData结构体
	var alertData models.AlertSyncData
	if err := json.Unmarshal(payload, &alertData); err != nil {
		utils.Error("Failed to unmarshal alert message",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	// 构建告警同步请求（复用现有的HTTP同步格式）
	syncRequest := &models.AlertSyncRequest{
		CabinetID: cabinetID,
		Timestamp: time.Now(),
		Alerts:    []models.AlertSyncData{alertData},
	}

	// 调用告警服务保存数据
	if err := s.alertService.SyncAlerts(s.ctx, syncRequest); err != nil {
		utils.Error("Failed to save alert from MQTT",
			zap.String("cabinet_id", cabinetID),
			zap.String("device_id", alertData.DeviceID),
			zap.Error(err),
		)
		return
	}

	utils.Info("Alert received via MQTT and saved successfully",
		zap.String("cabinet_id", cabinetID),
		zap.String("device_id", alertData.DeviceID),
		zap.String("alert_type", alertData.AlertType),
		zap.String("severity", alertData.Severity),
	)

	// 通过WebSocket广播告警（如果需要）
	if s.wsHub != nil {
		s.wsHub.BroadcastSensorData(map[string]interface{}{
			"type":       "alert",
			"cabinet_id": cabinetID,
			"alert":      alertData,
			"timestamp":  time.Now(),
		})
	}
}
