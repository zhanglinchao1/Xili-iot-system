package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"cloud-system/internal/utils"
	"go.uber.org/zap"
)

// LoggingMiddleware 请求日志中间件
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求开始时间
		start := time.Now()
		
		// 请求路径
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		// 处理请求
		c.Next()
		
		// 计算延迟
		latency := time.Since(start)
		
		// 获取状态码
		statusCode := c.Writer.Status()
		
		// 获取客户端IP
		clientIP := c.ClientIP()
		
		// 获取方法
		method := c.Request.Method
		
		// 获取用户信息（如果有）
		userID, _ := c.Get("user_id")
		
		// 构建日志字段
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		
	// 添加用户ID（如果存在）
	if userID != nil {
		if uid, ok := userID.(int); ok {
			fields = append(fields, zap.Int("user_id", uid))
		} else if uid, ok := userID.(string); ok {
			fields = append(fields, zap.String("user_id", uid))
		}
	}
		
		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}
		
		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			utils.Error("HTTP Request", fields...)
		case statusCode >= 400:
			utils.Warn("HTTP Request", fields...)
		default:
			utils.Info("HTTP Request", fields...)
		}
	}
}

// AccessLogMiddleware 访问日志中间件（简化版）
func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		
		utils.Info("Access Log",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

