package handlers

import (
	"net/http"

	"github.com/edge/storage-cabinet/internal/abac"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ABACHandler ABAC策略处理器(只读API)
type ABACHandler struct {
	repo   abac.Repository
	logger *zap.Logger
}

// NewABACHandler 创建ABAC处理器
func NewABACHandler(repo abac.Repository, logger *zap.Logger) *ABACHandler {
	return &ABACHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListPolicies 查询本地策略列表(只读)
func (h *ABACHandler) ListPolicies(c *gin.Context) {
	enabledOnly := c.Query("enabled_only") == "true"

	var policies []*abac.AccessPolicy
	var err error

	if enabledOnly {
		policies, err = h.repo.GetEnabledPolicies(c.Request.Context())
	} else {
		policies, err = h.repo.GetAllPolicies(c.Request.Context())
	}

	if err != nil {
		h.logger.Error("查询策略列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "QUERY_FAILED",
			"message": "查询策略失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "success",
		"data":     policies,
		"count":    len(policies),
		"readonly": true, // 标识为只读API
	})
}

// GetPolicy 获取策略详情(只读)
func (h *ABACHandler) GetPolicy(c *gin.Context) {
	policyID := c.Param("id")

	policy, err := h.repo.GetPolicy(c.Request.Context(), policyID)
	if err != nil {
		h.logger.Error("查询策略失败",
			zap.String("policy_id", policyID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": "策略不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "success",
		"data":     policy,
		"readonly": true,
	})
}

// GetPolicyStats 获取策略统计信息
func (h *ABACHandler) GetPolicyStats(c *gin.Context) {
	policies, err := h.repo.GetAllPolicies(c.Request.Context())
	if err != nil {
		h.logger.Error("查询策略统计失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "QUERY_FAILED",
			"message": "查询失败",
		})
		return
	}

	stats := gin.H{
		"total_policies":   len(policies),
		"enabled_policies": 0,
		"device_policies":  0,
	}

	for _, p := range policies {
		if p.Enabled {
			stats["enabled_policies"] = stats["enabled_policies"].(int) + 1
		}
		if p.SubjectType == "device" {
			stats["device_policies"] = stats["device_policies"].(int) + 1
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}
