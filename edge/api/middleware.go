/*
 * API中间件
 * 提供认证、日志、CORS等中间件功能
 */
package api

import (
	"crypto/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/edge/storage-cabinet/internal/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 日志中间件
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP请求",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在生产环境中应该设置具体的域名而不是 *
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 添加安全头
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// AuthMiddleware 认证中间件
func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "AUTH_001",
				"message": "缺少认证令牌",
			})
			c.Abort()
			return
		}

		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "AUTH_001",
				"message": "认证令牌格式错误",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// 验证令牌
		session, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "AUTH_002",
				"message": "认证令牌无效或已过期",
			})
			c.Abort()
			return
		}

		// 将会话信息存储到上下文
		c.Set("session", session)
		c.Set("device_id", session.DeviceID)

		c.Next()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
	// 简单的内存限流实现，使用互斥锁保护并发访问
	clients := make(map[string][]time.Time)
	var mu sync.RWMutex // 读写锁，保护 clients map 的并发访问

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// 使用写锁保护 map 的读写操作
		mu.Lock()
		defer mu.Unlock()

		// 清理过期记录
		if requests, exists := clients[clientIP]; exists {
			validRequests := make([]time.Time, 0)
			for _, reqTime := range requests {
				if now.Sub(reqTime) < window {
					validRequests = append(validRequests, reqTime)
				}
			}
			clients[clientIP] = validRequests
		}

		// 检查是否超过限制
		if len(clients[clientIP]) >= maxRequests {
			mu.Unlock() // 提前释放锁，避免响应时持有锁
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "RATE_LIMIT",
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		// 记录当前请求
		clients[clientIP] = append(clients[clientIP], now)

		c.Next()
	}
}

// ErrorHandlerMiddleware 错误处理中间件
func ErrorHandlerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("服务器内部错误",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": "服务器内部错误",
		})
	})
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 简单的时间戳+随机数实现
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成加密安全的随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	randBytes := make([]byte, length)

	if _, err := rand.Read(randBytes); err != nil {
		// 降级到时间戳方式
		for i := range b {
			b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		}
		return string(b)
	}

	for i := range b {
		b[i] = charset[randBytes[i]%byte(len(charset))]
	}
	return string(b)
}
