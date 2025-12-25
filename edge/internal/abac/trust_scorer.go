package abac

import (
	"math"
	"time"
)

// TrustScorer 信任度评估器
type TrustScorer struct{}

// NewTrustScorer 创建信任度评估器
func NewTrustScorer() *TrustScorer {
	return &TrustScorer{}
}

// CalculateDeviceTrustScore 计算传感器设备信任度
func (ts *TrustScorer) CalculateDeviceTrustScore(attr *DeviceAttributes) float64 {
	// 如果已有预设TrustScore，直接返回
	if attr.TrustScore > 0 {
		return attr.TrustScore
	}

	score := 100.0

	// 1. 设备状态影响
	switch attr.Status {
	case "active":
		// 不扣分
	case "inactive":
		score -= 40
	case "error":
		score -= 60
	default:
		score -= 20
	}

	// 2. 数据质量影响
	qualityScore := float64(attr.Quality)
	if qualityScore < 60 {
		penalty := (60 - qualityScore) * 0.5
		score -= penalty
	}

	// 3. 最后读取时间影响 (超过5分钟扣分)
	if !attr.LastReadingAt.IsZero() {
		minutesSinceReading := time.Since(attr.LastReadingAt).Minutes()
		if minutesSinceReading > 5 {
			penalty := math.Min(minutesSinceReading, 30)
			score -= penalty
		}
	} else {
		score -= 30
	}

	return math.Max(0, math.Min(100, score))
}

// CalculateTrustScore 统一的信任度计算接口
func (ts *TrustScorer) CalculateTrustScore(attr Attributes) float64 {
	if attr.GetTrustScore() > 0 {
		return attr.GetTrustScore()
	}

	switch v := attr.(type) {
	case *DeviceAttributes:
		return ts.CalculateDeviceTrustScore(v)
	default:
		return 0
	}
}
