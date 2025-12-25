package handlers

import (
	"time"

	"cloud-system/internal/abac"
	"cloud-system/internal/mqtt"
	"cloud-system/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ABACHandler struct {
	policyRepo      abac.PolicyRepository
	policyPublisher *mqtt.PolicyPublisher
}

func NewABACHandler(policyRepo abac.PolicyRepository) *ABACHandler {
	return &ABACHandler{
		policyRepo: policyRepo,
	}
}

// SetPolicyPublisher 设置策略发布器
func (h *ABACHandler) SetPolicyPublisher(publisher *mqtt.PolicyPublisher) {
	h.policyPublisher = publisher
}

// ListPolicies 列出所有策略
func (h *ABACHandler) ListPolicies(c *gin.Context) {
	var filter abac.PolicyListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "无效的查询参数")
		return
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}

	policies, total, err := h.policyRepo.List(c.Request.Context(), &filter)
	if err != nil {
		utils.Error("查询策略列表失败", zap.Error(err))
		utils.InternalServerError(c, "查询失败")
		return
	}

	utils.SuccessPaginated(c, policies, filter.Page, filter.PageSize, total)
}

// GetPolicy 获取策略详情
func (h *ABACHandler) GetPolicy(c *gin.Context) {
	id := c.Param("id")

	policy, err := h.policyRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.Error("查询策略失败", zap.Error(err), zap.String("policy_id", id))
		utils.NotFound(c, "策略不存在")
		return
	}

	utils.Success(c, gin.H{"policy": policy})
}

// CreatePolicy 创建策略
func (h *ABACHandler) CreatePolicy(c *gin.Context) {
	var req abac.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	policy := &abac.AccessPolicy{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		SubjectType: req.SubjectType,
		Conditions:  req.Conditions,
		Permissions: req.Permissions,
		Priority:    req.Priority,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.policyRepo.Create(c.Request.Context(), policy); err != nil {
		utils.Error("创建策略失败", zap.Error(err))
		utils.InternalServerError(c, "创建失败")
		return
	}

	utils.Info("创建策略成功", zap.String("policy_id", req.ID))
	utils.Success(c, gin.H{"policy": policy})
}

// UpdatePolicy 更新策略
func (h *ABACHandler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")

	var req abac.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	if err := h.policyRepo.Update(c.Request.Context(), id, &req); err != nil {
		utils.Error("更新策略失败", zap.Error(err), zap.String("policy_id", id))
		utils.InternalServerError(c, "更新失败")
		return
	}

	utils.Info("更新策略成功", zap.String("policy_id", id))
	utils.Success(c, gin.H{"message": "更新成功"})
}

// DeletePolicy 删除策略
func (h *ABACHandler) DeletePolicy(c *gin.Context) {
	id := c.Param("id")

	// 保护核心策略：禁止删除管理员完全访问策略
	if id == "policy_admin_full" {
		utils.Warn("尝试删除受保护的管理员策略", zap.String("policy_id", id))
		utils.BadRequest(c, "管理员完全访问策略是系统核心策略，不能删除")
		return
	}

	if err := h.policyRepo.Delete(c.Request.Context(), id); err != nil {
		utils.Error("删除策略失败", zap.Error(err), zap.String("policy_id", id))
		utils.InternalServerError(c, "删除失败")
		return
	}

	utils.Info("删除策略成功", zap.String("policy_id", id))
	utils.Success(c, gin.H{"message": "删除成功"})
}

// TogglePolicy 切换策略启用状态
func (h *ABACHandler) TogglePolicy(c *gin.Context) {
	id := c.Param("id")

	// 保护核心策略：禁止禁用管理员完全访问策略
	if id == "policy_admin_full" {
		// 先检查当前状态
		policy, err := h.policyRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			utils.Error("获取策略失败", zap.Error(err), zap.String("policy_id", id))
			utils.InternalServerError(c, "操作失败")
			return
		}
		// 如果当前是启用状态，阻止禁用
		if policy.Enabled {
			utils.Warn("尝试禁用受保护的管理员策略", zap.String("policy_id", id))
			utils.BadRequest(c, "管理员完全访问策略是系统核心策略，不能禁用")
			return
		}
	}

	if err := h.policyRepo.ToggleEnabled(c.Request.Context(), id); err != nil {
		utils.Error("切换策略状态失败", zap.Error(err), zap.String("policy_id", id))
		utils.InternalServerError(c, "操作失败")
		return
	}

	utils.Info("切换策略状态成功", zap.String("policy_id", id))
	utils.Success(c, gin.H{"message": "操作成功"})
}

// ListAccessLogs 列出访问日志
func (h *ABACHandler) ListAccessLogs(c *gin.Context) {
	var filter abac.AccessLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "无效的查询参数")
		return
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	logs, total, err := h.policyRepo.GetAccessLogs(c.Request.Context(), &filter)
	if err != nil {
		utils.Error("查询访问日志失败", zap.Error(err))
		utils.InternalServerError(c, "查询失败")
		return
	}

	utils.SuccessPaginated(c, logs, filter.Page, filter.PageSize, total)
}

