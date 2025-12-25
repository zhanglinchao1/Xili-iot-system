package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// CabinetService 储能柜服务接口
type CabinetService interface {
	// CreateCabinet 创建储能柜
	CreateCabinet(ctx context.Context, input *models.CreateCabinetInput) (*models.Cabinet, error)

	// GetCabinet 获取储能柜详情
	GetCabinet(ctx context.Context, cabinetID string) (*models.Cabinet, error)

	// ListCabinets 获取储能柜列表
	ListCabinets(ctx context.Context, filter *models.CabinetListFilter) ([]*models.Cabinet, int64, error)

	// UpdateCabinet 更新储能柜信息
	UpdateCabinet(ctx context.Context, cabinetID string, input *models.UpdateCabinetInput) error

	// DeleteCabinet 删除储能柜
	DeleteCabinet(ctx context.Context, cabinetID string) error

	// PreRegisterCabinet 预注册储能柜
	PreRegisterCabinet(ctx context.Context, input *models.PreRegisterCabinetInput) (*models.PreRegisterResponse, error)

	// GetActivationInfo 获取储能柜激活信息
	GetActivationInfo(ctx context.Context, cabinetID string, req interface{}) (*models.ActivationInfoResponse, error)

	// RegenerateToken 重新生成注册Token
	RegenerateToken(ctx context.Context, cabinetID string) (*models.RegenerateTokenResponse, error)

	// ActivateCabinet Edge端激活储能柜
	ActivateCabinet(ctx context.Context, input *models.ActivateCabinetInput) (*models.ActivateCabinetResponse, error)

	// RegisterCabinet Edge端直接注册储能柜（一步完成注册和激活）
	RegisterCabinet(ctx context.Context, input *models.RegisterCabinetInput) (*models.RegisterCabinetResponse, error)

	// GetLocations 获取所有储能柜位置信息（用于地图展示）
	GetLocations(ctx context.Context) ([]*models.CabinetLocation, error)

	// GetStatistics 获取储能柜统计信息
	GetStatistics(ctx context.Context) (*models.CabinetStatistics, error)

	// GetAPIKeyInfo 获取API Key信息（脱敏显示）
	GetAPIKeyInfo(ctx context.Context, cabinetID string) (map[string]interface{}, error)

	// RegenerateAPIKey 重新生成API Key和Secret
	RegenerateAPIKey(ctx context.Context, cabinetID string) (map[string]string, error)

	// RevokeAPIKey 撤销API Key
	RevokeAPIKey(ctx context.Context, cabinetID string) error
}

// cabinetService 储能柜服务实现
type cabinetService struct {
	cabinetRepo    repository.CabinetRepository
	licenseService LicenseService
}

// NewCabinetService 创建储能柜服务实例
func NewCabinetService(cabinetRepo repository.CabinetRepository, licenseService LicenseService) CabinetService {
	return &cabinetService{
		cabinetRepo:    cabinetRepo,
		licenseService: licenseService,
	}
}

