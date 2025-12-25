/*
 * ZKP验证器接口定义
 */
package zkp

// ZKPVerifier ZKP验证器接口
type ZKPVerifier interface {
	Initialize() error
	GenerateChallenge() (string, error)
	VerifyProof(deviceID, challenge, commitment, response string, proofData []byte) (bool, error)
	ComputeCommitment(secret, deviceID string) (string, error)
	ComputeResponse(secret, challenge string) (string, error)
	GenerateProof(secret, deviceID, challenge, commitment, response string) ([]byte, error)
}
