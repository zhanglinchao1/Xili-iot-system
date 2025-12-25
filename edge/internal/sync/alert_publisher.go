/*
 * 告警MQTT发布器
 * 负责将告警通过MQTT实时推送到Cloud端
 */
package sync

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// AlertPublisher 告警MQTT发布器
type AlertPublisher struct {
	mqttClient mqtt.Client
	cabinetID  string
	logger     *zap.Logger
	enabled    bool
}

// NewAlertPublisher 创建告警MQTT发布器
func NewAlertPublisher(mqttClient mqtt.Client, cabinetID string, logger *zap.Logger) *AlertPublisher {
	enabled := mqttClient != nil
	if enabled {
		logger.Info("告警MQTT发布器已创建", zap.String("cabinet_id", cabinetID))
	} else {
		logger.Warn("MQTT客户端未提供，告警将仅通过HTTP同步")
	}

	return &AlertPublisher{
		mqttClient: mqttClient,
		cabinetID:  cabinetID,
		logger:     logger,
		enabled:    enabled,
	}
}

// PublishAlert 通过MQTT发布告警
func (p *AlertPublisher) PublishAlert(alert *models.Alert) error {
	if !p.enabled || p.mqttClient == nil {
		return fmt.Errorf("MQTT发布器未启用")
	}

	if !p.mqttClient.IsConnected() {
		return fmt.Errorf("MQTT客户端未连接")
	}

	// 映射severity值到Cloud端期望的格式
	// Edge: low/medium/high/critical -> Cloud: info/warning/error/critical
	severityMapped := mapSeverityToCloud(alert.Severity)

	// 构建消息负载
	payload, err := json.Marshal(map[string]interface{}{
		"alert_id":    alert.ID,
		"device_id":   alert.DeviceID,
		"alert_type":  alert.AlertType,
		"severity":    severityMapped,
		"message":     alert.Message,
		"value":       alert.Value,
		"threshold":   alert.Threshold,
		"timestamp":   alert.Timestamp,
		"resolved":    alert.Resolved,
		"resolved_at": alert.ResolvedAt,
	})
	if err != nil {
		p.logger.Error("序列化告警数据失败", zap.Error(err))
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	// 发布到MQTT - Topic: edge/cabinet/{cabinet_id}/alerts
	topic := fmt.Sprintf("edge/cabinet/%s/alerts", p.cabinetID)
	token := p.mqttClient.Publish(topic, 1, false, payload)

	// 等待发布完成（带超时）
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("MQTT告警发布超时")
	}

	if err := token.Error(); err != nil {
		p.logger.Error("MQTT告警发布失败",
			zap.String("device_id", alert.DeviceID),
			zap.String("alert_type", alert.AlertType),
			zap.Error(err))
		return err
	}

	p.logger.Info("MQTT告警发布成功",
		zap.String("topic", topic),
		zap.String("device_id", alert.DeviceID),
		zap.String("alert_type", alert.AlertType),
		zap.String("severity", alert.Severity))

	return nil
}

// IsEnabled 返回发布器是否启用
func (p *AlertPublisher) IsEnabled() bool {
	return p.enabled && p.mqttClient != nil && p.mqttClient.IsConnected()
}

// mapSeverityToCloud函数已在cloud_sync.go中定义，可直接使用
