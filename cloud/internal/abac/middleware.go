package abac

import (
	"context"
	"encoding/json"
	"time"

	"cloud-system/internal/repository"
	"cloud-system/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ABACMiddleware ABAC访问控制中间件
func ABACMiddleware(policyRepo PolicyRepository, cabinetRepo repository.CabinetRepository, vulnRepo repository.VulnerabilityRepository) gin.HandlerFunc {
	evaluator := NewEvaluator()
	scorer := NewTrustScorer()

	return func(c *gin.Context) {
		// 1. 确定主体类型和提取属性
		var attrs Attributes
		var subjectType SubjectType
		var subjectID string

		// 检查是否有cabinet_id (Edge端)
		if cabinetID, exists := c.Get("cabinet_id"); exists && cabinetRepo != nil {
			subjectType = SubjectTypeCabinet
			subjectID = cabinetID.(string)

			// 从数据库加载Cabinet属性
			cabinet, err := cabinetRepo.GetByID(context.Background(), subjectID)
			if err != nil {
				logAccessDenied(c, policyRepo, string(subjectType), subjectID, "获取储能柜信息失败")
				utils.Unauthorized(c, "无法验证储能柜身份")
				c.Abort()
				return
			}

			// 构建Cabinet属性
			// 简化方案：根据激活状态判断许可证状态(activated=有效许可证)
			hasValidLicense := cabinet.ActivationStatus == "activated"

			cabinetAttrs := &CabinetAttributes{
				CabinetID:           cabinet.CabinetID,
				MACAddress:          cabinet.MACAddress,
				IPAddress:           getStringOrEmpty(cabinet.IPAddress),
				Status:              cabinet.Status,
				ActivationStatus:    cabinet.ActivationStatus,
				VulnerabilityScore:  cabinet.LatestVulnerabilityScore,
				RiskLevel:           cabinet.LatestRiskLevel,
				LastSyncAt:          getTimeOrZero(cabinet.LastSyncAt),
				HasValidLicense:     hasValidLicense,
			}

			// 从脆弱性评估获取信任分数
			if vulnRepo != nil {
				if vulnAssessment, err := vulnRepo.GetLatestByCabinetID(context.Background(), subjectID); err == nil && vulnAssessment != nil {
					cabinetAttrs.TrustScore = vulnAssessment.OverallScore
				} else {
					// 如果没有脆弱性评估，使用默认计算
					cabinetAttrs.TrustScore = scorer.CalculateCabinetTrustScore(cabinetAttrs)
				}
			} else {
				cabinetAttrs.TrustScore = scorer.CalculateCabinetTrustScore(cabinetAttrs)
			}
			attrs = cabinetAttrs

		} else if userID, exists := c.Get("user_id"); exists {
			// 用户端
			subjectType = SubjectTypeUser
			subjectID = c.GetString("username")

			// 构建User属性
			userAttrs := &UserAttributes{
				UserID:      userID.(int),
				Username:    c.GetString("username"),
				Role:        c.GetString("role"),
				Status:      "active", // 简化：认证通过即为active
				LastLoginIP: c.ClientIP(),
				LastLoginAt: time.Now(),
			}

			// 计算信任度
			userAttrs.TrustScore = scorer.CalculateUserTrustScore(userAttrs)
			attrs = userAttrs
		} else {
			// 未认证的请求，不应该到达这里（应该被认证中间件拦截）
			logAccessDenied(c, policyRepo, "unknown", "anonymous", "未认证")
			utils.Unauthorized(c, "未认证")
			c.Abort()
			return
		}

		// 2. 加载启用的策略
		policies, err := policyRepo.GetBySubjectType(context.Background(), string(subjectType), true)
		if err != nil {
			utils.Error("加载策略失败", zap.Error(err))
			logAccessDenied(c, policyRepo, string(subjectType), subjectID, "加载策略失败")
			utils.InternalServerError(c, "访问控制系统错误")
			c.Abort()
			return
		}

		// 3. 执行策略评估
		evalReq := &EvaluateRequest{
			SubjectAttrs: attrs,
			Resource:     c.Request.URL.Path,
			Action:       c.Request.Method,
			Policies:     policies,
		}

		evalResp := evaluator.Evaluate(evalReq)

		// 4. 记录访问日志
		logAccess(c, policyRepo, string(subjectType), subjectID, evalResp)

		// 5. 授权决策
		if !evalResp.Allowed {
			utils.Forbidden(c, evalResp.Reason)
			c.Abort()
			return
		}

		// 将评估结果存入上下文
		c.Set("abac_evaluation", evalResp)
		c.Set("trust_score", evalResp.TrustScore)

		c.Next()
	}
}

// logAccess 记录访问日志
func logAccess(c *gin.Context, policyRepo PolicyRepository, subjectType, subjectID string, evalResp *EvaluateResponse) {
	// 过滤admin用户的日志，避免日志量过大
	if shouldSkipAccessLog(subjectType, subjectID) {
		return
	}

	// 序列化属性
	attrsJSON, _ := json.Marshal(map[string]interface{}{
		"trust_score": evalResp.TrustScore,
		"ip_address":  c.ClientIP(),
	})

	log := &AccessLog{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Resource:    c.Request.URL.Path,
		Action:      c.Request.Method,
		Allowed:     evalResp.Allowed,
		TrustScore:  &evalResp.TrustScore,
		Timestamp:   time.Now(),
		Attributes:  attrsJSON,
	}

	if evalResp.MatchedPolicy != nil {
		log.PolicyID = &evalResp.MatchedPolicy.ID
	}

	ipAddr := c.ClientIP()
	log.IPAddress = &ipAddr

	// 异步记录日志，不阻塞请求
	go func() {
		if err := policyRepo.LogAccess(context.Background(), log); err != nil {
			utils.Error("记录访问日志失败", zap.Error(err))
		}
	}()
}

// logAccessDenied 记录访问拒绝日志
func logAccessDenied(c *gin.Context, policyRepo PolicyRepository, subjectType, subjectID, reason string) {
	// 过滤admin用户的日志，避免日志量过大
	if shouldSkipAccessLog(subjectType, subjectID) {
		return
	}

	attrsJSON, _ := json.Marshal(map[string]interface{}{
		"reason":     reason,
		"ip_address": c.ClientIP(),
	})

	log := &AccessLog{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Resource:    c.Request.URL.Path,
		Action:      c.Request.Method,
		Allowed:     false,
		Timestamp:   time.Now(),
		Attributes:  attrsJSON,
	}

	ipAddr := c.ClientIP()
	log.IPAddress = &ipAddr

	go func() {
		if err := policyRepo.LogAccess(context.Background(), log); err != nil {
			utils.Error("记录访问日志失败", zap.Error(err))
		}
	}()
}

// 辅助函数
func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getTimeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

// getTimeOrZero 返回时间或零值(用于信任度计算)
func getTimeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{} // 返回零值,让trust scorer检测"从未同步"
	}
	return *t
}

// shouldSkipAccessLog 判断是否应该跳过访问日志记录
// 跳过admin用户的日志，避免日志量过大
func shouldSkipAccessLog(subjectType, subjectID string) bool {
	return subjectType == "user" && subjectID == "admin"
}