// GetAccessStats 获取访问统计
func (h *ABACHandler) GetAccessStats(c *gin.Context) {
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	var startTimePtr, endTimePtr *string
	if startTime != "" {
		startTimePtr = &startTime
	}
	if endTime != "" {
		endTimePtr = &endTime
	}

	stats, err := h.policyRepo.GetAccessStats(c.Request.Context(), startTimePtr, endTimePtr)
	if err != nil {
		utils.Error("查询访问统计失败", zap.Error(err))
		utils.InternalServerError(c, "查询失败")
		return
	}

	utils.Success(c, gin.H{"stats": stats})
}

// EvaluatePolicy 测试策略评估
func (h *ABACHandler) EvaluatePolicy(c *gin.Context) {
	var req abac.EvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	// 加载策略
	policies, err := h.policyRepo.GetBySubjectType(c.Request.Context(), req.SubjectType, true)
	if err != nil {
		utils.Error("加载策略失败", zap.Error(err))
		utils.InternalServerError(c, "评估失败")
		return
	}

	// 构建属性对象
	var attrs abac.Attributes
	switch req.SubjectType {
	case "user":
		userAttrs := &abac.UserAttributes{}
		// 从map填充属性
		if v, ok := req.Attributes["user_id"].(float64); ok {
			userAttrs.UserID = int(v)
		}
		if v, ok := req.Attributes["username"].(string); ok {
			userAttrs.Username = v
		}
		if v, ok := req.Attributes["role"].(string); ok {
			userAttrs.Role = v
		}
		if v, ok := req.Attributes["status"].(string); ok {
			userAttrs.Status = v
		}
		if v, ok := req.Attributes["trust_score"].(float64); ok {
			userAttrs.TrustScore = v
		}
		attrs = userAttrs

	case "cabinet":
		cabinetAttrs := &abac.CabinetAttributes{}
		if v, ok := req.Attributes["cabinet_id"].(string); ok {
			cabinetAttrs.CabinetID = v
		}
		if v, ok := req.Attributes["status"].(string); ok {
			cabinetAttrs.Status = v
		}
		if v, ok := req.Attributes["activation_status"].(string); ok {
			cabinetAttrs.ActivationStatus = v
		}
		if v, ok := req.Attributes["vulnerability_score"].(float64); ok {
			cabinetAttrs.VulnerabilityScore = v
		}
		if v, ok := req.Attributes["risk_level"].(string); ok {
			cabinetAttrs.RiskLevel = v
		}
		if v, ok := req.Attributes["trust_score"].(float64); ok {
			cabinetAttrs.TrustScore = v
		}
		attrs = cabinetAttrs

	default:
		utils.BadRequest(c, "不支持的主体类型")
		return
	}

	// 评估
	evaluator := abac.NewEvaluator()
	evalReq := &abac.EvaluateRequest{
		SubjectAttrs: attrs,
		Resource:     req.Resource,
		Action:       req.Action,
		Policies:     policies,
	}

	evalResp := evaluator.Evaluate(evalReq)

	result := &abac.EvaluationResult{
		Allowed:       evalResp.Allowed,
		MatchedPolicy: evalResp.MatchedPolicy,
		TrustScore:    evalResp.TrustScore,
		Permissions:   evalResp.Permissions,
		Reason:        evalResp.Reason,
	}

	utils.Success(c, gin.H{"result": result})
}

// DistributeRequest 策略分发请求
type DistributeRequest struct {
	CabinetIDs []string `json:"cabinet_ids" binding:"required"`
	PolicyIDs  []string `json:"policy_ids,omitempty"`
	FullSync   bool     `json:"full_sync,omitempty"`
}

