// Package handlers 实现用户管理相关的HTTP处理器
package handlers

import (
	"net/http"
	"strconv"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile 获取个人信息
// @Summary 获取个人信息
// @Description 获取当前登录用户的个人信息
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserInfo
// @Failure 401 {object} errors.ErrorResponse
// @Router /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未授权")
		return
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), userID.(int))
	if err != nil {
		appErr := err.(*errors.AppError)
		utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
		return
	}

	utils.Success(c, profile)
}

// UpdateProfile 更新个人信息
// @Summary 更新个人信息
// @Description 更新当前登录用户的个人信息（邮箱）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.UpdateProfileRequest true "更新信息"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未授权")
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), userID.(int), &req); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrConflict {
			statusCode = http.StatusConflict
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "更新成功")
}

// UpdatePassword 修改密码
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.UpdatePasswordRequest true "修改密码"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/users/password [put]
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未授权")
		return
	}

	var req models.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), userID.(int), &req); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrUnauthorized {
			statusCode = http.StatusUnauthorized
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "密码修改成功")
}

// ListUsers 获取用户列表（管理员）
// @Summary 获取用户列表
// @Description 获取所有用户列表（仅管理员）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param role query string false "角色过滤"
// @Param status query string false "状态过滤"
// @Success 200 {object} utils.PaginatedResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// 权限检查在中间件中完成

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	filter := &models.UserListFilter{
		Page:  page,
		Limit: pageSize,
	}

	if role := c.Query("role"); role != "" {
		filter.Role = &role
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), filter)
	if err != nil {
		appErr := err.(*errors.AppError)
		utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
		return
	}

	utils.SuccessPaginated(c, users, page, pageSize, total)
}

// GetUser 获取用户详情（管理员）
// @Summary 获取用户详情
// @Description 获取指定用户的详细信息（仅管理员）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "用户ID"
// @Success 200 {object} models.UserInfo
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.ValidationError(c, "用户ID格式错误")
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, user)
}

// CreateUser 创建用户（管理员）
// @Summary 创建用户
// @Description 创建新用户（仅管理员）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateUserRequest true "创建用户"
// @Success 201 {object} models.UserInfo
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrConflict {
			statusCode = http.StatusConflict
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
		"message": "用户创建成功",
	})
}

// UpdateUser 更新用户（管理员）
// @Summary 更新用户
// @Description 更新指定用户的信息（仅管理员）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "用户ID"
// @Param body body models.UpdateUserRequest true "更新信息"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.ValidationError(c, "用户ID格式错误")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.userService.UpdateUser(c.Request.Context(), userID, &req); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrConflict {
			statusCode = http.StatusConflict
		} else if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "更新成功")
}

// DeleteUser 删除用户（管理员）
// @Summary 删除用户
// @Description 删除指定用户（仅管理员）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "用户ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.ValidationError(c, "用户ID格式错误")
		return
	}

	// 防止管理员删除自己
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(int) == userID {
		utils.ErrorResponse(c, http.StatusBadRequest, errors.NewValidationError("不能删除自己"))
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "删除成功")
}

// ResetUserPassword 重置用户密码（管理员）
// @Summary 重置用户密码
// @Description 重置指定用户的密码（仅管理员）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "用户ID"
// @Param body body models.ResetPasswordRequest true "新密码"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/users/{user_id}/reset-password [put]
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.ValidationError(c, "用户ID格式错误")
		return
	}

	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.userService.ResetUserPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "密码重置成功")
}
