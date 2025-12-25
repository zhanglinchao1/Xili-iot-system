package utils

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"cloud-system/pkg/errors"
)

// SuccessResponse 统一成功响应格式
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse 分页响应格式
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Page    int         `json:"page"`
	PageSize int        `json:"page_size"`
	Total   int64       `json:"total"`
	Message string      `json:"message,omitempty"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage 返回带消息的成功响应
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// SuccessPaginated 返回分页成功响应
func SuccessPaginated(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Success:  true,
		Data:     data,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// ErrorResponse 返回错误响应
func ErrorResponse(c *gin.Context, statusCode int, err *errors.AppError) {
	c.JSON(statusCode, errors.NewErrorResponse(err))
}

// BadRequest 返回400错误
func BadRequest(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, errors.NewBadRequestError(message))
}

// Unauthorized 返回401错误
func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, errors.NewUnauthorizedError(message))
}

// Forbidden 返回403错误
func Forbidden(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, errors.NewForbiddenError(message))
}

// NotFound 返回404错误
func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, errors.NewNotFoundError(message))
}

// InternalServerError 返回500错误
func InternalServerError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, errors.NewInternalServerError(message))
}

// ValidationError 返回验证错误
func ValidationError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, errors.NewValidationError(message))
}

