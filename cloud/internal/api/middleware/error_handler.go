package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"
	"go.uber.org/zap"
)

// ErrorHandlerMiddleware 错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 处理panic
				utils.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				
				appErr := errors.NewInternalServerError("服务器内部错误")
				utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
			}
		}()
		
		c.Next()
		
		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// 如果是AppError，直接返回
			if appErr, ok := err.Err.(*errors.AppError); ok {
				statusCode := getHTTPStatusCode(appErr.Code)
				utils.ErrorResponse(c, statusCode, appErr)
				return
			}
			
			// 否则返回通用错误
			appErr := errors.NewInternalServerError(err.Error())
			utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
		}
	}
}

// getHTTPStatusCode 根据错误代码获取HTTP状态码
func getHTTPStatusCode(code errors.ErrorCode) int {
	switch code {
	case errors.ErrBadRequest, errors.ErrValidation:
		return http.StatusBadRequest
	case errors.ErrUnauthorized:
		return http.StatusUnauthorized
	case errors.ErrForbidden:
		return http.StatusForbidden
	case errors.ErrNotFound, errors.ErrCabinetNotFound, 
	     errors.ErrLicenseNotFound, errors.ErrRecordNotFound:
		return http.StatusNotFound
	case errors.ErrConflict, errors.ErrRecordExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				utils.Error("Panic occurred",
					zap.Any("panic", err),
					zap.String("path", c.Request.URL.Path),
				)
				
				utils.InternalServerError(c, "服务器内部错误")
			}
		}()
		
		c.Next()
	}
}

