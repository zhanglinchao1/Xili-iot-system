package handlers

import (
	"net/http"

	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// MapHandler 地图处理器
type MapHandler struct {
	mapService services.MapService
}

// NewMapHandler 创建地图处理器实例
func NewMapHandler(mapService services.MapService) *MapHandler {
	return &MapHandler{
		mapService: mapService,
	}
}

// GeocodeRequest 地理编码请求
type GeocodeRequest struct {
	Address string `json:"address" binding:"required"`
}

// Geocode 地理编码：将地址转换为经纬度
// @Summary 地理编码
// @Tags Map
// @Accept json
// @Produce json
// @Param request body GeocodeRequest true "地理编码请求"
// @Success 200 {object} utils.SuccessResponse{data=services.GeocodeResult}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/map/geocode [post]
func (h *MapHandler) Geocode(c *gin.Context) {
	var req GeocodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	result, err := h.mapService.Geocode(c.Request.Context(), req.Address)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrBadRequest {
			statusCode = http.StatusBadRequest
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, result)
}

// SearchPlaceRequest 地点搜索请求
type SearchPlaceRequest struct {
	Keyword string `json:"keyword" binding:"required"`
	Region  string `json:"region,omitempty"`
}

// SearchPlace 地点搜索：根据关键词搜索地点
// @Summary 地点搜索
// @Tags Map
// @Accept json
// @Produce json
// @Param request body SearchPlaceRequest true "地点搜索请求"
// @Success 200 {object} utils.SuccessResponse{data=[]services.PlaceSuggestion}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/map/search [post]
func (h *MapHandler) SearchPlace(c *gin.Context) {
	var req SearchPlaceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	results, err := h.mapService.SearchPlace(c.Request.Context(), req.Keyword, req.Region)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrBadRequest {
			statusCode = http.StatusBadRequest
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, results)
}

// SearchLocationGET 地点搜索 (GET请求)
// @Summary 地点搜索
// @Tags Map
// @Param keyword query string true "搜索关键词"
// @Param region query string false "限定区域"
// @Success 200 {object} utils.SuccessResponse{data=object{results=[]services.PlaceSuggestion}}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/map/search [get]
func (h *MapHandler) SearchLocationGET(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		utils.ValidationError(c, "搜索关键词不能为空")
		return
	}

	region := c.DefaultQuery("region", "")

	results, err := h.mapService.SearchPlace(c.Request.Context(), keyword, region)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrBadRequest {
			statusCode = http.StatusBadRequest
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, gin.H{
		"results": results,
	})
}

// ReverseGeocodeGET 逆地理编码 (GET请求)
// @Summary 逆地理编码
// @Tags Map
// @Param latitude query number true "纬度"
// @Param longitude query number true "经度"
// @Success 200 {object} utils.SuccessResponse{data=object{title=string,address=string}}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/map/geocode/reverse [get]
func (h *MapHandler) ReverseGeocodeGET(c *gin.Context) {
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")

	if latitude == "" || longitude == "" {
		utils.ValidationError(c, "经纬度不能为空")
		return
	}

	// 调用地理编码服务的逆向功能
	result, err := h.mapService.ReverseGeocode(c.Request.Context(), latitude, longitude)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusInternalServerError
		if appErr.Code == errors.ErrBadRequest {
			statusCode = http.StatusBadRequest
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.Success(c, result)
}
