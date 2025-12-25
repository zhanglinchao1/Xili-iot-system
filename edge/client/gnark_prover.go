/*
 * Gnark客户端证明生成器
 * 基于gnark库实现的零知识证明客户端
 * 文档: https://docs.gnark.consensys.io/overview
 */
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/std/hash/mimc"
)

// AuthCircuit 认证电路定义（与服务端保持一致）
type AuthCircuit struct {
	// 私有输入（证明者知道，验证者不知道）
	Secret frontend.Variable `gnark:",secret"`

	// 公开输入（双方都知道）
	DeviceID   frontend.Variable `gnark:",public"` // 设备ID
	Challenge  frontend.Variable `gnark:",public"` // 挑战随机数
	Commitment frontend.Variable `gnark:",public"` // 承诺值 = hash(secret, deviceID)
	Response   frontend.Variable `gnark:",public"` // 响应值 = hash(secret, challenge)
}

// Define 定义电路约束
func (circuit *AuthCircuit) Define(api frontend.API) error {
	// 1. 验证设备身份：检查 hash(secret, deviceID) == commitment
	mimc1, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	mimc1.Write(circuit.Secret)
	mimc1.Write(circuit.DeviceID)
	computedCommitment := mimc1.Sum()
	
	// 断言计算的承诺值等于公开的承诺值
	api.AssertIsEqual(computedCommitment, circuit.Commitment)

	// 2. 验证挑战响应：检查 hash(secret, challenge) == response
	mimc2, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	mimc2.Write(circuit.Secret)
	mimc2.Write(circuit.Challenge)
	computedResponse := mimc2.Sum()
	
	// 断言计算的响应值等于公开的响应值
	api.AssertIsEqual(computedResponse, circuit.Response)

	return nil
}

// DeviceCredentials 设备凭据
type DeviceCredentials struct {
	DeviceID   string `json:"device_id"`
	Secret     string `json:"secret"`
	PublicKey  string `json:"public_key"`
	Commitment string `json:"commitment"`
}

// ChallengeRequest 挑战请求
type ChallengeRequest struct {
	DeviceID string `json:"device_id"`
}

// ChallengeResponse 挑战响应
type ChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// AuthRequest 认证请求
type AuthRequest struct {
	DeviceID    string `json:"device_id"`
	ChallengeID string `json:"challenge_id"`
	Proof       struct {
		Proof         []byte   `json:"proof"`
		PublicWitness []string `json:"public_witness"`
	} `json:"proof"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Success   bool      `json:"success"`
	SessionID string    `json:"session_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// GnarkProver gnark证明生成器
type GnarkProver struct {
	constraintSystem constraint.ConstraintSystem
	provingKey       groth16.ProvingKey
	curve            ecc.ID
	initialized      bool
}

// NewGnarkProver 创建新的证明生成器
func NewGnarkProver() *GnarkProver {
	return &GnarkProver{
		curve: ecc.BN254, // 使用BN254曲线
	}
}

// Initialize 初始化证明生成器
func (p *GnarkProver) Initialize() error {
	if p.initialized {
		return nil
	}

	fmt.Println("🔧 初始化gnark证明生成器...")

	// 创建电路实例
	var circuit AuthCircuit

	// 编译电路
	r1cs, err := frontend.Compile(p.curve.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		return fmt.Errorf("编译电路失败: %w", err)
	}
	p.constraintSystem = r1cs

	// 生成证明密钥（注意：在实际部署中，这应该从服务端获取或预先生成）
	pk, _, err := groth16.Setup(r1cs)
	if err != nil {
		return fmt.Errorf("生成密钥失败: %w", err)
	}
	p.provingKey = pk

	p.initialized = true
	fmt.Println("✅ gnark证明生成器初始化成功")
	return nil
}

// ComputeMiMCHash 计算MiMC哈希
func (p *GnarkProver) ComputeMiMCHash(input1, input2 string) (string, error) {
	// 使用与电路一致的MiMC哈希
	mimcHash := hash.MIMC_BN254.New()

	// 将字符串转换为字节并写入哈希
	mimcHash.Write([]byte(input1))
	mimcHash.Write([]byte(input2))

	// 计算哈希值
	hashBytes := mimcHash.Sum(nil)
	result := new(big.Int).SetBytes(hashBytes)
	return result.Text(16), nil
}

// GenerateProof 生成零知识证明
func (p *GnarkProver) GenerateProof(
	secret string,
	deviceID string,
	challenge string,
	commitment string,
	response string,
) ([]byte, error) {
	if !p.initialized {
		return nil, fmt.Errorf("证明生成器未初始化")
	}

	fmt.Println("🔐 生成零知识证明...")

	// 创建完整见证（包括私有输入）
	assignment := &AuthCircuit{
		Secret:     secret,
		DeviceID:   deviceID,
		Challenge:  challenge,
		Commitment: commitment,
		Response:   response,
	}

	witness, err := frontend.NewWitness(assignment, p.curve.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("创建见证失败: %w", err)
	}

	// 生成证明
	proof, err := groth16.Prove(p.constraintSystem, p.provingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("生成证明失败: %w", err)
	}

	// 序列化证明
	var buf bytes.Buffer
	if _, err := proof.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("序列化证明失败: %w", err)
	}

	fmt.Println("✅ 零知识证明生成成功")
	return buf.Bytes(), nil
}

// EdgeClient Edge系统客户端
type EdgeClient struct {
	serverURL string
	prover    *GnarkProver
}

// NewEdgeClient 创建新的Edge客户端
func NewEdgeClient(serverURL string) *EdgeClient {
	return &EdgeClient{
		serverURL: serverURL,
		prover:    NewGnarkProver(),
	}
}

