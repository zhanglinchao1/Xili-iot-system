package licensing

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type licenseClaims struct {
	LicenseID  string `json:"lic"`
	MACAddress string `json:"mac"`
	MaxDevices int    `json:"max"`
	jwt.RegisteredClaims
}

// GenerateToken 使用RSA私钥生成许可证JWT
func GenerateToken(licenseID, macAddress string, maxDevices int, expiresAt time.Time, privateKeyPath string) (string, error) {
	if privateKeyPath == "" {
		return "", fmt.Errorf("签名私钥路径未配置")
	}

	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("读取私钥失败: %w", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return "", fmt.Errorf("解析私钥PEM失败")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析RSA私钥失败: %w", err)
	}

	now := time.Now()
	claims := &licenseClaims{
		LicenseID:  licenseID,
		MACAddress: macAddress,
		MaxDevices: maxDevices,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cloud-system",
			Subject:   licenseID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("签名许可证失败: %w", err)
	}

	return signedToken, nil
}
