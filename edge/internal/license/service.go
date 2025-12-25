/*
 * 许可证服务
 * 负责JWT许可证的加载、验证和设备数管理
 */
package license

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// LicenseClaims JWT许可证声明
type LicenseClaims struct {
	LicenseID  string `json:"lic"` // 许可证ID
	MACAddress string `json:"mac"` // 绑定的MAC地址
	MaxDevices int    `json:"max"` // 最大设备数
	jwt.RegisteredClaims
}

// Service 许可证服务
type Service struct {
	logger      *zap.Logger
	enabled     bool
	licenseJWT  string
	claims      *LicenseClaims
	macAddress  string
	gracePeriod time.Duration
	pubKey      *rsa.PublicKey
	licensePath string
	lastEvent   string
	lastEventAt *time.Time
}

// NewService 创建许可证服务
func NewService(
	enabled bool,
	licensePath string,
	pubKeyPath string,
	gracePeriod time.Duration,
	logger *zap.Logger,
) (*Service, error) {
	s := &Service{
		logger:      logger,
		enabled:     enabled,
		gracePeriod: gracePeriod,
		licensePath: licensePath,
	}

	// 如果未启用，直接返回
	if !enabled {
		logger.Info("许可证验证已禁用")
		return s, nil
	}

	// 获取本机MAC地址
	macAddr, err := s.getMACAddress()
	if err != nil {
		return nil, fmt.Errorf("获取MAC地址失败: %w", err)
	}
	s.macAddress = macAddr

	// 加载厂商公钥
	pubKey, err := s.loadPublicKey(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载厂商公钥失败: %w", err)
	}
	s.pubKey = pubKey

	// 尝试加载许可证（如果文件不存在，允许启动但标记为未授权状态）
	if err := s.LoadLicense(licensePath); err != nil {
		logger.Warn("许可证加载失败，系统将以未授权模式运行",
			zap.Error(err),
			zap.String("license_path", licensePath))

		// 记录最后事件为"许可证未加载"
		now := time.Now()
		s.lastEvent = "LICENSE_NOT_LOADED"
		s.lastEventAt = &now

		// 允许启动，但返回未授权的服务实例
		return s, nil
	}

	logger.Info("许可证服务初始化成功",
		zap.String("license_id", s.claims.LicenseID),
		zap.String("mac_address", s.macAddress),
		zap.Int("max_devices", s.claims.MaxDevices),
		zap.Time("expires_at", s.claims.ExpiresAt.Time),
	)

	return s, nil
}

// LoadLicense 加载许可证文件
func (s *Service) LoadLicense(licensePath string) error {
	if licensePath != "" {
		s.licensePath = licensePath
	}

	// 读取许可证文件
	data, err := os.ReadFile(s.licensePath)
	if err != nil {
		return fmt.Errorf("读取许可证文件失败: %w", err)
	}

	if err := s.applyToken(string(data)); err != nil {
		return err
	}
	s.recordEvent("loaded_from_file")
	return nil
}

// applyToken 解析并应用许可证JWT
func (s *Service) applyToken(token string) error {
	s.licenseJWT = token

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, err := parser.ParseWithClaims(s.licenseJWT, &LicenseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("无效的签名算法: %v", token.Header["alg"])
		}
		return s.pubKey, nil
	})

	if err != nil {
		return fmt.Errorf("解析许可证失败: %w", err)
	}

	// 提取声明（注意：我们不检查token.Valid，因为我们手动处理过期）
	claims, ok := parsedToken.Claims.(*LicenseClaims)
	if !ok {
		return fmt.Errorf("许可证声明无效")
	}

	s.claims = claims
	return nil
}

// ApplyLicenseToken 接收来自Cloud的许可证并写入本地
func (s *Service) ApplyLicenseToken(token string) error {
	if !s.enabled {
		return fmt.Errorf("许可证服务未启用")
	}

	if s.licensePath == "" {
		return fmt.Errorf("未配置许可证文件路径")
	}

	if err := os.WriteFile(s.licensePath, []byte(token), 0600); err != nil {
		return fmt.Errorf("写入许可证文件失败: %w", err)
	}

	if err := s.applyToken(token); err != nil {
		return err
	}

	s.logger.Info("许可证已更新",
		zap.String("license_id", s.claims.LicenseID),
		zap.Time("expires_at", s.claims.ExpiresAt.Time))
	s.recordEvent("updated_from_cloud")
	return nil
}

// RevokeLicense 清除本地许可证
func (s *Service) RevokeLicense() error {
	if s.licensePath == "" {
		return fmt.Errorf("未配置许可证文件路径")
	}

	if err := os.Remove(s.licensePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除许可证文件失败: %w", err)
	}

	s.licenseJWT = ""
	s.claims = nil
	s.logger.Warn("许可证已吊销，本地文件已删除")
	s.recordEvent("license_revoked")
	return nil
}

