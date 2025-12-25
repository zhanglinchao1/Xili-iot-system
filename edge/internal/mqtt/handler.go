/*
 * MQTT 消息处理器
 * 处理不同类型的 MQTT 消息
 */
package mqtt

import (
	"encoding/json"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/edge/storage-cabinet/internal/cloud"
	"github.com/edge/storage-cabinet/internal/license"
	"go.uber.org/zap"
)

// Handler MQTT 消息处理器
type Handler struct {
	logger           *zap.Logger
	collectorService CollectorService
	deviceManager    DeviceManager
	wsHub            *WebSocketHub // WebSocket管理器
	stats            *MQTTStats    // MQTT统计数据
	licenseService   *license.Service
	ackClient        *cloud.CommandClient
}

// CollectorService 数据采集服务接口
type CollectorService interface {
	SaveSensorData(data interface{}) error
	SaveAlert(alert interface{}) error
	ResolveAlert(alertID int64) error // 解决告警
}

// DeviceManager 设备管理器接口
type DeviceManager interface {
	UpdateDeviceStatusString(deviceID, status string) error
	UpdateLastSeen(deviceID string) error
}

// NewHandler 创建消息处理器
func NewHandler(logger *zap.Logger, collector CollectorService, deviceMgr DeviceManager, stats *MQTTStats, licenseSvc *license.Service, ackClient *cloud.CommandClient) *Handler {
	return &Handler{
		logger:           logger,
		collectorService: collector,
		deviceManager:    deviceMgr,
		wsHub:            NewWebSocketHub(logger),
		stats:            stats,
		licenseService:   licenseSvc,
		ackClient:        ackClient,
	}
}

// SetWebSocketHub 设置WebSocket管理器
func (h *Handler) SetWebSocketHub(wsHub *WebSocketHub) {
	h.wsHub = wsHub
}

// GetWebSocketHub 获取WebSocket管理器
func (h *Handler) GetWebSocketHub() *WebSocketHub {
	return h.wsHub
}

