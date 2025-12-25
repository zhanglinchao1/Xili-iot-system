package abac

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessStats 访问统计
type AccessStats struct {
	TotalRequests  int64
	DeniedRequests int64
	LastResetTime  time.Time
}

// DeviceABACMiddleware 设备ABAC中间件
type DeviceABACMiddleware struct {
	repo          Repository
	evaluator     *Evaluator
	stats         AccessStats
	alertCallback SyncAlertCallback // 复用告警回调类型
}

// NewDeviceABACMiddleware 创建设备ABAC中间件
func NewDeviceABACMiddleware(repo Repository) *DeviceABACMiddleware {
	return &DeviceABACMiddleware{
		repo:      repo,
		evaluator: NewEvaluator(),
		stats:     AccessStats{LastResetTime: time.Now()},
	}
}

// SetAlertCallback 设置告警回调
func (m *DeviceABACMiddleware) SetAlertCallback(callback SyncAlertCallback) {
	m.alertCallback = callback
}

// GetStats 获取访问统计
func (m *DeviceABACMiddleware) GetStats() AccessStats {
	return m.stats
}

// GetDenialRate 获取拒绝率
func (m *DeviceABACMiddleware) GetDenialRate() float64 {
	if m.stats.TotalRequests == 0 {
		return 0
	}
	return float64(m.stats.DeniedRequests) / float64(m.stats.TotalRequests) * 100
}

// Handle 中间件处理函数
func (m *DeviceABACMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从上下文获取设备属性 (由DeviceAuthMiddleware设置)
		deviceAttrs, exists := c.Get("device_attrs")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "未找到设备认证信息",
			})
			c.Abort()
			return
		}

		attrs, ok := deviceAttrs.(*DeviceAttributes)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "设备属性类型错误",
			})
			c.Abort()
			return
		}

		// 2. 获取启用的策略
		policies, err := m.repo.GetEnabledPolicies(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "加载策略失败",
			})
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

		evalResp := m.evaluator.Evaluate(evalReq)

		// 4. 更新统计并记录访问日志
		m.stats.TotalRequests++
		if !evalResp.Allowed {
			m.stats.DeniedRequests++
		}
		go m.logAccess(attrs, c.Request.URL.Path, c.Request.Method, evalResp)

		// 5. 检查拒绝率并触发告警（每100次请求检查一次）
		if m.stats.TotalRequests%100 == 0 {
			denialRate := m.GetDenialRate()
			if denialRate > 50 && m.alertCallback != nil {
				m.alertCallback("HIGH_DENIAL_RATE", "设备权限拒绝率过高", map[string]interface{}{
					"denial_rate":      denialRate,
					"total_requests":   m.stats.TotalRequests,
					"denied_requests":  m.stats.DeniedRequests,
				})
			}
		}

		// 6. 处理评估结果
		if !evalResp.Allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"success":     false,
				"error":       "访问被拒绝",
				"reason":      evalResp.Reason,
				"trust_score": evalResp.TrustScore,
			})
			c.Abort()
			return
		}

		// 6. 将评估结果存入上下文
		c.Set("abac_result", evalResp)
		c.Set("trust_score", evalResp.TrustScore)
		c.Set("permissions", evalResp.Permissions)

		c.Next()
	}
}

func (m *DeviceABACMiddleware) logAccess(attrs *DeviceAttributes, resource, action string, resp *EvaluateResponse) {
	attrsJSON, _ := json.Marshal(attrs)

	log := &AccessLog{
		SubjectType: string(SubjectTypeDevice),
		SubjectID:   attrs.DeviceID,
		Resource:    resource,
		Action:      action,
		Allowed:     resp.Allowed,
		TrustScore:  &resp.TrustScore,
		Reason:      resp.Reason,
		Timestamp:   time.Now(),
		Attributes:  attrsJSON,
		Synced:      false,
	}

	if resp.MatchedPolicy != nil {
		log.PolicyID = &resp.MatchedPolicy.ID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.repo.LogAccess(ctx, log)
}

// DeviceInfoProvider 设备信息提供接口
type DeviceInfoProvider interface {
	GetDeviceInfo(deviceID string) (status string, quality int, lastReading time.Time, err error)
}

// DeviceAuthMiddleware 设备认证中间件
// 从请求中提取设备身份信息并构建DeviceAttributes
type DeviceAuthMiddleware struct {
	deviceProvider DeviceInfoProvider // 可选，用于获取真实设备状态
}

// NewDeviceAuthMiddleware 创建设备认证中间件
func NewDeviceAuthMiddleware() *DeviceAuthMiddleware {
	return &DeviceAuthMiddleware{}
}

// NewDeviceAuthMiddlewareWithProvider 创建带设备信息提供者的中间件
func NewDeviceAuthMiddlewareWithProvider(provider DeviceInfoProvider) *DeviceAuthMiddleware {
	return &DeviceAuthMiddleware{deviceProvider: provider}
}

// Handle 设备认证处理
func (m *DeviceAuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取设备ID
		deviceID := c.GetHeader("X-Device-ID")
		if deviceID == "" {
			// 尝试从查询参数获取
			deviceID = c.Query("device_id")
		}

		if deviceID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "缺少设备标识",
			})
			c.Abort()
			return
		}

		// 构建设备属性
		attrs := &DeviceAttributes{
			DeviceID:   deviceID,
			CabinetID:  c.GetHeader("X-Cabinet-ID"),
			SensorType: c.GetHeader("X-Sensor-Type"),
			Status:     "active", // 默认值
			Quality:    80,       // 默认值
		}

		// 从设备管理模块获取真实状态
		if m.deviceProvider != nil {
			status, quality, lastReading, err := m.deviceProvider.GetDeviceInfo(deviceID)
			if err == nil {
				attrs.Status = status
				attrs.Quality = quality
				attrs.LastReadingAt = lastReading
			}
		}

		// 允许请求头覆盖（用于测试）
		if quality := c.GetHeader("X-Data-Quality"); quality != "" {
			var q int
			if err := json.Unmarshal([]byte(quality), &q); err == nil {
				attrs.Quality = q
			}
		}

		c.Set("device_attrs", attrs)
		c.Set("device_id", deviceID)

		c.Next()
	}
}
