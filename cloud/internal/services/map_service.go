package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"cloud-system/internal/config"
	"cloud-system/pkg/errors"

	"go.uber.org/zap"
)

// MapService 地图服务接口
type MapService interface {
	// Geocode 地理编码：将地址转换为经纬度
	Geocode(ctx context.Context, address string) (*GeocodeResult, error)
	// SearchPlace 地点搜索：根据关键词搜索地点
	SearchPlace(ctx context.Context, keyword string, region string) ([]*PlaceSuggestion, error)
	// ReverseGeocode 逆地理编码：将经纬度转换为地址
	ReverseGeocode(ctx context.Context, latitude string, longitude string) (map[string]interface{}, error)
}

// mapService 地图服务实现
type mapService struct {
	config *config.Config
	logger *zap.Logger
}

// NewMapService 创建地图服务实例
func NewMapService(cfg *config.Config, logger *zap.Logger) MapService {
	return &mapService{
		config: cfg,
		logger: logger,
	}
}

// GeocodeResult 地理编码结果
type GeocodeResult struct {
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Province  string  `json:"province,omitempty"`
	City      string  `json:"city,omitempty"`
	District  string  `json:"district,omitempty"`
}

// PlaceSuggestion 地点搜索建议
type PlaceSuggestion struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Province  string  `json:"province,omitempty"`
	City      string  `json:"city,omitempty"`
	District  string  `json:"district,omitempty"`
}

