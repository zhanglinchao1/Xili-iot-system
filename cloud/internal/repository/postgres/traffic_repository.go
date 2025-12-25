package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud-system/internal/models"
	"cloud-system/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TrafficRepository 基于PostgreSQL的流量数据实现
type TrafficRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewTrafficRepository 创建实例
func NewTrafficRepository(pool *pgxpool.Pool, logger *zap.Logger) repository.TrafficRepository {
	return &TrafficRepository{
		pool:   pool,
		logger: logger,
	}
}

type transmissionMetrics struct {
	LatencyAvg        float64 `json:"latency_avg"`
	PacketLossRate    float64 `json:"packet_loss_rate"`
	Throughput        float64 `json:"throughput"`
	MQTTSuccessRate   float64 `json:"mqtt_success_rate"`
	ReconnectionCount int     `json:"reconnection_count"`
}

// ListLatestStats 返回每个储能柜最近一次的流量统计
func (r *TrafficRepository) ListLatestStats(ctx context.Context) ([]*models.TrafficStat, error) {
	query := `
		WITH ranked AS (
			SELECT
				va.cabinet_id,
				va.timestamp,
				va.risk_level,
				va.transmission_metrics,
				ROW_NUMBER() OVER (PARTITION BY va.cabinet_id ORDER BY va.timestamp DESC) AS rn
			FROM vulnerability_assessments va
			WHERE va.transmission_metrics IS NOT NULL
		)
		SELECT r.cabinet_id, r.timestamp, r.risk_level, r.transmission_metrics,
		       c.location
		FROM ranked r
		LEFT JOIN cabinets c ON r.cabinet_id = c.cabinet_id
		WHERE r.rn = 1
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.TrafficStat
	for rows.Next() {
		stat, err := r.scanStat(rows)
		if err != nil {
			r.logger.Warn("解析流量统计失败", zap.Error(err))
			continue
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// GetLatestStat 返回指定储能柜最近一次流量统计
func (r *TrafficRepository) GetLatestStat(ctx context.Context, cabinetID string) (*models.TrafficStat, error) {
	query := `
		SELECT va.cabinet_id, va.timestamp, va.risk_level, va.transmission_metrics,
		       c.location
		FROM vulnerability_assessments va
		LEFT JOIN cabinets c ON va.cabinet_id = c.cabinet_id
		WHERE va.cabinet_id = $1 AND va.transmission_metrics IS NOT NULL
		ORDER BY va.timestamp DESC
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, cabinetID)
	return r.scanStat(row)
}

// GetHistory 返回时间范围内的流量样本
func (r *TrafficRepository) GetHistory(ctx context.Context, cabinetID string, start time.Time, limit int) ([]models.TrafficSample, error) {
	if limit <= 0 {
		limit = 288 // 默认最多返回一天内5分钟粒度的数据
	}

	query := `
		SELECT timestamp, transmission_metrics
		FROM vulnerability_assessments
		WHERE cabinet_id = $1
			AND transmission_metrics IS NOT NULL
			AND timestamp >= $2
		ORDER BY timestamp ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, cabinetID, start, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []models.TrafficSample
	for rows.Next() {
		var ts time.Time
		var payload *string
		if err := rows.Scan(&ts, &payload); err != nil {
			return nil, err
		}
		metrics, err := parseTransmissionMetrics(payload)
		if err != nil {
			r.logger.Warn("解析历史流量数据失败", zap.Error(err))
			continue
		}
		samples = append(samples, models.TrafficSample{
			CabinetID: cabinetID,
			Timestamp: ts,
			FlowKbps:  metrics.Throughput,
		})
	}

	return samples, rows.Err()
}

func (r *TrafficRepository) scanStat(row pgx.Row) (*models.TrafficStat, error) {
	var (
		cabinetID string
		ts        time.Time
		riskLevel string
		payload   *string
		location  *string
	)

	if err := row.Scan(&cabinetID, &ts, &riskLevel, &payload, &location); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	metrics, err := parseTransmissionMetrics(payload)
	if err != nil {
		return nil, err
	}

	stat := &models.TrafficStat{
		CabinetID:         cabinetID,
		Timestamp:         ts,
		FlowKbps:          metrics.Throughput,
		LatencyMs:         metrics.LatencyAvg,
		PacketLossRate:    metrics.PacketLossRate,
		MQTTSuccessRate:   metrics.MQTTSuccessRate,
		ReconnectionCount: metrics.ReconnectionCount,
		RiskLevel:         riskLevel,
		BaselineDeviation: fmt.Sprintf("%.1f%%", metrics.MQTTSuccessRate*100),
	}

	// 设置位置信息（如果存在）
	if location != nil {
		stat.Location = *location
	} else {
		stat.Location = ""
	}

	return stat, nil
}

func parseTransmissionMetrics(raw *string) (*transmissionMetrics, error) {
	if raw == nil || *raw == "" {
		return nil, fmt.Errorf("transmission metrics is empty")
	}

	var metrics transmissionMetrics
	if err := json.Unmarshal([]byte(*raw), &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}
