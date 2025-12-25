package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CabinetHandler 储能柜处理器
type CabinetHandler struct {
	cabinetService services.CabinetService
}

// NewCabinetHandler 创建储能柜处理器实例
func NewCabinetHandler(cabinetService services.CabinetService) *CabinetHandler {
	return &CabinetHandler{
		cabinetService: cabinetService,
	}
}

// CreateCabinet 创建储能柜
// @Summary 创建储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param request body models.CreateCabinetInput true "创建储能柜请求"
// @Success 201 {object} utils.SuccessResponse{data=models.Cabinet}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /api/v1/cabinets [post]
func (h *CabinetHandler) CreateCabinet(c *gin.Context) {
	var input models.CreateCabinetInput

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	cabinet, err := h.cabinetService.CreateCabinet(c.Request.Context(), &input)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, cabinet, "储能柜创建成功")
}

// GetCabinet 获取储能柜详情
// @Summary 获取储能柜详情
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Cabinet}
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id} [get]
func (h *CabinetHandler) GetCabinet(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	cabinet, err := h.cabinetService.GetCabinet(c.Request.Context(), cabinetID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, cabinet)
}

// ListCabinets 获取储能柜列表
// @Summary 获取储能柜列表
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param status query string false "状态过滤"
// @Param location query string false "位置过滤"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} utils.PaginatedResponse{data=[]models.Cabinet}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/cabinets [get]
func (h *CabinetHandler) ListCabinets(c *gin.Context) {
	var filter models.CabinetListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, "查询参数格式错误")
		return
	}

	cabinets, total, err := h.cabinetService.ListCabinets(c.Request.Context(), &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessPaginated(c, cabinets, filter.Page, filter.PageSize, total)
}

// UpdateCabinet 更新储能柜
// @Summary 更新储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Param request body models.UpdateCabinetInput true "更新储能柜请求"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id} [put]
func (h *CabinetHandler) UpdateCabinet(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var input models.UpdateCabinetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.cabinetService.UpdateCabinet(c.Request.Context(), cabinetID, &input); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "储能柜更新成功")
}

// DeleteCabinet 删除储能柜
// @Summary 删除储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id} [delete]
func (h *CabinetHandler) DeleteCabinet(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	if err := h.cabinetService.DeleteCabinet(c.Request.Context(), cabinetID); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, nil, "储能柜删除成功")
}

// PreRegisterCabinet 预注册储能柜
// @Summary 预注册储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param request body models.PreRegisterCabinetInput true "预注册储能柜请求"
// @Success 201 {object} utils.SuccessResponse{data=models.PreRegisterResponse}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/pre-register [post]
func (h *CabinetHandler) PreRegisterCabinet(c *gin.Context) {
	var input models.PreRegisterCabinetInput

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	response, err := h.cabinetService.PreRegisterCabinet(c.Request.Context(), &input)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrRecordExists {
			statusCode = http.StatusConflict
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, response, "储能柜预注册成功")
}

// GetActivationInfo 获取储能柜激活信息
// @Summary 获取储能柜激活信息
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} utils.SuccessResponse{data=models.ActivationInfoResponse}
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/activation-info [get]
func (h *CabinetHandler) GetActivationInfo(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	info, err := h.cabinetService.GetActivationInfo(c.Request.Context(), cabinetID, c.Request)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, info)
}

// RegenerateToken 重新生成注册Token
// @Summary 重新生成注册Token
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} utils.SuccessResponse{data=models.RegenerateTokenResponse}
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/regenerate-token [post]
func (h *CabinetHandler) RegenerateToken(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	response, err := h.cabinetService.RegenerateToken(c.Request.Context(), cabinetID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessWithMessage(c, response, "Token重新生成成功")
}

// ActivateCabinet Edge端激活储能柜
// @Summary Edge端激活储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param request body models.ActivateCabinetInput true "激活储能柜请求"
// @Success 200 {object} utils.SuccessResponse{data=models.ActivateCabinetResponse}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/activate [post]
func (h *CabinetHandler) ActivateCabinet(c *gin.Context) {
	var input models.ActivateCabinetInput

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	response, err := h.cabinetService.ActivateCabinet(c.Request.Context(), &input)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, response, "储能柜激活成功")
}

