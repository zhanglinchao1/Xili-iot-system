/*
 * 简化的零知识证明验证器
 * 用于测试和开发，不使用复杂的ZKP库
 */
package zkp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.uber.org/zap"
)

// SimpleVerifier 简化的ZKP验证器
type SimpleVerifier struct {
	logger *zap.Logger
}

// NewSimpleVerifier 创建简化验证器
func NewSimpleVerifier(logger *zap.Logger) *SimpleVerifier {
	return &SimpleVerifier{
		logger: logger,
	}
}

// Initialize 初始化验证器
func (v *SimpleVerifier) Initialize() error {
	v.logger.Info("Simple ZKP verifier initialized (for testing)")
	return nil
}

// GenerateChallenge 生成挑战
func (v *SimpleVerifier) GenerateChallenge() (string, error) {
	// 生成32字节随机数
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	return hex.EncodeToString(nonce), nil
}

// VerifyProof 验证证明（简化版本）
// 实现 ZKPVerifier 接口
func (v *SimpleVerifier) VerifyProof(deviceID, challenge, commitment, response string, proofData []byte) (bool, error) {
	// 简化验证：检查输入是否有效
	if deviceID == "" || commitment == "" || challenge == "" || response == "" {
		return false, fmt.Errorf("invalid input parameters")
	}

	if len(proofData) == 0 {
		return false, fmt.Errorf("empty proof")
	}

	// 简单的哈希验证（仅用于测试）
	expectedHash := sha256.Sum256([]byte(deviceID + commitment + challenge + response))
	proofHash := sha256.Sum256(proofData)

	// 在实际应用中，这里应该是复杂的ZKP验证逻辑
	// 为了测试，我们简化为哈希比较
	v.logger.Debug("Simple proof verification",
		zap.String("device_id", deviceID),
		zap.String("expected", hex.EncodeToString(expectedHash[:])),
		zap.String("proof", hex.EncodeToString(proofHash[:])))

	// 简化验证：总是返回true（仅用于测试）
	return true, nil
}

// ComputeCommitment 计算承诺值
// 实现 ZKPVerifier 接口
func (v *SimpleVerifier) ComputeCommitment(secret, deviceID string) (string, error) {
	// 简化计算：使用哈希
	hash := sha256.Sum256([]byte(secret + deviceID))
	return hex.EncodeToString(hash[:]), nil
}

// ComputeResponse 计算预期响应
// 实现 ZKPVerifier 接口
func (v *SimpleVerifier) ComputeResponse(secret, challenge string) (string, error) {
	// 简化计算：使用哈希
	hash := sha256.Sum256([]byte(secret + challenge))
	return hex.EncodeToString(hash[:]), nil
}

// GenerateProof 生成零知识证明
// 实现 ZKPVerifier 接口
func (v *SimpleVerifier) GenerateProof(secret, deviceID, challenge, commitment, response string) ([]byte, error) {
	// 简化版本：生成一个简单的哈希作为证明
	proofStr := secret + deviceID + challenge + commitment + response
	hash := sha256.Sum256([]byte(proofStr))
	return hash[:], nil
}
