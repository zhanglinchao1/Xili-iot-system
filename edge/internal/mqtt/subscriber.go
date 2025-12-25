/*
 * MQTT 订阅器
 * 负责连接 MQTT Broker 并订阅 Topic
 */
package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/edge/storage-cabinet/internal/cloud"
	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/license"
	"go.uber.org/zap"
)

// ABACPolicyHandler ABAC策略处理接口
type ABACPolicyHandler interface {
	GetPolicyTopic() string
	HandlePolicySync(payload []byte) error
}

// Subscriber MQTT 订阅器
type Subscriber struct {
	logger    *zap.Logger
	config    config.MQTTConfig
	client    mqtt.Client
	handler   *Handler
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc

	// 统计数据
	stats       *MQTTStats
	cabinetID   string
	commandTopic string

	// ABAC策略处理器
	abacHandler ABACPolicyHandler
	abacTopic   string
}

// NewSubscriber 创建 MQTT 订阅器
func NewSubscriber(
	cfg config.MQTTConfig,
	collector CollectorService,
	deviceMgr DeviceManager,
	licenseService *license.Service,
	ackClient *cloud.CommandClient,
	logger *zap.Logger,
	cabinetID string,
) *Subscriber {
	// 先创建统计对象
	stats := NewMQTTStats()

	// 创建处理器时注入统计对象
	handler := NewHandler(logger, collector, deviceMgr, stats, licenseService, ackClient)

	return &Subscriber{
		logger:    logger,
		config:    cfg,
		handler:   handler,
		stats:     stats,
		cabinetID: cabinetID,
	}
}

// SetABACHandler 设置ABAC策略处理器
func (s *Subscriber) SetABACHandler(handler ABACPolicyHandler) {
	s.abacHandler = handler
	if handler != nil {
		s.abacTopic = handler.GetPolicyTopic()
		s.logger.Info("ABAC策略处理器已注册", zap.String("topic", s.abacTopic))
	}
}

// Start 启动 MQTT 订阅器
func (s *Subscriber) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("MQTT 订阅器已禁用")
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动WebSocket Hub
	if s.handler.GetWebSocketHub() != nil {
		go s.handler.GetWebSocketHub().Run()
	}

	// 配置 MQTT 客户端选项
	opts := mqtt.NewClientOptions()
	opts.AddBroker(s.config.BrokerAddress)
	opts.SetClientID(s.config.ClientID)
	opts.SetUsername(s.config.Username)
	opts.SetPassword(s.config.Password)
	opts.SetKeepAlive(time.Duration(s.config.KeepAlive) * time.Second)
	opts.SetCleanSession(s.config.CleanSession)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(s.config.ReconnectInterval)

	// 检查broker地址是否使用SSL/TLS协议
	brokerUsesTLS := false
	if len(s.config.BrokerAddress) > 6 {
		protocol := s.config.BrokerAddress[:6]
		if protocol == "ssl://" || protocol == "tls://" || protocol == "tcps://" {
			brokerUsesTLS = true
		}
	}

	// 配置TLS（如果启用或broker地址使用SSL/TLS协议）
	if s.config.TLS.Enabled || brokerUsesTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: s.config.TLS.InsecureSkipVerify,
		}

		// 加载CA证书（如果提供）
		if s.config.TLS.CAFile != "" {
			caCertPEM, err := os.ReadFile(s.config.TLS.CAFile)
			if err != nil {
				return fmt.Errorf("读取CA证书失败: %w", err)
			}

			// 创建证书池并添加CA证书
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caCertPEM) {
				return fmt.Errorf("解析CA证书失败: 无法添加到证书池")
			}

			tlsConfig.RootCAs = certPool
			s.logger.Info("已加载并配置CA证书",
				zap.String("ca_file", s.config.TLS.CAFile),
				zap.Bool("insecure_skip_verify", s.config.TLS.InsecureSkipVerify))
		}

		// 加载客户端证书和私钥（如果提供）
		if s.config.TLS.CertFile != "" && s.config.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("加载客户端证书失败: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
			s.logger.Info("已加载客户端证书",
				zap.String("cert_file", s.config.TLS.CertFile),
				zap.String("key_file", s.config.TLS.KeyFile))
		}

		opts.SetTLSConfig(tlsConfig)
		if brokerUsesTLS && !s.config.TLS.Enabled {
			s.logger.Info("MQTT TLS已自动启用（检测到SSL/TLS协议）",
				zap.String("broker", s.config.BrokerAddress),
				zap.Bool("insecure_skip_verify", s.config.TLS.InsecureSkipVerify))
		} else {
			s.logger.Info("MQTT TLS已启用",
				zap.Bool("insecure_skip_verify", s.config.TLS.InsecureSkipVerify),
				zap.String("broker", s.config.BrokerAddress))
		}
	}

	// 设置连接回调
	opts.SetOnConnectHandler(s.onConnect)
	opts.SetConnectionLostHandler(s.onConnectionLost)

	// 设置默认消息处理器
	opts.SetDefaultPublishHandler(s.createMessageHandler())

	// 创建客户端
	s.client = mqtt.NewClient(opts)

	// 连接到 Broker
	s.logger.Info("正在连接到 MQTT Broker...",
		zap.String("broker", s.config.BrokerAddress),
		zap.String("client_id", s.config.ClientID))

	token := s.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("连接 MQTT Broker 超时")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("连接 MQTT Broker 失败: %w", err)
	}

	s.connected = true
	s.logger.Info("✅ 已连接到 MQTT Broker",
		zap.String("broker", s.config.BrokerAddress))

	return nil
}

