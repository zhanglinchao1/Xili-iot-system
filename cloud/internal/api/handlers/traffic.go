package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TrafficHandler struct {
	trafficService *services.TrafficService
	cabinetService services.CabinetService
	trafficRepo    repository.TrafficRepository
}

func NewTrafficHandler(ts *services.TrafficService, cs services.CabinetService, tr repository.TrafficRepository) *TrafficHandler {
	return &TrafficHandler{
		trafficService: ts,
		cabinetService: cs,
		trafficRepo:    tr,
	}
}

func (h *TrafficHandler) GetSummary(c *gin.Context) {
	duration := parseRange(c.DefaultQuery("range", "1h"))
	summary, labels, totals, avg, protocol := h.trafficService.Summary(duration)

	// 如果内存中没有实时数据，尝试从持久化存储中加载
	if summary.CabinetCount == 0 && h.trafficRepo != nil {
		repoSummary, repoLabels, repoTotals, repoAvg, repoProtocol, err := h.buildSummaryFromRepo(c.Request.Context(), duration)
		if err != nil {
			utils.Warn("加载持久化流量统计失败", zap.Error(err))
		} else if repoSummary.CabinetCount > 0 {
			summary = repoSummary
			labels = repoLabels
			totals = repoTotals
			avg = repoAvg
			protocol = repoProtocol
		}
	}

	utils.Success(c, gin.H{
		"summary": summary,
		"trend": gin.H{
			"labels":  labels,
			"total":   totals,
			"average": avg,
		},
		"protocol": protocol,
	})
}

func (h *TrafficHandler) ListCabinets(c *gin.Context) {
	ctx := c.Request.Context()
	stats := h.mergeStats(ctx, h.trafficService.List())
	if len(stats) == 0 {
		utils.Success(c, []models.TrafficStat{})
		return
	}

	// 位置信息已经在 mergeStats 中通过 LEFT JOIN 获取，无需再次查询
	// 只需要确保基线对比字段正确设置
	for _, stat := range stats {
		// 设置基线对比（使用MQTT成功率）
		stat.BaselineDeviation = fmt.Sprintf("%.1f%%", stat.MQTTSuccessRate*100)
	}
	utils.Success(c, stats)
}

func (h *TrafficHandler) GetCabinetDetail(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	duration := parseRange(c.DefaultQuery("range", "1h"))
	ctx := c.Request.Context()

	stat, history := h.trafficService.GetDetail(cabinetID, duration)
	if stat == nil && h.trafficRepo != nil {
		var err error
		stat, err = h.trafficRepo.GetLatestStat(ctx, cabinetID)
		if err != nil {
			utils.Warn("查询持久化流量数据失败", zap.String("cabinet_id", cabinetID), zap.Error(err))
		}
	}

	if len(history) == 0 && h.trafficRepo != nil {
		start := time.Now().Add(-duration)
		repoHistory, err := h.trafficRepo.GetHistory(ctx, cabinetID, start, 288)
		if err != nil {
			utils.Warn("查询流量历史失败", zap.String("cabinet_id", cabinetID), zap.Error(err))
		} else {
			history = repoHistory
		}
	}

	if stat == nil {
		utils.ErrorResponse(c, http.StatusNotFound, errors.New(errors.ErrNotFound, "未找到流量数据"))
		return
	}

	cabinet, err := h.cabinetService.GetCabinet(c.Request.Context(), cabinetID)
	if err == nil && cabinet != nil && cabinet.Location != nil {
		stat.Location = *cabinet.Location
	}

	utils.Success(c, gin.H{
		"stat":    stat,
		"history": history,
		"protocol": []models.ProtocolSlice{
			{Name: "MQTT", Value: stat.FlowKbps * stat.MQTTSuccessRate},
			{Name: "HTTPS", Value: stat.FlowKbps * (1 - stat.MQTTSuccessRate)},
		},
	})
}

