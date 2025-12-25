package handlers

import (
	"fmt"
	"net/http"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService services.AlertService
}

// NewAlertHandler 创建告警处理器实例
func NewAlertHandler(alertService services.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

// ListAlerts 获取告警列表
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	var filter models.AlertListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, "查询参数格式错误")
		return
	}

	alerts, total, err := h.alertService.ListAlerts(c.Request.Context(), &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessPaginated(c, alerts, filter.Page, filter.PageSize, total)
}

// GetAlert 获取告警详情
func (h *AlertHandler) GetAlert(c *gin.Context) {
	alertID := c.Param("alert_id")

	alert, err := h.alertService.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, alert)
}

// ResolveAlert 解决告警
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("alert_id")

	resolvedBy := "admin"
	if user, exists := c.Get("user_id"); exists {
		if username, ok := c.Get("username"); ok {
			resolvedBy = username.(string)
		} else {
			// 如果没有username，使用user_id转换为字符串
			resolvedBy = fmt.Sprintf("user_%d", user.(int))
		}
	}

	if err := h.alertService.ResolveAlert(c.Request.Context(), alertID, resolvedBy); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		} else if appErr.Code == errors.ErrDatabaseQuery || appErr.Code == errors.ErrInternalServer {
			statusCode = http.StatusInternalServerError
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "告警已解决")
}

// GetCabinetAlerts 获取储能柜的告警
func (h *AlertHandler) GetCabinetAlerts(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	alerts, err := h.alertService.GetCabinetAlerts(c.Request.Context(), cabinetID)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, alerts)
}

// GetHealthScore 获取储能柜健康评分
func (h *AlertHandler) GetHealthScore(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	score, err := h.alertService.CalculateHealthScore(c.Request.Context(), cabinetID)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, gin.H{
		"cabinet_id":   cabinetID,
		"health_score": score,
	})
}

// SyncAlerts 接收Edge端同步的告警数据
// @Summary 同步告警数据
// @Tags Alerts
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Param request body models.AlertSyncRequest true "告警同步数据"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/alerts/sync [post]
func (h *AlertHandler) SyncAlerts(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var request models.AlertSyncRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	// 确保请求中的cabinet_id与路径参数一致
	request.CabinetID = cabinetID

	if err := h.alertService.SyncAlerts(c.Request.Context(), &request); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrBadRequest {
			statusCode = http.StatusBadRequest
		} else if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "告警数据同步成功")
}

// BatchResolveAlerts 批量解决告警
// @Summary 批量解决告警
// @Tags Alerts
// @Accept json
// @Produce json
// @Param request body models.BatchResolveRequest true "告警ID列表"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/alerts/batch-resolve [post]
func (h *AlertHandler) BatchResolveAlerts(c *gin.Context) {
	var request models.BatchResolveRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if len(request.AlertIDs) == 0 {
		utils.ValidationError(c, "告警ID列表不能为空")
		return
	}

	resolvedBy := "admin"
	if user, exists := c.Get("user_id"); exists {
		if username, ok := c.Get("username"); ok {
			resolvedBy = username.(string)
		} else {
			// 如果没有username，使用user_id转换为字符串
			resolvedBy = fmt.Sprintf("user_%d", user.(int))
		}
	}

	if err := h.alertService.BatchResolveAlerts(c.Request.Context(), request.AlertIDs, resolvedBy); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrDatabaseQuery || appErr.Code == errors.ErrInternalServer {
			statusCode = http.StatusInternalServerError
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, gin.H{
		"resolved_count": len(request.AlertIDs),
	}, "批量解决告警成功")
}