// DistributePolicy 分发策略到指定储能柜
func (h *ABACHandler) DistributePolicy(c *gin.Context) {
	policyID := c.Param("id")

	var req struct {
		CabinetIDs []string `json:"cabinet_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	if h.policyPublisher == nil {
		utils.InternalServerError(c, "策略发布器未配置")
		return
	}

	// 获取操作者信息
	operatorID, _ := c.Get("user_id")
	operatorName, _ := c.Get("username")
	var operatorIDInt *int
	var operatorNameStr *string
	if id, ok := operatorID.(int); ok {
		operatorIDInt = &id
	}
	if name, ok := operatorName.(string); ok {
		operatorNameStr = &name
	}

	// 分发到每个储能柜
	successCount := 0
	var lastErr error
	for _, cabinetID := range req.CabinetIDs {
		// 记录分发日志
		distributionLog := &abac.DistributionLog{
			PolicyID:      policyID,
			CabinetID:     cabinetID,
			OperationType: "distribute",
			Status:        "pending",
			OperatorID:    operatorIDInt,
			OperatorName:  operatorNameStr,
			DistributedAt: time.Now(),
		}

		if err := h.policyPublisher.DistributePolicyToCabinet(c.Request.Context(), cabinetID, policyID); err != nil {
			utils.Warn("分发策略失败", zap.String("cabinet_id", cabinetID), zap.Error(err))
			lastErr = err
			// 记录失败状态
			distributionLog.Status = "failed"
			errMsg := err.Error()
			distributionLog.ErrorMessage = &errMsg
			h.policyRepo.LogDistribution(c.Request.Context(), distributionLog)
			continue
		}

		// 记录成功分发的日志(状态为pending,等待Edge端确认)
		if err := h.policyRepo.LogDistribution(c.Request.Context(), distributionLog); err != nil {
			utils.Warn("记录分发日志失败", zap.Error(err))
		}
		successCount++
	}

	if successCount == 0 && lastErr != nil {
		utils.InternalServerError(c, "分发失败: "+lastErr.Error())
		return
	}

	utils.Info("策略分发完成", zap.String("policy_id", policyID), zap.Int("success", successCount))
	utils.Success(c, gin.H{
		"message":       "分发完成",
		"success_count": successCount,
		"total":         len(req.CabinetIDs),
	})
}

// FullSyncPolicies 全量同步策略到储能柜
func (h *ABACHandler) FullSyncPolicies(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	if h.policyPublisher == nil {
		utils.InternalServerError(c, "策略发布器未配置")
		return
	}

	if err := h.policyPublisher.FullSyncToCabinet(c.Request.Context(), cabinetID); err != nil {
		utils.Error("全量同步失败", zap.String("cabinet_id", cabinetID), zap.Error(err))
		utils.InternalServerError(c, "同步失败: "+err.Error())
		return
	}

	utils.Info("全量策略同步完成", zap.String("cabinet_id", cabinetID))
	utils.Success(c, gin.H{"message": "同步完成"})
}

// BroadcastPolicy 广播策略到所有储能柜
func (h *ABACHandler) BroadcastPolicy(c *gin.Context) {
	policyID := c.Param("id")

	if h.policyPublisher == nil {
		utils.InternalServerError(c, "策略发布器未配置")
		return
	}

	if err := h.policyPublisher.BroadcastPolicyToAllCabinets(c.Request.Context(), policyID); err != nil {
		utils.Error("广播策略失败", zap.String("policy_id", policyID), zap.Error(err))
		utils.InternalServerError(c, "广播失败: "+err.Error())
		return
	}

	utils.Info("策略广播完成", zap.String("policy_id", policyID))
	utils.Success(c, gin.H{"message": "广播完成"})
}

// SyncDeviceAccessLogs 接收Edge端上报的设备访问日志
func (h *ABACHandler) SyncDeviceAccessLogs(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var req struct {
		Logs []abac.AccessLog `json:"logs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	// 批量保存日志
	successCount := 0
	skippedCount := 0
	for _, log := range req.Logs {
		log.SubjectID = cabinetID + ":" + log.SubjectID // 添加储能柜前缀

		// 过滤admin用户的日志，避免日志量过大
		if log.SubjectType == "user" && log.SubjectID == "admin" {
			skippedCount++
			continue
		}

		if err := h.policyRepo.LogAccess(c.Request.Context(), &log); err != nil {
			utils.Warn("保存访问日志失败", zap.Error(err))
			continue
		}
		successCount++
	}

	utils.Info("设备访问日志同步完成",
		zap.String("cabinet_id", cabinetID),
		zap.Int("received", len(req.Logs)),
		zap.Int("saved", successCount),
		zap.Int("skipped", skippedCount),
	)

	utils.Success(c, gin.H{
		"message":       "同步完成",
		"received":      len(req.Logs),
		"success_count": successCount,
		"skipped_count": skippedCount,
	})
}

// ListDistributionLogs 获取策略分发历史
func (h *ABACHandler) ListDistributionLogs(c *gin.Context) {
	var filter abac.DistributionLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.BadRequest(c, "无效的查询参数")
		return
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	logs, total, err := h.policyRepo.GetDistributionLogs(c.Request.Context(), &filter)
	if err != nil {
		utils.Error("查询分发日志失败", zap.Error(err))
		utils.InternalServerError(c, "查询失败")
		return
	}

	utils.SuccessPaginated(c, logs, filter.Page, filter.PageSize, total)
}

// GetPolicyDistributionStatus 获取策略分发状态
func (h *ABACHandler) GetPolicyDistributionStatus(c *gin.Context) {
	policyID := c.Param("id")

	filter := &abac.DistributionLogFilter{
		PolicyID: &policyID,
		Page:     1,
		PageSize: 100, // 获取最近100条记录
	}

	logs, total, err := h.policyRepo.GetDistributionLogs(c.Request.Context(), filter)
	if err != nil {
		utils.Error("查询策略分发状态失败", zap.Error(err), zap.String("policy_id", policyID))
		utils.InternalServerError(c, "查询失败")
		return
	}

	// 统计各状态数量
	stats := map[string]int{
		"pending": 0,
		"success": 0,
		"failed":  0,
	}
	for _, log := range logs {
		stats[log.Status]++
	}

	utils.Success(c, gin.H{
		"policy_id": policyID,
		"logs":      logs,
		"total":     total,
		"stats":     stats,
	})
}
