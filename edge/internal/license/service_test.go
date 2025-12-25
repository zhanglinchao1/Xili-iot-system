/*
 * 许可证服务单元测试
 */
package license

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// 测试辅助函数：生成测试用RSA密钥对
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成RSA密钥失败: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

// 测试辅助函数：保存公钥到文件
func savePublicKey(t *testing.T, pubKey *rsa.PublicKey, path string) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("序列化公钥失败: %v", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	if err := os.WriteFile(path, pubKeyPEM, 0644); err != nil {
		t.Fatalf("保存公钥失败: %v", err)
	}
}

// 测试辅助函数：生成测试许可证
func generateTestLicense(t *testing.T, privateKey *rsa.PrivateKey, macAddr string, maxDevices int, expiresIn time.Duration) string {
	claims := &LicenseClaims{
		LicenseID:  "TEST-LICENSE-001",
		MACAddress: macAddr,
		MaxDevices: maxDevices,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("签名许可证失败: %v", err)
	}

	return signedToken
}

func TestNewService_Disabled(t *testing.T) {
	logger := zap.NewNop()

	// 测试禁用状态
	service, err := NewService(false, "", "", 72*time.Hour, logger)
	if err != nil {
		t.Fatalf("创建禁用的许可证服务失败: %v", err)
	}

	if service.enabled {
		t.Error("许可证服务应该是禁用状态")
	}

	// 禁用状态下Check应该直接通过
	if err := service.Check(); err != nil {
		t.Errorf("禁用状态下Check应该通过: %v", err)
	}
}

func TestLicenseService_ValidLicense(t *testing.T) {
	logger := zap.NewNop()

	// 1. 生成测试密钥对
	privateKey, publicKey := generateTestKeyPair(t)

	// 2. 创建临时文件
	tempDir := t.TempDir()
	pubKeyPath := tempDir + "/test_pubkey.pem"
	licensePath := tempDir + "/test_license.lic"

	// 3. 保存公钥
	savePublicKey(t, publicKey, pubKeyPath)

	// 4. 获取本机MAC地址（用于生成匹配的许可证）
	service := &Service{logger: logger}
	macAddr, err := service.getMACAddress()
	if err != nil {
		t.Skipf("无法获取MAC地址，跳过测试: %v", err)
	}

	// 5. 生成有效许可证（未来30天过期）
	licenseJWT := generateTestLicense(t, privateKey, macAddr, 100, 30*24*time.Hour)
	if err := os.WriteFile(licensePath, []byte(licenseJWT), 0644); err != nil {
		t.Fatalf("保存许可证失败: %v", err)
	}

	// 6. 创建许可证服务
	service, err = NewService(true, licensePath, pubKeyPath, 72*time.Hour, logger)
	if err != nil {
		t.Fatalf("创建许可证服务失败: %v", err)
	}

	// 7. 验证许可证有效
	if err := service.Check(); err != nil {
		t.Errorf("有效许可证验证失败: %v", err)
	}

	// 8. 验证最大设备数
	if service.GetMaxDevices() != 100 {
		t.Errorf("期望最大设备数100，实际: %d", service.GetMaxDevices())
	}
}

func TestLicenseService_ExpiredLicense(t *testing.T) {
	logger := zap.NewNop()

	// 1. 生成测试密钥对
	privateKey, publicKey := generateTestKeyPair(t)

	// 2. 创建临时文件
	tempDir := t.TempDir()
	pubKeyPath := tempDir + "/test_pubkey.pem"
	licensePath := tempDir + "/test_license.lic"

	// 3. 保存公钥
	savePublicKey(t, publicKey, pubKeyPath)

	// 4. 获取本机MAC地址
	service := &Service{logger: logger}
	macAddr, err := service.getMACAddress()
	if err != nil {
		t.Skipf("无法获取MAC地址，跳过测试: %v", err)
	}

	// 5. 生成已过期的许可证（过期1小时，宽限期内）
	licenseJWT := generateTestLicense(t, privateKey, macAddr, 100, -1*time.Hour)
	if err := os.WriteFile(licensePath, []byte(licenseJWT), 0644); err != nil {
		t.Fatalf("保存许可证失败: %v", err)
	}

	// 6. 创建许可证服务（宽限期72小时）
	service, err = NewService(true, licensePath, pubKeyPath, 72*time.Hour, logger)
	if err != nil {
		t.Fatalf("创建许可证服务失败: %v", err)
	}

	// 7. 在宽限期内应该通过（只有警告）
	if err := service.Check(); err != nil {
		t.Errorf("宽限期内的过期许可证应该通过: %v", err)
	}
}

func TestLicenseService_ExpiredBeyondGrace(t *testing.T) {
	logger := zap.NewNop()

	// 1. 生成测试密钥对
	privateKey, publicKey := generateTestKeyPair(t)

	// 2. 创建临时文件
	tempDir := t.TempDir()
	pubKeyPath := tempDir + "/test_pubkey.pem"
	licensePath := tempDir + "/test_license.lic"

	// 3. 保存公钥
	savePublicKey(t, publicKey, pubKeyPath)

	// 4. 获取本机MAC地址
	service := &Service{logger: logger}
	macAddr, err := service.getMACAddress()
	if err != nil {
		t.Skipf("无法获取MAC地址，跳过测试: %v", err)
	}

	// 5. 生成已过期的许可证（过期100小时，超过72小时宽限期）
	licenseJWT := generateTestLicense(t, privateKey, macAddr, 100, -100*time.Hour)
	if err := os.WriteFile(licensePath, []byte(licenseJWT), 0644); err != nil {
		t.Fatalf("保存许可证失败: %v", err)
	}

	// 6. 创建许可证服务（宽限期72小时）
	service, err = NewService(true, licensePath, pubKeyPath, 72*time.Hour, logger)
	if err != nil {
		t.Fatalf("创建许可证服务失败: %v", err)
	}

	// 7. 超过宽限期应该拒绝
	if err := service.Check(); err == nil {
		t.Error("超过宽限期的过期许可证应该被拒绝")
	}
}

func TestLicenseService_MACMismatch(t *testing.T) {
	logger := zap.NewNop()

	// 1. 生成测试密钥对
	privateKey, publicKey := generateTestKeyPair(t)

	// 2. 创建临时文件
	tempDir := t.TempDir()
	pubKeyPath := tempDir + "/test_pubkey.pem"
	licensePath := tempDir + "/test_license.lic"

	// 3. 保存公钥
	savePublicKey(t, publicKey, pubKeyPath)

	// 4. 生成许可证，使用错误的MAC地址
	wrongMAC := "00:11:22:33:44:55"
	licenseJWT := generateTestLicense(t, privateKey, wrongMAC, 100, 30*24*time.Hour)
	if err := os.WriteFile(licensePath, []byte(licenseJWT), 0644); err != nil {
		t.Fatalf("保存许可证失败: %v", err)
	}

	// 5. 创建许可证服务
	service, err := NewService(true, licensePath, pubKeyPath, 72*time.Hour, logger)
	if err != nil {
		t.Fatalf("创建许可证服务失败: %v", err)
	}

	// 6. MAC地址不匹配应该拒绝
	if err := service.Check(); err == nil {
		t.Error("MAC地址不匹配的许可证应该被拒绝")
	}
}
