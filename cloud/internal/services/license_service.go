package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud-system/internal/licensing"
	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"go.uber.org/zap"
)

const (
	defaultLicenseValidDays  = 90
	defaultLicenseMaxDevices = 100
)

// LicenseService 许可证服务接口
type LicenseService interface {
	// CreateLicense 创建许可证
	CreateLicense(ctx context.Context, request *models.CreateLicenseRequest, createdBy string) (*models.License, error)

	// GetLicense 获取许可证
	GetLicense(ctx context.Context, cabinetID string) (*models.License, error)

	// ListLicenses 获取许可证列表
	ListLicenses(ctx context.Context, filter *models.LicenseListFilter) ([]*models.License, int64, error)

	// RenewLicense 续期许可证
	RenewLicense(ctx context.Context, cabinetID string, request *models.RenewLicenseRequest) error

	// RevokeLicense 吊销许可证
	RevokeLicense(ctx context.Context, cabinetID string, request *models.RevokeLicenseRequest, revokedBy string) error

	// DeleteLicense 删除许可证
	DeleteLicense(ctx context.Context, cabinetID string) error

	// ValidateLicense 验证许可证（Edge端调用）
	ValidateLicense(ctx context.Context, request *models.ValidateLicenseRequest) (*models.License, bool, error)

	// GenerateLicenseToken 生成下发给Edge的许可证令牌
	GenerateLicenseToken(ctx context.Context, cabinetID string) (string, error)

	// SyncLicenses 为没有许可证的储能柜创建默认许可证
	SyncLicenses(ctx context.Context, cabinetIDs []string, validDays, maxDevices int, permissions []string) (int, error)
}

// licenseService 许可证服务实现
type licenseService struct {
	licenseRepo    repository.LicenseRepository
	cabinetRepo    repository.CabinetRepository
	signingKeyPath string
}

// NewLicenseService 创建许可证服务实例
func NewLicenseService(
	licenseRepo repository.LicenseRepository,
	cabinetRepo repository.CabinetRepository,
	signingKeyPath string,
) LicenseService {
	return &licenseService{
		licenseRepo:    licenseRepo,
		cabinetRepo:    cabinetRepo,
		signingKeyPath: signingKeyPath,
	}
}

