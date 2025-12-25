/*
 * 许可证生成工具
 * 用于生成Edge系统的JWT许可证和RSA密钥对
 */
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LicenseClaims JWT许可证声明
type LicenseClaims struct {
	LicenseID  string `json:"lic"`  // 许可证ID
	MACAddress string `json:"mac"`  // 绑定的MAC地址
	MaxDevices int    `json:"max"`  // 最大设备数
	jwt.RegisteredClaims
}

func main() {
	// 命令行参数
	var (
		mode        = flag.String("mode", "license", "操作模式: keygen (生成密钥对) 或 license (生成许可证)")
		licenseID   = flag.String("id", "", "许可证ID (例如: LIC-2025-001)")
		macAddress  = flag.String("mac", "", "绑定的MAC地址 (例如: 00:11:22:33:44:55)")
		maxDevices  = flag.Int("max", 100, "最大设备数")
		expireDays  = flag.Int("days", 365, "有效期天数")
		privateKey  = flag.String("privkey", "./vendor_privkey.pem", "私钥文件路径")
		publicKey   = flag.String("pubkey", "./vendor_pubkey.pem", "公钥文件路径")
		outputFile  = flag.String("output", "./license.lic", "许可证输出文件")
	)
	flag.Parse()

	switch *mode {
	case "keygen":
		if err := generateKeyPair(*privateKey, *publicKey); err != nil {
			fmt.Fprintf(os.Stderr, "生成密钥对失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ RSA密钥对生成成功\n")
		fmt.Printf("  私钥: %s\n", *privateKey)
		fmt.Printf("  公钥: %s\n", *publicKey)

	case "license":
		// 参数校验
		if *licenseID == "" {
			fmt.Fprintf(os.Stderr, "错误: 必须提供许可证ID (-id)\n")
			flag.Usage()
			os.Exit(1)
		}
		if *macAddress == "" {
			fmt.Fprintf(os.Stderr, "错误: 必须提供MAC地址 (-mac)\n")
			flag.Usage()
			os.Exit(1)
		}

		if err := generateLicense(*licenseID, *macAddress, *maxDevices, *expireDays, *privateKey, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "生成许可证失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ 许可证生成成功\n")
		fmt.Printf("  许可证ID: %s\n", *licenseID)
		fmt.Printf("  MAC地址:  %s\n", *macAddress)
		fmt.Printf("  最大设备: %d\n", *maxDevices)
		fmt.Printf("  有效期:   %d天\n", *expireDays)
		fmt.Printf("  过期时间: %s\n", time.Now().AddDate(0, 0, *expireDays).Format("2006-01-02"))
		fmt.Printf("  输出文件: %s\n", *outputFile)

	default:
		fmt.Fprintf(os.Stderr, "错误: 无效的模式 '%s'\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

// generateKeyPair 生成RSA密钥对
func generateKeyPair(privateKeyPath, publicKeyPath string) error {
	// 生成2048位RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("生成RSA密钥失败: %w", err)
	}

	// 保存私钥
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("保存私钥失败: %w", err)
	}

	// 保存公钥
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("序列化公钥失败: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0644); err != nil {
		return fmt.Errorf("保存公钥失败: %w", err)
	}

	return nil
}

// generateLicense 生成许可证
func generateLicense(licenseID, macAddress string, maxDevices, expireDays int, privateKeyPath, outputPath string) error {
	// 读取私钥
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("读取私钥失败: %w", err)
	}

	// 解析PEM块
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("解析私钥PEM失败")
	}

	// 解析RSA私钥
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析RSA私钥失败: %w", err)
	}

	// 创建许可证声明
	now := time.Now()
	expiresAt := now.AddDate(0, 0, expireDays)

	claims := &LicenseClaims{
		LicenseID:  licenseID,
		MACAddress: macAddress,
		MaxDevices: maxDevices,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "Edge-Vendor",
			Subject:   licenseID,
		},
	}

	// 生成JWT令牌
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return fmt.Errorf("签名许可证失败: %w", err)
	}

	// 保存许可证文件
	if err := os.WriteFile(outputPath, []byte(signedToken), 0644); err != nil {
		return fmt.Errorf("保存许可证失败: %w", err)
	}

	return nil
}
