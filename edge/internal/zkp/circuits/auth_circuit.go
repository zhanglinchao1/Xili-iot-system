package circuits

import (
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// AuthCircuit 定义认证电路
// 证明客户端知道 secret，使得:
// 1. commitment = MiMC(secret || device_id)
// 2. response = MiMC(secret || challenge)
type AuthCircuit struct {
	// 私有输入 (仅证明者知道)
	Secret frontend.Variable `gnark:",secret"`

	// 公开输入 (证明者和验证者都知道)
	DeviceID    frontend.Variable `gnark:",public"`
	Challenge   frontend.Variable `gnark:",public"`
	Commitment  frontend.Variable `gnark:",public"`
	Response    frontend.Variable `gnark:",public"`
}

// Define 定义电路约束
func (circuit *AuthCircuit) Define(api frontend.API) error {
	// 创建 MiMC 哈希函数
	mimcHasher, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	// 约束 1: commitment = MiMC(secret, device_id)
	// ✅ 修复: 使用两次 Write 调用，与实际计算保持一致
	mimcHasher.Reset()
	mimcHasher.Write(circuit.Secret)
	mimcHasher.Write(circuit.DeviceID)
	computedCommitment := mimcHasher.Sum()
	api.AssertIsEqual(circuit.Commitment, computedCommitment)

	// 约束 2: response = MiMC(secret, challenge)
	// ✅ 修复: 使用两次 Write 调用，与实际计算保持一致
	mimcHasher.Reset()
	mimcHasher.Write(circuit.Secret)
	mimcHasher.Write(circuit.Challenge)
	computedResponse := mimcHasher.Sum()
	api.AssertIsEqual(circuit.Response, computedResponse)

	return nil
}

// GetCurve 返回使用的椭圆曲线
func GetCurve() ecc.ID {
	return ecc.BN254
}
