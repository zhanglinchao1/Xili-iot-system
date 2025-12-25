package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// Client MQTT客户端
type Client struct {
	client mqtt.Client
	cfg    *config.Config
}

// NewClient 创建MQTT客户端
func NewClient(cfg *config.Config) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTT.Broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.SetCleanSession(cfg.MQTT.CleanSession)
	opts.SetKeepAlive(time.Duration(cfg.MQTT.KeepAlive) * time.Second)
	opts.SetAutoReconnect(true)

	// 解析重连延迟
	reconnectDelay, _ := time.ParseDuration(cfg.MQTT.ReconnectDelay)
	maxReconnectInterval, _ := time.ParseDuration(cfg.MQTT.MaxReconnectInterval)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(reconnectDelay)
	opts.SetMaxReconnectInterval(maxReconnectInterval)

	// 配置TLS（如果启用）
	if cfg.MQTT.TLS.Enabled {
		tlsConfig, err := newTLSConfig(&cfg.MQTT.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
		utils.Info("MQTT TLS enabled", zap.String("ca_file", cfg.MQTT.TLS.CAFile))
	}

	// 设置连接回调
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		utils.Info("MQTT connected", zap.String("broker", cfg.MQTT.Broker))
	})

	// 设置连接丢失回调
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		utils.Warn("MQTT connection lost", zap.Error(err))
	})

	// 设置重连回调
	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		utils.Info("MQTT reconnecting...")
	})

	// 创建客户端
	client := mqtt.NewClient(opts)

	// 连接（带超时，默认30秒）
	connectTimeout := 30 * time.Second
	if token := client.Connect(); !token.WaitTimeout(connectTimeout) {
		return nil, fmt.Errorf("failed to connect to MQTT broker: connection timeout after %v", connectTimeout)
	} else if token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	utils.Info("MQTT client initialized", zap.String("broker", cfg.MQTT.Broker))

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// GetClient 获取MQTT客户端
func (c *Client) GetClient() mqtt.Client {
	return c.client
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		utils.Info("MQTT connection closed")
	}
	return nil
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish 发布消息
func (c *Client) Publish(topic string, payload interface{}) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := c.client.Publish(topic, c.cfg.MQTT.QoS, false, payload)
	token.Wait()

	if token.Error() != nil {
		utils.Error("Failed to publish MQTT message",
			zap.String("topic", topic),
			zap.Error(token.Error()),
		)
		return token.Error()
	}

	utils.Debug("MQTT message published", zap.String("topic", topic))
	return nil
}

// Subscribe 订阅主题
func (c *Client) Subscribe(topic string, callback mqtt.MessageHandler) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := c.client.Subscribe(topic, c.cfg.MQTT.QoS, callback)
	token.Wait()

	if token.Error() != nil {
		utils.Error("Failed to subscribe MQTT topic",
			zap.String("topic", topic),
			zap.Error(token.Error()),
		)
		return token.Error()
	}

	utils.Info("MQTT topic subscribed", zap.String("topic", topic))
	return nil
}

// Unsubscribe 取消订阅
func (c *Client) Unsubscribe(topics ...string) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := c.client.Unsubscribe(topics...)
	token.Wait()

	if token.Error() != nil {
		utils.Error("Failed to unsubscribe MQTT topics",
			zap.Strings("topics", topics),
			zap.Error(token.Error()),
		)
		return token.Error()
	}

	utils.Info("MQTT topics unsubscribed", zap.Strings("topics", topics))
	return nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}
	return nil
}

// 定义MQTT主题常量（基于senddata.md规范）
// Topic 格式: cloud/cabinets/{cabinet_id}/commands/{category}
const (
	// 命令主题模板 - 符合 senddata.md 规范
	TopicCommandConfig  = "cloud/cabinets/%s/commands/config"  // 配置更新
	TopicCommandLicense = "cloud/cabinets/%s/commands/license" // 许可证指令
	TopicCommandQuery   = "cloud/cabinets/%s/commands/query"   // 查询指令
	TopicCommandControl = "cloud/cabinets/%s/commands/control" // 控制指令

	// 响应主题模板 - Edge 端发布响应到此 Topic
	TopicResponse = "cloud/cabinets/%s/responses/%s" // 响应: {cabinet_id}/responses/{command_id}
)

// GetCommandTopic 获取命令主题
// 根据 senddata.md 规范，命令 Topic 格式为: cloud/cabinets/{cabinet_id}/commands/{category}
func GetCommandTopic(cabinetID, commandType string) string {
	switch commandType {
	case "config", "config_update", "config_push":
		return fmt.Sprintf(TopicCommandConfig, cabinetID)
	case "license", "license_update", "license_push", "license_revoke":
		return fmt.Sprintf(TopicCommandLicense, cabinetID)
	case "query", "query_status", "query_logs":
		return fmt.Sprintf(TopicCommandQuery, cabinetID)
	case "control", "restart", "mode_switch", "cache_clear", "resolve_alert":
		return fmt.Sprintf(TopicCommandControl, cabinetID)
	default:
		// 默认使用 control 类别
		return fmt.Sprintf(TopicCommandControl, cabinetID)
	}
}

// GetResponseTopic 获取响应主题
// 根据 senddata.md 规范，响应 Topic 格式为: cloud/cabinets/{cabinet_id}/responses/{command_id}
func GetResponseTopic(cabinetID, commandID string) string {
	return fmt.Sprintf(TopicResponse, cabinetID, commandID)
}

// GetResponseSubscribeTopic 获取响应订阅主题（通配符）
// Cloud 端订阅所有储能柜的响应: cloud/cabinets/+/responses/+
func GetResponseSubscribeTopic() string {
	return "cloud/cabinets/+/responses/+"
}

// newTLSConfig 创建TLS配置
func newTLSConfig(tlsCfg *config.EdgeMQTTTLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
	}

	// 加载CA证书
	if tlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