// CreateCabinet 创建储能柜
func (s *cabinetService) CreateCabinet(ctx context.Context, input *models.CreateCabinetInput) (*models.Cabinet, error) {
	// 验证输入
	if err := utils.ValidateCabinetID(input.CabinetID); err != nil {
		return nil, err
	}

	if err := utils.ValidateMACAddress(input.MACAddress); err != nil {
		return nil, err
	}

	if err := utils.ValidateRequired("name", input.Name); err != nil {
		return nil, err
	}

	// 检查储能柜是否已存在
	exists, err := s.cabinetRepo.Exists(ctx, input.CabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence",
			zap.String("cabinet_id", input.CabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	if exists {
		return nil, errors.New(errors.ErrRecordExists, "储能柜ID已存在")
	}

	// 检查MAC地址是否已被使用
	_, err = s.cabinetRepo.GetByMACAddress(ctx, input.MACAddress)
	if err == nil {
		return nil, errors.New(errors.ErrRecordExists, "MAC地址已被使用")
	} else if err.(*errors.AppError).Code != errors.ErrCabinetNotFound {
		// 如果不是"不存在"错误，说明是其他数据库错误
		utils.Error("Failed to check MAC address uniqueness",
			zap.String("mac_address", input.MACAddress),
			zap.Error(err),
		)
		return nil, err
	}

	// 创建储能柜对象
	cabinet := &models.Cabinet{
		CabinetID:   input.CabinetID,
		Name:        input.Name,
		Location:    input.Location,
		CapacityKwh: input.CapacityKwh,
		MACAddress:  input.MACAddress,
	}

	// 保存到数据库
	if err := s.cabinetRepo.Create(ctx, cabinet); err != nil {
		utils.Error("Failed to create cabinet",
			zap.String("cabinet_id", input.CabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	utils.Info("Cabinet created successfully",
		zap.String("cabinet_id", cabinet.CabinetID),
		zap.String("name", cabinet.Name),
	)

	s.ensureDefaultLicense(ctx, cabinet.CabinetID, nil, nil, nil)

	return cabinet, nil
}

// GetCabinet 获取储能柜详情
func (s *cabinetService) GetCabinet(ctx context.Context, cabinetID string) (*models.Cabinet, error) {
	// 验证输入
	if err := utils.ValidateCabinetID(cabinetID); err != nil {
		return nil, err
	}

	// 从数据库获取
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		utils.Warn("Failed to get cabinet",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	return cabinet, nil
}

// ListCabinets 获取储能柜列表
func (s *cabinetService) ListCabinets(ctx context.Context, filter *models.CabinetListFilter) ([]*models.Cabinet, int64, error) {
	// 验证和设置默认值
	if filter.Page < 1 {
		filter.Page = 1
	}

	var err error
	filter.PageSize, err = utils.ValidatePageSize(filter.PageSize, 100)
	if err != nil {
		return nil, 0, err
	}

	// 验证状态（如果提供）
	if filter.Status != nil && *filter.Status != "" {
		if err := utils.ValidateStatus(*filter.Status, models.ValidStatuses); err != nil {
			return nil, 0, err
		}
	}

	// 从数据库获取
	cabinets, total, err := s.cabinetRepo.List(ctx, filter)
	if err != nil {
		utils.Error("Failed to list cabinets", zap.Error(err))
		return nil, 0, err
	}

	return cabinets, total, nil
}

// UpdateCabinet 更新储能柜信息
func (s *cabinetService) UpdateCabinet(ctx context.Context, cabinetID string, input *models.UpdateCabinetInput) error {
	// 验证输入
	if err := utils.ValidateCabinetID(cabinetID); err != nil {
		return err
	}

	// 验证状态（如果提供）
	if input.Status != nil {
		if err := utils.ValidateStatus(*input.Status, models.ValidStatuses); err != nil {
			return err
		}
	}

	// 检查储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	if !exists {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	// 更新数据库
	if err := s.cabinetRepo.Update(ctx, cabinetID, input); err != nil {
		utils.Error("Failed to update cabinet",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("Cabinet updated successfully", zap.String("cabinet_id", cabinetID))

	return nil
}

// DeleteCabinet 删除储能柜
func (s *cabinetService) DeleteCabinet(ctx context.Context, cabinetID string) error {
	// 验证输入
	if err := utils.ValidateCabinetID(cabinetID); err != nil {
		return err
	}

	// 删除储能柜
	if err := s.cabinetRepo.Delete(ctx, cabinetID); err != nil {
		utils.Error("Failed to delete cabinet",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("Cabinet deleted successfully", zap.String("cabinet_id", cabinetID))

	return nil
}

// PreRegisterCabinet 预注册储能柜
func (s *cabinetService) PreRegisterCabinet(ctx context.Context, input *models.PreRegisterCabinetInput) (*models.PreRegisterResponse, error) {
	// 验证输入
	if err := utils.ValidateCabinetID(input.CabinetID); err != nil {
		return nil, err
	}

	if err := utils.ValidateMACAddress(input.MACAddress); err != nil {
		return nil, err
	}

	if err := utils.ValidateRequired("name", input.Name); err != nil {
		return nil, err
	}

	// 检查储能柜是否已存在
	exists, err := s.cabinetRepo.Exists(ctx, input.CabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence", zap.String("cabinet_id", input.CabinetID), zap.Error(err))
		return nil, err
	}

	if exists {
		return nil, errors.New(errors.ErrRecordExists, "储能柜ID已存在")
	}

	// 检查MAC地址是否已被使用
	_, err = s.cabinetRepo.GetByMACAddress(ctx, input.MACAddress)
	if err == nil {
		return nil, errors.New(errors.ErrRecordExists, "MAC地址已被使用")
	} else if err.(*errors.AppError).Code != errors.ErrCabinetNotFound {
		utils.Error("Failed to check MAC address uniqueness", zap.String("mac_address", input.MACAddress), zap.Error(err))
		return nil, err
	}

	// 生成注册Token (使用JWT，24小时有效期)
	tokenExpiresAt := time.Now().Add(24 * time.Hour)
	token, err := generateRegistrationToken(input.CabinetID, input.MACAddress, tokenExpiresAt)
	if err != nil {
		utils.Error("Failed to generate registration token", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "生成注册Token失败")
	}

	// 创建储能柜记录
	cabinet := &models.Cabinet{
		CabinetID:         input.CabinetID,
		Name:              input.Name,
		Location:          input.Location,
		CapacityKwh:       input.CapacityKwh,
		MACAddress:        input.MACAddress,
		RegistrationToken: &token,
		TokenExpiresAt:    &tokenExpiresAt,
		IPAddress:         input.IPAddress,
		DeviceModel:       input.DeviceModel,
		Notes:             input.Notes,
	}

	if err := s.cabinetRepo.Create(ctx, cabinet); err != nil {
		utils.Error("Failed to create cabinet", zap.String("cabinet_id", input.CabinetID), zap.Error(err))
		return nil, err
	}

	// TODO: 如果提供了许可证信息，创建许可证记录
	// if input.LicenseExpiresAt != nil { ... }

	utils.Info("Cabinet pre-registered successfully", zap.String("cabinet_id", input.CabinetID))

	var licenseExpiresAt *time.Time
	if input.LicenseExpiresAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *input.LicenseExpiresAt); err == nil {
			licenseExpiresAt = &parsed
		}
	}
	s.ensureDefaultLicense(ctx, input.CabinetID, licenseExpiresAt, input.Permissions, nil)

	return &models.PreRegisterResponse{
		CabinetID:         input.CabinetID,
		RegistrationToken: token,
		TokenExpiresAt:    tokenExpiresAt,
	}, nil
}

// GetActivationInfo 获取储能柜激活信息
func (s *cabinetService) GetActivationInfo(ctx context.Context, cabinetID string, req interface{}) (*models.ActivationInfoResponse, error) {
	// 验证输入
	if err := utils.ValidateCabinetID(cabinetID); err != nil {
		return nil, err
	}

	// 获取储能柜信息
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	// 检查是否已激活
	if cabinet.ActivationStatus == "activated" {
		return nil, errors.New(errors.ErrBadRequest, "储能柜已激活")
	}

	// 检查Token是否存在
	if cabinet.RegistrationToken == nil || *cabinet.RegistrationToken == "" {
		return nil, errors.New(errors.ErrBadRequest, "注册Token不存在")
	}

	// 检查Token是否过期
	tokenExpired := false
	if cabinet.TokenExpiresAt != nil && time.Now().After(*cabinet.TokenExpiresAt) {
		tokenExpired = true
	}

	// 构建Cloud API URL - 优先从配置读取，如果有HTTP请求则从请求中获取
	cfg := config.Get()
	cloudAPIURL := cfg.Business.Frontend.APIBaseURL // 从配置读取
	if cloudAPIURL == "" || cloudAPIURL == "/api/v1" {
		// 如果配置是相对路径，使用服务器配置构建完整URL
		cloudAPIURL = fmt.Sprintf("http://%s:%d/api/v1", cfg.Server.Host, cfg.Server.Port)
	}
	// 如果有HTTP请求，优先使用请求中的Host（更准确的外部地址）
	if httpReq, ok := req.(*http.Request); ok {
		scheme := "http"
		if httpReq.TLS != nil {
			scheme = "https"
		}
		cloudAPIURL = fmt.Sprintf("%s://%s/api/v1", scheme, httpReq.Host)
	}

	return &models.ActivationInfoResponse{
		CabinetID:         cabinet.CabinetID,
		Name:              cabinet.Name,
		MACAddress:        cabinet.MACAddress,
		RegistrationToken: *cabinet.RegistrationToken,
		TokenExpiresAt:    *cabinet.TokenExpiresAt,
		TokenExpired:      tokenExpired,
		CloudAPIURL:       cloudAPIURL,
	}, nil
}

// RegenerateToken 重新生成注册Token
func (s *cabinetService) RegenerateToken(ctx context.Context, cabinetID string) (*models.RegenerateTokenResponse, error) {
	// 验证输入
	if err := utils.ValidateCabinetID(cabinetID); err != nil {
		return nil, err
	}

	// 获取储能柜信息
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	// 检查是否已激活
	if cabinet.ActivationStatus == "activated" {
		return nil, errors.New(errors.ErrBadRequest, "储能柜已激活，无法重新生成Token")
	}

	// 生成新的Token
	tokenExpiresAt := time.Now().Add(24 * time.Hour)
	token, err := generateRegistrationToken(cabinet.CabinetID, cabinet.MACAddress, tokenExpiresAt)
	if err != nil {
		utils.Error("Failed to generate registration token", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "生成注册Token失败")
	}

	// 更新Token
	if err := s.cabinetRepo.UpdateRegistrationToken(ctx, cabinetID, token, tokenExpiresAt); err != nil {
		utils.Error("Failed to update registration token", zap.String("cabinet_id", cabinetID), zap.Error(err))
		return nil, err
	}

	utils.Info("Registration token regenerated successfully", zap.String("cabinet_id", cabinetID))

	return &models.RegenerateTokenResponse{
		RegistrationToken: token,
		TokenExpiresAt:    tokenExpiresAt,
	}, nil
}

func (s *cabinetService) ensureDefaultLicense(
	ctx context.Context,
	cabinetID string,
	expiresAt *time.Time,
	permissions []string,
	createdBy *string,
) {
	if s.licenseService == nil {
		return
	}

	if _, err := s.licenseService.GetLicense(ctx, cabinetID); err == nil {
		return
	} else {
		if appErr, ok := err.(*errors.AppError); !ok || appErr.Code != errors.ErrLicenseNotFound {
			utils.Warn("Failed to check license before provisioning",
				zap.String("cabinet_id", cabinetID),
				zap.Error(err),
			)
			return
		}
	}

	validDays := 90 // 默认许可证有效期90天
	if expiresAt != nil {
		diff := int(math.Ceil(time.Until(*expiresAt).Hours() / 24))
		if diff > 0 {
			validDays = diff
		}
	}

	provisionPermissions := permissions
	if len(provisionPermissions) == 0 {
		provisionPermissions = []string{"sensor:rw"}
	}

	maxDevices := 100 // 默认最大设备数100
	createdByVal := "system"
	if createdBy != nil && *createdBy != "" {
		createdByVal = *createdBy
	}

	if _, err := s.licenseService.CreateLicense(ctx, &models.CreateLicenseRequest{
		CabinetID:   cabinetID,
		ValidDays:   validDays,
		MaxDevices:  maxDevices,
		Permissions: provisionPermissions,
	}, createdByVal); err != nil {
		utils.Warn("Failed to auto provision license",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
	}
}

// ActivateCabinet Edge端激活储能柜
func (s *cabinetService) ActivateCabinet(ctx context.Context, input *models.ActivateCabinetInput) (*models.ActivateCabinetResponse, error) {
	// 验证输入
	if input.RegistrationToken == "" {
		return nil, errors.New(errors.ErrBadRequest, "注册Token不能为空")
	}

	if err := utils.ValidateMACAddress(input.MACAddress); err != nil {
		return nil, err
	}

	// 根据Token查找储能柜
	cabinet, err := s.cabinetRepo.GetByRegistrationToken(ctx, input.RegistrationToken)
	if err != nil {
		return nil, errors.New(errors.ErrBadRequest, "无效的注册Token")
	}

	// 验证MAC地址是否匹配
	if cabinet.MACAddress != input.MACAddress {
		utils.Warn("MAC address mismatch during activation",
			zap.String("cabinet_id", cabinet.CabinetID),
			zap.String("expected_mac", cabinet.MACAddress),
			zap.String("provided_mac", input.MACAddress),
		)
		return nil, errors.New(errors.ErrBadRequest, "MAC地址不匹配")
	}

	// 检查Token是否过期
	if cabinet.TokenExpiresAt != nil && time.Now().After(*cabinet.TokenExpiresAt) {
		return nil, errors.New(errors.ErrBadRequest, "注册Token已过期")
	}

	// 检查是否已激活
	if cabinet.ActivationStatus == "activated" {
		return nil, errors.New(errors.ErrBadRequest, "储能柜已激活")
	}

	// 生成API Key (不再生成API Secret)
	apiKey, err := generateAPIKey()
	if err != nil {
		utils.Error("Failed to generate API key", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "生成API Key失败")
	}

	// 更新储能柜激活信息 (api_secret_hash传空字符串)
	if err := s.cabinetRepo.UpdateActivation(ctx, cabinet.CabinetID, apiKey, ""); err != nil {
		utils.Error("Failed to update cabinet activation", zap.String("cabinet_id", cabinet.CabinetID), zap.Error(err))
		return nil, err
	}

	utils.Info("Cabinet activated successfully", zap.String("cabinet_id", cabinet.CabinetID))

	return &models.ActivateCabinetResponse{
		CabinetID: cabinet.CabinetID,
		APIKey:    apiKey,
		APISecret: "", // API Secret已废弃
	}, nil
}

// RegisterCabinet Edge端直接注册储能柜（一步完成注册和激活）
// 只需提供cabinet_id即可注册，其他字段可选，可在注册后使用API Key更新
func (s *cabinetService) RegisterCabinet(ctx context.Context, input *models.RegisterCabinetInput) (*models.RegisterCabinetResponse, error) {
	// 验证输入 - 只需验证cabinet_id
	if err := utils.ValidateCabinetID(input.CabinetID); err != nil {
		return nil, err
	}

	// 如果提供了MAC地址，验证格式
	if input.MACAddress != nil && *input.MACAddress != "" {
		if err := utils.ValidateMACAddress(*input.MACAddress); err != nil {
			return nil, err
		}
	}

	// 检查储能柜是否已存在
	exists, err := s.cabinetRepo.Exists(ctx, input.CabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence", zap.String("cabinet_id", input.CabinetID), zap.Error(err))
		return nil, err
	}

	if exists {
		return nil, errors.New(errors.ErrRecordExists, "储能柜ID已存在")
	}

	// 处理name：如果为空，使用cabinet_id作为默认值
	name := input.CabinetID
	if input.Name != nil && *input.Name != "" {
		name = *input.Name
	}

	// 处理MAC地址：如果为空，生成一个临时MAC地址（格式：00:00:00:XX:XX:XX，后6位基于cabinet_id的hash）
	var macAddress string
	if input.MACAddress != nil && *input.MACAddress != "" {
		macAddress = *input.MACAddress
		// 检查MAC地址是否已被使用
		_, err = s.cabinetRepo.GetByMACAddress(ctx, macAddress)
		if err == nil {
			return nil, errors.New(errors.ErrRecordExists, "MAC地址已被使用")
		} else if err.(*errors.AppError).Code != errors.ErrCabinetNotFound {
			utils.Error("Failed to check MAC address uniqueness", zap.String("mac_address", macAddress), zap.Error(err))
			return nil, err
		}
	} else {
		// 生成临时MAC地址：基于cabinet_id生成唯一MAC地址
		// 格式：00:00:00:XX:XX:XX，后6位基于cabinet_id的hash值
		hash := fmt.Sprintf("%x", input.CabinetID)
		if len(hash) < 6 {
			hash = hash + strings.Repeat("0", 6-len(hash))
		}
		macAddress = fmt.Sprintf("00:00:00:%s:%s:%s", hash[0:2], hash[2:4], hash[4:6])
		utils.Info("Generated temporary MAC address for cabinet", zap.String("cabinet_id", input.CabinetID), zap.String("mac_address", macAddress))
	}

	// 生成API Key
	apiKey, err := generateAPIKey()
	if err != nil {
		utils.Error("Failed to generate API key", zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrInternalServer, "生成API Key失败")
	}

	// 创建储能柜记录（直接设置为已激活状态，不再使用API Secret）
	now := time.Now()
	emptyHash := "" // API Secret Hash已废弃，传空字符串
	cabinet := &models.Cabinet{
		CabinetID:        input.CabinetID,
		Name:             name,
		Location:         input.Location,
		CapacityKwh:      input.CapacityKwh,
		DeviceModel:      input.DeviceModel,
		IPAddress:        input.IPAddress,
		MACAddress:       macAddress,
		Status:           models.CabinetStatusOffline, // 初始状态为离线，等待Edge端首次同步后变为active
		ActivationStatus: "activated", // 直接注册方式立即激活
		APIKey:           &apiKey,
		APISecretHash:    &emptyHash, // 不再使用API Secret
		ActivatedAt:      &now,
	}

	// 保存储能柜
	if err := s.cabinetRepo.Create(ctx, cabinet); err != nil {
		utils.Error("Failed to create cabinet", zap.String("cabinet_id", input.CabinetID), zap.Error(err))
		return nil, err
	}

	utils.Info("Cabinet registered successfully", zap.String("cabinet_id", input.CabinetID), zap.String("name", name), zap.String("mac_address", macAddress))
	s.ensureDefaultLicense(ctx, cabinet.CabinetID, nil, nil, nil)

	return &models.RegisterCabinetResponse{
		CabinetID: cabinet.CabinetID,
		APIKey:    apiKey,
		APISecret: "", // API Secret已废弃
	}, nil
}

// GetLocations 获取所有储能柜位置信息（用于地图展示）
func (s *cabinetService) GetLocations(ctx context.Context) ([]*models.CabinetLocation, error) {
	locations, err := s.cabinetRepo.GetLocations(ctx)
	if err != nil {
		utils.Error("Failed to get cabinet locations", zap.Error(err))
		return nil, err
	}

	utils.Debug("Cabinet locations retrieved successfully", zap.Int("count", len(locations)))

	return locations, nil
}

// GetStatistics 获取储能柜统计信息
func (s *cabinetService) GetStatistics(ctx context.Context) (*models.CabinetStatistics, error) {
	stats, err := s.cabinetRepo.GetStatistics(ctx)
	if err != nil {
		utils.Error("Failed to get cabinet statistics", zap.Error(err))
		return nil, err
	}

	utils.Debug("Cabinet statistics retrieved successfully",
		zap.Int64("total", stats.TotalCabinets),
		zap.Int64("active", stats.ActiveCabinets),
		zap.Int64("activated", stats.ActivatedCabinets),
	)

	return stats, nil
}

// ========== 辅助函数 ==========

// generateRegistrationToken 生成注册Token (JWT)
func generateRegistrationToken(cabinetID, macAddress string, expiresAt time.Time) (string, error) {
	claims := jwt.MapClaims{
		"cabinet_id":  cabinetID,
		"mac_address": macAddress,
		"exp":         expiresAt.Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// TODO: 从配置文件读取secret
	secret := []byte("your-secret-key-change-this-in-production")

	return token.SignedString(secret)
}

// generateAPIKey 生成API Key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ck_" + base64.URLEncoding.EncodeToString(bytes)[:43], nil
}

// 注意: API Secret已被移除,只使用API Key进行认证

// GetAPIKeyInfo 获取API Key信息（脱敏显示，不返回Secret）
func (s *cabinetService) GetAPIKeyInfo(ctx context.Context, cabinetID string) (map[string]interface{}, error) {
	// 查询储能柜
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	// 构造返回信息
	result := map[string]interface{}{
		"cabinet_id":        cabinet.CabinetID,
		"activation_status": cabinet.ActivationStatus,
		"has_api_key":       cabinet.APIKey != nil && *cabinet.APIKey != "",
	}

	// 脱敏显示API Key
	if cabinet.APIKey != nil && *cabinet.APIKey != "" {
		apiKey := *cabinet.APIKey
		if len(apiKey) > 13 {
			// 显示前10字符 + *** + 后3字符
			maskedKey := apiKey[:10] + "***" + apiKey[len(apiKey)-3:]
			result["api_key_masked"] = maskedKey
		} else {
			result["api_key_masked"] = "***"
		}
	}

	// 添加生成时间（使用updated_at作为近似值）
	if !cabinet.UpdatedAt.IsZero() {
		result["generated_at"] = cabinet.UpdatedAt
	}

	return result, nil
}

// RegenerateAPIKey 重新生成API Key和Secret
func (s *cabinetService) RegenerateAPIKey(ctx context.Context, cabinetID string) (map[string]string, error) {
	// 查询储能柜
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	// 检查是否已激活
	if cabinet.ActivationStatus != "activated" {
		return nil, errors.New(errors.ErrBadRequest, "只有已激活的储能柜才能重新生成API Key")
	}

	// 生成新的API Key和Secret
	newAPIKey, err := generateAPIKey()
	if err != nil {
		utils.Error("生成API Key失败", zap.Error(err))
		return nil, errors.New(errors.ErrInternalServer, "生成API Key失败")
	}

	// 更新数据库 (不再使用API Secret，传入空字符串)
	if err := s.cabinetRepo.UpdateActivation(ctx, cabinetID, newAPIKey, ""); err != nil {
		utils.Error("更新储能柜API Key失败",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, errors.New(errors.ErrInternalServer, "更新API Key失败")
	}

	utils.Info("API Key重新生成成功",
		zap.String("cabinet_id", cabinetID),
		zap.String("api_key_prefix", newAPIKey[:10]+"..."),
	)

	// 返回新的API Key (不再返回api_secret)
	return map[string]string{
		"cabinet_id": cabinetID,
		"api_key":    newAPIKey,
	}, nil
}

// RevokeAPIKey 撤销API Key（清空API Key和Secret）
func (s *cabinetService) RevokeAPIKey(ctx context.Context, cabinetID string) error {
	// 查询储能柜
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return err
	}

	// 检查是否有API Key
	if cabinet.APIKey == nil || *cabinet.APIKey == "" {
		return errors.New(errors.ErrBadRequest, "储能柜没有API Key")
	}

	// 清空API Key和Secret（使用UpdateActivation方法，传入空字符串）
	if err := s.cabinetRepo.UpdateActivation(ctx, cabinetID, "", ""); err != nil {
		utils.Error("撤销储能柜API Key失败",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return errors.New(errors.ErrInternalServer, "撤销API Key失败")
	}

	utils.Info("API Key已撤销",
		zap.String("cabinet_id", cabinetID),
	)

	return nil
}
