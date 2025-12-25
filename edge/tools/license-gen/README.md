# Edge 许可证生成工具

用于生成Edge系统的JWT许可证和RSA密钥对。

## 功能

1. **生成RSA密钥对**: 生成用于签名许可证的厂商密钥对
2. **生成许可证**: 基于MAC地址和设备限制生成JWT许可证

## 使用方法

### 1. 生成密钥对（厂商侧操作，仅需一次）

```bash
# 编译工具
go build -o license-gen main.go

# 生成RSA密钥对
./license-gen -mode keygen

# 自定义输出路径
./license-gen -mode keygen \
  -privkey ./keys/vendor_privkey.pem \
  -pubkey ./keys/vendor_pubkey.pem
```

**输出文件**:
- `vendor_privkey.pem`: 厂商私钥（保密，仅用于签发许可证）
- `vendor_pubkey.pem`: 厂商公钥（公开，部署到Edge设备）

**重要**: 私钥必须妥善保管，泄露将导致安全风险！

---

### 2. 生成许可证（为每个客户生成）

```bash
# 基本用法
./license-gen -mode license \
  -id "LIC-2025-001" \
  -mac "00:11:22:33:44:55" \
  -max 100 \
  -days 365

# 完整参数示例
./license-gen -mode license \
  -id "LIC-CUSTOMER-001" \
  -mac "a4:83:e7:12:34:56" \
  -max 200 \
  -days 730 \
  -privkey ./vendor_privkey.pem \
  -output ./customer_license.lic
```

**参数说明**:
- `-id`: 许可证唯一标识（建议格式: `LIC-客户名-序号`）
- `-mac`: Edge设备的MAC地址（使用 `ip link` 或 `ifconfig` 查看）
- `-max`: 该许可证允许的最大设备数
- `-days`: 许可证有效期天数
- `-privkey`: 厂商私钥路径（默认 `./vendor_privkey.pem`）
- `-output`: 许可证输出路径（默认 `./license.lic`）

**输出示例**:
```
✓ 许可证生成成功
  许可证ID: LIC-2025-001
  MAC地址:  00:11:22:33:44:55
  最大设备: 100
  有效期:   365天
  过期时间: 2026-10-21
  输出文件: ./license.lic
```

---

## 部署流程

### 厂商侧（一次性操作）

1. 生成密钥对:
   ```bash
   ./license-gen -mode keygen
   ```

2. 保管私钥:
   - 将 `vendor_privkey.pem` 保存到安全位置
   - 建议备份到多个介质
   - 设置文件权限为 600

3. 公开公钥:
   - 将 `vendor_pubkey.pem` 集成到Edge安装包
   - 路径: `configs/vendor_pubkey.pem`

### 客户侧（每次购买时操作）

1. 获取客户的Edge设备MAC地址:
   ```bash
   # 在客户的Edge设备上执行
   ip link show | grep -A1 "^2:" | grep link/ether | awk '{print $2}'
   ```

2. 生成许可证:
   ```bash
   ./license-gen -mode license \
     -id "LIC-CUSTOMER-001" \
     -mac "<客户MAC地址>" \
     -max 100 \
     -days 365
   ```

3. 交付给客户:
   - 将生成的 `license.lic` 发送给客户
   - 指导客户将许可证放到 `configs/license.lic`

4. 客户启用许可证:
   ```yaml
   # configs/config.yaml
   license:
     enabled: true
     path: "./configs/license.lic"
     pubkey_path: "./configs/vendor_pubkey.pem"
     grace_period: 72h
   ```

5. 重启Edge服务:
   ```bash
   ./start_all.sh --force
   ```

---

## 许可证续期

当许可证即将过期时，为同一客户生成新许可证:

```bash
# 使用相同的MAC地址，但延长有效期
./license-gen -mode license \
  -id "LIC-CUSTOMER-001-RENEWAL" \
  -mac "<原MAC地址>" \
  -max 100 \
  -days 365 \
  -output ./renewed_license.lic
```

客户替换许可证文件并重启服务即可。

---

## 故障排查

### 错误: "许可证MAC地址不匹配"

**原因**: 许可证绑定的MAC地址与设备实际MAC不符

**解决**:
1. 在Edge设备查看实际MAC:
   ```bash
   ip link | grep -A1 "^2:" | grep link/ether
   ```
2. 使用正确的MAC重新生成许可证

---

### 错误: "许可证已过期"

**原因**: 许可证超过有效期且宽限期已过

**解决**:
1. 生成新许可证（使用原MAC地址）
2. 替换 `configs/license.lic`
3. 重启服务

---

### 错误: "许可证签名验证失败"

**原因**: 公钥与签发许可证的私钥不匹配

**解决**:
1. 确认 `vendor_pubkey.pem` 与签发时的私钥配对
2. 如更换密钥，需重新生成所有许可证

---

## 安全建议

1. **私钥保护**:
   - 永远不要将私钥发送给客户
   - 使用加密存储保管私钥
   - 定期备份私钥

2. **许可证管理**:
   - 记录每个许可证的签发信息（客户、MAC、过期时间）
   - 使用唯一的许可证ID便于追溯
   - 设置合理的有效期（建议1年）

3. **MAC地址绑定**:
   - 许可证与MAC强绑定，无法在其他设备使用
   - 客户更换硬件时需重新签发许可证

---

## 示例工作流

### 场景: 为新客户签发许可证

```bash
# Step 1: 客户提供Edge设备MAC地址
# 客户在Edge设备执行: ip link show

# Step 2: 厂商签发许可证
./license-gen -mode license \
  -id "LIC-ACME-CORP-001" \
  -mac "52:54:00:12:34:56" \
  -max 150 \
  -days 730 \
  -output ./acme_corp_license.lic

# Step 3: 将 acme_corp_license.lic 发送给客户

# Step 4: 客户部署
# 1. 将许可证复制到 configs/license.lic
# 2. 修改 configs/config.yaml 启用许可证
# 3. 重启服务
```

输出:
```
✓ 许可证生成成功
  许可证ID: LIC-ACME-CORP-001
  MAC地址:  52:54:00:12:34:56
  最大设备: 150
  有效期:   730天
  过期时间: 2027-10-21
  输出文件: ./acme_corp_license.lic
```

---

## 技术细节

- **签名算法**: RS256 (RSA with SHA-256)
- **密钥长度**: 2048 bits
- **JWT标准**: RFC 7519
- **宽限期**: 许可证过期后默认72小时宽限期
- **验证点**: ZKP认证入口 (`GenerateChallenge()`)
