# 许可证MAC地址不匹配问题修复指南

## 问题描述

当Edge设备启动时，如果许可证绑定的MAC地址与设备实际MAC地址不匹配，会出现以下错误：

```
LICENSE_001: 许可证校验失败 - 许可证MAC地址不匹配
  许可证绑定MAC: 00:0C:29:3C:42:FE
  当前设备MAC: 70:b5:e8:30:6a:5f
  【修复方法】请使用当前设备MAC地址重新申请许可证
```

## 问题原因

1. **认证流程依赖许可证**：
   - Edge端在每次设备认证时都会调用 `license.Check()` 检查许可证
   - 如果MAC地址不匹配，会返回 `LICENSE_001` 错误，拒绝认证
   - 这导致设备无法获取挑战，无法进行认证，无法传输数据

2. **无法自动修复的原因**：
   - Edge端虽然有 `ApplyLicenseToken` 方法可以通过MQTT接收新许可证
   - 但是，由于许可证校验失败，设备无法通过认证，无法建立MQTT连接
   - 这是一个"鸡生蛋，蛋生鸡"的问题：需要许可证才能认证，但需要认证才能接收新许可证

## 解决方案

### 方案1：手动更新许可证文件（推荐）

**步骤**：

1. **获取当前设备MAC地址**：
   ```bash
   ip link show | grep -A1 "^2:" | grep link/ether | awk '{print $2}'
   # 或者
   ip link show | grep -E "eth0|en0|ens33|enp0s3" | grep link/ether | head -1 | awk '{print $2}'
   ```
   输出示例：`70:b5:e8:30:6a:5f`

2. **使用license-gen工具生成新许可证**：
   ```bash
   cd /home/zhang/xili/edge/tools/license-gen
   
   # 生成新许可证（使用当前设备MAC地址）
   ./license-gen -mode license \
     -id "LIC-CABINET-001-FIXED" \
     -mac "70:b5:e8:30:6a:5f" \
     -max 100 \
     -days 365 \
     -privkey ./vendor_privkey.pem \
     -output ./license_fixed.lic
   ```

3. **替换许可证文件**：
   ```bash
   # 备份旧许可证
   cp /home/zhang/xili/edge/configs/license.lic /home/zhang/xili/edge/configs/license.lic.backup
   
   # 复制新许可证
   cp /home/zhang/xili/edge/tools/license-gen/license_fixed.lic /home/zhang/xili/edge/configs/license.lic
   
   # 设置正确的权限
   chmod 600 /home/zhang/xili/edge/configs/license.lic
   ```

4. **重启Edge服务**：
   ```bash
   cd /home/zhang/xili/edge
   ./stop_all.sh
   ./start_all.sh
   ```

5. **验证修复**：
   ```bash
   # 查看Edge日志，确认许可证校验通过
   tail -f logs/edge.log | grep -i license
   
   # 应该看到类似以下日志：
   # "许可证服务初始化成功"
   # "license_id": "LIC-CABINET-001-FIXED"
   # "mac_address": "70:b5:e8:30:6a:5f"
   ```

### 方案2：临时禁用许可证验证（仅用于紧急情况）

如果无法立即生成新许可证，可以临时禁用许可证验证：

1. **修改配置文件**：
   ```yaml
   # configs/config.yaml
   license:
     enabled: false  # 临时禁用许可证验证
   ```

2. **重启Edge服务**：
   ```bash
   cd /home/zhang/xili/edge
   ./stop_all.sh
   ./start_all.sh
   ```

**注意**：禁用许可证验证会降低系统安全性，仅应在紧急情况下使用。

### 方案3：修改代码支持MAC不匹配时仍可接收许可证（需要代码修改）

如果需要支持在MAC不匹配时仍可通过MQTT接收新许可证，需要修改代码逻辑：

1. **修改 `edge/internal/license/service.go`**：
   - 在 `Check()` 方法中，当MAC不匹配时，不直接返回错误
   - 而是记录警告，但允许通过（等待Cloud下发新许可证）

2. **修改 `edge/internal/auth/service.go`**：
   - 在 `GenerateChallenge()` 方法中，当许可证MAC不匹配时，允许通过
   - 但记录警告日志

**注意**：此方案会降低安全性，不推荐在生产环境使用。

## 预防措施

1. **部署前检查MAC地址**：
   - 在部署Edge设备前，先获取设备MAC地址
   - 使用正确的MAC地址生成许可证

2. **使用自动化脚本**：
   - 创建部署脚本，自动获取MAC地址并生成许可证
   - 避免手动输入错误

3. **定期检查许可证状态**：
   - 定期检查Edge日志，确认许可证状态正常
   - 在许可证即将过期前提前续期

## 常见问题

### Q: 为什么会出现MAC地址不匹配？

A: 可能的原因：
- 设备更换了网卡
- 虚拟机迁移到新主机
- 手动修改了MAC地址
- 许可证生成时使用了错误的MAC地址

### Q: 能否通过Cloud端自动下发新许可证？

A: 理论上可以，但需要满足以下条件：
- Edge设备能够通过认证（需要有效的许可证）
- MQTT连接正常
- Cloud端有权限下发许可证

如果许可证MAC不匹配导致无法认证，则无法通过MQTT接收新许可证。

### Q: 如何避免此问题？

A: 
- 在部署前确认设备MAC地址
- 使用自动化脚本生成许可证
- 定期检查许可证状态
- 在设备迁移前更新许可证

## 相关文件

- 许可证生成工具：`edge/tools/license-gen/license-gen`
- 许可证服务代码：`edge/internal/license/service.go`
- 认证服务代码：`edge/internal/auth/service.go`
- 许可证配置文件：`edge/configs/license.lic`
- 许可证公钥：`edge/configs/vendor_pubkey.pem`

