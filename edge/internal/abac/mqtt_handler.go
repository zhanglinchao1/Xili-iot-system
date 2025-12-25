package abac

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// SyncAlertCallback 同步告警回调函数
type SyncAlertCallback func(alertType string, message string, details map[string]interface{})

// MQTTHandler 处理Cloud下发的ABAC策略
type MQTTHandler struct {
	repo          Repository
	cabinetID     string
	syncFailCount int                                      // 连续同步失败次数
	alertCallback SyncAlertCallback                        // 告警回调
	publishFunc   func(topic string, payload []byte) error // MQTT发布函数
}

// NewMQTTHandler 创建MQTT处理器
func NewMQTTHandler(repo Repository, cabinetID string) *MQTTHandler {
	return &MQTTHandler{
		repo:      repo,
		cabinetID: cabinetID,
	}
}

// SetPublishFunc 设置MQTT发布函数
func (h *MQTTHandler) SetPublishFunc(publishFunc func(string, []byte) error) {
	h.publishFunc = publishFunc
}

// SetAlertCallback 设置告警回调
func (h *MQTTHandler) SetAlertCallback(callback SyncAlertCallback) {
	h.alertCallback = callback
}

// triggerAlert 触发告警
func (h *MQTTHandler) triggerAlert(alertType, message string, details map[string]interface{}) {
	if h.alertCallback != nil {
		h.alertCallback(alertType, message, details)
	}
	log.Printf("[ABAC-ALERT] %s: %s", alertType, message)
}

// GetPolicyTopic 获取策略订阅主题
func (h *MQTTHandler) GetPolicyTopic() string {
	return "cloud/cabinet/" + h.cabinetID + "/policy/sync"
}

// HandlePolicySync 处理策略同步消息
func (h *MQTTHandler) HandlePolicySync(payload []byte) error {
	var msg PolicySyncMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("[ABAC] 解析策略同步消息失败: %v", err)
		return err
	}

	log.Printf("[ABAC] 收到策略同步消息: action=%s, policies=%d", msg.Action, len(msg.Policies))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch msg.Action {
	case "sync":
		return h.handleSync(ctx, msg.Policies)
	case "delete":
		return h.handleDelete(ctx, msg.PolicyIDs)
	case "full_sync":
		return h.handleFullSync(ctx, msg.Policies)
	default:
		log.Printf("[ABAC] 未知的策略同步动作: %s", msg.Action)
		return nil
	}
}

// handleSync 增量同步策略
func (h *MQTTHandler) handleSync(ctx context.Context, policies []*AccessPolicy) error {
	failedPolicies := []string{}
	successPolicies := []string{}

	for _, policy := range policies {
		// 只处理device类型的策略
		if policy.SubjectType != "device" {
			continue
		}

		if err := h.repo.SavePolicy(ctx, policy); err != nil {
			log.Printf("[ABAC] 保存策略失败 %s: %v", policy.ID, err)
			failedPolicies = append(failedPolicies, policy.ID)
			continue
		}
		log.Printf("[ABAC] 策略已保存: %s (%s)", policy.ID, policy.Name)
		successPolicies = append(successPolicies, policy.ID)
	}

	// 发送ACK给Cloud端
	for _, policyID := range successPolicies {
		h.sendAck(policyID, "success")
	}

	if len(failedPolicies) > 0 {
		h.syncFailCount++
		if h.syncFailCount >= 3 {
			h.triggerAlert("POLICY_SYNC_FAILED", "策略同步连续失败", map[string]interface{}{
				"failed_policies": failedPolicies,
				"fail_count":      h.syncFailCount,
			})
		}
		return nil // 部分失败不返回error，避免阻塞
	}
	h.syncFailCount = 0 // 重置失败计数
	return nil
}

// handleDelete 删除策略
func (h *MQTTHandler) handleDelete(ctx context.Context, policyIDs []string) error {
	for _, id := range policyIDs {
		if err := h.repo.DeletePolicy(ctx, id); err != nil {
			log.Printf("[ABAC] 删除策略失败 %s: %v", id, err)
			return err
		}
		log.Printf("[ABAC] 策略已删除: %s", id)
	}
	return nil
}