// Check 检查许可证有效性
func (s *Service) Check() error {
	// 如果未启用，直接通过
	if !s.enabled {
		return nil
	}

	// 如果许可证未加载，记录警告但允许通过（等待Cloud端下发许可证）
	if s.claims == nil {
		s.logger.Debug("许可证未加载，系统以未授权模式运行，等待Cloud端下发许可证")
		// 不返回错误，允许设备认证通过，但功能可能受限
		return nil
	}

	// 1. 检查MAC地址绑定（不区分大小写）
	if !strings.EqualFold(s.claims.MACAddress, s.macAddress) {
		return fmt.Errorf("许可证MAC地址不匹配\n"+
			"  许可证绑定MAC: %s\n"+
			"  当前设备MAC: %s\n"+
			"  【修复方法】请使用当前设备MAC地址重新申请许可证",
			s.claims.MACAddress, s.macAddress)
	}

	// 2. 检查过期时间
	now := time.Now()
	expiresAt := s.claims.ExpiresAt.Time

	if now.After(expiresAt) {
		// 已过期，检查宽限期
		gracePeriodEnd := expiresAt.Add(s.gracePeriod)
		gracePeriodRemaining := gracePeriodEnd.Sub(now)

		if now.After(gracePeriodEnd) {
			// 宽限期已结束，拒绝访问
			return fmt.Errorf("许可证已过期且宽限期已结束\n"+
				"  许可证ID: %s\n"+
				"  过期时间: %s\n"+
				"  宽限期: %v\n"+
				"  宽限期结束时间: %s\n"+
				"  【修复方法】\n"+
				"    1. 联系厂商续期许可证\n"+
				"    2. 将新许可证文件替换到: configs/license.lic\n"+
				"    3. 重启Edge服务",
				s.claims.LicenseID,
				expiresAt.Format("2006-01-02 15:04:05"),
				s.gracePeriod,
				gracePeriodEnd.Format("2006-01-02 15:04:05"))
		}

		// 在宽限期内，记录警告但允许继续
		s.logger.Warn("⚠️  许可证已过期但在宽限期内，请尽快续期",
			zap.String("license_id", s.claims.LicenseID),
			zap.Time("expires_at", expiresAt),
			zap.Time("grace_period_end", gracePeriodEnd),
			zap.Duration("remaining", gracePeriodRemaining),
			zap.String("action", "请联系厂商续期许可证"),
		)

		// 每小时提醒一次（避免日志刷屏）
		if gracePeriodRemaining < 24*time.Hour {
			s.logger.Error("🚨 许可证即将在宽限期内失效",
				zap.Duration("remaining", gracePeriodRemaining),
				zap.String("urgent_action", "请立即联系厂商续期"))
		}
	}

	return nil
}

// GetMaxDevices 获取许可证允许的最大设备数
func (s *Service) GetMaxDevices() int {
	if !s.enabled || s.claims == nil {
		return 0 // 返回0表示无限制
	}
	return s.claims.MaxDevices
}

// IsEnabled 检查许可证验证是否启用
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// GetLicenseInfo 获取许可证信息（用于状态查询）
func (s *Service) GetLicenseInfo() map[string]interface{} {
	if !s.enabled || s.claims == nil {
		return map[string]interface{}{
			"enabled":       false,
			"last_event":    s.lastEvent,
			"last_event_at": formatTimePtr(s.lastEventAt),
		}
	}

	now := time.Now()
	expiresAt := s.claims.ExpiresAt.Time
	isExpired := now.After(expiresAt)
	inGracePeriod := isExpired && now.Before(expiresAt.Add(s.gracePeriod))

	return map[string]interface{}{
		"enabled":         true,
		"license_id":      s.claims.LicenseID,
		"mac_address":     s.macAddress,
		"max_devices":     s.claims.MaxDevices,
		"expires_at":      expiresAt.Format(time.RFC3339),
		"is_expired":      isExpired,
		"in_grace_period": inGracePeriod,
		"last_event":      s.lastEvent,
		"last_event_at":   formatTimePtr(s.lastEventAt),
	}
}

func (s *Service) recordEvent(event string) {
	now := time.Now()
	s.lastEvent = event
	s.lastEventAt = &now
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// loadPublicKey 加载RSA公钥
func (s *Service) loadPublicKey(pubKeyPath string) (*rsa.PublicKey, error) {
	// 读取公钥文件
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("读取公钥文件失败: %w", err)
	}

	// 解析PEM块
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("解析PEM块失败")
	}

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败: %w", err)
	}

	// 类型断言为RSA公钥
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是RSA公钥")
	}

	return rsaPub, nil
}

// getMACAddress 获取本机主网卡的MAC地址
// 优先获取默认路由对应的网卡，如果失败则返回第一个非回环网卡
func (s *Service) getMACAddress() (string, error) {
	// 方案1: 尝试获取默认路由对应的主网卡
	if mainMAC, err := s.getMainInterfaceMAC(); err == nil && mainMAC != "" {
		s.logger.Debug("使用主网卡MAC地址", zap.String("mac", mainMAC))
		return mainMAC, nil
	}

	// 方案2: 回退到第一个非回环网卡（兼容性）
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("获取网络接口失败: %w", err)
	}

	for _, iface := range interfaces {
		// 跳过回环接口、未启用接口和没有MAC地址的接口
		if iface.Flags&net.FlagLoopback != 0 ||
			iface.Flags&net.FlagUp == 0 ||
			len(iface.HardwareAddr) == 0 {
			continue
		}

		mac := iface.HardwareAddr.String()
		s.logger.Warn("使用第一个可用网卡MAC地址（非主网卡）",
			zap.String("interface", iface.Name),
			zap.String("mac", mac))
		return mac, nil
	}

	return "", fmt.Errorf("未找到有效的网络接口")
}

// getMainInterfaceMAC 获取默认路由对应的主网卡MAC地址
func (s *Service) getMainInterfaceMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// 遍历所有接口，查找有默认路由的接口
	for _, iface := range interfaces {
		// 跳过回环接口和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 检查是否有IP地址（有IP的接口更可能是主网卡）
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		// 检查是否有MAC地址
		if len(iface.HardwareAddr) > 0 {
			// 检查接口名称是否为常见的主网卡名称
			name := iface.Name
			if name == "eth0" || name == "en0" || name == "ens33" ||
				name == "enp0s3" || name == "wlan0" {
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	return "", fmt.Errorf("未找到主网卡")
}
