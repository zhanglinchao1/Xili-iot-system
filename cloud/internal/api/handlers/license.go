package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// LicenseHandler 许可证处理器
type LicenseHandler struct {
	licenseService services.LicenseService
	commandService services.CommandService
}

const (
	defaultSyncValidDays  = 90
	defaultSyncMaxDevices = 100
)

// NewLicenseHandler 创建许可证处理器实例
func NewLicenseHandler(licenseService services.LicenseService, commandService services.CommandService) *LicenseHandler {
	return &LicenseHandler{
		licenseService: licenseService,
		commandService: commandService,
	}
}

// CreateLicense 创建许可证
func (h *LicenseHandler) CreateLicense(c *gin.Context) {
	var request models.CreateLicenseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	createdBy := "admin"
	if user, exists := c.Get("user_id"); exists {
		if userID, ok := user.(int); ok {
			createdBy = fmt.Sprintf("%d", userID)
		} else if userIDStr, ok := user.(string); ok {
			createdBy = userIDStr
		}
	}

	license, err := h.licenseService.CreateLicense(c.Request.Context(), &request, createdBy)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, license, "许可证创建成功")
}

// GetLicense 获取许可证
func (h *LicenseHandler) GetLicense(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	license, err := h.licenseService.GetLicense(c.Request.Context(), cabinetID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, license)
}

// ListLicenses 获取许可证列表
func (h *LicenseHandler) ListLicenses(c *gin.Context) {
	var filter models.LicenseListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, "查询参数格式错误")
		return
	}

	licenses, total, err := h.licenseService.ListLicenses(c.Request.Context(), &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessPaginated(c, licenses, filter.Page, filter.PageSize, total)
}

// RenewLicense 续期许可证
func (h *LicenseHandler) RenewLicense(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var request models.RenewLicenseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.licenseService.RenewLicense(c.Request.Context(), cabinetID, &request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	requestedBy := "system"
	if user, exists := c.Get("user_id"); exists {
		if userID, ok := user.(int); ok {
			requestedBy = fmt.Sprintf("%d", userID)
		} else if userIDStr, ok := user.(string); ok {
			requestedBy = userIDStr
		}
	}

	if _, err := h.pushLicenseToEdge(c.Request.Context(), cabinetID, requestedBy); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, nil, "许可证续期成功并已同步Edge")
}

// RevokeLicense 吊销许可证
func (h *LicenseHandler) RevokeLicense(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var request models.RevokeLicenseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	revokedBy := "admin"
	if user, exists := c.Get("user_id"); exists {
		if userID, ok := user.(int); ok {
			revokedBy = fmt.Sprintf("%d", userID)
		} else if userIDStr, ok := user.(string); ok {
			revokedBy = userIDStr
		}
	}

	if err := h.licenseService.RevokeLicense(c.Request.Context(), cabinetID, &request, revokedBy); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	if _, err := h.sendLicenseCommand(c.Request.Context(), cabinetID, "license_revoke", revokedBy, map[string]interface{}{
		"cabinet_id":    cabinetID,
		"revoke_reason": request.Reason,
	}); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, nil, "许可证已吊销并下发Edge")
}

// ValidateLicense 验证许可证（Edge端调用）
func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
	var request models.ValidateLicenseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	license, isValid, err := h.licenseService.ValidateLicense(c.Request.Context(), &request)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, gin.H{
		"valid":   isValid,
		"license": license,
	})
}

// DeleteLicense 删除许可证记录
func (h *LicenseHandler) DeleteLicense(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	deletedBy := "system"
	if user, exists := c.Get("user_id"); exists {
		if userID, ok := user.(int); ok {
			deletedBy = fmt.Sprintf("%d", userID)
		} else if userIDStr, ok := user.(string); ok {
			deletedBy = userIDStr
		}
	}

	if err := h.licenseService.DeleteLicense(c.Request.Context(), cabinetID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	if _, err := h.sendLicenseCommand(c.Request.Context(), cabinetID, "license_revoke", deletedBy, map[string]interface{}{
		"cabinet_id":    cabinetID,
		"revoke_reason": "deleted",
	}); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, nil, "许可证已删除并通知Edge吊销")
}

// PushLicense 生成许可证并下发MQTT命令
func (h *LicenseHandler) PushLicense(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	createdBy := "system"
	if user, exists := c.Get("user_id"); exists {
		if userID, ok := user.(int); ok {
			createdBy = fmt.Sprintf("%d", userID)
		} else if userIDStr, ok := user.(string); ok {
			createdBy = userIDStr
		}
	}

	command, err := h.pushLicenseToEdge(c.Request.Context(), cabinetID, createdBy)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, command, "许可证下发命令已发送")
}

func (h *LicenseHandler) sendLicenseCommand(ctx context.Context, cabinetID, commandType, createdBy string, payload map[string]interface{}) (*models.Command, error) {
	if h.commandService == nil {
		return nil, errors.New(errors.ErrInternalServer, "命令服务不可用")
	}

	commandReq := &models.SendCommandRequest{
		CommandType: commandType,
		Payload:     payload,
	}

	return h.commandService.SendCommand(ctx, cabinetID, commandReq, createdBy)
}

func (h *LicenseHandler) pushLicenseToEdge(ctx context.Context, cabinetID, requestedBy string) (*models.Command, error) {
	token, err := h.licenseService.GenerateLicenseToken(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	licenseDetail, err := h.licenseService.GetLicense(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"license_token": token,
		"license_id":    licenseDetail.LicenseID,
		"cabinet_id":    licenseDetail.CabinetID,
		"mac_address":   licenseDetail.MACAddress,
		"max_devices":   licenseDetail.MaxDevices,
		"expires_at":    licenseDetail.ExpiresAt.Format(time.RFC3339),
	}

	return h.sendLicenseCommand(ctx, cabinetID, "license_push", requestedBy, payload)
}

// SyncLicenses 同步历史许可证
func (h *LicenseHandler) SyncLicenses(c *gin.Context) {
	var req models.SyncLicensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err != io.EOF {
			utils.ValidationError(c, "请求参数格式错误")
			return
		}
	}

	validDays := defaultIfNilInt(req.ValidDays, defaultSyncValidDays)
	maxDevices := defaultIfNilInt(req.MaxDevices, defaultSyncMaxDevices)

	created, err := h.licenseService.SyncLicenses(c.Request.Context(), req.CabinetIDs, validDays, maxDevices, req.Permissions)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, gin.H{
		"created": created,
	}, fmt.Sprintf("同步完成，新建 %d 条许可证记录", created))
}

func defaultIfNilInt(value *int, def int) int {
	if value == nil || *value <= 0 {
		return def
	}
	return *value
}
