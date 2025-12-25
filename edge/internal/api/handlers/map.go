package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MapSearchRequest 地点搜索请求
type MapSearchRequest struct {
	Keyword string `json:"keyword" binding:"required"` // 搜索关键词
	Region  string `json:"region"`                     // 搜索区域(可选)
}

// MapSearchResponse 腾讯地图搜索响应
type MapSearchResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Data    []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Address  string `json:"address"`
		Category string `json:"category"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		AdInfo struct {
			Adcode   string `json:"adcode"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"ad_info"`
	} `json:"data"`
}

// SearchPlace 腾讯地图地点搜索接口
// 代理腾讯地图WebService API的地点搜索功能
func SearchPlace(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查地图功能是否启用
		if !cfg.Map.Enabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "MAP_DISABLED",
				"message": "地图功能未启用",
			})
			return
		}

		// 检查API密钥是否配置
		if cfg.Map.TencentMapKey == "" {
			logger.Error("腾讯地图API密钥未配置")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "MAP_KEY_NOT_CONFIGURED",
				"message": "地图API密钥未配置",
			})
			return
		}

		// 解析请求
		var req MapSearchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 构建腾讯地图WebService API URL
		// 使用地点搜索API: https://lbs.qq.com/service/webService/webServiceGuide/webServiceSuggestion
		region := req.Region
		if region == "" {
			region = "全国" // 默认全国范围
		}

		apiURL := fmt.Sprintf("https://apis.map.qq.com/ws/place/v1/suggestion?keyword=%s&region=%s&key=%s&output=json",
			url.QueryEscape(req.Keyword),
			url.QueryEscape(region),
			cfg.Map.TencentMapKey,
		)

		// 调用腾讯地图API
		logger.Info("调用腾讯地图搜索API",
			zap.String("keyword", req.Keyword),
			zap.String("region", region),
		)

		resp, err := http.Get(apiURL)
		if err != nil {
			logger.Error("调用腾讯地图API失败",
				zap.Error(err),
				zap.String("keyword", req.Keyword),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "MAP_API_ERROR",
				"message": "地图API调用失败",
			})
			return
		}
		defer resp.Body.Close()

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("读取地图API响应失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "READ_RESPONSE_ERROR",
				"message": "读取响应失败",
			})
			return
		}

		// 解析响应并转换格式
		var tencentResp struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
			Count   int    `json:"count"`
			Data    []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Address  string `json:"address"`
				Category string `json:"category"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				AdInfo struct {
					Adcode   string `json:"adcode"`
					Province string `json:"province"`
					City     string `json:"city"`
					District string `json:"district"`
				} `json:"ad_info"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &tencentResp); err != nil {
			logger.Error("解析地图API响应失败",
				zap.Error(err),
				zap.String("body", string(body)),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "PARSE_RESPONSE_ERROR",
				"message": "解析响应失败",
			})
			return
		}

		// 检查腾讯地图API返回状态
		if tencentResp.Status != 0 {
			logger.Warn("腾讯地图API返回错误",
				zap.Int("status", tencentResp.Status),
				zap.String("message", tencentResp.Message),
			)
			c.JSON(http.StatusOK, gin.H{
				"status":  tencentResp.Status,
				"message": tencentResp.Message,
				"count":   0,
				"data":    []interface{}{},
			})
			return
		}

		// 返回成功响应
		logger.Info("地图搜索成功",
			zap.String("keyword", req.Keyword),
			zap.Int("count", tencentResp.Count),
		)

		c.JSON(http.StatusOK, gin.H{
			"status":  tencentResp.Status,
			"message": tencentResp.Message,
			"count":   tencentResp.Count,
			"data":    tencentResp.Data,
		})
	}
}
