/*
 * ABAC日志处理器
 * 处理Edge端上报的设备访问日志
 */
package services

import (
	"context"
	"encoding/json"
	"time"

	"cloud-system/internal/abac"
	"cloud-system/internal/utils"

	"go.uber.org/zap"
)

// ABACLogHandlerImpl ABAC日志处理器实现
type ABACLogHandlerImpl struct {
	policyRepo abac.PolicyRepository
}

// NewABACLogHandler 创建ABAC日志处理器
func NewABACLogHandler(policyRepo abac.PolicyRepository) *ABACLogHandlerImpl {
	return &ABACLogHandlerImpl{
		policyRepo: policyRepo,
	}
}

// HandleAccessLogs 处理Edge上报的访问日志
func (h *ABACLogHandlerImpl) HandleAccessLogs(ctx context.Context, cabinetID string, logs []map[string]interface{}) error {
	successCount := 0
	skippedCount := 0
	for _, logData := range logs {
		log, err := h.parseAccessLog(cabinetID, logData)
		if err != nil {
			utils.Warn("解析访问日志失败", zap.Error(err))
			continue
		}

		// 过滤admin用户的日志，避免日志量过大
		if h.shouldSkipLog(log) {
			skippedCount++
			continue
		}

		if err := h.policyRepo.LogAccess(ctx, log); err != nil {
			utils.Warn("保存访问日志失败", zap.Error(err))
			continue
		}
		successCount++
	}

	utils.Info("ABAC访问日志处理完成",
		zap.String("cabinet_id", cabinetID),
		zap.Int("received", len(logs)),
		zap.Int("saved", successCount),
		zap.Int("skipped", skippedCount),
	)

	return nil
}

// parseAccessLog 解析访问日志
func (h *ABACLogHandlerImpl) parseAccessLog(cabinetID string, data map[string]interface{}) (*abac.AccessLog, error) {
	log := &abac.AccessLog{
		SubjectType: "device",
	}

	// 解析字段
	if v, ok := data["subject_type"].(string); ok {
		log.SubjectType = v
	}
	if v, ok := data["subject_id"].(string); ok {
		log.SubjectID = cabinetID + ":" + v // 添加储能柜前缀
	}
	if v, ok := data["resource"].(string); ok {
		log.Resource = v
	}
	if v, ok := data["action"].(string); ok {
		log.Action = v
	}
	if v, ok := data["allowed"].(bool); ok {
		log.Allowed = v
	}
	if v, ok := data["policy_id"].(string); ok && v != "" {
		log.PolicyID = &v
	}
	if v, ok := data["trust_score"].(float64); ok {
		log.TrustScore = &v
	}
	// reason字段存储在attributes中
	if v, ok := data["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			log.Timestamp = t
		}
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	// 保存原始属性
	if attrs, ok := data["attributes"]; ok {
		if attrsJSON, err := json.Marshal(attrs); err == nil {
			log.Attributes = attrsJSON
		}
	}

	return log, nil
}

// HandlePolicyAck 处理Edge端策略分发ACK确认
func (h *ABACLogHandlerImpl) HandlePolicyAck(ctx context.Context, cabinetID, policyID string) error {
	// 更新分发日志状态
	if err := h.policyRepo.UpdateDistributionAck(ctx, policyID, cabinetID); err != nil {
		utils.Error("更新策略分发ACK失败",
			zap.String("cabinet_id", cabinetID),
			zap.String("policy_id", policyID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("策略分发ACK确认成功",
		zap.String("cabinet_id", cabinetID),
		zap.String("policy_id", policyID),
	)

	return nil
}

// shouldSkipLog 判断是否应该跳过访问日志记录
// 跳过admin用户的日志，避免日志量过大
func (h *ABACLogHandlerImpl) shouldSkipLog(log *abac.AccessLog) bool {
	return log.SubjectType == "user" && log.SubjectID == "admin"
}
