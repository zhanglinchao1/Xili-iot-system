/*
 * 认证数据模型
 * 定义零知识认证相关的数据结构
 */
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Challenge 认证挑战
type Challenge struct {
	ChallengeID string    `json:"challenge_id" db:"challenge_id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	Nonce       string    `json:"nonce" db:"nonce"` // 随机数
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	Used        bool      `json:"used" db:"used"`
}

// PublicWitness 公开见证
// 包含零知识证明验证所需的所有公开输入
type PublicWitness struct {
	DeviceID   string `json:"device_id"`  // 设备ID
	Challenge  string `json:"challenge"`  // 挑战值（从服务端获取的nonce）
	Commitment string `json:"commitment"` // 身份承诺 MiMC(secret, deviceID)
	Response   string `json:"response"`   // 挑战响应 MiMC(secret, challenge)
}

// ZKProof 零知识证明
// 符合Gnark Groth16格式
type ZKProof struct {
	Proof         string        `json:"proof"`          // Base64编码的Groth16 proof（~192字节）
	PublicWitness PublicWitness `json:"public_witness"` // 公开见证（公开输入集合）
}

// AuthRequest 认证请求
type AuthRequest struct {
	DeviceID    string   `json:"device_id" binding:"required"`
	ChallengeID string   `json:"challenge_id" binding:"required"`
	Proof       *ZKProof `json:"proof" binding:"required"`
}

// Session 会话信息
type Session struct {
	SessionID  string    `json:"session_id" db:"session_id"`
	DeviceID   string    `json:"device_id" db:"device_id"`
	Token      string    `json:"token" db:"token"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at" db:"last_used_at"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
}

// ChallengeRequest 挑战请求
type ChallengeRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// ChallengeResponse 挑战响应
type ChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Success   bool      `json:"success"`
	SessionID string    `json:"session_id,omitempty"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// TokenClaims JWT令牌声明
type TokenClaims struct {
	DeviceID  string `json:"device_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}