// Stop 停止 MQTT 订阅器
func (s *Subscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.client != nil && s.client.IsConnected() {
		// 取消订阅
		s.unsubscribeAll()

		// 断开连接
		s.client.Disconnect(250)
		s.logger.Info("MQTT 订阅器已停止")
	}

	s.connected = false
}

// onConnect 连接成功回调
func (s *Subscriber) onConnect(client mqtt.Client) {
	s.logger.Info("🔗 MQTT 连接已建立，开始订阅 Topic...")

	s.connected = true
	s.stats.RecordConnect()

	// 订阅所有需要的 Topic
	topics := map[string]byte{
		TopicSensors:   s.config.QoS,
		TopicDevices:   s.config.QoS,
		TopicAlerts:    s.config.QoS,
		TopicHeartbeat: s.config.QoS,
	}

	// 订阅Cloud命令主题（移除licenseService依赖，允许在许可证问题情况下接收许可证更新）
	// 修复循环依赖问题：即使许可证MAC不匹配，也能接收许可证更新命令
	if s.cabinetID != "" {
		s.commandTopic = fmt.Sprintf("cloud/cabinets/%s/commands/#", s.cabinetID)
		topics[s.commandTopic] = s.config.QoS
	}

	// 订阅ABAC策略同步topic
	if s.abacTopic != "" {
		topics[s.abacTopic] = s.config.QoS
	}

	for topic, qos := range topics {
		token := client.Subscribe(topic, qos, nil)
		if token.Wait() && token.Error() != nil {
			s.logger.Error("❌ 订阅 Topic 失败",
				zap.String("topic", topic),
				zap.Error(token.Error()))
			s.stats.RecordMessageFailed()
		} else {
			s.logger.Info("✅ 已订阅 Topic",
				zap.String("topic", topic),
				zap.Uint8("qos", uint8(qos)))
			s.stats.RecordMessageSent()
		}
	}
}

// onConnectionLost 连接丢失回调
func (s *Subscriber) onConnectionLost(client mqtt.Client, err error) {
	s.logger.Error("❌ MQTT 连接丢失",
		zap.Error(err),
		zap.String("broker", s.config.BrokerAddress))

	s.connected = false
	s.stats.RecordDisconnect()

	s.logger.Info("⏳ 将自动重连...")
}

// unsubscribeAll 取消所有订阅
func (s *Subscriber) unsubscribeAll() {
	topics := []string{
		TopicSensors,
		TopicDevices,
		TopicAlerts,
		TopicHeartbeat,
	}

	if s.commandTopic != "" {
		topics = append(topics, s.commandTopic)
	}
	if s.abacTopic != "" {
		topics = append(topics, s.abacTopic)
	}

	for _, topic := range topics {
		token := s.client.Unsubscribe(topic)
		if token.Wait() && token.Error() != nil {
			s.logger.Warn("取消订阅失败",
				zap.String("topic", topic),
				zap.Error(token.Error()))
		}
	}
}

// IsConnected 检查是否已连接
func (s *Subscriber) IsConnected() bool {
	return s.connected && s.client != nil && s.client.IsConnected()
}

// GetWebSocketHub 获取WebSocket管理器
func (s *Subscriber) GetWebSocketHub() *WebSocketHub {
	if s.handler != nil {
		return s.handler.GetWebSocketHub()
	}
	return nil
}

// GetStats 获取MQTT统计数据
func (s *Subscriber) GetStats() *MQTTStats {
	return s.stats
}

// GetMQTTClient 获取MQTT客户端（用于发布消息）
func (s *Subscriber) GetMQTTClient() mqtt.Client {
	return s.client
}

// createMessageHandler 创建消息处理器（包装原有handler，增加ABAC处理）
func (s *Subscriber) createMessageHandler() mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		topic := msg.Topic()

		// 检查是否是ABAC策略消息
		if s.abacHandler != nil && s.abacTopic != "" && topic == s.abacTopic {
			s.logger.Info("收到ABAC策略同步消息", zap.String("topic", topic))
			if err := s.abacHandler.HandlePolicySync(msg.Payload()); err != nil {
				s.logger.Error("处理ABAC策略消息失败", zap.Error(err))
			}
			return
		}

		// 其他消息交给原有handler处理
		s.handler.HandleMessage(client, msg)
	}
}