// CreateLicense 创建许可证
func (s *licenseService) CreateLicense(ctx context.Context, request *models.CreateLicenseRequest, createdBy string) (*models.License, error) {
	// 验证储能柜是否存在
	cabinet, err := s.cabinetRepo.GetByID(ctx, request.CabinetID)
	if err != nil {
		return nil, err
	}

	// 检查是否已存在许可证(cabinet_id有UNIQUE约束)
	existingLicense, err := s.licenseRepo.GetByCabinetID(ctx, request.CabinetID)
	if err == nil {
		// 许可证已存在,返回友好错误提示
		utils.Warn("Cabinet already has a license",
			zap.String("cabinet_id", request.CabinetID),
			zap.String("existing_license_id", existingLicense.LicenseID),
			zap.String("status", existingLicense.Status),
		)
		return nil, errors.New(errors.ErrValidation,
			fmt.Sprintf("储能柜 %s 已存在许可证(状态: %s),请先删除或吊销旧许可证",
				request.CabinetID, existingLicense.Status))
	} else if appErr, ok := err.(*errors.AppError); !ok || appErr.Code != errors.ErrLicenseNotFound {
		// 查询出错(非"未找到"错误)
		return nil, err
	}
	// 许可证不存在,继续创建

	// 序列化权限列表
	permissionsBytes, err := json.Marshal(request.Permissions)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrBadRequest, "权限列表格式错误")
	}

	// 创建许可证
	if request.MaxDevices <= 0 {
		return nil, errors.New(errors.ErrValidation, "max_devices 必须大于0")
	}

	license := &models.License{
		CabinetID:   request.CabinetID,
		MACAddress:  cabinet.MACAddress,
		LicenseID:   fmt.Sprintf("LIC-%s-%d", request.CabinetID, time.Now().Unix()),
		ExpiresAt:   time.Now().AddDate(0, 0, request.ValidDays),
		Permissions: string(permissionsBytes),
		MaxDevices:  request.MaxDevices,
		CreatedBy:   createdBy,
	}

	if err := s.licenseRepo.Create(ctx, license); err != nil {
		utils.Error("Failed to create license",
			zap.String("cabinet_id", request.CabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	utils.Info("License created successfully",
		zap.String("cabinet_id", license.CabinetID),
		zap.Time("expires_at", license.ExpiresAt),
	)

	return license, nil
}

// GetLicense 获取许可证
func (s *licenseService) GetLicense(ctx context.Context, cabinetID string) (*models.License, error) {
	license, err := s.licenseRepo.GetByCabinetID(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	return license, nil
}

// ListLicenses 获取许可证列表
func (s *licenseService) ListLicenses(ctx context.Context, filter *models.LicenseListFilter) ([]*models.License, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}

	var err error
	filter.PageSize, err = utils.ValidatePageSize(filter.PageSize, 100)
	if err != nil {
		return nil, 0, err
	}

	if filter.Status != nil && *filter.Status != "" {
		if !models.IsValidLicenseStatus(*filter.Status) {
			return nil, 0, errors.New(errors.ErrValidation, "无效的许可证状态")
		}
	}

	licenses, total, err := s.licenseRepo.List(ctx, filter)
	if err != nil {
		utils.Error("Failed to list licenses", zap.Error(err))
		return nil, 0, err
	}

	return licenses, total, nil
}

// RenewLicense 续期许可证
func (s *licenseService) RenewLicense(ctx context.Context, cabinetID string, request *models.RenewLicenseRequest) error {
	if err := s.licenseRepo.Renew(ctx, cabinetID, request.ExtendDays); err != nil {
		utils.Error("Failed to renew license",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("License renewed successfully",
		zap.String("cabinet_id", cabinetID),
		zap.Int("extend_days", request.ExtendDays),
	)

	return nil
}

// RevokeLicense 吊销许可证
func (s *licenseService) RevokeLicense(ctx context.Context, cabinetID string, request *models.RevokeLicenseRequest, revokedBy string) error {
	if err := s.licenseRepo.Revoke(ctx, cabinetID, request.Reason, revokedBy); err != nil {
		utils.Error("Failed to revoke license",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("License revoked successfully",
		zap.String("cabinet_id", cabinetID),
		zap.String("reason", request.Reason),
	)

	return nil
}

// DeleteLicense 删除许可证
func (s *licenseService) DeleteLicense(ctx context.Context, cabinetID string) error {
	if err := s.licenseRepo.Delete(ctx, cabinetID); err != nil {
		utils.Error("Failed to delete license",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("License deleted",
		zap.String("cabinet_id", cabinetID),
	)
	return nil
}

// GenerateLicenseToken 生成许可证令牌
func (s *licenseService) GenerateLicenseToken(ctx context.Context, cabinetID string) (string, error) {
	if s.signingKeyPath == "" {
		return "", errors.New(errors.ErrInternalServer, "未配置许可证签名私钥")
	}

	license, err := s.licenseRepo.GetByCabinetID(ctx, cabinetID)
	if err != nil {
		return "", err
	}

	if license.MaxDevices <= 0 {
		return "", errors.New(errors.ErrValidation, "许可证未配置最大设备数")
	}

	token, err := licensing.GenerateToken(license.LicenseID, license.MACAddress, license.MaxDevices, license.ExpiresAt, s.signingKeyPath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrInternalServer, "生成许可证令牌失败")
	}

	return token, nil
}

// ValidateLicense 验证许可证（Edge端调用）
func (s *licenseService) ValidateLicense(ctx context.Context, request *models.ValidateLicenseRequest) (*models.License, bool, error) {
	license, err := s.licenseRepo.Validate(ctx, request.CabinetID, request.MACAddress)
	if err != nil {
		return nil, false, err
	}

	// 检查状态
	if license.Status == "revoked" {
		return license, false, nil
	}

	// 检查过期时间
	if time.Now().After(license.ExpiresAt) {
		return license, false, nil
	}

	return license, true, nil
}

// SyncLicenses 为没有许可证的储能柜创建默认许可证
func (s *licenseService) SyncLicenses(ctx context.Context, cabinetIDs []string, validDays, maxDevices int, permissions []string) (int, error) {
	if validDays <= 0 {
		validDays = defaultLicenseValidDays
	}
	if maxDevices <= 0 {
		maxDevices = defaultLicenseMaxDevices
	}
	if len(permissions) == 0 {
		permissions = []string{"sensor:rw"}
	}

	created := 0
	createdBy := "system"

	ensure := func(cabinetID string) error {
		if cabinetID == "" {
			return nil
		}
		if _, err := s.licenseRepo.GetByCabinetID(ctx, cabinetID); err == nil {
			return nil
		} else if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.ErrLicenseNotFound {
			req := &models.CreateLicenseRequest{
				CabinetID:   cabinetID,
				ValidDays:   validDays,
				MaxDevices:  maxDevices,
				Permissions: permissions,
			}
			if _, err := s.CreateLicense(ctx, req, createdBy); err != nil {
				return err
			}
			created++
			return nil
		} else {
			return err
		}
	}

	if len(cabinetIDs) > 0 {
		for _, id := range cabinetIDs {
			if err := ensure(id); err != nil {
				return created, err
			}
		}
		return created, nil
	}

	page := 1
	pageSize := 100
	for {
		filter := &models.CabinetListFilter{
			Page:     page,
			PageSize: pageSize,
		}

		cabinets, total, err := s.cabinetRepo.List(ctx, filter)
		if err != nil {
			return created, err
		}

		if len(cabinets) == 0 {
			break
		}

		for _, cab := range cabinets {
			if err := ensure(cab.CabinetID); err != nil {
				return created, err
			}
		}

		if page*pageSize >= int(total) {
			break
		}
		page++
	}

	return created, nil
}
