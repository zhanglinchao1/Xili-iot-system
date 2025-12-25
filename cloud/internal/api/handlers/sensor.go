package handlers

import (
	"net/http"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// SensorHandler 传感器处理器
type SensorHandler struct {
	sensorService services.SensorService
}

// NewSensorHandler 创建传感器处理器实例
func NewSensorHandler(sensorService services.SensorService) *SensorHandler {
	return &SensorHandler{
		sensorService: sensorService,
	}
}

// SyncSensorData 同步传感器数据（Edge端调用）
// @Summary 同步传感器数据
// @Tags Sensor
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Param request body models.SyncDataRequest true "传感器数据"
// @Success 200 {object} utils.SuccessResponse{data=map[string]int}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/sync [post]
func (h *SensorHandler) SyncSensorData(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var request models.SyncDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	count, err := h.sensorService.SyncSensorData(c.Request.Context(), cabinetID, &request)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound || appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, gin.H{
		"synced_count": count,
	}, "数据同步成功")
}

// GetLatestSensorData 获取储能柜的最新传感器数据
// @Summary 获取最新传感器数据
// @Tags Sensor
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} utils.SuccessResponse{data=[]models.LatestSensorData}
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/sensors/latest [get]
func (h *SensorHandler) GetLatestSensorData(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	data, err := h.sensorService.GetLatestSensorData(c.Request.Context(), cabinetID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, data)
}

// ListCabinetDevices 获取储能柜的传感器设备列表
func (h *SensorHandler) ListCabinetDevices(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	devices, err := h.sensorService.ListDevices(c.Request.Context(), cabinetID)
	if err != nil {
		appErr := err.(*errors.AppError)
		status := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			status = http.StatusNotFound
		}
		utils.ErrorResponse(c, status, appErr)
		return
	}

	utils.Success(c, devices)
}

// GetHistoricalData 获取历史数据
// @Summary 获取历史数据
// @Tags Sensor
// @Accept json
// @Produce json
// @Param device_id query string true "设备ID"
// @Param start_time query string true "开始时间"
// @Param end_time query string true "结束时间"
// @Param aggregation query string false "聚合方式"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(100)
// @Success 200 {object} utils.PaginatedResponse{data=[]models.SensorData}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/devices/data [get]
func (h *SensorHandler) GetHistoricalData(c *gin.Context) {
	var query models.HistoricalDataQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ValidationError(c, "查询参数格式错误")
		return
	}

	// 如果有聚合方式，返回聚合数据
	if query.Aggregation != "" && query.Aggregation != "raw" {
		aggregatedData, err := h.sensorService.GetAggregatedData(c.Request.Context(), &query)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
			return
		}

		utils.Success(c, aggregatedData)
		return
	}

	// 否则返回原始数据
	data, total, err := h.sensorService.GetHistoricalData(c.Request.Context(), &query)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessPaginated(c, data, query.Page, query.PageSize, total)
}
