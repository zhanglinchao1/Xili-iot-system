package errors

import "fmt"

// ErrorCode 错误代码类型
type ErrorCode string

// 错误代码常量
const (
	// 通用错误
	ErrInternalServer   ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrBadRequest       ErrorCode = "BAD_REQUEST"
	ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrForbidden        ErrorCode = "FORBIDDEN"
	ErrNotFound         ErrorCode = "NOT_FOUND"
	ErrConflict         ErrorCode = "CONFLICT"
	ErrValidation       ErrorCode = "VALIDATION_ERROR"
	ErrExternalService  ErrorCode = "EXTERNAL_SERVICE_ERROR"

	// 数据库错误
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrDatabaseQuery      ErrorCode = "DATABASE_QUERY_ERROR"
	ErrRecordNotFound     ErrorCode = "RECORD_NOT_FOUND"
	ErrRecordExists       ErrorCode = "RECORD_ALREADY_EXISTS"

	// 储能柜相关错误
	ErrCabinetNotFound  ErrorCode = "CABINET_NOT_FOUND"
	ErrCabinetOffline   ErrorCode = "CABINET_OFFLINE"
	ErrCabinetInvalidID ErrorCode = "CABINET_INVALID_ID"

	// 许可证相关错误
	ErrLicenseNotFound ErrorCode = "LICENSE_NOT_FOUND"
	ErrLicenseExpired  ErrorCode = "LICENSE_EXPIRED"
	ErrLicenseRevoked  ErrorCode = "LICENSE_REVOKED"
	ErrLicenseInvalid  ErrorCode = "LICENSE_INVALID"

	// 命令相关错误
	ErrCommandTimeout ErrorCode = "COMMAND_TIMEOUT"
	ErrCommandFailed  ErrorCode = "COMMAND_FAILED"
	ErrMQTTConnection ErrorCode = "MQTT_CONNECTION_ERROR"

	// 数据同步错误
	ErrSyncFailed            ErrorCode = "SYNC_FAILED"
	ErrSyncBatchSizeExceeded ErrorCode = "SYNC_BATCH_SIZE_EXCEEDED"
)

// AppError 应用错误结构
type AppError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Err     error                  `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 返回内部错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// ErrorResponse 统一错误响应格式
type ErrorResponse struct {
	Error struct {
		Code    ErrorCode              `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err *AppError) *ErrorResponse {
	resp := &ErrorResponse{}
	resp.Error.Code = err.Code
	resp.Error.Message = err.Message
	resp.Error.Details = err.Details
	return resp
}

// 常用错误构造函数
func NewBadRequestError(message string) *AppError {
	return New(ErrBadRequest, message)
}

func NewUnauthorizedError(message string) *AppError {
	return New(ErrUnauthorized, message)
}

func NewForbiddenError(message string) *AppError {
	return New(ErrForbidden, message)
}

func NewNotFoundError(message string) *AppError {
	return New(ErrNotFound, message)
}

func NewInternalServerError(message string) *AppError {
	return New(ErrInternalServer, message)
}

func NewValidationError(message string) *AppError {
	return New(ErrValidation, message)
}

func NewExternalServiceError(message string) *AppError {
	return New(ErrExternalService, message)
}
