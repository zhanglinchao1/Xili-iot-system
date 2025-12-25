package middleware

import (
	"context"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   float64 `json:"user_id"` // JWT数字类型默认为float64
	Username string  `json:"username"`
	Role     string  `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "缺少认证令牌")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Unauthorized(c, "认证令牌格式无效")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析JWT
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			utils.Unauthorized(c, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", int(claims.UserID))
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件
// 如果提供了有效token则解析，否则继续
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})

		if err == nil && token.Valid {
			c.Set("user_id", int(claims.UserID))
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
		}

		c.Next()
	}
}

// GenerateToken 生成JWT令牌
func GenerateToken(cfg *config.Config, userID int, username, role string) (string, error) {
	// 解析过期时间
	expiry, err := time.ParseDuration(cfg.JWT.Expiry)
	if err != nil {
		expiry = 24 * time.Hour
	}

	claims := JWTClaims{
		UserID:   float64(userID),
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// RefreshToken 刷新JWT令牌
func RefreshToken(cfg *config.Config, tokenString string) (string, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.NewUnauthorizedError("令牌无效")
	}

	// 生成新令牌
	return GenerateToken(cfg, int(claims.UserID), claims.Username, claims.Role)
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	return username.(string), true
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	return role.(string), true
}

// AdminMiddleware 管理员权限检查中间件
// 必须在AuthMiddleware之后使用
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, 403, errors.NewForbiddenError("权限不足"))
			c.Abort()
			return
		}

		if role.(string) != "admin" {
			utils.ErrorResponse(c, 403, errors.NewForbiddenError("需要管理员权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// EdgeAPIKeyMiddleware Edge端API Key认证中间件（带数据库验证）
// 用于Edge端同步数据的端点，验证Bearer token格式的API Key
func EdgeAPIKeyMiddleware(cabinetRepo repository.CabinetRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// API Key认证是可选的，允许无认证访问（向后兼容）
			c.Next()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			utils.Unauthorized(c, "API Key格式无效")
			c.Abort()
			return
		}

		apiKey := parts[1]

		// 基本长度检查
		if len(apiKey) < 10 {
			utils.Unauthorized(c, "API Key格式无效")
			c.Abort()
			return
		}

		// 从数据库验证API Key
		ctx := context.Background()
		cabinet, err := cabinetRepo.GetByAPIKey(ctx, apiKey)
		if err != nil {
			utils.Warn("API Key验证失败",
				zap.String("api_key_prefix", apiKey[:10]+"..."),
				zap.Error(err),
			)
			utils.Unauthorized(c, "无效的API Key")
			c.Abort()
			return
		}

		// 检查储能柜是否已激活
		if cabinet.ActivationStatus != "activated" {
			utils.Warn("储能柜未激活",
				zap.String("cabinet_id", cabinet.CabinetID),
				zap.String("status", cabinet.ActivationStatus),
			)
			utils.Unauthorized(c, "储能柜未激活")
			c.Abort()
			return
		}

		// 验证成功，将信息存入上下文
		c.Set("edge_api_key", apiKey)
		c.Set("cabinet_id", cabinet.CabinetID)
		c.Set("cabinet", cabinet)

		utils.Debug("API Key验证成功",
			zap.String("cabinet_id", cabinet.CabinetID),
			zap.String("api_key_prefix", apiKey[:10]+"..."),
		)

		c.Next()
	}
}
