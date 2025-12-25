package middleware

import (
	"strconv"

	"cloud-system/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.CORS.Enabled {
			c.Next()
			return
		}

		// 检查是否允许所有来源
		allowAllOrigins := false
		for _, allowed := range cfg.CORS.AllowOrigins {
			if allowed == "*" {
				allowAllOrigins = true
				break
			}
		}

		// 设置允许的来源
		origin := c.Request.Header.Get("Origin")
		if allowAllOrigins {
			// 允许所有来源时，不能设置Credentials
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && isOriginAllowed(origin, cfg.CORS.AllowOrigins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			// 只有指定来源时才允许凭证
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// 设置允许的方法
		c.Writer.Header().Set("Access-Control-Allow-Methods",
			joinStrings(cfg.CORS.AllowMethods, ", "))

		// 设置允许的头
		c.Writer.Header().Set("Access-Control-Allow-Headers",
			joinStrings(cfg.CORS.AllowHeaders, ", "))

		// 设置预检请求缓存时间
		c.Writer.Header().Set("Access-Control-Max-Age",
			strconv.Itoa(cfg.CORS.MaxAge))

		// 处理OPTIONS请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查来源是否被允许
func isOriginAllowed(origin string, allowOrigins []string) bool {
	for _, allowed := range allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// joinStrings 连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
