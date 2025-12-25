package services

import (
	"sort"
	"sync"
	"time"

	"cloud-system/internal/models"
)

// TrafficService 存储并提供流量统计
type TrafficService struct {
	mu         sync.RWMutex
	latest     map[string]*models.TrafficStat
	history    map[string][]models.TrafficSample
	maxHistory time.Duration
}

// NewTrafficService 创建实例
func NewTrafficService() *TrafficService {
	return &TrafficService{
		latest:     make(map[string]*models.TrafficStat),
		history:    make(map[string][]models.TrafficSample),
		maxHistory: 24 * time.Hour,
	}
}

// Update 更新最新统计
func (s *TrafficService) Update(msg *models.TrafficMQTTMessage) {
	if msg == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stat := &models.TrafficStat{
		CabinetID:         msg.CabinetID,
		Timestamp:         msg.Timestamp,
		FlowKbps:          msg.ThroughputKbps,
		LatencyMs:         msg.LatencyMs,
		PacketLossRate:    msg.PacketLossRate,
		MQTTSuccessRate:   msg.MQTTSuccessRate,
		ReconnectionCount: msg.ReconnectionCount,
		RiskLevel:         msg.RiskLevel,
		BaselineDeviation: "--",
	}

	s.latest[msg.CabinetID] = stat

	history := append(s.history[msg.CabinetID], models.TrafficSample{
		CabinetID: msg.CabinetID,
		Timestamp: msg.Timestamp,
		FlowKbps:  msg.ThroughputKbps,
	})

	cutoff := time.Now().Add(-s.maxHistory)
	idx := 0
	for idx < len(history) && history[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx > 0 {
		history = history[idx:]
	}
	if len(history) > 2000 {
		history = history[len(history)-2000:]
	}

	s.history[msg.CabinetID] = history
}

// List 返回最新统计
func (s *TrafficService) List() []*models.TrafficStat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*models.TrafficStat, 0, len(s.latest))
	for _, stat := range s.latest {
		copy := *stat
		list = append(list, &copy)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CabinetID < list[j].CabinetID
	})
	return list
}

// Summary 返回概览
func (s *TrafficService) Summary(rangeDuration time.Duration) (models.TrafficSummary, []string, []float64, []float64, []models.ProtocolSlice) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := models.TrafficSummary{}
	var totalFlow float64
	var totalLatency float64
	var totalPacketLoss float64
	var totalSuccess float64

	for _, stat := range s.latest {
		summary.CabinetCount++
		totalFlow += stat.FlowKbps
		totalLatency += stat.LatencyMs
		totalPacketLoss += stat.PacketLossRate
		totalSuccess += stat.MQTTSuccessRate
		if stat.RiskLevel != "healthy" && stat.RiskLevel != "low" {
			summary.AnomalyCount++
		}
	}

	if summary.CabinetCount > 0 {
		summary.AvgLatency = totalLatency / float64(summary.CabinetCount)
		summary.AvgPacketLoss = totalPacketLoss / float64(summary.CabinetCount)
		summary.AvgMQTTSucess = totalSuccess / float64(summary.CabinetCount)
	}
	summary.TotalFlow = totalFlow

	labels, totals, avg := s.aggregateTrend(rangeDuration)
	protocol := s.protocolSlices()
	return summary, labels, totals, avg, protocol
}

// GetDetail 返回指定柜子详情
func (s *TrafficService) GetDetail(cabinetID string, rangeDuration time.Duration) (*models.TrafficStat, []models.TrafficSample) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stat, ok := s.latest[cabinetID]
	if !ok {
		return nil, nil
	}

	history := s.history[cabinetID]
	if rangeDuration > 0 {
		cutoff := time.Now().Add(-rangeDuration)
		idx := 0
		for idx < len(history) && history[idx].Timestamp.Before(cutoff) {
			idx++
		}
		if idx > 0 {
			history = history[idx:]
		}
	}

	result := make([]models.TrafficSample, len(history))
	copy(result, history)
	return stat, result
}

func (s *TrafficService) aggregateTrend(rangeDuration time.Duration) ([]string, []float64, []float64) {
	if rangeDuration <= 0 {
		rangeDuration = time.Hour
	}
	bucketMinutes := time.Duration(5) * time.Minute
	if rangeDuration > 6*time.Hour {
		bucketMinutes = 30 * time.Minute
	}
	if rangeDuration > 24*time.Hour {
		bucketMinutes = time.Hour
	}

	end := time.Now()
	start := end.Add(-rangeDuration)
	bucketCount := int(rangeDuration / bucketMinutes)
	if bucketCount < 1 {
		bucketCount = 1
	}

	labels := make([]string, bucketCount)
	totals := make([]float64, bucketCount)
	counts := make([]int, bucketCount)

	var timeFormat string
	if rangeDuration > 24*time.Hour {
		timeFormat = "01-02 15:04"
	} else {
		timeFormat = "15:04"
	}

	for i := 0; i < bucketCount; i++ {
		t := start.Add(time.Duration(i) * bucketMinutes)
		labels[i] = t.Format(timeFormat)
	}

	for _, hist := range s.history {
		for _, sample := range hist {
			if sample.Timestamp.Before(start) {
				continue
			}
			if sample.Timestamp.After(end) {
				continue
			}
			idx := int(sample.Timestamp.Sub(start) / bucketMinutes)
			if idx >= bucketCount {
				idx = bucketCount - 1
			}
			totals[idx] += sample.FlowKbps / 1024 // 转换为MB/s
			counts[idx]++
		}
	}

	avg := make([]float64, bucketCount)
	runTotal := 0.0
	runCount := 0
	for i := 0; i < bucketCount; i++ {
		if counts[i] > 0 {
			runTotal += totals[i]
			runCount++
		}
		if runCount > 0 {
			avg[i] = runTotal / float64(runCount)
		}
	}

	return labels, totals, avg
}

func (s *TrafficService) protocolSlices() []models.ProtocolSlice {
	mqttSum := 0.0
	httpsSum := 0.0

	for _, stat := range s.latest {
		mqttShare := stat.FlowKbps * stat.MQTTSuccessRate
		mqttSum += mqttShare
		httpsSum += stat.FlowKbps - mqttShare
	}

	if mqttSum == 0 && httpsSum == 0 {
		return []models.ProtocolSlice{{Name: "MQTT", Value: 1}}
	}

	return []models.ProtocolSlice{
		{Name: "MQTT", Value: mqttSum},
		{Name: "HTTPS", Value: httpsSum},
	}
}
