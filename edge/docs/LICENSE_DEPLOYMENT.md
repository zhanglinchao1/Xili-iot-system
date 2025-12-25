# Edge系统 - 单包授权(SPA)部署指南

**Single Package Authorization Deployment Guide**

**版本**: v1.0
**日期**: 2025-10-21
**状态**: 生产就绪

---

## 📋 目录

- [1. 概述](#1-概述)
- [2. 架构说明](#2-架构说明)
- [3. 部署流程](#3-部署流程)
- [4. 许可证管理](#4-许可证管理)
- [5. 故障排查](#5-故障排查)
- [6. 安全建议](#6-安全建议)

---

## 1. 概述

### 1.1 什么是SPA

Edge系统的单包授权(SPA)是一种基于JWT的许可证管理方案，用于控制软件的商业使用。主要特性：

- ✅ **设备绑定**: 许可证与MAC地址强绑定，防止未授权复制
- ✅ **设备限额**: 控制单个Edge实例可管理的最大设备数
- ✅ **自动过期**: 设定有效期，到期后自动失效
- ✅ **宽限期**: 过期后72小时宽限期，避免服务突然中断
- ✅ **无感集成**: 网关设备无需任何代码改动
- ✅ **单点校验**: 仅在认证入口检查，性能开销极低

### 1.2 工作原理

```
┌─────────────────────────────────────────────────────────┐
│                   许可证验证流程                          │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  网关设备                Edge服务端                       │
│     │                       │                            │
│     │  1. 请求认证挑战      │                            │
│     ├──────────────────────>│                            │
│     │                       │                            │
│     │                   【许可证检查】                    │
│     │                       ├─ 验证签名                  │
│     │                       ├─ 检查MAC地址               │
│     │                       ├─ 检查过期时间              │
│     │                       └─ 检查设备限额              │
│     │                       │                            │
│     │  2a. 许可证有效       │                            │
│     │     返回挑战          │                            │
│     │<──────────────────────┤                            │
│     │                       │                            │
│     │  2b. 许可证无效       │                            │
│     │     返回错误          │                            │
│     │<─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┤                            │
│     │   LICENSE_001         │                            │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

**关键点**: 网关设备不知道许可证的存在，只会看到认证成功或失败。

---

## 2. 架构说明

### 2.1 系统组件

| 组件 | 描述 | 位置 |
|------|------|------|
| **许可证生成工具** | 厂商侧工具，生成JWT许可证 | `tools/license-gen/` |
| **许可证文件** | JWT格式的许可证 | `configs/license.lic` |
| **厂商公钥** | RSA公钥，验证许可证签名 | `configs/vendor_pubkey.pem` |
| **许可证服务** | Edge内部服务，验证许可证 | `internal/license/` |
| **认证服务** | 集成许可证检查的ZKP认证 | `internal/auth/` |

### 2.2 许可证格式

JWT Header:
```json
{
  "alg": "RS256",
  "typ": "JWT"
}
```

JWT Payload (Claims):
```json
{
  "lic": "LIC-2025-001",          // 许可证ID
  "mac": "00:11:22:33:44:55",     // 绑定的MAC地址
  "max": 100,                      // 最大设备数
  "iat": 1700000000,               // 签发时间（Unix时间戳）
  "exp": 1731536000,               // 过期时间（Unix时间戳）
  "iss": "Edge-Vendor",            // 签发者
  "sub": "LIC-2025-001"            // 主题（许可证ID）
}
```

JWT Signature:
```
RS256(base64UrlEncode(header) + "." + base64UrlEncode(payload), vendor_private_key)
```

---

## 3. 部署流程

### 3.1 厂商侧准备（一次性操作）

#### Step 1: 生成RSA密钥对

```bash
cd tools/license-gen

# 编译工具
go build -o license-gen main.go

# 生成密钥对
./license-gen -mode keygen \
  -privkey ./vendor_privkey.pem \
  -pubkey ./vendor_pubkey.pem
```

**输出**:
```
✓ RSA密钥对生成成功
  私钥: ./vendor_privkey.pem
  公钥: ./vendor_pubkey.pem
```

**重要**:
- 私钥 (`vendor_privkey.pem`) 必须保密存储，仅厂商持有
- 公钥 (`vendor_pubkey.pem`) 将集成到Edge安装包中

#### Step 2: 集成公钥到安装包

```bash
# 将公钥复制到configs目录
cp vendor_pubkey.pem /path/to/edge/configs/

# 打包Edge发行版时包含此公钥
# 客户收到的安装包结构:
# edge/
#   ├── edge                  (二进制)
#   ├── configs/
#   │   ├── config.yaml
#   │   └── vendor_pubkey.pem (厂商公钥)
#   └── ...
```

---

### 3.2 为客户生成许可证

#### Step 1: 获取客户Edge设备的MAC地址

要求客户在Edge设备上执行:
```bash
# Linux
ip link show | grep -A1 "^2:" | grep link/ether | awk '{print $2}'

# 或使用
ifconfig | grep ether | head -1 | awk '{print $2}'
```

示例输出:
```
00:15:5d:41:5b:ca
```

#### Step 2: 生成许可证

```bash
./license-gen -mode license \
  -id "LIC-CUSTOMER-001" \
  -mac "00:15:5d:41:5b:ca" \
  -max 100 \
  -days 365 \
  -privkey ./vendor_privkey.pem \
  -output ./customer_license.lic
```

**参数说明**:
- `-id`: 许可证唯一标识（建议格式: `LIC-客户名-序号`）
- `-mac`: 客户Edge设备的MAC地址
- `-max`: 允许管理的最大设备数
- `-days`: 有效期天数（推荐365天）

**输出**:
```
✓ 许可证生成成功
  许可证ID: LIC-CUSTOMER-001
  MAC地址:  00:15:5d:41:5b:ca
  最大设备: 100
  有效期:   365天
  过期时间: 2026-10-21
  输出文件: ./customer_license.lic
```

#### Step 3: 交付许可证

将生成的 `customer_license.lic` 文件通过安全渠道发送给客户。

---

### 3.3 客户侧部署

#### Step 1: 将许可证放到configs目录

```bash
# 将许可证文件复制到Edge安装目录
cp customer_license.lic /path/to/edge/configs/license.lic
```

#### Step 2: 启用许可证验证

编辑 `configs/config.yaml`:
```yaml
# 许可证配置（单包授权SPA）
license:
  enabled: true                              # 启用许可证验证
  path: "./configs/license.lic"              # 许可证文件路径
  pubkey_path: "./configs/vendor_pubkey.pem" # 厂商公钥路径
  grace_period: 72h                          # 过期宽限期
```

#### Step 3: 重启Edge服务

```bash
# 使用启动脚本（推荐）
./start_all.sh --force

# 或手动启动
./edge -config ./configs/config.yaml
```

#### Step 4: 验证许可证生效

检查启动日志:
```bash
tail -f logs/edge.log | grep -i license
```

期望输出:
```
INFO    许可证服务已启用    {"max_devices": 100}
```

---

## 4. 许可证管理

### 4.1 查看许可证信息

许可证信息在Edge启动时会记录到日志:
```
INFO    许可证服务初始化成功
  license_id: LIC-CUSTOMER-001
  mac_address: 00:15:5d:41:5b:ca
  max_devices: 100
  expires_at: 2026-10-21T14:40:18+08:00
```

### 4.2 许可证续期

#### 方式1: 生成新许可证（推荐）

```bash
# 使用相同MAC但新的许可证ID
./license-gen -mode license \
  -id "LIC-CUSTOMER-001-RENEWAL" \
  -mac "00:15:5d:41:5b:ca" \
  -max 100 \
  -days 365 \
  -output ./renewed_license.lic
```

客户替换 `configs/license.lic` 并重启服务。

#### 方式2: 延长现有许可证有效期

重新生成相同ID的许可证，延长 `-days` 参数。

### 4.3 修改设备限额

如果客户需要增加设备数:
```bash
# 生成新许可证，增加max_devices
./license-gen -mode license \
  -id "LIC-CUSTOMER-001-UPGRADE" \
  -mac "00:15:5d:41:5b:ca" \
  -max 200 \  # 从100增加到200
  -days 365 \
  -output ./upgraded_license.lic
```

### 4.4 许可证过期处理

**过期后72小时内（宽限期）**:
- Edge继续正常运行
- 日志中会有警告信息
- 建议客户尽快续期

**过期超过72小时**:
- Edge拒绝所有新的设备认证请求
- 已认证的设备会话不受影响（直到session过期）
- 客户必须获取新许可证才能恢复服务

---

## 5. 故障排查

### 5.1 常见问题

#### 问题1: 启动时报"初始化许可证服务失败"

**可能原因**:
1. 许可证文件不存在或路径错误
2. 公钥文件不存在或路径错误
3. 许可证文件损坏或格式错误

**解决方案**:
```bash
# 1. 检查文件是否存在
ls -l configs/license.lic configs/vendor_pubkey.pem

# 2. 检查配置路径
cat configs/config.yaml | grep -A 4 "^license:"

# 3. 验证许可证格式
head -c 50 configs/license.lic
# 应该看到 "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." 这样的JWT格式
```

---

#### 问题2: 认证时报"LICENSE_001: 许可证校验失败"

**可能原因**:
1. MAC地址不匹配
2. 许可证已过期且超过宽限期
3. 许可证签名验证失败
4. 设备数超限

**解决方案**:
```bash
# 1. 检查当前设备MAC地址
ip link show | grep -A1 "^2:" | grep link/ether

# 2. 查看详细错误日志
tail -50 logs/edge.log | grep LICENSE

# 3. 如果MAC不匹配，需要重新生成许可证
# 4. 如果过期，需要续期许可证
```

---

#### 问题3: 日志显示"许可证已过期但在宽限期内"

**说明**: 这是警告，服务仍正常运行

**操作**:
1. 联系厂商申请续期许可证
2. 在宽限期结束前（72小时）完成替换
3. 替换许可证后重启服务

---

### 5.2 诊断命令

#### 查看许可证内容（仅供调试）

```bash
# 提取JWT payload并Base64解码
cat configs/license.lic | cut -d'.' -f2 | base64 -d 2>/dev/null | jq .
```

示例输出:
```json
{
  "lic": "LIC-CUSTOMER-001",
  "mac": "00:15:5d:41:5b:ca",
  "max": 100,
  "iat": 1729489218,
  "exp": 1761025218,
  "iss": "Edge-Vendor",
  "sub": "LIC-CUSTOMER-001"
}
```

#### 检查公钥信息

```bash
openssl rsa -pubin -in configs/vendor_pubkey.pem -text -noout
```

---

## 6. 安全建议

### 6.1 厂商侧

1. **私钥保护**:
   - 使用加密存储保管 `vendor_privkey.pem`
   - 限制私钥文件权限: `chmod 600 vendor_privkey.pem`
   - 定期备份私钥到安全介质
   - 永远不要将私钥发送给客户或第三方

2. **许可证签发记录**:
   - 维护许可证签发台账（客户、MAC、过期时间、设备限额）
   - 使用唯一的许可证ID便于追溯
   - 记录每次续期和升级操作

3. **密钥轮换**:
   - 定期（如每2年）更换密钥对
   - 新旧密钥共存一段时间，平滑过渡
   - 通知所有客户更新公钥

### 6.2 客户侧

1. **许可证保护**:
   - 限制许可证文件权限: `chmod 644 configs/license.lic`
   - 不要将许可证共享给其他设备
   - 定期备份许可证文件

2. **硬件更换**:
   - 更换网卡会导致MAC变化，需重新申请许可证
   - 联系厂商提供新的MAC地址

3. **过期提醒**:
   - 在许可证过期前30天联系厂商续期
   - 避免在宽限期最后时刻才处理

---

## 7. 附录

### 7.1 完整部署检查清单

#### 厂商侧检查清单
- [ ] 生成RSA密钥对
- [ ] 安全保管私钥（加密、备份、权限控制）
- [ ] 将公钥集成到Edge安装包
- [ ] 测试许可证生成工具
- [ ] 建立许可证签发记录台账

#### 客户部署检查清单
- [ ] 获取Edge设备MAC地址
- [ ] 收到厂商签发的许可证文件
- [ ] 将许可证放到 `configs/license.lic`
- [ ] 确认 `configs/vendor_pubkey.pem` 存在
- [ ] 修改 `config.yaml` 启用许可证
- [ ] 重启Edge服务
- [ ] 检查日志确认许可证生效

---

### 7.2 许可证配置示例

#### 开发环境配置（禁用许可证）
```yaml
license:
  enabled: false
  path: "./configs/license.lic"
  pubkey_path: "./configs/vendor_pubkey.pem"
  grace_period: 72h
```

#### 生产环境配置（启用许可证）
```yaml
license:
  enabled: true
  path: "./configs/license.lic"
  pubkey_path: "./configs/vendor_pubkey.pem"
  grace_period: 72h
```

---

### 7.3 支持联系

如有问题，请联系技术支持：
- 技术支持邮箱: support@edge-vendor.com
- 许可证申请: license@edge-vendor.com

---

**文档版本**: v1.0
**最后更新**: 2025-10-21
