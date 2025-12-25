/*
 * 储能柜自动激活服务
 * 负责在Edge端启动时自动激活储能柜
 */
package activation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/storage"
	"go.uber.org/zap"
)

// StorageInterface 存储接口(用于依赖注入)
type StorageInterface interface {
	SaveCloudCredentials(cabinetID, apiKey, apiSecret, cloudEndpoint string) error
	GetFirstCloudCredentials() (*storage.CloudCredential, error)
	GetCloudCredentials(cabinetID string) (*storage.CloudCredential, error)
}

// Service 激活服务
type Service struct {
	config  *config.Config
	storage StorageInterface
	logger  *zap.Logger
	client  *http.Client
}

// NewService 创建激活服务实例
func NewService(cfg *config.Config, storage StorageInterface, logger *zap.Logger) *Service {
	return &Service{
		config:  cfg,
		storage: storage,
		logger:  logger,
		client: &http.Client{
			Timeout: cfg.Cloud.Timeout,
		},
	}
}

// getEndpoint 获取Cloud端点（优先从数据库读取）
func (s *Service) getEndpoint() string {
	if s.storage != nil {
		cabinetID := s.config.Cloud.CabinetID
		if cred, err := s.storage.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
			return cred.CloudEndpoint
		}
		if cred, err := s.storage.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
			return cred.CloudEndpoint
		}
	}
	return s.config.Cloud.Endpoint
}

// ActivateRequest 激活请求
type ActivateRequest struct {
	RegistrationToken string `json:"registration_token"`
	MACAddress        string `json:"mac_address"`
}

// ActivateResponse 激活响应
type ActivateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    struct {
		CabinetID string `json:"cabinet_id"`
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
	} `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// TryAutoActivate 尝试自动激活储能柜
func (s *Service) TryAutoActivate(ctx context.Context) error {
	// 检查是否启用自动激活
	if !s.config.Registration.Enabled {
		s.logger.Info("自动激活未启用，跳过")
		return nil
	}

	// 检查是否已经有API Key (已激活)
	if s.config.Cloud.APIKey != "" {
		s.logger.Info("储能柜已激活，跳过自动激活",
			zap.String("api_key", maskAPIKey(s.config.Cloud.APIKey)))
		return nil
	}

	// 检查注册Token是否存在
	if s.config.Registration.Token == "" {
		s.logger.Warn("注册Token为空，无法自动激活")
		return fmt.Errorf("注册Token为空")
	}

	// 检查MAC地址是否存在
	if s.config.Registration.MACAddress == "" {
		s.logger.Warn("MAC地址为空，无法自动激活")
		return fmt.Errorf("MAC地址为空")
	}

	s.logger.Info("开始自动激活储能柜",
		zap.String("mac_address", s.config.Registration.MACAddress))

	// 调用激活API
	apiKey, apiSecret, err := s.activateCabinet(ctx)
	if err != nil {
		s.logger.Error("激活失败", zap.Error(err))
		return err
	}

	// 保存API Key到数据库(而不是配置文件)
	cabinetID := s.config.Cloud.CabinetID
	if cabinetID == "" {
		// 如果配置中没有cabinet_id,使用config中的registration信息
		cabinetID = "default" // 默认cabinet_id
	}

	// 保存时使用当前有效的endpoint（可能来自数据库）
	endpoint := s.getEndpoint()
	if err := s.storage.SaveCloudCredentials(cabinetID, apiKey, apiSecret, endpoint); err != nil {
		s.logger.Error("保存API凭证到数据库失败", zap.Error(err))
		return fmt.Errorf("保存API凭证失败: %w", err)
	}

	s.logger.Info("储能柜激活成功",
		zap.String("api_key", maskAPIKey(apiKey)),
		zap.String("cabinet_id", cabinetID))

	// 打印重要提示
	if apiSecret != "" {
		s.logger.Warn("⚠️  重要提示: API Secret 请妥善保管",
			zap.String("api_secret", apiSecret))
	}
	s.logger.Info("API凭证已保存到数据库，可立即使用无需重启服务")

	return nil
}

// activateCabinet 调用Cloud端激活API
func (s *Service) activateCabinet(ctx context.Context) (apiKey, apiSecret string, err error) {
	// 构建激活URL（优先从数据库获取endpoint）
	endpoint := s.getEndpoint()
	activateURL := fmt.Sprintf("%s/cabinets/activate", endpoint)

	// 构建请求体
	reqBody := ActivateRequest{
		RegistrationToken: s.config.Registration.Token,
		MACAddress:        s.config.Registration.MACAddress,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", activateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	s.logger.Info("发送激活请求", zap.String("url", activateURL))
	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("发送激活请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var activateResp ActivateResponse
	if err := json.Unmarshal(body, &activateResp); err != nil {
		return "", "", fmt.Errorf("解析响应失败: %w, body: %s", err, string(body))
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		errMsg := activateResp.Error
		if errMsg == "" {
			errMsg = activateResp.Message
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "", "", fmt.Errorf("激活失败: %s", errMsg)
	}

	if !activateResp.Success {
		return "", "", fmt.Errorf("激活失败: %s", activateResp.Error)
	}

	// 提取API凭证
	apiKey = activateResp.Data.APIKey
	apiSecret = activateResp.Data.APISecret

	if apiKey == "" || apiSecret == "" {
		return "", "", fmt.Errorf("激活响应缺少API凭证")
	}

	return apiKey, apiSecret, nil
}

// maskAPIKey 脱敏显示API Key
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// CheckActivationStatus 检查激活状态
func (s *Service) CheckActivationStatus() (activated bool, message string) {
	if s.config.Cloud.APIKey != "" {
		return true, fmt.Sprintf("已激活 (API Key: %s)", maskAPIKey(s.config.Cloud.APIKey))
	}

	if s.config.Registration.Enabled && s.config.Registration.Token != "" {
		return false, "待激活 (已配置注册Token)"
	}

	return false, "未激活 (无API Key或注册Token)"
}

// GetActivationInfo 获取激活信息 (用于API返回)
func (s *Service) GetActivationInfo() map[string]interface{} {
	activated, message := s.CheckActivationStatus()

	info := map[string]interface{}{
		"activated":        activated,
		"status_message":   message,
		"cabinet_id":       s.config.Cloud.CabinetID,
		"cloud_endpoint":   s.getEndpoint(), // 优先从数据库获取endpoint
		"auto_activation":  s.config.Registration.Enabled,
	}

	if s.config.Cloud.APIKey != "" {
		info["api_key"] = maskAPIKey(s.config.Cloud.APIKey)
	}

	return info
}
