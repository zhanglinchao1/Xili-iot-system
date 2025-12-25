# Cloud→Edge 指令下发功能清单

**版本**: 1.0  
**日期**: 2025-11-03  
**架构**: HTTP(批量同步) + MQTT(实时指令)

---

## 📋 目录

1. [架构设计](#架构设计)
2. [指令分类](#指令分类)
3. [MQTT Topic设计](#mqtt-topic设计)
4. [指令详细清单](#指令详细清单)
5. [实施优先级](#实施优先级)

---

## 架构设计

### 通信模式对比

| 维度 | HTTP批量同步 | MQTT实时指令 |
|------|------------|-------------|
| **方向** | Edge → Cloud | Cloud ↔ Edge |
| **用途** | 历史数据上传 | 实时指令下发、配置更新 |
| **延迟** | 5分钟 | 秒级 |
| **数据量** | 大（1000条/批） | 小（单条指令） |
| **可靠性** | 批量确认 | QoS 1保证 |

### 混合架构示意图

```
┌────────────────────────────────────────────────────────┐
│                    Cloud云端系统                        │
│                                                        │
│  ┌──────────────┐         ┌──────────────┐           │
│  │ HTTP Server  │         │ MQTT Broker  │           │
│  │ (接收数据)    │         │ (指令下发)    │           │
│  └──────┬───────┘         └──────┬───────┘           │
└─────────┼──────────────────────┼─────────────────────┘
          │                      │
          │ ↑ HTTP POST         │ ↕ MQTT Pub/Sub
          │ (批量数据)          │ (实时指令)
┌─────────▼──────────────────────▼─────────────────────┐
│                    Edge端系统                          │
│                                                        │
│  ┌──────────────┐         ┌──────────────┐           │
│  │ Cloud Sync   │         │ MQTT Client  │           │
│  │ (定期上报)    │         │ (指令接收)    │           │
│  └──────────────┘         └──────┬───────┘           │
│                                  │                    │
│  ┌──────────────────────────────▼──────────────────┐ │
│  │          指令处理模块                             │ │
│  │  - ConfigManager (配置管理)                      │ │
│  │  - LicenseService (许可证管理)                   │ │
│  │  - ControlService (远程控制)                     │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```

---

## 指令分类

### 按功能分类

| 分类 | 优先级 | 说明 | 实施阶段 |
|------|--------|------|---------|
| **配置管理** | P0 | 动态更新Edge端配置参数 | 第一阶段 |
| **许可证管理** | P0 | 许可证更新、吊销 | 第一阶段 |
| **远程查询** | P1 | 查询Edge状态、日志 | 第二阶段 |
| **远程控制** | P2 | 重启、模式切换等 | 第三阶段 |
| **固件升级** | P3 | OTA固件升级 | 未来扩展 |

---

## MQTT Topic设计

### Topic命名规范

```
格式: cloud/cabinets/{cabinet_id}/{category}/{action}

示例:
- cloud/cabinets/CABINET-001/commands/config       # 配置更新指令
- cloud/cabinets/CABINET-001/commands/license      # 许可证指令
- cloud/cabinets/CABINET-001/commands/query        # 查询指令
- cloud/cabinets/CABINET-001/commands/control      # 控制指令
- cloud/cabinets/CABINET-001/responses/{cmd_id}    # Edge响应
```

### Topic权限设计

```
Cloud端 (发布者):
- 发布权限: cloud/cabinets/+/commands/#
- 订阅权限: cloud/cabinets/+/responses/#

Edge端 (订阅者):
- 订阅权限: cloud/cabinets/{cabinet_id}/commands/#
- 发布权限: cloud/cabinets/{cabinet_id}/responses/#
```

---

## 指令详细清单

### 1. 配置管理指令 (P0)

#### 1.1 更新储能柜ID

**功能**: 动态修改Edge端的cabinet_id  
**优先级**: P0  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/config`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_001",
  "command_type": "config_update",
  "timestamp": "2025-11-03T10:00:00+08:00",
  "params": {
    "config_type": "cabinet_id",
    "old_value": "CABINET-001",
    "new_value": "CABINET-002",
    "operator": "admin",
    "reason": "设备重新分配"
  }
}
```

**Edge端处理**:
1. 验证新cabinet_id格式
2. 更新配置文件 `configs/config.yaml`
3. 更新MQTT Client ID
4. 重新订阅新的Topic
5. 返回执行结果

**响应格式**:
```json
{
  "command_id": "cmd_uuid_001",
  "status": "success",
  "message": "Cabinet ID更新成功",
  "details": {
    "old_id": "CABINET-001",
    "new_id": "CABINET-002",
    "updated_at": "2025-11-03T10:00:05+08:00"
  }
}
```

**失败场景**:
- 新ID格式错误
- 新ID已被占用
- 配置文件写入失败
- 权限不足

---

#### 1.2 更新告警阈值

**功能**: 动态调整7种传感器的告警阈值  
**优先级**: P0  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/config`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_002",
  "command_type": "config_update",
  "timestamp": "2025-11-03T10:05:00+08:00",
  "params": {
    "config_type": "alert_threshold",
    "sensor_type": "co2",
    "threshold": {
      "max": 5500.0,
      "unit": "ppm"
    },
    "operator": "admin",
    "reason": "根据现场环境调整"
  }
}
```

**支持的传感器类型**:
- `co2`: CO2浓度 (max)
- `co`: CO浓度 (max)
- `smoke`: 烟雾浓度 (max)
- `liquid_level`: 液位 (min/max)
- `conductivity`: 电导率 (min/max)
- `temperature`: 温度 (min/max)
- `flow`: 流速 (min/max)

**Edge端处理**:
1. 验证sensor_type有效性
2. 验证阈值范围合理性
3. 更新配置文件
4. 更新内存中的告警检测器
5. 返回执行结果

**响应格式**:
```json
{
  "command_id": "cmd_uuid_002",
  "status": "success",
  "message": "告警阈值更新成功",
  "details": {
    "sensor_type": "co2",
    "old_threshold": 5000.0,
    "new_threshold": 5500.0,
    "unit": "ppm",
    "updated_at": "2025-11-03T10:05:03+08:00"
  }
}
```

---

#### 1.3 更新同步间隔

**功能**: 调整Edge→Cloud批量同步的时间间隔  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/config`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_003",
  "command_type": "config_update",
  "timestamp": "2025-11-03T10:10:00+08:00",
  "params": {
    "config_type": "sync_interval",
    "interval_seconds": 600,
    "operator": "admin",
    "reason": "减少网络负载"
  }
}
```

**可配置范围**: 60秒 ~ 3600秒

**Edge端处理**:
1. 验证间隔值在有效范围内
2. 更新配置文件
3. 重启CloudSync定时器
4. 返回执行结果

---

#### 1.4 更新实时推送策略

**功能**: 配置传感器数据的实时推送行为  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/config`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_004",
  "command_type": "config_update",
  "timestamp": "2025-11-03T10:15:00+08:00",
  "params": {
    "config_type": "realtime_publish",
    "settings": {
      "enabled": true,
      "mode": "on_change",
      "change_threshold": 5.0,
      "sample_interval": 10
    },
    "operator": "admin"
  }
}
```

**推送模式**:
- `all`: 全部推送
- `on_change`: 变化推送（超过阈值才推送）
- `periodic`: 定期推送（按sample_interval）
- `off`: 关闭实时推送

---

### 2. 许可证管理指令 (P0)

#### 2.1 许可证更新推送

**功能**: 主动推送新许可证到Edge端  
**优先级**: P0  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/license`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_005",
  "command_type": "license_update",
  "timestamp": "2025-11-03T11:00:00+08:00",
  "params": {
    "action": "update",
    "license": {
      "license_id": "LIC-CABINET-001",
      "cabinet_id": "CABINET-001",
      "mac_address": "00:0c:29:3c:42:fe",
      "max_devices": -1,
      "expires_at": "2026-11-03T11:00:00+08:00",
      "status": "active",
      "permissions": ["auth", "collect", "alert", "statistics"]
    },
    "operator": "admin",
    "reason": "许可证续期"
  }
}
```

**Edge端处理**:
1. 验证MAC地址匹配
2. 更新内存中的许可证
3. 保存到缓存文件 `data/license_cache.json`
4. 记录审计日志
5. 返回执行结果

**响应格式**:
```json
{
  "command_id": "cmd_uuid_005",
  "status": "success",
  "message": "许可证更新成功",
  "details": {
    "license_id": "LIC-CABINET-001",
    "old_expires_at": "2025-11-03T11:00:00+08:00",
    "new_expires_at": "2026-11-03T11:00:00+08:00",
    "cached_at": "2025-11-03T11:00:02+08:00"
  }
}
```

---

#### 2.2 许可证吊销通知

**功能**: 立即吊销Edge端的许可证  
**优先级**: P0  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/license`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_006",
  "command_type": "license_revoke",
  "timestamp": "2025-11-03T11:05:00+08:00",
  "params": {
    "action": "revoke",
    "license_id": "LIC-CABINET-001",
    "reason": "客户欠费",
    "operator": "admin",
    "revoked_at": "2025-11-03T11:05:00+08:00"
  }
}
```

**Edge端处理**:
1. 立即标记许可证为已吊销
2. 停止所有需要许可证的功能
   - 停止ZKP认证
   - 停止数据采集
   - 停止告警生成
3. 记录吊销日志
4. 返回执行结果

**响应格式**:
```json
{
  "command_id": "cmd_uuid_006",
  "status": "success",
  "message": "许可证已吊销",
  "details": {
    "license_id": "LIC-CABINET-001",
    "revoked_at": "2025-11-03T11:05:00+08:00",
    "reason": "客户欠费",
    "services_stopped": ["auth", "collect", "alert"]
  }
}
```

---

#### 2.3 权限更新

**功能**: 动态调整许可证权限  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/license`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_007",
  "command_type": "license_permission_update",
  "timestamp": "2025-11-03T11:10:00+08:00",
  "params": {
    "action": "update_permissions",
    "permissions": {
      "add": ["firmware_update", "remote_control"],
      "remove": []
    },
    "operator": "admin",
    "reason": "客户升级到高级版"
  }
}
```

**可用权限列表**:
- `auth`: 设备认证
- `collect`: 数据采集
- `alert`: 告警功能
- `statistics`: 统计查询
- `firmware_update`: 固件升级（高级）
- `remote_control`: 远程控制（高级）
- `export_data`: 数据导出（高级）

---

### 3. 远程查询指令 (P1)

#### 3.1 查询Edge状态

**功能**: 获取Edge端实时运行状态  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/query`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_008",
  "command_type": "query_status",
  "timestamp": "2025-11-03T11:20:00+08:00",
  "params": {
    "query_type": "status",
    "include": ["system", "license", "devices", "services"]
  }
}
```

**Edge端响应**:
```json
{
  "command_id": "cmd_uuid_008",
  "status": "success",
  "data": {
    "system": {
      "cabinet_id": "CABINET-001",
      "version": "2.0.1",
      "uptime": "72h35m",
      "cpu_usage": 45.2,
      "memory_usage": 62.8,
      "disk_usage": 38.5
    },
    "license": {
      "status": "active",
      "expires_at": "2026-11-03T11:00:00+08:00",
      "remaining_days": 365,
      "permissions": ["auth", "collect", "alert", "statistics"]
    },
    "devices": {
      "total": 7,
      "online": 6,
      "offline": 1,
      "fault": 0
    },
    "services": {
      "mqtt": "running",
      "cloud_sync": "running",
      "alert": "running",
      "collector": "running"
    },
    "timestamp": "2025-11-03T11:20:02+08:00"
  }
}
```

---

#### 3.2 查询配置信息

**功能**: 获取Edge端当前配置  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/query`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_009",
  "command_type": "query_config",
  "timestamp": "2025-11-03T11:25:00+08:00",
  "params": {
    "query_type": "config",
    "sections": ["cloud", "alert", "mqtt"]
  }
}
```

**Edge端响应**:
```json
{
  "command_id": "cmd_uuid_009",
  "status": "success",
  "data": {
    "cloud": {
      "enabled": true,
      "endpoint": "https://cloud.example.com/api/v1",
      "cabinet_id": "CABINET-001",
      "sync_interval": "5m",
      "realtime": {
        "enabled": true,
        "mode": "on_change",
        "change_threshold": 5.0
      }
    },
    "alert": {
      "enabled": true,
      "thresholds": {
        "co2_max": 5500.0,
        "co_max": 50.0,
        "temperature_max": 60.0
      }
    },
    "mqtt": {
      "broker_address": "tcp://127.0.0.1:1883",
      "client_id": "edge-server-subscriber"
    },
    "timestamp": "2025-11-03T11:25:02+08:00"
  }
}
```

---

#### 3.3 查询设备列表

**功能**: 获取所有已注册设备信息  
**优先级**: P1  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/query`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_010",
  "command_type": "query_devices",
  "timestamp": "2025-11-03T11:30:00+08:00",
  "params": {
    "query_type": "devices",
    "include_offline": true
  }
}
```

**Edge端响应**:
```json
{
  "command_id": "cmd_uuid_010",
  "status": "success",
  "data": {
    "total": 7,
    "devices": [
      {
        "device_id": "CO2_SENSOR_001",
        "sensor_type": "co2",
        "status": "online",
        "last_seen_at": "2025-11-03T11:29:45+08:00",
        "last_value": 520.5,
        "model": "CO2-X200"
      },
      {
        "device_id": "CO_SENSOR_001",
        "sensor_type": "co",
        "status": "offline",
        "last_seen_at": "2025-11-03T10:15:30+08:00",
        "last_value": 12.3,
        "model": "CO-M100"
      }
      // ... 其他5个设备
    ],
    "timestamp": "2025-11-03T11:30:02+08:00"
  }
}
```

---

#### 3.4 查询日志

**功能**: 获取Edge端最近日志  
**优先级**: P2  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/query`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_011",
  "command_type": "query_logs",
  "timestamp": "2025-11-03T11:35:00+08:00",
  "params": {
    "query_type": "logs",
    "level": "error",
    "limit": 50,
    "time_range": {
      "start": "2025-11-03T00:00:00+08:00",
      "end": "2025-11-03T11:35:00+08:00"
    }
  }
}
```

---

### 4. 远程控制指令 (P2)

#### 4.1 重启服务

**功能**: 远程重启Edge端服务  
**优先级**: P2  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/control`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_012",
  "command_type": "service_restart",
  "timestamp": "2025-11-03T12:00:00+08:00",
  "params": {
    "action": "restart",
    "service": "cloud_sync",
    "operator": "admin",
    "reason": "同步服务异常"
  }
}
```

**可重启的服务**:
- `cloud_sync`: 云端同步服务
- `mqtt`: MQTT客户端
- `collector`: 数据采集服务
- `alert`: 告警检测服务
- `all`: 重启整个Edge系统（慎用）

**Edge端处理**:
1. 验证权限（需要`remote_control`权限）
2. 停止指定服务
3. 等待3秒
4. 重启服务
5. 返回执行结果

**响应格式**:
```json
{
  "command_id": "cmd_uuid_012",
  "status": "success",
  "message": "服务重启成功",
  "details": {
    "service": "cloud_sync",
    "stopped_at": "2025-11-03T12:00:02+08:00",
    "started_at": "2025-11-03T12:00:05+08:00",
    "status": "running"
  }
}
```

---

#### 4.2 切换运行模式

**功能**: 切换Edge端运行模式（调试/生产）  
**优先级**: P3  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/control`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_013",
  "command_type": "mode_switch",
  "timestamp": "2025-11-03T12:05:00+08:00",
  "params": {
    "action": "switch_mode",
    "mode": "debug",
    "operator": "admin",
    "reason": "故障排查"
  }
}
```

**可用模式**:
- `release`: 生产模式
- `debug`: 调试模式（增强日志）
- `test`: 测试模式

---

#### 4.3 清理缓存

**功能**: 清理Edge端缓存数据  
**优先级**: P3  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/control`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_014",
  "command_type": "cache_clear",
  "timestamp": "2025-11-03T12:10:00+08:00",
  "params": {
    "action": "clear_cache",
    "cache_type": "license",
    "operator": "admin"
  }
}
```

**可清理的缓存**:
- `license`: 许可证缓存
- `sensor_data`: 传感器数据缓存
- `all`: 所有缓存

---

### 5. 固件升级指令 (P3)

#### 5.1 固件升级通知

**功能**: 推送固件升级包信息  
**优先级**: P3  
**Topic**: `cloud/cabinets/{cabinet_id}/commands/control`

**指令格式**:
```json
{
  "command_id": "cmd_uuid_015",
  "command_type": "firmware_update",
  "timestamp": "2025-11-03T13:00:00+08:00",
  "params": {
    "action": "update_firmware",
    "firmware": {
      "version": "2.1.0",
      "download_url": "https://cloud.example.com/firmware/edge-v2.1.0.tar.gz",
      "checksum": "sha256:abc123...",
      "size_bytes": 52428800,
      "release_notes": "修复Bug、性能优化"
    },
    "schedule": {
      "immediate": false,
      "scheduled_at": "2025-11-03T02:00:00+08:00"
    },
    "operator": "admin"
  }
}
```

**Edge端处理流程**:
1. 验证权限（需要`firmware_update`权限）
2. 下载固件包到临时目录
3. 验证checksum
4. 等待scheduled_at时间
5. 停止服务
6. 备份当前版本
7. 安装新固件
8. 重启服务
9. 验证升级结果
10. 返回执行结果或回滚

---

## 通用指令格式规范

### 请求格式

所有Cloud→Edge指令必须包含以下字段：

```json
{
  "command_id": "string",           // 唯一指令ID (UUID)
  "command_type": "string",         // 指令类型
  "timestamp": "RFC3339",           // 指令时间戳
  "params": {                       // 指令参数（根据类型不同）
    ...
  },
  "timeout": 30,                    // 超时时间（秒），可选
  "retry": false                    // 是否允许重试，可选
}
```

### 响应格式

所有Edge→Cloud响应必须包含以下字段：

```json
{
  "command_id": "string",           // 对应的指令ID
  "status": "string",               // success/failed/timeout
  "message": "string",              // 执行结果消息
  "details": {                      // 详细信息（可选）
    ...
  },
  "timestamp": "RFC3339",           // 响应时间戳
  "error": {                        // 错误信息（失败时）
    "code": "string",
    "message": "string"
  }
}
```

### 错误码定义

| 错误码 | 说明 | 处理建议 |
|--------|------|---------|
| `INVALID_PARAMS` | 参数无效 | 检查参数格式 |
| `PERMISSION_DENIED` | 权限不足 | 检查许可证权限 |
| `CONFIG_ERROR` | 配置错误 | 检查配置合法性 |
| `SERVICE_UNAVAILABLE` | 服务不可用 | 重启服务或稍后重试 |
| `TIMEOUT` | 执行超时 | 增加超时时间或检查系统负载 |
| `INTERNAL_ERROR` | 内部错误 | 查看日志排查 |

---

## 实施优先级

### 第一阶段（P0）- 核心功能

**必须实现**：

1. ✅ **配置管理**
   - 更新储能柜ID
   - 更新告警阈值
   
2. ✅ **许可证管理**
   - 许可证更新推送
   - 许可证吊销通知

**目标**：实现基本的远程配置能力

---

### 第二阶段（P1）- 扩展功能

**推荐实现**：

3. ✅ **配置管理扩展**
   - 更新同步间隔
   - 更新实时推送策略
   
4. ✅ **远程查询**
   - 查询Edge状态
   - 查询配置信息
   - 查询设备列表
   
5. ✅ **许可证管理扩展**
   - 权限动态更新

**目标**：增强运维便利性

---

### 第三阶段（P2）- 高级功能

**可选实现**：

6. ✅ **远程控制**
   - 重启服务
   - 切换运行模式
   - 清理缓存
   
7. ✅ **远程查询扩展**
   - 查询日志

**目标**：提升故障排查效率

---

### 未来扩展（P3）

**长期规划**：

8. ⏰ **固件升级**
   - OTA固件升级
   - 版本回滚

**目标**：完整的设备生命周期管理

---

## Edge端实施要点

### 1. MQTT客户端配置

```go
// Edge/internal/mqtt/cloud_subscriber.go
type CloudSubscriber struct {
    client       mqtt.Client
    cabinetID    string
    configMgr    *config.Manager
    licenseMgr   *license.Service
    controlSvc   *control.Service
    logger       *zap.Logger
}

func (s *CloudSubscriber) Start() error {
    // 订阅所有指令主题
    topics := map[string]byte{
        fmt.Sprintf("cloud/cabinets/%s/commands/config", s.cabinetID):  1,
        fmt.Sprintf("cloud/cabinets/%s/commands/license", s.cabinetID): 1,
        fmt.Sprintf("cloud/cabinets/%s/commands/query", s.cabinetID):   1,
        fmt.Sprintf("cloud/cabinets/%s/commands/control", s.cabinetID): 1,
    }
    
    token := s.client.SubscribeMultiple(topics, s.handleCommand)
    return token.Error()
}
```

### 2. 指令路由器

```go
func (s *CloudSubscriber) handleCommand(client mqtt.Client, msg mqtt.Message) {
    topic := msg.Topic()
    
    if strings.Contains(topic, "/config") {
        s.handleConfigCommand(msg.Payload())
    } else if strings.Contains(topic, "/license") {
        s.handleLicenseCommand(msg.Payload())
    } else if strings.Contains(topic, "/query") {
        s.handleQueryCommand(msg.Payload())
    } else if strings.Contains(topic, "/control") {
        s.handleControlCommand(msg.Payload())
    }
}
```

### 3. 权限验证

```go
func (s *CloudSubscriber) checkPermission(commandType string) error {
    requiredPermissions := map[string]string{
        "service_restart":   "remote_control",
        "firmware_update":   "firmware_update",
        "cache_clear":       "remote_control",
    }
    
    if perm, ok := requiredPermissions[commandType]; ok {
        if !s.licenseMgr.HasPermission(perm) {
            return fmt.Errorf("权限不足: 需要%s权限", perm)
        }
    }
    return nil
}
```

### 4. 响应发送

```go
func (s *CloudSubscriber) sendResponse(commandID string, status string, data interface{}) {
    response := Response{
        CommandID: commandID,
        Status:    status,
        Timestamp: time.Now(),
        Data:      data,
    }
    
    payload, _ := json.Marshal(response)
    topic := fmt.Sprintf("cloud/cabinets/%s/responses/%s", s.cabinetID, commandID)
    
    s.client.Publish(topic, 1, false, payload)
}
```

---

## Cloud端实施要点

### 1. 指令发送服务

```go
// Cloud/internal/service/command_service.go
type CommandService struct {
    mqttClient mqtt.Client
    db         *database.DB
    logger     *zap.Logger
}

func (s *CommandService) SendCommand(cabinetID string, cmd *Command) error {
    // 1. 保存指令到数据库
    if err := s.db.SaveCommand(cmd); err != nil {
        return err
    }
    
    // 2. 发送MQTT指令
    topic := fmt.Sprintf("cloud/cabinets/%s/commands/%s", cabinetID, cmd.Category)
    payload, _ := json.Marshal(cmd)
    
    token := s.mqttClient.Publish(topic, 1, false, payload)
    token.Wait()
    
    if err := token.Error(); err != nil {
        return fmt.Errorf("发送指令失败: %w", err)
    }
    
    // 3. 等待响应（可选，异步处理）
    return nil
}
```

### 2. 响应监听器

```go
func (s *CommandService) SubscribeResponses() error {
    topic := "cloud/cabinets/+/responses/+"
    
    token := s.mqttClient.Subscribe(topic, 1, s.handleResponse)
    return token.Error()
}

func (s *CommandService) handleResponse(client mqtt.Client, msg mqtt.Message) {
    var response Response
    json.Unmarshal(msg.Payload(), &response)
    
    // 更新数据库中的指令状态
    s.db.UpdateCommandStatus(response.CommandID, response.Status, response.Data)
    
    // 记录审计日志
    s.logger.Info("收到指令响应",
        zap.String("command_id", response.CommandID),
        zap.String("status", response.Status))
}
```

---

## 测试用例

### 1. 配置更新测试

```bash
# 发送Cabinet ID更新指令
mosquitto_pub -h cloud.example.com -p 1883 \
  -u cloud_admin -P password \
  -t "cloud/cabinets/CABINET-001/commands/config" \
  -m '{
    "command_id": "test_001",
    "command_type": "config_update",
    "timestamp": "2025-11-03T14:00:00+08:00",
    "params": {
      "config_type": "cabinet_id",
      "new_value": "CABINET-NEW-001",
      "operator": "admin"
    }
  }'

# 监听响应
mosquitto_sub -h cloud.example.com -p 1883 \
  -u cloud_admin -P password \
  -t "cloud/cabinets/CABINET-001/responses/#" \
  -v
```

### 2. 许可证吊销测试

```bash
mosquitto_pub -h cloud.example.com -p 1883 \
  -u cloud_admin -P password \
  -t "cloud/cabinets/CABINET-001/commands/license" \
  -m '{
    "command_id": "test_002",
    "command_type": "license_revoke",
    "timestamp": "2025-11-03T14:05:00+08:00",
    "params": {
      "action": "revoke",
      "license_id": "LIC-CABINET-001",
      "reason": "测试吊销",
      "operator": "admin"
    }
  }'
```

---

## 安全考虑

### 1. 认证与授权

- ✅ MQTT使用用户名/密码认证
- ✅ Cloud端ACL权限控制
- ✅ Edge端验证指令签名（可选）
- ✅ 许可证权限验证

### 2. 数据加密

- ✅ 使用MQTT over TLS (端口8883)
- ✅ 敏感参数加密传输
- ✅ 指令完整性校验

### 3. 审计日志

- ✅ 记录所有指令发送
- ✅ 记录所有指令执行结果
- ✅ 记录操作员信息

---

## 监控指标

### Edge端监控

- 指令接收计数
- 指令执行成功率
- 指令平均响应时间
- MQTT连接状态

### Cloud端监控

- 指令发送计数
- 指令超时率
- 响应接收延迟
- MQTT Broker负载

---

## 总结

本文档详细列出了Cloud→Edge需要下发的所有指令类型，包括：

1. **配置管理**（4个指令）- P0/P1
2. **许可证管理**（3个指令）- P0/P1
3. **远程查询**（4个指令）- P1/P2
4. **远程控制**（3个指令）- P2/P3
5. **固件升级**（1个指令）- P3

建议**优先实现P0和P1功能**，满足基本的远程管理需求。后续根据实际需要逐步增加P2和P3功能。

---

**下一步行动**:
1. ✅ 确认功能优先级
2. ✅ 实施Edge端MQTT订阅
3. ✅ 实施Cloud端指令发送
4. ✅ 编写单元测试和集成测试
5. ✅ 部署到测试环境验证

---

**文档版本历史**:
- v1.0 (2025-11-03): 初始版本，完整指令清单