// LoadCredentials 加载设备凭据
func (c *EdgeClient) LoadCredentials(filePath string) (*DeviceCredentials, error) {
	fmt.Printf("📂 加载设备凭据: %s\n", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取凭据文件失败: %w", err)
	}

	var creds DeviceCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("解析凭据文件失败: %w", err)
	}

	// 验证必需字段
	if creds.DeviceID == "" || creds.Secret == "" || creds.Commitment == "" {
		return nil, fmt.Errorf("凭据文件缺少必需字段")
	}

	fmt.Println("✅ 设备凭据加载成功")
	return &creds, nil
}

// GetChallenge 获取认证挑战
func (c *EdgeClient) GetChallenge(deviceID string) (*ChallengeResponse, error) {
	fmt.Println("📡 获取认证挑战...")

	reqBody := ChallengeRequest{
		DeviceID: deviceID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := http.Post(
		c.serverURL+"/api/v1/auth/challenge",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器错误 %d: %s", resp.StatusCode, string(body))
	}

	var challenge ChallengeResponse
	if err := json.Unmarshal(body, &challenge); err != nil {
		return nil, fmt.Errorf("解析挑战响应失败: %w", err)
	}

	fmt.Println("✅ 认证挑战获取成功")
	return &challenge, nil
}

// SubmitProof 提交零知识证明
func (c *EdgeClient) SubmitProof(
	deviceID string,
	challengeID string,
	proofData []byte,
	publicWitness []string,
) (*AuthResponse, error) {
	fmt.Println("📤 提交零知识证明...")

	reqBody := AuthRequest{
		DeviceID:    deviceID,
		ChallengeID: challengeID,
	}
	reqBody.Proof.Proof = proofData
	reqBody.Proof.PublicWitness = publicWitness

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := http.Post(
		c.serverURL+"/api/v1/auth/verify",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("解析认证响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !authResp.Success {
		return nil, fmt.Errorf("认证失败: %s", authResp.Message)
	}

	fmt.Println("✅ 零知识证明验证成功")
	return &authResp, nil
}

// Authenticate 执行完整的认证流程
func (c *EdgeClient) Authenticate(credentialsPath string) (*AuthResponse, error) {
	// 初始化证明生成器
	if err := c.prover.Initialize(); err != nil {
		return nil, err
	}

	// 加载设备凭据
	creds, err := c.LoadCredentials(credentialsPath)
	if err != nil {
		return nil, err
	}

	// 获取挑战
	challenge, err := c.GetChallenge(creds.DeviceID)
	if err != nil {
		return nil, err
	}

	// 计算响应值
	response, err := c.prover.ComputeMiMCHash(creds.Secret, challenge.Nonce)
	if err != nil {
		return nil, fmt.Errorf("计算响应值失败: %w", err)
	}

	// 生成零知识证明
	proofData, err := c.prover.GenerateProof(
		creds.Secret,
		creds.DeviceID,
		challenge.Nonce,
		creds.Commitment,
		response,
	)
	if err != nil {
		return nil, err
	}

	// 准备公开见证
	publicWitness := []string{
		creds.DeviceID,
		challenge.Nonce,
		creds.Commitment,
		response,
	}

	// 提交证明
	authResp, err := c.SubmitProof(
		creds.DeviceID,
		challenge.ChallengeID,
		proofData,
		publicWitness,
	)
	if err != nil {
		return nil, err
	}

	return authResp, nil
}

// TestAuthenticatedAPI 测试需要认证的API
func (c *EdgeClient) TestAuthenticatedAPI(token string, deviceID string) error {
	fmt.Println("🧪 测试认证API访问...")

	// 测试数据收集API
	testData := map[string]interface{}{
		"device_id":   deviceID,
		"sensor_type": "co2",
		"value":       420.5,
		"unit":        "ppm",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"quality":     95,
	}

	jsonData, _ := json.Marshal(testData)

	req, err := http.NewRequest("POST", c.serverURL+"/api/v1/data/collect", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ 数据收集API测试成功")
	} else {
		fmt.Printf("⚠️  数据收集API测试失败: %d\n", resp.StatusCode)
	}

	return nil
}

func main() {
	fmt.Println("🔐 Edge系统Gnark零知识证明客户端")
	fmt.Println("基于gnark库: https://docs.gnark.consensys.io/overview")
	fmt.Println("==================================================")

	if len(os.Args) < 2 {
		fmt.Println("用法: go run gnark_prover.go <credentials_file> [server_url]")
		fmt.Println("示例: go run gnark_prover.go device_credentials_CO2_SENSOR_20251015_140552.json")
		os.Exit(1)
	}

	credentialsPath := os.Args[1]
	serverURL := "http://localhost:8080"
	if len(os.Args) > 2 {
		serverURL = os.Args[2]
	}

	// 创建客户端
	client := NewEdgeClient(serverURL)

	// 执行认证
	authResp, err := client.Authenticate(credentialsPath)
	if err != nil {
		log.Fatalf("❌ 认证失败: %v", err)
	}

	fmt.Println("\n🎉 认证成功!")
	fmt.Printf("会话ID: %s\n", authResp.SessionID)
	fmt.Printf("JWT令牌: %s...\n", authResp.Token[:50])
	fmt.Printf("过期时间: %s\n", authResp.ExpiresAt.Format(time.RFC3339))

	// 测试认证API
	deviceID := ""
	if creds, err := client.LoadCredentials(credentialsPath); err == nil {
		deviceID = creds.DeviceID
	}

	if err := client.TestAuthenticatedAPI(authResp.Token, deviceID); err != nil {
		fmt.Printf("⚠️  API测试失败: %v\n", err)
	}

	fmt.Println("\n==================================================")
	fmt.Println("✅ gnark零知识证明认证流程完成!")
}