func parseRange(value string) time.Duration {
	switch value {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func (h *TrafficHandler) mergeStats(ctx context.Context, live []*models.TrafficStat) []*models.TrafficStat {
	statMap := make(map[string]*models.TrafficStat, len(live))
	for _, stat := range live {
		statMap[stat.CabinetID] = stat
	}

	if h.trafficRepo != nil {
		persisted, err := h.trafficRepo.ListLatestStats(ctx)
		if err != nil {
			utils.Warn("加载持久化流量数据失败", zap.Error(err))
		} else {
			for _, stat := range persisted {
				if stat == nil {
					continue
				}
				if existing, ok := statMap[stat.CabinetID]; ok {
					// 如果时间戳更新，使用新的数据
					if stat.Timestamp.After(existing.Timestamp) {
						statMap[stat.CabinetID] = stat
					} else {
						// 如果时间戳相同或更旧，但位置信息为空，使用持久化数据的位置信息
						if existing.Location == "" && stat.Location != "" {
							existing.Location = stat.Location
						}
					}
				} else {
					statMap[stat.CabinetID] = stat
				}
			}
		}
	}

	list := make([]*models.TrafficStat, 0, len(statMap))
	for _, stat := range statMap {
		list = append(list, stat)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CabinetID < list[j].CabinetID
	})
	return list
}

func (h *TrafficHandler) buildSummaryFromRepo(ctx context.Context, rangeDuration time.Duration) (models.TrafficSummary, []string, []float64, []float64, []models.ProtocolSlice, error) {
	summary := models.TrafficSummary{}

	// 获取时间范围内的起始时间
	end := time.Now()
	start := end.Add(-rangeDuration)

	// 获取所有储能柜的最新统计（用于获取储能柜列表）
	stats, err := h.trafficRepo.ListLatestStats(ctx)
	if err != nil {
		return summary, nil, nil, nil, nil, err
	}
	if len(stats) == 0 {
		return summary, nil, nil, nil, nil, nil
	}

	// 根据时间范围内的历史数据计算统计数据
	var totalFlow float64
	var cabinetLatencyMap = make(map[string][]float64) // 用于计算平均延迟
	var cabinetPacketLossMap = make(map[string][]float64)
	var cabinetSuccessMap = make(map[string][]float64)

	const defaultLimit = 288
	visited := make(map[string]struct{}, len(stats))
	for _, stat := range stats {
		if stat == nil || stat.CabinetID == "" {
			continue
		}
		if _, ok := visited[stat.CabinetID]; ok {
			continue
		}
		visited[stat.CabinetID] = struct{}{}

		// 获取时间范围内的历史数据
		history, err := h.trafficRepo.GetHistory(ctx, stat.CabinetID, start, defaultLimit)
		if err != nil {
			utils.Warn("加载储能柜历史流量失败",
				zap.String("cabinet_id", stat.CabinetID),
				zap.Error(err),
			)
			// 如果无法获取历史数据，使用最新统计数据
			summary.CabinetCount++
			totalFlow += stat.FlowKbps
			cabinetLatencyMap[stat.CabinetID] = []float64{stat.LatencyMs}
			cabinetPacketLossMap[stat.CabinetID] = []float64{stat.PacketLossRate}
			cabinetSuccessMap[stat.CabinetID] = []float64{stat.MQTTSuccessRate}
			if stat.RiskLevel != "healthy" && stat.RiskLevel != "low" {
				summary.AnomalyCount++
			}
			continue
		}

		// 过滤时间范围内的数据
		var validSamples []models.TrafficSample
		for _, sample := range history {
			if sample.Timestamp.After(start) && !sample.Timestamp.After(end) {
				validSamples = append(validSamples, sample)
			}
		}

		if len(validSamples) == 0 {
			// 如果没有有效数据，使用最新统计数据
			summary.CabinetCount++
			totalFlow += stat.FlowKbps
			cabinetLatencyMap[stat.CabinetID] = []float64{stat.LatencyMs}
			cabinetPacketLossMap[stat.CabinetID] = []float64{stat.PacketLossRate}
			cabinetSuccessMap[stat.CabinetID] = []float64{stat.MQTTSuccessRate}
			if stat.RiskLevel != "healthy" && stat.RiskLevel != "low" {
				summary.AnomalyCount++
			}
			continue
		}

		// 计算该储能柜在时间范围内的统计数据
		var cabinetTotalFlow float64
		for _, sample := range validSamples {
			cabinetTotalFlow += sample.FlowKbps
		}

		// 使用最新统计数据中的延迟、丢包率、成功率（因为历史数据中没有这些字段）
		cabinetLatencyMap[stat.CabinetID] = []float64{stat.LatencyMs}
		cabinetPacketLossMap[stat.CabinetID] = []float64{stat.PacketLossRate}
		cabinetSuccessMap[stat.CabinetID] = []float64{stat.MQTTSuccessRate}

		summary.CabinetCount++
		totalFlow += cabinetTotalFlow
		if stat.RiskLevel != "healthy" && stat.RiskLevel != "low" {
			summary.AnomalyCount++
		}
	}

	// 计算平均值
	if summary.CabinetCount > 0 {
		count := float64(summary.CabinetCount)
		// 计算平均延迟
		var sumLatency float64
		var sumPacketLoss float64
		var sumSuccess float64
		for _, latencies := range cabinetLatencyMap {
			for _, lat := range latencies {
				sumLatency += lat
			}
		}
		for _, losses := range cabinetPacketLossMap {
			for _, loss := range losses {
				sumPacketLoss += loss
			}
		}
		for _, successes := range cabinetSuccessMap {
			for _, success := range successes {
				sumSuccess += success
			}
		}
		summary.AvgLatency = sumLatency / count
		summary.AvgPacketLoss = sumPacketLoss / count
		summary.AvgMQTTSucess = sumSuccess / count
	}
	summary.TotalFlow = totalFlow

	labels, totals, avg, err := h.buildTrendFromRepo(ctx, rangeDuration, stats)
	if err != nil {
		return summary, nil, nil, nil, nil, err
	}

	// 根据时间范围内的数据计算协议分布
	protocol := calculateProtocolFromStatsWithRange(ctx, h.trafficRepo, stats, start, end)
	return summary, labels, totals, avg, protocol, nil
}

func (h *TrafficHandler) buildTrendFromRepo(ctx context.Context, rangeDuration time.Duration, stats []*models.TrafficStat) ([]string, []float64, []float64, error) {
	if rangeDuration <= 0 {
		rangeDuration = time.Hour
	}

	bucketMinutes := 5 * time.Minute
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
	
	// 根据时间范围选择不同的时间标签格式
	var timeFormat string
	if rangeDuration > 24*time.Hour {
		// 跨天显示日期和时间
		timeFormat = "01-02 15:04"
	} else {
		// 单天只显示时间
		timeFormat = "15:04"
	}
	
	for i := 0; i < bucketCount; i++ {
		t := start.Add(time.Duration(i) * bucketMinutes)
		labels[i] = t.Format(timeFormat)
	}

	const defaultLimit = 288
	visited := make(map[string]struct{}, len(stats))
	for _, stat := range stats {
		if stat == nil || stat.CabinetID == "" {
			continue
		}
		if _, ok := visited[stat.CabinetID]; ok {
			continue
		}
		visited[stat.CabinetID] = struct{}{}

		history, err := h.trafficRepo.GetHistory(ctx, stat.CabinetID, start, defaultLimit)
		if err != nil {
			utils.Warn("加载储能柜历史流量失败",
				zap.String("cabinet_id", stat.CabinetID),
				zap.Error(err),
			)
			continue
		}

		for _, sample := range history {
			ts := sample.Timestamp
			if ts.Before(start) || ts.After(end) {
				continue
			}
			idx := int(ts.Sub(start) / bucketMinutes)
			if idx < 0 {
				idx = 0
			} else if idx >= bucketCount {
				idx = bucketCount - 1
			}

			// 转换为MB/s以匹配前端显示
			totals[idx] += sample.FlowKbps / 1024
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

	return labels, totals, avg, nil
}

func calculateProtocolFromStats(stats []*models.TrafficStat) []models.ProtocolSlice {
	mqttSum := 0.0
	httpsSum := 0.0

	for _, stat := range stats {
		if stat == nil {
			continue
		}
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

// calculateProtocolFromStatsWithRange 根据时间范围内的历史数据计算协议分布
func calculateProtocolFromStatsWithRange(ctx context.Context, repo repository.TrafficRepository, stats []*models.TrafficStat, start, end time.Time) []models.ProtocolSlice {
	mqttSum := 0.0
	httpsSum := 0.0

	const defaultLimit = 288
	visited := make(map[string]struct{}, len(stats))
	for _, stat := range stats {
		if stat == nil || stat.CabinetID == "" {
			continue
		}
		if _, ok := visited[stat.CabinetID]; ok {
			continue
		}
		visited[stat.CabinetID] = struct{}{}

		// 获取时间范围内的历史数据
		history, err := repo.GetHistory(ctx, stat.CabinetID, start, defaultLimit)
		if err != nil {
			// 如果无法获取历史数据，使用最新统计数据
			mqttShare := stat.FlowKbps * stat.MQTTSuccessRate
			mqttSum += mqttShare
			httpsSum += stat.FlowKbps - mqttShare
			continue
		}

		// 计算时间范围内的总流量
		var totalFlow float64
		for _, sample := range history {
			if sample.Timestamp.After(start) && !sample.Timestamp.After(end) {
				totalFlow += sample.FlowKbps
			}
		}

		// 如果没有历史数据，使用最新统计数据
		if totalFlow == 0 {
			totalFlow = stat.FlowKbps
		}

		// 使用最新统计数据中的MQTT成功率计算协议分布
		mqttShare := totalFlow * stat.MQTTSuccessRate
		mqttSum += mqttShare
		httpsSum += totalFlow - mqttShare
	}

	if mqttSum == 0 && httpsSum == 0 {
		return []models.ProtocolSlice{{Name: "MQTT", Value: 1}}
	}

	return []models.ProtocolSlice{
		{Name: "MQTT", Value: mqttSum},
		{Name: "HTTPS", Value: httpsSum},
	}
}
