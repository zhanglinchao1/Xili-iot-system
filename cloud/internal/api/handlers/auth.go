// Package handlers 实现认证相关的HTTP处理器
package handlers

import (
	"net/http"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码登录，返回JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "登录请求"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error":   "INVALID_REQUEST",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务层
	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		// 处理错误
		c.JSON(401, gin.H{
			"success": false,
			"error":   "AUTH_FAILED",
			"message": err.Error(),
		})
		return
	}

	utils.SuccessWithMessage(c, resp, "登录成功")
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if !ok {
			appErr = errors.Wrap(err, errors.ErrInternalServer, "注册失败")
		}

		statusCode := http.StatusBadRequest
		switch appErr.Code {
		case errors.ErrConflict:
			statusCode = http.StatusConflict
		case errors.ErrInternalServer, errors.ErrDatabaseQuery:
			statusCode = http.StatusInternalServerError
		case errors.ErrValidation:
			statusCode = http.StatusBadRequest
		default:
			statusCode = http.StatusBadRequest
		}

		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, resp, "注册成功")
}

// GetCurrentUser 获取当前登录用户信息
// @Summary 获取当前用户
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserInfo
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// 从中间件中获取用户信息
	username, exists := c.Get("username")
	if !exists {
		c.JSON(401, gin.H{"success": false, "error": "UNAUTHORIZED", "message": "未授权"})
		return
	}
	
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"success": false, "error": "UNAUTHORIZED", "message": "未授权"})
		return
	}
	
	role, _ := c.Get("role")

	userInfo := &models.UserInfo{
		ID:       userID.(int),
		Username: username.(string),
		Role:     role.(string),
	}

	utils.Success(c, userInfo)
}