// TencentGeocodeResponse 腾讯地理编码API响应
type TencentGeocodeResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Title    string `json:"title"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		AdInfo struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"ad_info"`
		AddressComponents struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"address_components"`
	} `json:"result"`
}

// Geocode 地理编码：将地址转换为经纬度
func (s *mapService) Geocode(ctx context.Context, address string) (*GeocodeResult, error) {
	if address == "" {
		return nil, errors.New(errors.ErrBadRequest, "地址不能为空")
	}

	// 获取配置
	mapKey := s.config.Business.Map.TencentMapKey
	webServiceKey := s.config.Business.Map.TencentWebServiceKey

	if mapKey == "" {
		return nil, errors.New(errors.ErrInternalServer, "地图服务未配置")
	}

	// 智能地址补全：如果地址太短或不包含城市信息，自动添加默认区域
	// 这样可以避免腾讯地图API返回348参数错误
	fullAddress := s.enhanceAddress(address)

	// 构建请求参数
	params := map[string]string{
		"address": fullAddress,
		"key":     mapKey,
	}

	// 生成签名(如果配置了webservice_key)
	if webServiceKey != "" {
		sig := s.generateSignature(params, webServiceKey)
		params["sig"] = sig
	}

	// 构建URL
	baseURL := "https://apis.map.qq.com/ws/geocoder/v1/"
	reqURL := s.buildURL(baseURL, params)

	s.logger.Info("Geocode request",
		zap.String("address", address),
		zap.String("url", reqURL),
	)

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		s.logger.Error("Failed to create geocode request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "创建地理编码请求失败")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send geocode request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "发送地理编码请求失败")
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read geocode response", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "读取地理编码响应失败")
	}

	s.logger.Info("Geocode response",
		zap.String("body", string(body)),
	)

	// 解析响应
	var tencentResp TencentGeocodeResponse
	if err := json.Unmarshal(body, &tencentResp); err != nil {
		s.logger.Error("Failed to parse geocode response",
			zap.Error(err),
			zap.String("body", string(body)),
		)
		return nil, errors.Wrap(err, errors.ErrInternalServer, "解析地理编码响应失败")
	}

	// 检查状态
	if tencentResp.Status != 0 {
		s.logger.Error("Geocode API error",
			zap.Int("status", tencentResp.Status),
			zap.String("message", tencentResp.Message),
		)
		return nil, errors.New(errors.ErrExternalService, fmt.Sprintf("地理编码失败: %s", tencentResp.Message))
	}

	// 构建结果
	result := &GeocodeResult{
		Address:   tencentResp.Result.Title,
		Latitude:  tencentResp.Result.Location.Lat,
		Longitude: tencentResp.Result.Location.Lng,
		Province:  tencentResp.Result.AddressComponents.Province,
		City:      tencentResp.Result.AddressComponents.City,
		District:  tencentResp.Result.AddressComponents.District,
	}

	s.logger.Info("Geocode success",
		zap.String("address", address),
		zap.Float64("latitude", result.Latitude),
		zap.Float64("longitude", result.Longitude),
	)

	return result, nil
}

// generateSignature 生成腾讯地图WebService API签名
func (s *mapService) generateSignature(params map[string]string, sk string) string {
	// 按key升序排列参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	paramStr := strings.Join(parts, "&")

	// 拼接请求路径和参数
	signStr := "/ws/geocoder/v1/?" + paramStr + sk

	// 计算MD5
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// TencentPlaceSearchResponse 腾讯地点搜索API响应
type TencentPlaceSearchResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Data    []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Address  string `json:"address"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		AdInfo struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"ad_info"`
	} `json:"data"`
}

// SearchPlace 地点搜索：根据关键词搜索地点
func (s *mapService) SearchPlace(ctx context.Context, keyword string, region string) ([]*PlaceSuggestion, error) {
	if keyword == "" {
		return nil, errors.New(errors.ErrBadRequest, "搜索关键词不能为空")
	}

	// 获取配置
	mapKey := s.config.Business.Map.TencentMapKey
	webServiceKey := s.config.Business.Map.TencentWebServiceKey

	if mapKey == "" {
		return nil, errors.New(errors.ErrInternalServer, "地图服务未配置")
	}

	// 构建请求参数
	params := map[string]string{
		"keyword": keyword,
		"key":     mapKey,
	}
	// 腾讯地图Place Search API需要boundary参数(必填)
	// 使用region格式进行城市搜索: boundary=region(city_name,auto_extend)
	if region != "" {
		// auto_extend: 0=仅当前城市, 1=自动扩大范围
		params["boundary"] = fmt.Sprintf("region(%s,0)", region)
	} else {
		// 如果没有指定城市,搜索全国范围(使用中国的边界)
		params["boundary"] = "region(全国,1)"
	}

	// 生成签名(如果配置了webservice_key)
	if webServiceKey != "" {
		sig := s.generatePlaceSearchSignature(params, webServiceKey)
		params["sig"] = sig
	}

	// 构建URL
	baseURL := "https://apis.map.qq.com/ws/place/v1/search"
	reqURL := s.buildURL(baseURL, params)

	s.logger.Info("Place search request",
		zap.String("keyword", keyword),
		zap.String("region", region),
		zap.String("url", reqURL),
	)

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		s.logger.Error("Failed to create place search request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "创建地点搜索请求失败")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send place search request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "发送地点搜索请求失败")
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read place search response", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "读取地点搜索响应失败")
	}

	s.logger.Info("Place search response",
		zap.String("body", string(body)),
	)

	// 解析响应
	var tencentResp TencentPlaceSearchResponse
	if err := json.Unmarshal(body, &tencentResp); err != nil {
		s.logger.Error("Failed to parse place search response",
			zap.Error(err),
			zap.String("body", string(body)),
		)
		return nil, errors.Wrap(err, errors.ErrInternalServer, "解析地点搜索响应失败")
	}

	// 检查状态
	if tencentResp.Status != 0 {
		s.logger.Error("Place search API error",
			zap.Int("status", tencentResp.Status),
			zap.String("message", tencentResp.Message),
		)
		return nil, errors.New(errors.ErrExternalService, fmt.Sprintf("地点搜索失败: %s", tencentResp.Message))
	}

	// 构建结果
	results := make([]*PlaceSuggestion, 0, len(tencentResp.Data))
	for _, item := range tencentResp.Data {
		results = append(results, &PlaceSuggestion{
			ID:        item.ID,
			Title:     item.Title,
			Address:   item.Address,
			Latitude:  item.Location.Lat,
			Longitude: item.Location.Lng,
			Province:  item.AdInfo.Province,
			City:      item.AdInfo.City,
			District:  item.AdInfo.District,
		})
	}

	s.logger.Info("Place search success",
		zap.String("keyword", keyword),
		zap.Int("count", len(results)),
	)

	return results, nil
}

// generatePlaceSearchSignature 生成地点搜索API签名
func (s *mapService) generatePlaceSearchSignature(params map[string]string, sk string) string {
	// 按key升序排列参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	paramStr := strings.Join(parts, "&")

	// 拼接请求路径和参数
	signStr := "/ws/place/v1/search?" + paramStr + sk

	// 计算MD5
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// buildURL 构建请求URL
func (s *mapService) buildURL(baseURL string, params map[string]string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// ReverseGeocodeResponse 逆地理编码响应
type ReverseGeocodeResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Address          string `json:"address"`
		FormattedAddress struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Street   string `json:"street"`
		} `json:"formatted_addresses"`
		AddressComponent struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Street   string `json:"street"`
		} `json:"address_component"`
	} `json:"result"`
}

// ReverseGeocode 逆地理编码：将经纬度转换为地址
func (s *mapService) ReverseGeocode(ctx context.Context, latitude string, longitude string) (map[string]interface{}, error) {
	if latitude == "" || longitude == "" {
		return nil, errors.New(errors.ErrBadRequest, "经纬度不能为空")
	}

	// 获取配置
	mapKey := s.config.Business.Map.TencentMapKey
	webServiceKey := s.config.Business.Map.TencentWebServiceKey

	if mapKey == "" {
		return nil, errors.New(errors.ErrInternalServer, "地图服务未配置")
	}

	// 构建请求参数
	location := fmt.Sprintf("%s,%s", latitude, longitude)
	params := map[string]string{
		"location": location,
		"key":      mapKey,
	}

	// 生成签名(如果配置了webservice_key)
	if webServiceKey != "" {
		sig := s.generateReverseGeocodeSignature(params, webServiceKey)
		params["sig"] = sig
	}

	// 构建URL
	baseURL := "https://apis.map.qq.com/ws/geocoder/v1/"
	reqURL := s.buildURL(baseURL, params)

	s.logger.Info("Reverse geocode request",
		zap.String("latitude", latitude),
		zap.String("longitude", longitude),
		zap.String("url", reqURL),
	)

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		s.logger.Error("Failed to create reverse geocode request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "创建逆地理编码请求失败")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send reverse geocode request", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "发送逆地理编码请求失败")
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read reverse geocode response", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "读取逆地理编码响应失败")
	}

	s.logger.Info("Reverse geocode response",
		zap.String("body", string(body)),
	)

	// 解析响应
	var tencentResp ReverseGeocodeResponse
	if err := json.Unmarshal(body, &tencentResp); err != nil {
		s.logger.Error("Failed to parse reverse geocode response",
			zap.Error(err),
			zap.String("body", string(body)),
		)
		return nil, errors.Wrap(err, errors.ErrInternalServer, "解析逆地理编码响应失败")
	}

	// 检查状态
	if tencentResp.Status != 0 {
		s.logger.Error("Reverse geocode API error",
			zap.Int("status", tencentResp.Status),
			zap.String("message", tencentResp.Message),
		)
		return nil, errors.New(errors.ErrExternalService, fmt.Sprintf("逆地理编码失败: %s", tencentResp.Message))
	}

	// 构建结果
	address := tencentResp.Result.Address
	title := address

	// 尝试构建更友好的标题
	addrComp := tencentResp.Result.AddressComponent
	if addrComp.Province != "" || addrComp.City != "" || addrComp.District != "" {
		title = fmt.Sprintf("%s%s%s%s",
			addrComp.Province,
			addrComp.City,
			addrComp.District,
			addrComp.Street,
		)
	}

	result := map[string]interface{}{
		"title":   title,
		"address": address,
	}

	s.logger.Info("Reverse geocode success",
		zap.String("latitude", latitude),
		zap.String("longitude", longitude),
		zap.String("address", address),
	)

	return result, nil
}

// generateReverseGeocodeSignature 生成逆地理编码API签名
func (s *mapService) generateReverseGeocodeSignature(params map[string]string, sk string) string {
	// 按key升序排列参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	paramStr := strings.Join(parts, "&")

	// 拼接请求路径和参数
	signStr := "/ws/geocoder/v1/?" + paramStr + sk

	// 计算MD5
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// enhanceAddress 智能地址补全
// 如果地址太短或不包含城市/省份信息，自动添加默认区域前缀
// 这样可以避免腾讯地图API返回348"参数错误"
func (s *mapService) enhanceAddress(address string) string {
	// 常见的省份和城市关键词
	locationKeywords := []string{
		"省", "市", "区", "县", "镇", "街道", "路", "街",
		"北京", "上海", "天津", "重庆",
		"杭州", "广州", "深圳", "南京", "苏州", "成都", "武汉",
	}

	// 检查地址是否已经包含位置关键词
	hasLocationInfo := false
	for _, keyword := range locationKeywords {
		if strings.Contains(address, keyword) {
			hasLocationInfo = true
			break
		}
	}

	// 如果地址已经包含城市信息，直接返回
	if hasLocationInfo {
		return address
	}

	// 否则，添加默认城市前缀（从配置中获取）
	// 默认使用"杭州市"作为前缀，因为系统主要部署在杭州
	defaultCity := "杭州市"

	s.logger.Debug("Address enhanced",
		zap.String("original", address),
		zap.String("enhanced", defaultCity+address),
	)

	return defaultCity + address
}
