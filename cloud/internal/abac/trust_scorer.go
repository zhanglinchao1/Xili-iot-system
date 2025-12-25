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

// CalculateUserTrustScore 计算用户信任度
func (ts *TrustScorer) CalculateUserTrustScore(attr *UserAttributes) float64 {
	// 如果已有预设TrustScore，直接返回
	if attr.TrustScore > 0 {
		return attr.TrustScore
	}
	score := 100.0

	// 1. 角色基础分
	if attr.Role == "admin" {
		score = 100
	} else {
		score = 80
	}

	// 2. 账号状态
	if attr.Status != "active" {
		return 0
	}

	// 3. 最后登录时间 (超过30天扣分)
	if !attr.LastLoginAt.IsZero() {
		daysSinceLogin := time.Since(attr.LastLoginAt).Hours() / 24
		if daysSinceLogin > 30 {
			penalty := math.Min(daysSinceLogin, 50)
			score -= penalty
		}
	}

	return math.Max(0, math.Min(100, score))
}

// CalculateCabinetTrustScore 计算储能柜信任度
// 优化后的算法：基于脆弱性评分,结合同步状态、许可证、激活状态
func (ts *TrustScorer) CalculateCabinetTrustScore(attr *CabinetAttributes) float64 {
	// 如果已有预设TrustScore，直接返回
	if attr.TrustScore > 0 {
		return attr.TrustScore
	}

	// 基础分使用脆弱性评分(由Edge端评估,更全面)
	// 如果没有脆弱性评分,使用100分作为基础
	score := 100.0
	if attr.VulnerabilityScore > 0 {
		score = attr.VulnerabilityScore
	}

	// 1. 激活状态检查 (未激活扣30分)
	if attr.ActivationStatus != "activated" {
		score -= 30
	}

	// 2. 最后同步时间影响 (最多扣20分)
	if !attr.LastSyncAt.IsZero() {
		hoursSinceSync := time.Since(attr.LastSyncAt).Hours()
		if hoursSinceSync > 1 {
			penalty := math.Min(hoursSinceSync*2, 20) // 每超1小时扣2分,最多扣20分
			score -= penalty
		}
	} else {
		score -= 15 // 从未同步扣15分
	}

	// 3. 许可证状态 (无有效许可证扣15分)
	if !attr.HasValidLicense {
		score -= 15
	}

	// 4. 状态检查 (不同状态不同扣分)
	switch attr.Status {
	case "online":
		// 不扣分
	case "maintenance":
		score -= 5
	case "offline":
		score -= 20
	case "error":
		score -= 25
	}

	// 确保分数在0-100范围内
	return math.Max(0, math.Min(100, score))
}

// CalculateDeviceTrustScore 计算传感器设备信任度
func (ts *TrustScorer) CalculateDeviceTrustScore(attr *DeviceAttributes) float64 {
	score := 100.0

	// 1. 设备状态影响
	switch attr.Status {
	case "active":
		// 不扣分
	case "inactive":
		score -= 40
	case "error":
		score -= 60
	}

	// 2. 数据质量影响 (如果有数据质量指标)
	// 注意: 需要先检查Quality字段是否有效
	qualityScore := float64(attr.Quality) // 假设Quality范围0-100
	// 数据质量低于60分时额外扣分
	if qualityScore < 60 {
		penalty := (60 - qualityScore) * 0.5 // 低于60分,每差1分扣0.5分
		score -= penalty
	}

	// 3. 最后读取时间影响 (超过5分钟扣分)
	if !attr.LastReadingAt.IsZero() {
		minutesSinceReading := time.Since(attr.LastReadingAt).Minutes()
		if minutesSinceReading > 5 {
			penalty := math.Min(minutesSinceReading, 30) // 最多扣30分
			score -= penalty
		}
	} else {
		score -= 30 // 从未读取扣30分
	}

	// 确保分数在0-100范围内
	return math.Max(0, math.Min(100, score))
}

// CalculateTrustScore 统一的信任度计算接口
// 如果属性中已有非零TrustScore，则直接返回（用于测试API）
func (ts *TrustScorer) CalculateTrustScore(attr Attributes) float64 {
	// 如果已经有预设的TrustScore，直接使用
	if attr.GetTrustScore() > 0 {
		return attr.GetTrustScore()
	}

	switch v := attr.(type) {
	case *UserAttributes:
		return ts.CalculateUserTrustScore(v)
	case *CabinetAttributes:
		return ts.CalculateCabinetTrustScore(v)
	case *DeviceAttributes:
		return ts.CalculateDeviceTrustScore(v)
	default:
		return 0
	}
}
