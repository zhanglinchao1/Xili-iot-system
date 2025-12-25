/*
 * 零知识证明验证器
 * 负责验证设备提交的零知识证明
 */
package zkp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/frontend"
	"github.com/edge/storage-cabinet/internal/zkp/circuits"
	"go.uber.org/zap"
)

// Verifier ZKP验证器
type Verifier struct {
	logger       *zap.Logger
	verifyingKey groth16.VerifyingKey
	curve        ecc.ID
	mu           sync.RWMutex
	initialized  bool
}

// NewVerifier 创建新的验证器
func NewVerifier(logger *zap.Logger) *Verifier {
	return &Verifier{
		logger: logger,
		curve:  ecc.BN254, // 使用BN254曲线
	}
}

// Initialize 初始化验证器 - 从文件加载 verifying key
func (v *Verifier) Initialize() error {
	return v.InitializeWithKeyPath("./auth_verifying.key")
}

// InitializeWithKeyPath 使用指定路径初始化验证器
func (v *Verifier) InitializeWithKeyPath(vkPath string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.initialized {
		return nil
	}

	v.logger.Info("Initializing ZKP verifier from key file...", zap.String("key_path", vkPath))

	// 检查文件是否存在
	if _, err := os.Stat(vkPath); os.IsNotExist(err) {
		return fmt.Errorf("verifying key file not found: %s\n"+
			"Please ensure auth_verifying.key exists (generated from Trusted Setup)", vkPath)
	}

	// 加载验证密钥
	vkFile, err := os.Open(vkPath)
	if err != nil {
		return fmt.Errorf("failed to open verifying key file: %w", err)
	}
	defer vkFile.Close()

	v.verifyingKey = groth16.NewVerifyingKey(v.curve)
	if _, err := v.verifyingKey.ReadFrom(vkFile); err != nil {
		return fmt.Errorf("failed to read verifying key: %w", err)
	}

	v.initialized = true
	v.logger.Info("ZKP verifier initialized successfully with pre-generated verifying key")
	return nil
}

// GenerateChallenge 生成认证挑战
func (v *Verifier) GenerateChallenge() (string, error) {
	// 生成32字节的随机数
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}
	return hex.EncodeToString(challenge), nil
}

// VerifyProof 验证零知识证明
func (v *Verifier) VerifyProof(
	deviceID string,
	challenge string,
	commitment string,
	response string,
	proofData []byte,
) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.initialized {
		return false, fmt.Errorf("verifier not initialized")
	}

	// 解析证明
	proof := groth16.NewProof(v.curve)
	if _, err := proof.ReadFrom(bytes.NewReader(proofData)); err != nil {
		v.logger.Error("Failed to parse proof", zap.Error(err))
		return false, fmt.Errorf("invalid proof format: %w", err)
	}

	// 准备公开输入
	publicWitness, err := v.preparePublicWitness(deviceID, challenge, commitment, response)
	if err != nil {
		return false, fmt.Errorf("failed to prepare public witness: %w", err)
	}

	// 验证证明
	err = groth16.Verify(proof, v.verifyingKey, publicWitness)
	if err != nil {
		v.logger.Debug("Proof verification failed",
			zap.String("device_id", deviceID),
			zap.Error(err))
		return false, nil
	}

	v.logger.Info("Proof verified successfully", zap.String("device_id", deviceID))
	return true, nil
}

// preparePublicWitness 准备公开见证
func (v *Verifier) preparePublicWitness(deviceID, challenge, commitment, response string) (witness.Witness, error) {
	// ✅ 修复：将字符串转换为 big.Int，与网关端保持一致
	
	// 1. DeviceID：字符串 -> 字节 -> big.Int
	deviceIDBytes := []byte(deviceID)
	deviceIDBig := new(big.Int).SetBytes(deviceIDBytes)
	
	// 2. Challenge：hex字符串 -> 字节 -> big.Int
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to decode challenge hex: %w", err)
	}
	challengeBig := new(big.Int).SetBytes(challengeBytes)
	
	// 3. Commitment：hex字符串 -> 字节 -> big.Int
	commitmentBytes, err := hex.DecodeString(commitment)
	if err != nil {
		return nil, fmt.Errorf("failed to decode commitment hex: %w", err)
	}
	commitmentBig := new(big.Int).SetBytes(commitmentBytes)
	
	// 4. Response：hex字符串 -> 字节 -> big.Int
	responseBytes, err := hex.DecodeString(response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response hex: %w", err)
	}
	responseBig := new(big.Int).SetBytes(responseBytes)
	
	// 创建见证赋值（使用 big.Int）
	assignment := &circuits.AuthCircuit{
		DeviceID:   deviceIDBig,
		Challenge:  challengeBig,
		Commitment: commitmentBig,
		Response:   responseBig,
	}

	// 创建公开见证
	witness, err := frontend.NewWitness(assignment, v.curve.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return nil, err
	}

	return witness, nil
}

// GenerateProof 服务端不需要生成证明，此方法保留仅用于接口兼容
func (v *Verifier) GenerateProof(
	secret string,
	deviceID string,
	challenge string,
	commitment string,
	response string,
) ([]byte, error) {
	return nil, fmt.Errorf("server-side verifier does not support proof generation")
}

// ComputeCommitment 计算承诺值（用于设备注册）
// ✅ 修复：与网关端 ComputeMiMCHash 保持一致
func (v *Verifier) ComputeCommitment(secret string, deviceID string) (string, error) {
	// 1. Secret 是 hex 字符串，需要解码为字节
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to decode secret hex: %w", err)
	}

	// 2. 将 deviceID 转换为域元素（32 字节）
	deviceIDBytes := []byte(deviceID)
	deviceIDBig := new(big.Int).SetBytes(deviceIDBytes)
	deviceIDFieldBytes := make([]byte, 32)
	deviceIDBig.FillBytes(deviceIDFieldBytes) // 填充为 32 字节

	// 3. 使用与电路一致的MiMC哈希
	mimcHash := hash.MIMC_BN254.New()
	mimcHash.Write(secretBytes)        // 第一个域元素（32 字节）
	mimcHash.Write(deviceIDFieldBytes) // 第二个域元素（32 字节）

	// 4. 计算哈希值并返回十六进制
	hashBytes := mimcHash.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

// ComputeResponse 计算响应值（用于测试）
// ✅ 修复：与网关端 ComputeMiMCHash 保持一致
func (v *Verifier) ComputeResponse(secret string, challenge string) (string, error) {
	// 1. Secret 是 hex 字符串，需要解码为字节
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to decode secret hex: %w", err)
	}

	// 2. Challenge 是 hex 字符串，需要解码为字节
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to decode challenge hex: %w", err)
	}

	// 3. 将 challenge 转换为域元素（32 字节）
	challengeBig := new(big.Int).SetBytes(challengeBytes)
	challengeFieldBytes := make([]byte, 32)
	challengeBig.FillBytes(challengeFieldBytes) // 填充为 32 字节

	// 4. 使用与电路一致的MiMC哈希
	mimcHash := hash.MIMC_BN254.New()
	mimcHash.Write(secretBytes)          // 第一个域元素（32 字节）
	mimcHash.Write(challengeFieldBytes)  // 第二个域元素（32 字节）

	// 5. 计算哈希值并返回十六进制
	hashBytes := mimcHash.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

