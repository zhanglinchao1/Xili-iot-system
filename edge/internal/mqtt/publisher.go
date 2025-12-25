package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// TrafficPublisher 负责将流量统计推送到MQTT Broker
type TrafficPublisher struct {
	cfg    config.MQTTConfig
	logger *zap.Logger
	client mqtt.Client
}

// NewTrafficPublisher 创建Publisher
func NewTrafficPublisher(cfg config.MQTTConfig, logger *zap.Logger) *TrafficPublisher {
	return &TrafficPublisher{cfg: cfg, logger: logger}
}

// Start 连接到MQTT Broker
func (p *TrafficPublisher) Start() error {
	if !p.cfg.Enabled {
		p.logger.Info("MQTT发布功能已禁用")
		return nil
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(p.cfg.BrokerAddress)
	clientID := p.cfg.ClientID
	if clientID == "" {
		clientID = "edge-traffic-publisher"
	}
	opts.SetClientID(clientID + "-pub")
	opts.SetUsername(p.cfg.Username)
	opts.SetPassword(p.cfg.Password)
	opts.SetKeepAlive(time.Duration(p.cfg.KeepAlive) * time.Second)
	opts.SetCleanSession(true)

	// TLS配置
	if p.cfg.TLS.Enabled {
		tlsConfig := &tls.Config{InsecureSkipVerify: p.cfg.TLS.InsecureSkipVerify}
		if p.cfg.TLS.CAFile != "" {
			caPEM, err := os.ReadFile(p.cfg.TLS.CAFile)
			if err != nil {
				return fmt.Errorf("读取CA证书失败: %w", err)
			}
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caPEM) {
				return fmt.Errorf("解析CA证书失败")
			}
			tlsConfig.RootCAs = certPool
		}
		if p.cfg.TLS.CertFile != "" && p.cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(p.cfg.TLS.CertFile, p.cfg.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("加载客户端证书失败: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		opts.SetTLSConfig(tlsConfig)
	}

	p.client = mqtt.NewClient(opts)
	token := p.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("连接MQTT发布通道超时")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("连接MQTT发布通道失败: %w", err)
	}

	p.logger.Info("MQTT流量发布器已连接",
		zap.String("broker", p.cfg.BrokerAddress))
	return nil
}

// Stop 断开连接
func (p *TrafficPublisher) Stop() {
	if p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(250)
		p.logger.Info("MQTT流量发布器已停止")
	}
}

// PublishTraffic 推送流量统计
func (p *TrafficPublisher) PublishTraffic(stat *TrafficStat) error {
	if p.client == nil || !p.client.IsConnected() {
		return fmt.Errorf("MQTT发布器未连接")
	}
	payload, err := json.Marshal(stat)
	if err != nil {
		return fmt.Errorf("序列化流量数据失败: %w", err)
	}
	topic := TopicTrafficPrefix + stat.CabinetID
	token := p.client.Publish(topic, p.cfg.QoS, false, payload)
	token.Wait()
	return token.Error()
}

// PublishReport 根据评估结果推送流量统计
func (p *TrafficPublisher) PublishReport(report *models.EdgeVulnerabilityReport) error {
	if report == nil || report.TransmissionMetrics == nil {
		return nil
	}

	stat := &TrafficStat{
		CabinetID:         report.CabinetID,
		Timestamp:         report.Timestamp,
		ThroughputKbps:    report.TransmissionMetrics.Throughput,
		LatencyMs:         report.TransmissionMetrics.LatencyAvg,
		PacketLossRate:    report.TransmissionMetrics.PacketLossRate,
		MQTTSuccessRate:   report.TransmissionMetrics.MQTTSuccessRate,
		ReconnectionCount: report.TransmissionMetrics.ReconnectionCount,
		RiskLevel:         report.RiskLevel,
	}

	return p.PublishTraffic(stat)
}
