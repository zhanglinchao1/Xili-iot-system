/*
 * MQTT统计数据追踪
 * 用于脆弱性评估的通信指标收集
 */
package mqtt

import (
	"sync"
	"time"
)

// MQTTStats MQTT统计数据
type MQTTStats struct {
	mu sync.RWMutex

	// 连接统计
	ConnectTime        time.Time
	LastDisconnectTime time.Time
	ReconnectCount     int

	// 消息统计
	MessagesSent     int64
	MessagesReceived int64
	MessagesFailed   int64
	LastMessageTime  time.Time

	// 延迟统计
	LatencySamples []float64
	MaxLatencySamples int

	// 统计窗口
	WindowStartTime time.Time
}

// NewMQTTStats 创建MQTT统计数据实例
func NewMQTTStats() *MQTTStats {
	return &MQTTStats{
		ConnectTime:       time.Now(),
		WindowStartTime:   time.Now(),
		MaxLatencySamples: 100, // 保留最近100个延迟样本
		LatencySamples:    make([]float64, 0, 100),
	}
}

// RecordConnect 记录连接事件
func (s *MQTTStats) RecordConnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ConnectTime = time.Now()
}

// RecordDisconnect 记录断连事件
func (s *MQTTStats) RecordDisconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastDisconnectTime = time.Now()
	s.ReconnectCount++
}

// RecordMessageSent 记录消息发送
func (s *MQTTStats) RecordMessageSent() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.MessagesSent++
	s.LastMessageTime = time.Now()
}

// RecordMessageReceived 记录消息接收(带延迟)
func (s *MQTTStats) RecordMessageReceived(latency float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.MessagesReceived++
	s.LastMessageTime = time.Now()

	// 记录延迟样本
	if len(s.LatencySamples) >= s.MaxLatencySamples {
		// 移除最旧的样本
		s.LatencySamples = s.LatencySamples[1:]
	}
	s.LatencySamples = append(s.LatencySamples, latency)
}

// RecordMessageFailed 记录消息失败
func (s *MQTTStats) RecordMessageFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.MessagesFailed++
}

// GetAverageLatency 获取平均延迟
func (s *MQTTStats) GetAverageLatency() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.LatencySamples) == 0 {
		return 0
	}

	var sum float64
	for _, latency := range s.LatencySamples {
		sum += latency
	}
	return sum / float64(len(s.LatencySamples))
}

// GetPacketLossRate 获取丢包率
func (s *MQTTStats) GetPacketLossRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := s.MessagesSent
	if total == 0 {
		return 0
	}

	failed := s.MessagesFailed
	return float64(failed) / float64(total)
}

// GetThroughput 获取吞吐量(消息/秒)
func (s *MQTTStats) GetThroughput() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	duration := time.Since(s.WindowStartTime).Seconds()
	if duration == 0 {
		return 0
	}

	return float64(s.MessagesReceived) / duration
}

// GetSuccessRate 获取成功率
func (s *MQTTStats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := s.MessagesSent
	if total == 0 {
		return 1.0 // 默认100%
	}

	successful := total - s.MessagesFailed
	return float64(successful) / float64(total)
}

// GetReconnectCount 获取重连次数
func (s *MQTTStats) GetReconnectCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ReconnectCount
}

// ResetWindow 重置统计窗口
func (s *MQTTStats) ResetWindow() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.MessagesSent = 0
	s.MessagesReceived = 0
	s.MessagesFailed = 0
	s.LatencySamples = make([]float64, 0, s.MaxLatencySamples)
	s.WindowStartTime = time.Now()
}

// Snapshot 获取统计数据快照
func (s *MQTTStats) Snapshot() MQTTStatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return MQTTStatsSnapshot{
		ConnectTime:        s.ConnectTime,
		LastDisconnectTime: s.LastDisconnectTime,
		ReconnectCount:     s.ReconnectCount,
		MessagesSent:       s.MessagesSent,
		MessagesReceived:   s.MessagesReceived,
		MessagesFailed:     s.MessagesFailed,
		LastMessageTime:    s.LastMessageTime,
		AvgLatency:         s.GetAverageLatency(),
		PacketLossRate:     s.GetPacketLossRate(),
		Throughput:         s.GetThroughput(),
		SuccessRate:        s.GetSuccessRate(),
		WindowDuration:     time.Since(s.WindowStartTime),
	}
}

// MQTTStatsSnapshot 统计数据快照(用于只读访问)
type MQTTStatsSnapshot struct {
	ConnectTime        time.Time
	LastDisconnectTime time.Time
	ReconnectCount     int
	MessagesSent       int64
	MessagesReceived   int64
	MessagesFailed     int64
	LastMessageTime    time.Time
	AvgLatency         float64
	PacketLossRate     float64
	Throughput         float64
	SuccessRate        float64
	WindowDuration     time.Duration
}