// RegisterCabinet Edge端直接注册储能柜（一步完成注册和激活）
// @Summary Edge端直接注册储能柜
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param request body models.RegisterCabinetInput true "注册储能柜请求"
// @Success 201 {object} utils.SuccessResponse{data=models.RegisterCabinetResponse}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/register [post]
func (h *CabinetHandler) RegisterCabinet(c *gin.Context) {
	utils.Info("RegisterCabinet handler called")
	var input models.RegisterCabinetInput

	// 直接读取JSON body并解析，避免gin binding对指针类型字段的验证问题
	body, err := io.ReadAll(c.Request.Body)
	utils.Info("Request body read", zap.String("body", string(body)), zap.Error(err))
	if err != nil {
		utils.Error("Failed to read request body", zap.Error(err))
		utils.ErrorResponse(c, http.StatusBadRequest, errors.NewValidationError("无法读取请求数据"))
		return
	}

	// 解析JSON
	if err := json.Unmarshal(body, &input); err != nil {
		utils.Error("JSON unmarshal error", zap.String("error", err.Error()), zap.String("body", string(body)))
		utils.ErrorResponse(c, http.StatusBadRequest, errors.NewValidationError(fmt.Sprintf("请求参数格式错误: %v", err)))
		return
	}

	// 手动验证cabinet_id（唯一必填字段）
	if input.CabinetID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, errors.NewValidationError("cabinet_id不能为空"))
		return
	}

	response, err := h.cabinetService.RegisterCabinet(c.Request.Context(), &input)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrRecordExists {
			statusCode = http.StatusConflict
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "储能柜注册成功",
		"data":    response,
	})
}

// SyncCabinetInfo Edge端同步储能柜信息（不需要JWT认证）
// @Summary Edge端同步储能柜信息
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Param request body models.UpdateCabinetInput true "储能柜信息"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/sync [put]
func (h *CabinetHandler) SyncCabinetInfo(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var input models.UpdateCabinetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.cabinetService.UpdateCabinet(c.Request.Context(), cabinetID, &input); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "储能柜信息同步成功")
}

// GetCabinetLocations 获取所有储能柜位置信息（用于地图展示）
// @Summary 获取所有储能柜位置信息
// @Tags Cabinet
// @Accept json
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=[]models.CabinetLocation}
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/locations [get]
func (h *CabinetHandler) GetCabinetLocations(c *gin.Context) {
	locations, err := h.cabinetService.GetLocations(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.(*errors.AppError))
		return
	}

	utils.Success(c, locations)
}

// GetCabinetStatistics 获取储能柜统计信息
// @Summary 获取储能柜统计信息
// @Tags Cabinet
// @Accept json
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=models.CabinetStatistics}
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/statistics [get]
func (h *CabinetHandler) GetCabinetStatistics(c *gin.Context) {
	stats, err := h.cabinetService.GetStatistics(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.(*errors.AppError))
		return
	}

	utils.Success(c, stats)
}

// GetAPIKey 获取储能柜的API Key信息（脱敏显示）
// @Summary 获取API Key信息
// @Description 获取储能柜的API Key（脱敏显示，不返回Secret）
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} map[string]interface{} "API Key信息"
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/api-key [get]
func (h *CabinetHandler) GetAPIKey(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	if cabinetID == "" {
		utils.BadRequest(c, "储能柜ID不能为空")
		return
	}

	info, err := h.cabinetService.GetAPIKeyInfo(c.Request.Context(), cabinetID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			utils.ErrorResponse(c, http.StatusNotFound, appErr)
		} else {
			utils.InternalServerError(c, "获取API Key信息失败")
		}
		return
	}

	utils.Success(c, info)
}

// RegenerateAPIKey 重新生成API Key和Secret
// @Summary 重新生成API Key
// @Description 重新生成储能柜的API Key和Secret（仅此一次返回完整Secret）
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} map[string]string "新的API凭证"
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/api-key/regenerate [post]
func (h *CabinetHandler) RegenerateAPIKey(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	if cabinetID == "" {
		utils.BadRequest(c, "储能柜ID不能为空")
		return
	}

	credentials, err := h.cabinetService.RegenerateAPIKey(c.Request.Context(), cabinetID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			if appErr.Code == errors.ErrBadRequest {
				utils.BadRequest(c, appErr.Message)
			} else if appErr.Code == errors.ErrNotFound {
				utils.ErrorResponse(c, http.StatusNotFound, appErr)
			} else {
				utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
			}
		} else {
			utils.InternalServerError(c, "重新生成API Key失败")
		}
		return
	}

	utils.Success(c, credentials)
}

// RevokeAPIKey 撤销API Key
// @Summary 撤销API Key
// @Description 撤销储能柜的API Key和Secret（清空）
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Success 200 {object} map[string]string "撤销成功消息"
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/cabinets/{cabinet_id}/api-key [delete]
func (h *CabinetHandler) RevokeAPIKey(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")
	if cabinetID == "" {
		utils.BadRequest(c, "储能柜ID不能为空")
		return
	}

	err := h.cabinetService.RevokeAPIKey(c.Request.Context(), cabinetID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			if appErr.Code == errors.ErrBadRequest {
				utils.BadRequest(c, appErr.Message)
			} else if appErr.Code == errors.ErrNotFound {
				utils.ErrorResponse(c, http.StatusNotFound, appErr)
			} else {
				utils.ErrorResponse(c, http.StatusInternalServerError, appErr)
			}
		} else {
			utils.InternalServerError(c, "撤销API Key失败")
		}
		return
	}

	utils.Success(c, map[string]string{
		"message": "API Key已撤销",
	})
}