// handleFullSync 全量同步策略 (清空后重新导入)
func (h *MQTTHandler) handleFullSync(ctx context.Context, policies []*AccessPolicy) error {
	// 清空现有策略
	if err := h.repo.ClearPolicies(ctx); err != nil {
		log.Printf("[ABAC] 清空策略失败: %v", err)
		return err
	}

	// 导入新策略
	for _, policy := range policies {
		if policy.SubjectType != "device" {
			continue
		}

		if err := h.repo.SavePolicy(ctx, policy); err != nil {
			log.Printf("[ABAC] 保存策略失败 %s: %v", policy.ID, err)
			return err
		}
	}

	log.Printf("[ABAC] 全量同步完成，共导入 %d 条策略", len(policies))
	return nil
}

// LogSyncService 日志同步服务 - 定期将访问日志同步到Cloud
type LogSyncService struct {
	repo        Repository
	cabinetID   string
	publishFunc func(topic string, payload []byte) error
}

// NewLogSyncService 创建日志同步服务
func NewLogSyncService(repo Repository, cabinetID string, publishFunc func(string, []byte) error) *LogSyncService {
	return &LogSyncService{
		repo:        repo,
		cabinetID:   cabinetID,
		publishFunc: publishFunc,
	}
}

// LogSyncPayload 日志同步消息格式
type LogSyncPayload struct {
	CabinetID string       `json:"cabinet_id"`
	Logs      []*AccessLog `json:"logs"`
	Timestamp time.Time    `json:"timestamp"`
}

// SyncLogs 同步日志到Cloud
func (s *LogSyncService) SyncLogs(ctx context.Context) error {
	// 获取未同步的日志
	logs, err := s.repo.GetUnsyncedLogs(ctx, 100)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		return nil
	}

	// 构建同步消息
	payload := LogSyncPayload{
		CabinetID: s.cabinetID,
		Logs:      logs,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 发送到Cloud
	topic := "edge/cabinet/" + s.cabinetID + "/abac/logs"
	if err := s.publishFunc(topic, data); err != nil {
		log.Printf("[ABAC] 同步日志失败: %v", err)
		return err
	}

	// 标记为已同步
	ids := make([]int64, len(logs))
	for i, l := range logs {
		ids[i] = l.ID
	}

	if err := s.repo.MarkLogsSynced(ctx, ids); err != nil {
		log.Printf("[ABAC] 标记日志已同步失败: %v", err)
		return err
	}

	log.Printf("[ABAC] 已同步 %d 条访问日志", len(logs))
	return nil
}

// StartPeriodicSync 启动定期同步
func (s *LogSyncService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	syncTicker := time.NewTicker(interval)
	cleanTicker := time.NewTicker(24 * time.Hour) // 每24小时清理一次旧日志
	defer syncTicker.Stop()
	defer cleanTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-syncTicker.C:
			if err := s.SyncLogs(ctx); err != nil {
				log.Printf("[ABAC] 定期日志同步失败: %v", err)
			}
		case <-cleanTicker.C:
			s.cleanOldLogs(ctx)
		}
	}
}

// cleanOldLogs 清理旧日志
func (s *LogSyncService) cleanOldLogs(ctx context.Context) {
	if cleaner, ok := s.repo.(interface{ CleanOldLogs(context.Context) (int64, error) }); ok {
		count, err := cleaner.CleanOldLogs(ctx)
		if err != nil {
			log.Printf("[ABAC] 清理旧日志失败: %v", err)
		} else if count > 0 {
			log.Printf("[ABAC] 已清理 %d 条旧日志", count)
		}
	}
}

// sendAck 发送策略应用确认到Cloud
func (h *MQTTHandler) sendAck(policyID, status string) {
	if h.publishFunc == nil {
		return
	}

	ackMsg := map[string]string{
		"policy_id": policyID,
		"status":    status,
	}

	payload, err := json.Marshal(ackMsg)
	if err != nil {
		log.Printf("[ABAC] 构建ACK消息失败: %v", err)
		return
	}

	topic := "edge/cabinet/" + h.cabinetID + "/policy/ack"
	if err := h.publishFunc(topic, payload); err != nil {
		log.Printf("[ABAC] 发送ACK失败 policy=%s: %v", policyID, err)
	} else {
		log.Printf("[ABAC] 已发送ACK: policy=%s status=%s", policyID, status)
	}
}
