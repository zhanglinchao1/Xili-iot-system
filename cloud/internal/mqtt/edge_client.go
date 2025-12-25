/*
 * Edge端MQTT客户端
 * 用于订阅Edge端传感器数据
 */
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

// EdgeClient Edge端MQTT客户端（用于订阅传感器数据）
type EdgeClient struct {
	client mqtt.Client
	cfg    *config.Config
}

// NewEdgeClient 创建Edge端MQTT客户端
func NewEdgeClient(cfg *config.Config) (*EdgeClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.EdgeMQTT.Broker)
	opts.SetClientID(cfg.EdgeMQTT.ClientID)
	opts.SetUsername(cfg.EdgeMQTT.Username)
	opts.SetPassword(cfg.EdgeMQTT.Password)
	opts.SetCleanSession(cfg.EdgeMQTT.CleanSession)
	opts.SetKeepAlive(time.Duration(cfg.EdgeMQTT.KeepAlive) * time.Second)
	opts.SetAutoReconnect(true)

	// 解析重连延迟
	reconnectDelay, _ := time.ParseDuration(cfg.EdgeMQTT.ReconnectDelay)
	maxReconnectInterval, _ := time.ParseDuration(cfg.EdgeMQTT.MaxReconnectInterval)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(reconnectDelay)
	opts.SetMaxReconnectInterval(maxReconnectInterval)

	// 配置TLS（如果启用）
	if cfg.EdgeMQTT.TLS.Enabled {
		tlsConfig, err := newEdgeTLSConfig(&cfg.EdgeMQTT.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
		utils.Info("Edge MQTT TLS enabled",
			zap.String("ca_file", cfg.EdgeMQTT.TLS.CAFile),
			zap.Bool("insecure_skip_verify", cfg.EdgeMQTT.TLS.InsecureSkipVerify),
		)
	}

	// 设置连接回调
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		utils.Info("Edge MQTT connected", zap.String("broker", cfg.EdgeMQTT.Broker))
	})

	// 设置连接丢失回调
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		utils.Warn("Edge MQTT connection lost", zap.Error(err))
	})

	// 设置重连回调
	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		utils.Info("Edge MQTT reconnecting...")
	})

	// 创建客户端
	client := mqtt.NewClient(opts)

	// 连接（带超时，默认30秒）
	connectTimeout := 30 * time.Second
	if token := client.Connect(); !token.WaitTimeout(connectTimeout) {
		return nil, fmt.Errorf("failed to connect to Edge MQTT broker: connection timeout after %v", connectTimeout)
	} else if token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to Edge MQTT broker: %w", token.Error())
	}

	utils.Info("Edge MQTT client initialized", zap.String("broker", cfg.EdgeMQTT.Broker))

	return &EdgeClient{
		client: client,
		cfg:    cfg,
	}, nil
}

// newEdgeTLSConfig 创建Edge端TLS配置
func newEdgeTLSConfig(tlsCfg *config.EdgeMQTTTLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
	}

	// 如果提供了CA证书文件，加载它
	if tlsCfg.CAFile != "" {
		certpool := x509.NewCertPool()
		ca, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		if !certpool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = certpool
	}

	return tlsConfig, nil
}

// GetClient 获取MQTT客户端
func (c *EdgeClient) GetClient() mqtt.Client {
	return c.client
}

// Close 关闭连接
func (c *EdgeClient) Close() error {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		utils.Info("Edge MQTT connection closed")
	}
	return nil
}

// IsConnected 检查连接状态
func (c *EdgeClient) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish 发布消息到指定topic
func (c *EdgeClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	token := c.client.Publish(topic, qos, retained, payload)
	token.Wait()
	if token.Error() != nil {
		utils.Error("Failed to publish MQTT message",
			zap.String("topic", topic),
			zap.Error(token.Error()))
		return token.Error()
	}

	utils.Debug("MQTT message published",
		zap.String("topic", topic),
		zap.Uint8("qos", qos))
	return nil
}
