package models

// Cabinet状态常量定义
// 与数据库约束保持一致: CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance'))
const (
	// CabinetStatusPending 待激活状态
	// 储能柜已预注册，等待Edge端首次激活
	CabinetStatusPending = "pending"

	// CabinetStatusActive 激活且同步中
	// Edge端已激活且正在定期同步数据（最近10分钟内有同步）
	CabinetStatusActive = "active"

	// CabinetStatusInactive 已停用
	// 管理员手动停用或长期未使用（超过30天未同步）
	CabinetStatusInactive = "inactive"

	// CabinetStatusOffline 离线
	// 超过10分钟未同步，但未达到停用阈值
	CabinetStatusOffline = "offline"

	// CabinetStatusMaintenance 维护中
	// 计划内维护，暂时不接收同步数据
	CabinetStatusMaintenance = "maintenance"
)

// ValidCabinetStatuses 所有有效的Cabinet状态
// 必须与数据库CHECK约束保持一致
var ValidCabinetStatuses = []string{
	CabinetStatusPending,
	CabinetStatusActive,
	CabinetStatusInactive,
	CabinetStatusOffline,
	CabinetStatusMaintenance,
}

// IsValidCabinetStatus 检查状态是否有效
func IsValidCabinetStatus(status string) bool {
	for _, s := range ValidCabinetStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GetCabinetStatusDescription 获取状态描述（用于前端显示）
func GetCabinetStatusDescription(status string) string {
	switch status {
	case CabinetStatusPending:
		return "待激活"
	case CabinetStatusActive:
		return "在线同步中"
	case CabinetStatusInactive:
		return "已停用"
	case CabinetStatusOffline:
		return "离线"
	case CabinetStatusMaintenance:
		return "维护中"
	default:
		return "未知状态"
	}
}

// Cabinet激活状态常量
const (
	// ActivationStatusPending 待激活
	ActivationStatusPending = "pending"

	// ActivationStatusActivated 已激活
	ActivationStatusActivated = "activated"
)

// ValidActivationStatuses 所有有效的激活状态
var ValidActivationStatuses = []string{
	ActivationStatusPending,
	ActivationStatusActivated,
}

// IsValidActivationStatus 检查激活状态是否有效
func IsValidActivationStatus(status string) bool {
	for _, s := range ValidActivationStatuses {
		if s == status {
			return true
		}
	}
	return false
}
