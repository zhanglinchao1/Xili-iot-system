package models

import (
	"time"
)

// Command 命令模型
type Command struct {
	CommandID   string     `json:"command_id" db:"command_id"`
	CabinetID   string     `json:"cabinet_id" db:"cabinet_id"`
	CommandType string     `json:"command_type" db:"command_type"` // config_update, license_push, license_revoke, restart
	Payload     string     `json:"payload" db:"payload"`           // JSON格式的命令内容
	Status      string     `json:"status" db:"status"`             // pending, sent, success, failed, timeout
	Result      *string    `json:"result,omitempty" db:"result"`   // Edge端返回的结果
	SentAt      *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedBy   string     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// SendCommandRequest 发送命令请求
type SendCommandRequest struct {
	CommandType string                 `json:"command_type" binding:"required"`
	Payload     map[string]interface{} `json:"payload" binding:"required"`
}

// CommandAckRequest Edge端命令回执
type CommandAckRequest struct {
	Status  string `json:"status" binding:"required,oneof=success failed"`
	Message string `json:"message,omitempty"`
}

// CommandListFilter 命令列表过滤参数
type CommandListFilter struct {
	CabinetID   *string `form:"cabinet_id"`
	CommandType *string `form:"command_type"`
	Status      *string `form:"status"`
	Page        int     `form:"page"`
	PageSize    int     `form:"page_size"`
}

// ValidCommandTypes 有效的命令类型列表
// 根据 senddata.md 规范定义的命令类型
var ValidCommandTypes = []string{
	// 配置类命令 (config)
	"config_update",       // 配置更新
	"config_push",         // 配置推送
	"config",              // 通用配置命令
	
	// 许可证类命令 (license)
	"license_push",        // 许可证推送
	"license_revoke",      // 许可证吊销
	"license_update",      // 许可证更新
	"license",             // 通用许可证命令
	
	// 查询类命令 (query)
	"query_status",        // 查询状态
	"query_logs",          // 查询日志
	"query",               // 通用查询命令
	
	// 控制类命令 (control)
	"restart",             // 重启服务
	"mode_switch",         // 切换运行模式
	"cache_clear",         // 清理缓存
	"resolve_alert",       // 解决告警
	"control",             // 通用控制命令
}

// ValidCommandStatuses 有效的命令状态列表
var ValidCommandStatuses = []string{
	"pending",
	"sent",
	"success",
	"failed",
	"timeout",
}

// IsValidCommandType 检查命令类型是否有效
func IsValidCommandType(commandType string) bool {
	for _, valid := range ValidCommandTypes {
		if commandType == valid {
			return true
		}
	}
	return false
}

// IsValidCommandStatus 检查命令状态是否有效
func IsValidCommandStatus(status string) bool {
	for _, valid := range ValidCommandStatuses {
		if status == valid {
			return true
		}
	}
	return false
}