// HandleMessage 处理 MQTT 消息（路由器）
func (h *Handler) HandleMessage(client mqtt.Client, msg mqtt.Message) {
	// 记录消息接收时间（用于计算延迟）
	receiveTime := time.Now()

	topic := msg.Topic()
	payload := msg.Payload()

	h.logger.Debug("📥 收到 MQTT 消息",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	// 根据 Topic 前缀路由到不同处理器
	switch {
	case strings.HasPrefix(topic, "cloud/cabinets/"):
		// 新格式: cloud/cabinets/{cabinet_id}/commands/{category}
		if strings.Contains(topic, "/commands/") {
			h.handleCommand(topic, payload)
		}
	case strings.HasPrefix(topic, "sensors/"):
		h.handleSensorData(topic, payload, receiveTime)
	case strings.Contains(topic, "/status"):
		h.handleDeviceStatus(topic, payload, receiveTime)
	case strings.HasPrefix(topic, "alerts/"):
		h.handleAlert(topic, payload, receiveTime)
	case strings.Contains(topic, "/heartbeat"):
		h.handleHeartbeat(topic, payload, receiveTime)
	default:
		h.logger.Warn("⚠️  未知的 MQTT Topic", zap.String("topic", topic))
	}
}

type commandMessage struct {
	CommandID   string                 `json:"command_id"`
	CommandType string                 `json:"command_type"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   int64                  `json:"timestamp"`
}

func (h *Handler) handleCommand(topic string, payload []byte) {
	h.logger.Info("📥 收到Cloud命令",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	// 许可证相关命令需要licenseService，其他命令可以正常处理
	// 注意：即使licenseService为nil或未启用，也允许处理许可证更新命令（用于修复循环依赖问题）

	var cmd commandMessage
	if err := json.Unmarshal(payload, &cmd); err != nil {
		h.logger.Error("解析命令失败",
			zap.String("topic", topic),
			zap.String("payload", string(payload)),
			zap.Error(err))
		return
	}

	h.logger.Info("解析命令成功",
		zap.String("command_id", cmd.CommandID),
		zap.String("command_type", cmd.CommandType),
		zap.Int64("timestamp", cmd.Timestamp))

	switch cmd.CommandType {
	case "license_push", "license_update":
		// 许可证更新命令：允许在licenseService为nil时处理（用于修复循环依赖问题）
		if h.licenseService == nil {
			h.logger.Warn("收到许可证更新命令但许可证服务未初始化，尝试初始化",
				zap.String("command_id", cmd.CommandID))
			h.ackCommand(cmd.CommandID, "failed", "license service not initialized")
			return
		}
		token, _ := cmd.Payload["license_token"].(string)
		if token == "" {
			h.logger.Error("许可证命令缺少token",
				zap.String("command_id", cmd.CommandID))
			h.ackCommand(cmd.CommandID, "failed", "missing token")
			return
		}
		if err := h.licenseService.ApplyLicenseToken(token); err != nil {
			h.logger.Error("应用许可证失败",
				zap.String("command_id", cmd.CommandID),
				zap.Error(err))
			h.ackCommand(cmd.CommandID, "failed", err.Error())
			return
		}
		h.logger.Info("许可证已更新",
			zap.String("command_id", cmd.CommandID))
		h.ackCommand(cmd.CommandID, "success", "license updated")
	case "license_revoke":
		// 许可证吊销命令：允许在licenseService为nil时处理
		if h.licenseService == nil {
			h.logger.Warn("收到许可证吊销命令但许可证服务未初始化",
				zap.String("command_id", cmd.CommandID))
			h.ackCommand(cmd.CommandID, "failed", "license service not initialized")
			return
		}
		if err := h.licenseService.RevokeLicense(); err != nil {
			h.logger.Error("吊销许可证失败",
				zap.String("command_id", cmd.CommandID),
				zap.Error(err))
			h.ackCommand(cmd.CommandID, "failed", err.Error())
			return
		}
		h.logger.Warn("许可证已吊销",
			zap.String("command_id", cmd.CommandID))
		h.ackCommand(cmd.CommandID, "success", "license revoked")
	case "resolve_alert":
		// 从payload中提取alert_id
		alertID, ok := cmd.Payload["alert_id"].(float64)
		if !ok {
			h.logger.Error("告警解决命令缺少alert_id或格式错误",
				zap.String("command_id", cmd.CommandID))
			h.ackCommand(cmd.CommandID, "failed", "missing or invalid alert_id")
			return
		}

		// 调用collectorService解决告警
		if err := h.collectorService.ResolveAlert(int64(alertID)); err != nil {
			h.logger.Error("解决告警失败",
				zap.String("command_id", cmd.CommandID),
				zap.Int64("alert_id", int64(alertID)),
				zap.Error(err))
			h.ackCommand(cmd.CommandID, "failed", err.Error())
			return
		}

		h.logger.Info("告警已解决（通过Cloud命令）",
			zap.String("command_id", cmd.CommandID),
			zap.Int64("alert_id", int64(alertID)))
		h.ackCommand(cmd.CommandID, "success", "alert resolved")
	default:
		h.logger.Warn("收到未知命令",
			zap.String("command_type", cmd.CommandType))
	}
}

func (h *Handler) ackCommand(commandID, status, message string) {
	if h.ackClient == nil || commandID == "" {
		return
	}
	if err := h.ackClient.AckCommand(commandID, status, message); err != nil {
		h.logger.Warn("回执命令失败",
			zap.String("command_id", commandID),
			zap.Error(err))
	}
}

// handleSensorData 处理传感器数据
func (h *Handler) handleSensorData(topic string, payload []byte, receiveTime time.Time) {
	var data SensorData
	if err := json.Unmarshal(payload, &data); err != nil {
		h.logger.Error("❌ 解析传感器数据失败",
			zap.String("topic", topic),
			zap.Error(err))
		return
	}

	// 计算延迟（消息时间戳到接收时间）
	latency := receiveTime.Sub(data.Timestamp).Seconds() * 1000 // 转换为毫秒
	if latency < 0 {
		latency = 0 // 防止时钟不同步导致负值
	}

	// 记录消息统计
	if h.stats != nil {
		h.stats.RecordMessageReceived(latency)
	}

	// 调用 collector 服务保存数据 (collector会记录保存日志,这里不再重复记录)
	if err := h.collectorService.SaveSensorData(&data); err != nil {
		h.logger.Error("❌ 保存传感器数据失败",
			zap.String("device_id", data.DeviceID),
			zap.String("sensor_type", data.SensorType),
			zap.Error(err))
		return
	}

	// 通过WebSocket广播传感器数据
	if h.wsHub != nil {
		h.wsHub.BroadcastSensorData(&data)
	}
}

// handleDeviceStatus 处理设备状态
func (h *Handler) handleDeviceStatus(topic string, payload []byte, receiveTime time.Time) {
	var status DeviceStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		h.logger.Error("❌ 解析设备状态失败",
			zap.String("topic", topic),
			zap.Error(err))
		return
	}

	// 计算延迟
	latency := receiveTime.Sub(status.Timestamp).Seconds() * 1000
	if latency < 0 {
		latency = 0
	}

	// 记录消息统计
	if h.stats != nil {
		h.stats.RecordMessageReceived(latency)
	}

	// 更新设备状态
	if err := h.deviceManager.UpdateDeviceStatusString(status.DeviceID, status.Status); err != nil {
		h.logger.Error("❌ 更新设备状态失败",
			zap.String("device_id", status.DeviceID),
			zap.String("status", status.Status),
			zap.Error(err))
		return
	}

	h.logger.Info("🔌 设备状态已更新",
		zap.String("device_id", status.DeviceID),
		zap.String("status", status.Status))

	// 通过WebSocket广播设备状态
	if h.wsHub != nil {
		h.wsHub.BroadcastDeviceStatus(&status)
	}
}

// handleAlert 处理告警信息
func (h *Handler) handleAlert(topic string, payload []byte, receiveTime time.Time) {
	var alert Alert
	if err := json.Unmarshal(payload, &alert); err != nil {
		h.logger.Error("❌ 解析告警信息失败",
			zap.String("topic", topic),
			zap.Error(err))
		return
	}

	// 计算延迟
	latency := receiveTime.Sub(alert.Timestamp).Seconds() * 1000
	if latency < 0 {
		latency = 0
	}

	// 记录消息统计
	if h.stats != nil {
		h.stats.RecordMessageReceived(latency)
	}

	// 保存告警
	if err := h.collectorService.SaveAlert(&alert); err != nil {
		h.logger.Error("❌ 保存告警失败",
			zap.String("device_id", alert.DeviceID),
			zap.String("alert_type", alert.AlertType),
			zap.Error(err))
		return
	}

	h.logger.Warn("🚨 收到告警",
		zap.String("device_id", alert.DeviceID),
		zap.String("alert_type", alert.AlertType),
		zap.String("severity", alert.Severity),
		zap.String("message", alert.Message))

	// 通过WebSocket广播告警信息
	if h.wsHub != nil {
		h.wsHub.BroadcastAlert(&alert)
	}
}

// handleHeartbeat 处理心跳信息
func (h *Handler) handleHeartbeat(topic string, payload []byte, receiveTime time.Time) {
	var heartbeat Heartbeat
	if err := json.Unmarshal(payload, &heartbeat); err != nil {
		h.logger.Error("❌ 解析心跳信息失败",
			zap.String("topic", topic),
			zap.Error(err))
		return
	}

	// 计算延迟
	latency := receiveTime.Sub(heartbeat.Timestamp).Seconds() * 1000
	if latency < 0 {
		latency = 0
	}

	// 记录消息统计
	if h.stats != nil {
		h.stats.RecordMessageReceived(latency)
	}

	// 更新设备最后活跃时间
	if err := h.deviceManager.UpdateLastSeen(heartbeat.DeviceID); err != nil {
		h.logger.Error("❌ 更新设备心跳失败",
			zap.String("device_id", heartbeat.DeviceID),
			zap.Error(err))
		return
	}

	h.logger.Debug("💓 收到心跳",
		zap.String("device_id", heartbeat.DeviceID))

	// 通过WebSocket广播心跳信息
	if h.wsHub != nil {
		h.wsHub.BroadcastHeartbeat(&heartbeat)
	}
}
