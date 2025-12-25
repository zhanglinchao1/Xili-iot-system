package models

import "time"

// TrafficMQTTMessage Edge通过MQTT推送的流量数据
type TrafficMQTTMessage struct {
	CabinetID         string    `json:"cabinet_id"`
	Timestamp         time.Time `json:"timestamp"`
	ThroughputKbps    float64   `json:"throughput_kbps"`
	LatencyMs         float64   `json:"latency_ms"`
	PacketLossRate    float64   `json:"packet_loss_rate"`
	MQTTSuccessRate   float64   `json:"mqtt_success_rate"`
	ReconnectionCount int       `json:"reconnection_count"`
	RiskLevel         string    `json:"risk_level"`
}

// TrafficSample 用于历史趋势
type TrafficSample struct {
	CabinetID string    `json:"cabinet_id"`
	Timestamp time.Time `json:"timestamp"`
	FlowKbps  float64   `json:"flow_kbps"`
}

// TrafficStat 供API返回的最新统计
type TrafficStat struct {
	CabinetID         string    `json:"cabinet_id"`
	Location          string    `json:"location"`
	Timestamp         time.Time `json:"timestamp"`
	FlowKbps          float64   `json:"flow_kbps"`
	LatencyMs         float64   `json:"latency_ms"`
	PacketLossRate    float64   `json:"packet_loss_rate"`
	MQTTSuccessRate   float64   `json:"mqtt_success_rate"`
	ReconnectionCount int       `json:"reconnection_count"`
	RiskLevel         string    `json:"risk_level"`
	BaselineDeviation string    `json:"baseline_deviation"`
}

// TrafficSummary 概览数据
type TrafficSummary struct {
	CabinetCount  int     `json:"cabinet_count"`
	TotalFlow     float64 `json:"total_flow_kbps"`
	AvgLatency    float64 `json:"avg_latency_ms"`
	AvgPacketLoss float64 `json:"avg_packet_loss"`
	AvgMQTTSucess float64 `json:"avg_mqtt_success"`
	AnomalyCount  int     `json:"anomaly_count"`
}

// ProtocolSlice 协议分布
type ProtocolSlice struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
