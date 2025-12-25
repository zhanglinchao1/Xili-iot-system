# Edge系统 - 完整API接口文档

## 📋 概述

本文档详细描述了Edge储能柜边缘认证网关系统的所有API接口，包括认证、设备管理、数据采集、告警管理、实时推送等功能模块。

**系统版本**: v1.0.0
**API版本**: v1
**基础URL**: `http://localhost:8001`
**WebSocket URL**: `ws://localhost:8001/ws`
**文档更新日期**: 2025-11-03

### API端点索引

#### 系统接口
- `GET /health` - 系统健康检查
- `GET /ready` - 系统就绪检查
- `GET /ws` - WebSocket实时推送连接

#### 许可证管理接口 (`/api/v1/license`)
- `GET /api/v1/license/info` - 获取许可证信息

#### 认证接口 (`/api/v1/auth`)
- `POST /api/v1/auth/challenge` - 获取认证挑战
- `POST /api/v1/auth/verify` - 验证零知识证明
- `POST /api/v1/auth/refresh` - 刷新会话

#### 设备管理接口 (`/api/v1/devices`)
- `GET /api/v1/devices` - 获取设备列表
- `GET /api/v1/devices/statistics` - 设备统计信息
- `GET /api/v1/devices/:id` - 获取设备详情
- `GET /api/v1/devices/:id/latest-data` - 获取设备最新数据
- `POST /api/v1/devices` - 注册设备
- `PUT /api/v1/devices/:id` - 更新设备信息
- `DELETE /api/v1/devices/:id` - 注销设备
- `POST /api/v1/devices/:id/heartbeat` - 设备心跳

#### 储能柜管理接口 (`/api/v1/cabinets`)
- `GET /api/v1/cabinets` - 获取储能柜列表
- `GET /api/v1/cabinets/:cabinet_id/devices` - 按储能柜获取设备列表

#### 数据采集接口 (`/api/v1/data`)
- `POST /api/v1/data/collect` - 数据采集 (需要JWT认证)
- `GET /api/v1/data/query` - 查询历史数据
- `GET /api/v1/data/statistics` - 获取数据统计

#### 告警管理接口 (`/api/v1/alerts`)
- `GET /api/v1/alerts` - 获取告警列表
- `POST /api/v1/alerts` - 创建告警
- `PUT /api/v1/alerts/:id/resolve` - 解决告警
- `GET /api/v1/alerts/config` - 获取告警配置(阈值)

#### 日志记录接口 (`/api/v1/logs`)
- `GET /api/v1/logs/alerts` - 获取告警日志
- `GET /api/v1/logs/auth` - 获取认证日志
- `DELETE /api/v1/logs/alerts/batch` - 批量删除告警日志
- `DELETE /api/v1/logs/auth/batch` - 批量删除认证日志
- `DELETE /api/v1/logs/auth/clear` - 清空所有认证日志

### 系统架构

本系统采用**双通道数据接收架构**:

1. **HTTP/HTTPS 通道**: 用于设备管理、配置操作、历史数据查询(Web管理界面)
2. **MQTT 通道**: 用于高频传感器数据的实时传输(低延迟、高吞吐量)
3. **WebSocket 通道**: 用于Web前端实时数据推送

两种数据通道接收的数据**统一存储**到相同的数据库表,Web管理界面可以查看所有来源的数据。

```
网关设备 ─┬─→ HTTP API (/api/v1/data/collect) ──┐
          │                                      ├─→ 统一数据库 → Web管理界面
          └─→ MQTT (sensors/#) ─→ MQTT订阅器 ──┘
                                      ↓
                               WebSocket Hub
                                      ↓
                               实时推送到前端
```

---

## 🔧 系统接口

### 健康检查

#### 1. 系统健康检查
```http
GET /health
```

**功能**: 检查系统基本运行状态  
**认证**: 无需认证  
**响应**:
```json
{
  "status": "ok",
  "timestamp": 1728968468,
  "service": "edge-system"
}
```

#### 2. 系统就绪检查
```http
GET /ready
```

**功能**: 检查系统各服务就绪状态
**认证**: 无需认证
**响应**:
```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "zkp": "ok",
    "services": "ok"
  }
}
```

---

## 🔌 WebSocket 实时推送接口

### 连接 WebSocket

```
ws://localhost:8001/ws
```

**功能**: 建立WebSocket连接,接收实时传感器数据、设备状态、告警信息

**认证**: 无需认证(专为Web管理界面设计)

**连接示例**:
```javascript
const ws = new WebSocket('ws://localhost:8001/ws');

ws.onopen = function() {
    console.log('✅ WebSocket连接成功');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('收到消息:', message);
};

ws.onclose = function() {
    console.log('🔌 WebSocket连接断开');
};

ws.onerror = function(error) {
    console.error('❌ WebSocket错误:', error);
};
```

### WebSocket 消息类型

服务端通过WebSocket推送以下类型的实时消息:

#### 1. 传感器数据 (sensor_data)

**消息格式**:
```json
{
  "type": "sensor_data",
  "data": {
    "device_id": "CO2_SENSOR_001",
    "sensor_type": "co2",
    "value": 420.5,
    "unit": "ppm",
    "timestamp": "2025-10-27T15:52:01Z",
    "quality": 100
  }
}
```

**触发时机**: 当MQTT订阅器接收到网关发布的传感器数据时

#### 2. 设备状态 (device_status)

**消息格式**:
```json
{
  "type": "device_status",
  "data": {
    "device_id": "CO2_SENSOR_001",
    "status": "online",
    "timestamp": "2025-10-27T15:52:00Z",
    "metadata": {
      "mqtt_enabled": true,
      "zkp_auth": true
    }
  }
}
```

**触发时机**: 当设备状态发生变化(上线/离线/故障)时

#### 3. 告警信息 (alert)

**消息格式**:
```json
{
  "type": "alert",
  "data": {
    "device_id": "CO2_SENSOR_001",
    "alert_type": "threshold_exceeded",
    "severity": "high",
    "message": "CO2浓度超过阈值",
    "value": 1200.0,
    "threshold": 1000.0,
    "timestamp": "2025-10-27T15:52:00Z"
  }
}
```

**触发时机**: 当检测到传感器数值超过阈值时

#### 4. 心跳信息 (heartbeat)

**消息格式**:
```json
{
  "type": "heartbeat",
  "data": {
    "device_id": "CO2_SENSOR_001",
    "timestamp": "2025-10-27T15:52:00Z"
  }
}
```

**触发时机**: 当接收到设备心跳时

### 前端集成示例

```javascript
// 创建WebSocket连接
const RealtimeMonitor = {
    websocket: null,

    init() {
        this.connectWebSocket();
    },

    connectWebSocket() {
        this.websocket = new WebSocket('ws://localhost:8001/ws');

        this.websocket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };

        this.websocket.onclose = () => {
            // 断线重连
            setTimeout(() => this.connectWebSocket(), 5000);
        };
    },

    handleMessage(message) {
        switch(message.type) {
            case 'sensor_data':
                this.updateSensorDisplay(message.data);
                break;
            case 'device_status':
                this.updateDeviceStatus(message.data);
                break;
            case 'alert':
                this.showAlert(message.data);
                break;
            case 'heartbeat':
                this.updateHeartbeat(message.data);
                break;
        }
    },

    updateSensorDisplay(data) {
        // 更新传感器显示面板
        const panel = document.getElementById(`sensor-${data.sensor_type}`);
        if (panel) {
            panel.querySelector('.value').textContent = data.value;
            panel.querySelector('.unit').textContent = data.unit;
        }
    },

    showAlert(alert) {
        // 显示告警通知
        const notification = `🚨 ${alert.device_id}: ${alert.message}`;
        console.warn(notification);
    }
};

// 启动实时监控
RealtimeMonitor.init();
```

### WebSocket 连接说明

- **自动重连**: 客户端应实现断线自动重连机制
- **心跳保活**: WebSocket连接会自动保持活跃状态
- **消息格式**: 所有消息均为JSON格式
- **数据来源**: WebSocket推送的数据来自MQTT订阅器接收到的实时数据
- **并发支持**: 支持多个Web客户端同时连接

---

## 🔐 认证接口 (`/api/v1/auth`)

### 零知识证明认证流程

> **技术说明**: 本系统基于 [Gnark](https://github.com/Consensys/gnark) 实现零知识证明认证。详细协议文档请参考 [ZKP-PROTOCOL.md](docs/ZKP-PROTOCOL.md)

**认证流程概览**:
```
1. 客户端 → 服务端: 请求challenge
2. 服务端 → 客户端: 返回random nonce
3. 客户端: 使用secret生成proof（客户端完成）
4. 客户端 → 服务端: 提交proof + public witness
5. 服务端: 验证proof，颁发JWT token
```

#### 1. 获取认证挑战
```http
POST /api/v1/auth/challenge
```

**功能**: 设备请求认证挑战，开始零知识证明认证流程  
**认证**: 无需认证  

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 是 | 设备唯一标识，长度1-64字符 |

**响应**:
```json
{
  "challenge_id": "d612a4fc-41d6-4d63-8add-ffb83a6a118c",
  "nonce": "a1b2c3d4e5f6...64位十六进制字符串",
  "expires_at": "2025-10-15T15:00:00Z"
}
```

**响应参数说明**:
| 参数 | 类型 | 说明 |
|------|------|------|
| challenge_id | string | 挑战唯一标识（UUID格式），提交proof时使用 |
| nonce | string | 随机挑战值（64位十六进制），用于计算response |
| expires_at | datetime | 挑战过期时间（通常5分钟），过期后需重新获取 |

**错误码**:
- `INVALID_REQUEST`: 请求参数错误
- `INVALID_DEVICE_ID`: 设备ID格式错误（长度超限或包含非法字符）
- `CHALLENGE_FAILED`: 生成挑战失败（服务器内部错误）
- `DEVICE_NOT_FOUND`: 设备未注册

**示例**:
```bash
curl -X POST http://localhost:8001/api/v1/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"device_id":"CO2_SENSOR_20251015_140552"}'
```

---

#### 2. 验证零知识证明
```http
POST /api/v1/auth/verify
```

**功能**: 设备提交零知识证明进行身份验证  
**认证**: 无需认证  

**重要说明**: 
- ⚠️ **Proof生成在客户端完成**，服务端只负责验证
- 🔐 设备的 `secret` **永远不会**传输到服务端
- 🔒 每次认证的proof都不同，无法重放

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "challenge_id": "d612a4fc-41d6-4d63-8add-ffb83a6a118c",
  "proof": {
    "proof": "base64编码的二进制proof数据（约192字节）",
    "public_witness": {
      "device_id": "CO2_SENSOR_20251015_140552",
      "challenge": "a1b2c3d4e5f6...从步骤1获取的nonce",
      "commitment": "设备注册时的commitment值",
      "response": "MiMC(secret, challenge)的十六进制结果"
    }
  }
}
```

**请求参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 是 | 设备ID，必须与challenge请求时一致 |
| challenge_id | string | 是 | 步骤1返回的challenge_id |
| proof.proof | string | 是 | **Groth16 proof的Base64编码**<br>- 原始proof是二进制数据（~192字节）<br>- 客户端使用proving key生成<br>- 服务端使用verifying key验证 |
| proof.public_witness | object | 是 | **公开输入集合**（见下表） |

**Public Witness 详解**:

| 字段 | 类型 | 说明 | 如何计算 |
|------|------|------|----------|
| device_id | string | 设备标识 | 与请求中的device_id相同 |
| challenge | string | 挑战值 | 步骤1中获取的nonce（64位十六进制） |
| commitment | string | 身份承诺 | `MiMC(secret, device_id)`<br>注册时预先计算并存储 |
| response | string | 挑战响应 | `MiMC(secret, challenge)`<br>每次认证时现场计算 |


**响应（成功）**:
```json
{
  "success": true,
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-10-15T16:00:00Z",
  "message": "认证成功"
}
```

**响应（失败）**:
```json
{
  "error": "AUTH_FAILED",
  "message": "零知识证明验证失败",
  "details": "proof verification failed"
}
```

**响应参数说明**:
| 参数 | 类型 | 说明 |
|------|------|------|
| success | boolean | 认证是否成功 |
| session_id | string | 会话ID（UUID格式） |
| token | string | JWT令牌，用于后续API调用 |
| expires_at | datetime | 令牌过期时间（通常1小时） |

**错误码**:
- `INVALID_REQUEST`: 请求参数错误或格式不正确
- `INVALID_CHALLENGE`: challenge_id无效或已过期
- `CHALLENGE_USED`: challenge已被使用（防重放）
- `AUTH_FAILED`: 零知识证明验证失败
- `PROOF_PARSE_ERROR`: proof数据解析失败（格式错误）
- `TOO_MANY_ATTEMPTS`: 认证尝试次数过多（防暴力破解）

**示例（使用curl）**:
```bash
# 注意：实际proof需要使用gnark生成，这里是示例格式
curl -X POST http://localhost:8001/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "CO2_SENSOR_20251015_140552",
    "challenge_id": "d612a4fc-41d6-4d63-8add-ffb83a6a118c",
    "proof": {
      "proof": "SGVsbG8gV29ybGQh...base64编码的proof",
      "public_witness": {
        "device_id": "CO2_SENSOR_20251015_140552",
        "challenge": "a1b2c3d4e5f6789...",
        "commitment": "3f2a1b9c8d7e6f...",
        "response": "9c8b7a6d5e4f3a..."
      }
    }
  }'
```

**安全特性**:
- ✅ **零知识性**: secret永不传输，服务端无法获知
- ✅ **抗重放**: 每个challenge只能使用一次
- ✅ **不可伪造**: 没有secret无法生成有效proof
- ✅ **隐私保护**: 多次认证无法关联追踪
- ✅ **快速验证**: 服务端验证时间<20ms

#### 3. 刷新会话
```http
POST /api/v1/auth/refresh
```

**功能**: 刷新JWT令牌延长会话有效期  
**认证**: 需要JWT令牌  

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**响应**:
```json
{
  "success": true,
  "session_id": "session-uuid",
  "token": "new-jwt-token",
  "expires_at": "2025-10-15T17:00:00Z",
  "message": "会话刷新成功"
}
```

---

## 📱 设备管理接口 (`/api/v1/devices`)

**认证要求**: 无需认证（专为Web管理界面设计）

#### 1. 获取设备列表
```http
GET /api/v1/devices
```

**功能**: 获取设备列表，支持分页和筛选  

**查询参数**:
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20)
- `status`: 设备状态筛选 (online/offline/disabled/fault)
- `sensor_type`: 传感器类型筛选 (**固定枚举值，见下方说明**)

**sensor_type 枚举值** (系统支持的7种传感器类型):
| 值 | 说明 | 单位 |
|----|------|------|
| `co2` | 二氧化碳传感器 | ppm |
| `co` | 一氧化碳传感器 | ppm |
| `smoke` | 烟雾传感器 | ppm |
| `liquid_level` | 液位传感器 | mm |
| `conductivity` | 电导率传感器 | mS/cm |
| `temperature` | 温度传感器 | °C |
| `flow` | 流速传感器 | L/min |

> **重要说明**: `sensor_type` 字段为**固定枚举值**，仅支持上述7种类型。在注册设备和数据采集时必须使用这些确切的字符串值，系统会进行严格校验。

**响应**:
```json
{
  "devices": [
    {
      "device_id": "CO2_SENSOR_20251015_140552",
      "device_type": "sensor",
      "sensor_type": "co2",
      "cabinet_id": "CABINET_A1",
      "public_key": "hex-string",
      "commitment": "hex-string",
      "status": "offline",
      "model": "CO2-SENSOR-PRO-V2",
      "manufacturer": "EdgeTech Solutions",
      "firmware_ver": "2.1.0",
      "created_at": "2025-10-15T14:05:52Z",
      "updated_at": "2025-10-15T14:05:52Z",
      "last_seen_at": null
    }
  ],
  "total": 4,
  "page": 1,
  "limit": 20
}
```

#### 2. 获取设备详情
```http
GET /api/v1/devices/{device_id}
```

**功能**: 获取指定设备的详细信息  

**路径参数**:
- `device_id`: 设备ID

**响应**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "device_type": "sensor",
  "sensor_type": "co2",
  "cabinet_id": "CABINET_A1",
  "public_key": "hex-string",
  "commitment": "hex-string",
  "status": "offline",
  "model": "CO2-SENSOR-PRO-V2",
  "manufacturer": "EdgeTech Solutions",
  "firmware_ver": "2.1.0",
  "created_at": "2025-10-15T14:05:52Z",
  "updated_at": "2025-10-15T14:05:52Z",
  "last_seen_at": null
}
```

#### 3. 注册设备
```http
POST /api/v1/devices
```

**功能**: 注册新设备到系统

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "device_type": "sensor",
  "sensor_type": "co2",
  "cabinet_id": "CABINET_A1",
  "public_key": "hex-string",
  "commitment": "hex-string",
  "model": "CO2-SENSOR-PRO-V2",
  "manufacturer": "EdgeTech Solutions",
  "firmware_ver": "2.1.0"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 是 | 设备唯一标识 |
| device_type | string | 是 | 设备类型 (通常为 "sensor") |
| sensor_type | string | 是 | **传感器类型（固定枚举值）**，必须为以下7种之一：<br>`co2`, `co`, `smoke`, `liquid_level`, `conductivity`, `temperature`, `flow` |
| cabinet_id | string | 是 | 所属储能柜ID |
| public_key | string | 是 | ZKP公钥 (十六进制字符串) |
| commitment | string | 是 | ZKP承诺值 (十六进制字符串) |
| model | string | 否 | 设备型号 |
| manufacturer | string | 否 | 制造商 |
| firmware_ver | string | 否 | 固件版本 |

**错误码**:
- `INVALID_REQUEST`: 请求参数错误
- `UNSUPPORTED_SENSOR_TYPE`: 不支持的传感器类型 (sensor_type 不在枚举值范围内)
- `DEVICE_ALREADY_EXISTS`: 设备ID已存在
- `REGISTER_FAILED`: 注册失败

**响应**: 返回创建的设备信息 (状态码: 201)

#### 4. 更新设备信息
```http
PUT /api/v1/devices/{device_id}
```

**功能**: 更新设备信息  

**路径参数**:
- `device_id`: 设备ID

**请求体**:
```json
{
  "status": "online",
  "model": "CO2-SENSOR-PRO-V3",
  "firmware_ver": "2.2.0"
}
```

**响应**: 返回更新后的设备信息

#### 5. 注销设备
```http
DELETE /api/v1/devices/{device_id}
```

**功能**: 从系统中注销设备  

**路径参数**:
- `device_id`: 设备ID

**响应**:
```json
{
  "message": "设备注销成功"
}
```

#### 6. 设备心跳
```http
POST /api/v1/devices/{device_id}/heartbeat
```

**功能**: 设备发送心跳保持在线状态  

**路径参数**:
- `device_id`: 设备ID

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "timestamp": "2025-10-15T14:00:00Z",
  "status": "online",
  "metadata": {
    "cpu_usage": 15.5,
    "memory_usage": 45.2
  }
}
```

**响应**:
```json
{
  "message": "心跳处理成功"
}
```

#### 7. 设备统计信息
```http
GET /api/v1/devices/statistics
```

**功能**: 获取设备统计信息  

**响应**:
```json
{
  "total_devices": 4,
  "online_devices": 0,
  "offline_devices": 4,
  "sensor_types": {
    "co2": 4,
    "temperature": 0,
    "smoke": 0
  },
  "cabinet_distribution": {
    "CABINET_A1": 4
  }
}
```

---

## 🏢 储能柜管理接口 (`/api/v1/cabinets`)

**认证要求**: 无需认证（专为云端同步设计）

#### 1. 获取储能柜列表
```http
GET /api/v1/cabinets
```

**功能**: 获取所有储能柜及其统计信息  

**响应**:
```json
{
  "cabinets": [
    {
      "cabinet_id": "CABINET_A1",
      "device_count": 4,
      "online_count": 0,
      "offline_count": 4,
      "sensor_types": {
        "co2": 4
      }
    }
  ],
  "total": 1
}
```

#### 2. 按储能柜获取设备列表
```http
GET /api/v1/cabinets/{cabinet_id}/devices
```

**功能**: 获取指定储能柜下的所有设备  

**路径参数**:
- `cabinet_id`: 储能柜ID

**响应**:
```json
{
  "cabinet_id": "CABINET_A1",
  "devices": [
    {
      "device_id": "CO2_SENSOR_20251015_140552",
      "device_type": "sensor",
      "sensor_type": "co2",
      "cabinet_id": "CABINET_A1",
      "status": "offline",
      "created_at": "2025-10-15T14:05:52Z"
    }
  ],
  "total": 4
}
```

---

## 📊 数据采集接口 (`/api/v1/data`)

**认证要求**: 需要JWT令牌（设备零知识认证后获得）

#### 1. 数据采集
```http
POST /api/v1/data/collect
```

**功能**: 设备上传传感器数据

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "sensor_type": "co2",
  "value": 420.5,
  "unit": "ppm",
  "timestamp": "2025-10-15T14:00:00Z",
  "quality": 100
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 是 | 设备ID，必须与JWT token中的设备ID一致 |
| sensor_type | string | 是 | **传感器类型（固定枚举值）**，必须为以下7种之一：<br>`co2`, `co`, `smoke`, `liquid_level`, `conductivity`, `temperature`, `flow` |
| value | number | 是 | 传感器数值 |
| unit | string | 是 | 测量单位，应与sensor_type对应（见下表） |
| timestamp | datetime | 否 | 采集时间（ISO8601格式），默认使用服务器时间 |
| quality | integer | 否 | 数据质量（0-100），默认100 |

**sensor_type 与 unit 对应关系**:
| sensor_type | 推荐unit | 数值范围说明 |
|-------------|----------|--------------|
| co2 | ppm | 0-10000 (正常大气<1000) |
| co | ppm | 0-1000 (安全<50) |
| smoke | ppm | 0-10000 |
| liquid_level | mm | 0-2000 |
| conductivity | mS/cm | 0-20 |
| temperature | °C | -40至85 |
| flow | L/min | 0-200 |

**响应**:
```json
{
  "message": "数据采集成功"
}
```

**错误码**:
- `AUTH_001`: 缺少认证令牌
- `AUTH_002`: 认证令牌无效或已过期
- `INVALID_REQUEST`: 请求参数错误
- `INVALID_DATA`: 传感器数据验证失败 (sensor_type不在枚举范围或数值超限)
- `COLLECT_FAILED`: 数据采集失败

#### 2. 查询历史数据
```http
GET /api/v1/data/query
```

**功能**: 查询历史传感器数据

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**查询参数**:
- `device_id`: 设备ID (可选)
- `sensor_type`: 传感器类型 (可选，**固定枚举值**: `co2`/`co`/`smoke`/`liquid_level`/`conductivity`/`temperature`/`flow`)
- `start_time`: 开始时间 (可选，ISO8601格式)
- `end_time`: 结束时间 (可选，ISO8601格式)
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 100)

**响应**:
```json
{
  "data": [
    {
      "id": 1,
      "device_id": "CO2_SENSOR_20251015_140552",
      "sensor_type": "co2",
      "value": 420.5,
      "unit": "ppm",
      "timestamp": "2025-10-15T14:00:00Z",
      "quality": 100,
      "synced": false,
      "synced_at": null
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 100
}
```

#### 3. 获取数据统计
```http
GET /api/v1/data/statistics
```

**功能**: 获取数据统计信息

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**查询参数**:
- `device_id`: 设备ID (可选)
- `sensor_type`: 传感器类型 (可选，**固定枚举值**: `co2`/`co`/`smoke`/`liquid_level`/`conductivity`/`temperature`/`flow`)
- `period`: 统计周期 (1h/24h/7d/30d, 默认: 24h)

**响应**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "sensor_type": "co2",
  "count": 100,
  "min_value": 380.0,
  "max_value": 450.0,
  "avg_value": 415.5,
  "start_time": "2025-10-14T14:00:00Z",
  "end_time": "2025-10-15T14:00:00Z"
}
```

---

## 🚨 告警管理接口 (`/api/v1/alerts`)

**认证要求**: 无需认证（专为Web管理界面设计）

#### 1. 获取告警列表
```http
GET /api/v1/alerts
```

**功能**: 获取告警列表

**查询参数**:
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20)
- `severity`: 严重级别筛选 (low/medium/high/critical)
- `resolved`: 是否已解决 (true/false)

**响应**:
```json
{
  "alerts": [
    {
      "id": 1,
      "device_id": "CO2_SENSOR_20251015_140552",
      "alert_type": "threshold_exceeded",
      "severity": "high",
      "message": "CO2浓度超过阈值",
      "value": 1200.0,
      "threshold": 1000.0,
      "timestamp": "2025-10-15T14:00:00Z",
      "resolved": false,
      "resolved_at": null
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

#### 2. 创建告警
```http
POST /api/v1/alerts
```

**功能**: 创建新告警  

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**请求体**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "alert_type": "threshold_exceeded",
  "severity": "high",
  "message": "CO2浓度超过阈值",
  "value": 1200.0,
  "threshold": 1000.0
}
```

**响应**: 返回创建的告警信息 (状态码: 201)

#### 3. 解决告警
```http
PUT /api/v1/alerts/{alert_id}/resolve
```

**功能**: 标记告警为已解决

**请求头**:
```http
Authorization: Bearer <jwt-token>
```

**路径参数**:
- `alert_id`: 告警ID

**响应**:
```json
{
  "message": "告警已解决"
}
```

**示例**:
```bash
# 解决ID为123的告警
curl -X PUT http://localhost:8001/api/v1/alerts/123/resolve \
  -H "Authorization: Bearer <jwt-token>"
```

#### 4. 获取告警配置(阈值)
```http
GET /api/v1/alerts/config
```

**功能**: 获取系统告警配置,包括所有传感器类型的阈值设置

**认证**: 无需认证

**响应**:
```json
{
  "enabled": true,
  "thresholds": {
    "co2": {
      "min": 0,
      "max": 5000
    },
    "co": {
      "min": 0,
      "max": 50
    },
    "smoke": {
      "min": 0,
      "max": 1000
    },
    "liquid_level": {
      "min": 100,
      "max": 900
    },
    "conductivity": {
      "min": 0,
      "max": 10
    },
    "temperature": {
      "min": -10,
      "max": 60
    },
    "flow": {
      "min": 0.5,
      "max": 100
    }
  }
}
```

**字段说明**:
- `enabled`: 告警功能是否启用
- `thresholds`: 各类传感器的阈值配置
  - `min`: 最小阈值(0表示只有最大值限制)
  - `max`: 最大阈值

**使用场景**:
1. 前端页面动态显示阈值信息
2. 实时监控页面展示告警边界
3. 统计图表标注阈值线
4. 配置管理界面回显当前设置

**前端集成示例**:
```javascript
// 加载阈值配置
const config = await API.getAlertConfig();

// 显示CO2阈值
if (config.enabled && config.thresholds.co2) {
  const threshold = config.thresholds.co2;
  document.getElementById('co2Threshold').textContent =
    `${threshold.min}-${threshold.max} ppm`;
}

// 检查是否超出阈值
function isExceeded(sensorType, value) {
  const t = config.thresholds[sensorType];
  return (t.max > 0 && value > t.max) ||
         (t.min > 0 && value < t.min);
}
```

**配置文件位置**: `configs/config.yaml`
```yaml
alert:
  enabled: true
  thresholds:
    co2_max: 5000.0
    co_max: 50.0
    smoke_max: 1000.0
    liquid_level_min: 100.0
    liquid_level_max: 900.0
    # ...更多配置
```

### 告警严重级别说明

系统支持4种告警严重级别，根据阈值超出程度自动判定：

| 级别 | 英文标识 | 中文名称 | 判定规则 | 前端徽章颜色 |
|------|---------|---------|---------|-------------|
| critical | critical | 危急 | 超出阈值200%以上 | 深红色 (#7f1d1d) |
| high | high | 严重 | 超出阈值100-200% | 红色 (#ef4444) |
| medium | medium | 中等 | 超出阈值50-100% | 橙色 (#f59e0b) |
| low | low | 轻微 | 超出阈值50%以内 | 蓝色 (#06b6d4) |

**示例计算**:
```javascript
// CO2阈值为1000ppm
// 当前值1500ppm -> 超出50% -> medium级别
// 当前值2000ppm -> 超出100% -> high级别
// 当前值3000ppm -> 超出200% -> critical级别
```

### 告警前端集成示例

```javascript
// 1. 获取告警列表（分页）
const result = await API.getAlerts(
  1,           // page
  20,          // limit
  'high',      // severity (可选: 'critical', 'high', 'medium', 'low')
  'false'      // resolved (可选: 'true', 'false')
);

// 2. 渲染告警表格
result.alerts.forEach(alert => {
  console.log(`[${alert.severity}] ${alert.message}`);
  console.log(`设备: ${alert.device_id}, 值: ${alert.value}, 阈值: ${alert.threshold}`);
});

// 3. 解决告警
await API.resolveAlert(123);

// 4. 获取未解决告警数量
const unresolvedAlerts = await API.getAlerts(1, 1000, '', 'false');
const count = unresolvedAlerts.total;
document.getElementById('unresolvedAlertsCount').textContent = count;
```

**告警自动刷新示例**:
```javascript
// 每30秒自动刷新告警列表
setInterval(async () => {
  if (document.getElementById('alertsPage').classList.contains('active')) {
    await loadAlerts();
    await updateUnresolvedCount();
  }
}, 30000);
```

---

## 📊 统计分析接口 (`/api/v1/data/statistics`)

**认证要求**: 无需认证（专为Web管理界面设计）

**后端实现**: ✅ 已完成
- 文件位置: [internal/collector/service.go:530-572](internal/collector/service.go#L530-L572)
- Handler: [api/handlers.go:533-550](api/handlers.go#L533-L550)
- 路由配置: [cmd/edge/main.go:250](cmd/edge/main.go#L250)

### 获取统计数据

```http
GET /api/v1/data/statistics?device_id={device_id}&sensor_type={sensor_type}&period={period}
```

**功能**: 获取指定时间段内传感器数据的统计信息

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| device_id | string | 否 | 空（所有设备） | 设备ID，为空则统计所有设备 |
| sensor_type | string | 否 | 空（所有类型） | 传感器类型（co2/co/smoke/liquid_level/conductivity/temperature/flow） |
| period | string | 否 | 24h | 统计时间段：1h/24h/7d/30d |

**实现逻辑**:
```sql
SELECT
    COUNT(*) as count,
    COALESCE(MIN(value), 0) as min_value,
    COALESCE(MAX(value), 0) as max_value,
    COALESCE(AVG(value), 0) as avg_value
FROM sensor_data
WHERE timestamp BETWEEN ? AND ?
    AND (device_id = ? OR ? = '')       -- 可选筛选
    AND (sensor_type = ? OR ? = '')     -- 可选筛选
```

**响应**:
```json
{
  "device_id": "CO2_SENSOR_20251015_140552",
  "sensor_type": "co2",
  "count": 1440,
  "min_value": 380.5,
  "max_value": 1250.0,
  "avg_value": 420.8,
  "start_time": "2025-10-19T14:00:00+08:00",
  "end_time": "2025-10-20T14:00:00+08:00"
}
```

**响应参数说明**:
| 参数 | 类型 | 说明 |
|------|------|------|
| count | integer | 数据点数量（无数据时为0） |
| min_value | float | 最小值（无数据时为0） |
| max_value | float | 最大值（无数据时为0） |
| avg_value | float | 平均值（无数据时为0） |
| start_time | datetime | 统计开始时间（ISO 8601格式） |
| end_time | datetime | 统计结束时间（ISO 8601格式） |
| device_id | string | 设备ID（如果指定） |
| sensor_type | string | 传感器类型（如果指定） |

**时间段说明**:
| period | 时间范围 | 使用场景 |
|--------|---------|---------|
| `1h`   | 最近1小时 | 实时监控 |
| `24h`  | 最近24小时（默认） | 日常分析 |
| `7d`   | 最近7天 | 周趋势分析 |
| `30d`  | 最近30天 | 月度报表 |

**使用示例**:

```bash
# 示例1: 获取所有设备最近24小时的统计（默认）
curl http://localhost:8001/api/v1/data/statistics

# 示例2: 获取指定设备最近30天的CO2数据统计
curl "http://localhost:8001/api/v1/data/statistics?device_id=CO2_SENSOR_001&sensor_type=co2&period=30d"

# 示例3: 获取所有温度传感器最近1小时的统计
curl "http://localhost:8001/api/v1/data/statistics?sensor_type=temperature&period=1h"

# 示例4: 获取所有设备所有类型最近7天的统计
curl "http://localhost:8001/api/v1/data/statistics?period=7d"
```

**实际测试结果**:
```bash
$ curl "http://localhost:8001/api/v1/data/statistics?period=30d"
{
  "device_id": "",
  "sensor_type": "",
  "count": 51,
  "min_value": 25.5,
  "max_value": 420.5,
  "avg_value": 133.93,
  "start_time": "2025-09-20T12:00:00+08:00",
  "end_time": "2025-10-20T12:00:00+08:00"
}
```

**前端集成示例**:
```javascript
// 获取统计数据
const stats = await API.getStatistics('CO2_SENSOR_001', 'co2', '24h');

// 显示统计结果
document.getElementById('statsCount').textContent = stats.count;
document.getElementById('statsMin').textContent = stats.min_value.toFixed(2);
document.getElementById('statsMax').textContent = stats.max_value.toFixed(2);
document.getElementById('statsAvg').textContent = stats.avg_value.toFixed(2);
document.getElementById('statsTime').textContent =
    `${formatTime(stats.start_time)} ~ ${formatTime(stats.end_time)}`;
```

**性能指标**:
- 查询响应时间: <100ms (10万条数据内)
- 支持并发查询: ✅
- 索引优化: ✅ (device_id, sensor_type, timestamp)
- 空数据处理: ✅ (使用COALESCE返回0)

---

## 📋 数据模型定义

### 传感器类型 (SensorType)

> **代码定义**: [pkg/models/device.go:11-21](pkg/models/device.go#L11-L21) (type SensorType string)

系统支持**7种固定的传感器类型**，该字段为**字符串枚举**，不可扩展：

```go
const (
    SensorCO2         = "co2"          // 二氧化碳传感器
    SensorCO          = "co"           // 一氧化碳传感器
    SensorSmoke       = "smoke"        // 烟雾传感器
    SensorLiquidLevel = "liquid_level" // 液位传感器
    SensorConductivity = "conductivity" // 电导率传感器
    SensorTemperature = "temperature"  // 温度传感器
    SensorFlow        = "flow"         // 流速传感器
)
```

**重要说明**:
- ✅ 该枚举值是**系统级硬编码**，数据库存储时进行严格校验
- ✅ 所有API接口（设备注册、数据采集、查询）都会验证 `sensor_type` 是否在此枚举范围内
- ❌ 不支持自定义传感器类型，提交非法值会返回 `UNSUPPORTED_SENSOR_TYPE` 错误
- 🔧 **代码验证位置**:
  - 类型定义: [pkg/models/device.go:11-21](pkg/models/device.go#L11-L21)
  - 数据验证: [api/handlers.go:659-669](api/handlers.go#L659-L669)
  - 设备管理: [internal/device/manager.go:597-598](internal/device/manager.go#L597-L598)

### 设备状态
```go
const (
    DeviceStatusOnline   = "online"   // 在线
    DeviceStatusOffline  = "offline"  // 离线
    DeviceStatusDisabled = "disabled" // 禁用
    DeviceStatusFault    = "fault"    // 故障
)
```

### 告警严重级别
```go
const (
    SeverityLow      = "low"      // 低
    SeverityMedium   = "medium"   // 中
    SeverityHigh     = "high"     // 高
    SeverityCritical = "critical" // 严重
)
```

### 传感器单位映射
```go
var SensorUnit = map[SensorType]string{
    SensorCO2:         "ppm",     // 二氧化碳浓度
    SensorCO:          "ppm",     // 一氧化碳浓度
    SensorSmoke:       "ppm",     // 烟雾浓度
    SensorLiquidLevel: "mm",      // 液位高度
    SensorConductivity: "mS/cm",  // 电导率
    SensorTemperature: "°C",      // 温度
    SensorFlow:        "L/min",   // 流速
}
```

---

## 🔒 认证与授权

### JWT令牌
- **获取方式**: 通过零知识证明认证获得
- **使用方式**: 在请求头中添加 `Authorization: Bearer <token>`
- **有效期**: 默认1小时
- **刷新**: 通过 `/api/v1/auth/refresh` 接口刷新

### 接口认证级别

#### 🟢 无需认证
- 系统健康检查 (`/health`, `/ready`)
- 认证相关接口 (`/api/v1/auth/*`)
- 设备管理接口 (`/api/v1/devices/*`) - 用于Web管理界面
- 储能柜管理接口 (`/api/v1/cabinets/*`) - 用于云端同步

#### 🔒 需要认证
- 数据采集接口 (`/api/v1/data/*`) - 需要设备JWT令牌
- 告警管理接口 (`/api/v1/alerts/*`) - 需要设备JWT令牌

---

## 🚀 使用示例

### 设备认证流程
```bash
# 1. 获取认证挑战
curl -X POST http://localhost:8001/api/v1/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"device_id": "CO2_SENSOR_20251015_140552"}'

# 2. 提交零知识证明
curl -X POST http://localhost:8001/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "CO2_SENSOR_20251015_140552",
    "challenge_id": "uuid-from-step1",
    "proof": {
      "proof": [1,2,3,...],
      "public_witness": ["device_id", "challenge", "commitment", "response"]
    }
  }'

# 3. 使用JWT令牌上传数据
curl -X POST http://localhost:8001/api/v1/data/collect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt-token>" \
  -d '{
    "device_id": "CO2_SENSOR_20251015_140552",
    "sensor_type": "co2",
    "value": 420.5,
    "unit": "ppm"
  }'
```

### 云端同步示例
```bash
# 获取储能柜列表
curl http://localhost:8001/api/v1/cabinets

# 获取指定储能柜的所有设备
curl http://localhost:8001/api/v1/cabinets/CABINET_A1/devices
```

---

## 📖 日志记录接口 (`/api/v1/logs`)

**认证要求**: 无需认证（专为Web管理界面设计）

**后端实现**: ✅ 已完成
- Handler: [api/handlers.go:762-1030](api/handlers.go#L762-L1030)
- 路由配置: [cmd/edge/main.go:291-295](cmd/edge/main.go#L291-L295)

### 1. 获取告警日志

```http
GET /api/v1/logs/alerts
```

**功能**: 获取系统告警日志，支持多条件筛选

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| start_date | string | 否 | 空 | 开始日期（YYYY-MM-DD格式） |
| end_date | string | 否 | 空 | 结束日期（YYYY-MM-DD格式） |
| severity | string | 否 | 空 | 严重级别（critical/high/medium/low） |
| resolved | string | 否 | 空 | 是否已解决（true/false） |
| device_id | string | 否 | 空 | 设备ID（支持模糊查询） |
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 20 | 每页数量（1-100） |

**响应**:
```json
{
  "logs": [
    {
      "id": 1,
      "device_id": "CO2_SENSOR_111111111",
      "alert_type": "threshold_exceeded",
      "severity": "high",
      "message": "CO2浓度超过阈值",
      "value": 1200.5,
      "threshold": 1000.0,
      "timestamp": "2025-10-21T14:30:25Z",
      "resolved": false,
      "resolved_at": null
    }
  ],
  "total": 15,
  "page": 1,
  "limit": 20
}
```

**实现逻辑**:
```sql
-- 按时间范围和筛选条件查询告警日志
SELECT id, device_id, alert_type, severity, message, value, threshold,
       timestamp, resolved, resolved_at
FROM alerts
WHERE timestamp >= ? AND timestamp <= ?
  AND (severity = ? OR ? = '')
  AND (resolved = ? OR ? = '')
  AND (device_id LIKE ? OR ? = '')
ORDER BY timestamp DESC
LIMIT ? OFFSET ?
```

**示例**:
```bash
# 查询最近7天的严重级别告警
curl "http://localhost:8001/api/v1/logs/alerts?start_date=2025-10-14&end_date=2025-10-21&severity=high&page=1&limit=20"

# 查询未解决的告警
curl "http://localhost:8001/api/v1/logs/alerts?resolved=false"

# 查询特定设备的告警
curl "http://localhost:8001/api/v1/logs/alerts?device_id=CO2_SENSOR"
```

### 2. 获取认证日志

```http
GET /api/v1/logs/auth
```

**功能**: 获取设备认证日志，包括挑战请求和认证成功记录

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| start_date | string | 否 | 空 | 开始日期（YYYY-MM-DD格式） |
| end_date | string | 否 | 空 | 结束日期（YYYY-MM-DD格式） |
| status | string | 否 | 空 | 认证状态（success/pending） |
| device_id | string | 否 | 空 | 设备ID（支持模糊查询） |
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 20 | 每页数量（1-100） |

**响应**:
```json
{
  "logs": [
    {
      "id": "5475469a-4aeb-4d63-a4c0-3dc2e5be15ec",
      "device_id": "TH_SENSOR_20251016_637482",
      "action": "challenge_used",
      "status": "success",
      "timestamp": "2025-10-21T16:15:54.319675091+08:00",
      "session_id": null,
      "details": "挑战已使用（认证成功）"
    },
    {
      "id": "a16a2b2b-678d-456d-b1bc-2d7a5430e958",
      "device_id": "TH_SENSOR_20251016_637482",
      "action": "challenge_requested",
      "status": "pending",
      "timestamp": "2025-10-21T15:55:57.026581863+08:00",
      "session_id": null,
      "details": "生成认证挑战"
    }
  ],
  "total": 255,
  "page": 1,
  "limit": 10
}
```

**认证动作类型**:
| action | 中文说明 | 数据来源 | 状态 |
|--------|---------|---------|------|
| challenge_requested | 请求认证挑战 | challenges表 | pending |
| challenge_used | 认证成功 | challenges表（used=true） | success |
| session_created | 会话建立 | sessions表 | success |

**实现逻辑**:
```sql
-- 1. 查询挑战记录
SELECT challenge_id, device_id, created_at, used
FROM challenges
WHERE created_at >= ? AND created_at <= ?
  AND (device_id LIKE ? OR ? = '')
ORDER BY created_at DESC

-- 2. 查询会话记录
SELECT session_id, device_id, created_at, ip_address
FROM sessions
WHERE created_at >= ? AND created_at <= ?
  AND (device_id LIKE ? OR ? = '')
ORDER BY created_at DESC
```

**示例**:
```bash
# 查询最近7天的认证日志
curl "http://localhost:8001/api/v1/logs/auth?start_date=2025-10-14&end_date=2025-10-21&page=1&limit=10"

# 查询认证成功的记录
curl "http://localhost:8001/api/v1/logs/auth?status=success"

# 查询特定设备的认证历史
curl "http://localhost:8001/api/v1/logs/auth?device_id=TH_SENSOR"
```

### 前端集成示例

```javascript
// 日志管理模块（web/js/logs.js）

const Logs = {
  activeTab: 'alert-logs',

  // 加载告警日志
  async loadAlertLogs() {
    const filters = {
      startDate: '2025-10-14',
      endDate: '2025-10-21',
      severity: 'high',    // 可选
      resolved: 'false',   // 可选
      deviceID: '',        // 可选
      page: 1,
      limit: 20
    };

    const result = await API.getAlertLogs(filters);
    console.log(`加载了 ${result.logs.length} 条告警日志`);
  },

  // 加载认证日志
  async loadAuthLogs() {
    const filters = {
      startDate: '2025-10-14',
      endDate: '2025-10-21',
      status: 'success',   // 可选
      deviceID: '',        // 可选
      page: 1,
      limit: 20
    };

    const result = await API.getAuthLogs(filters);
    console.log(`加载了 ${result.logs.length} 条认证日志`);
  }
};
```

**Tab切换示例**:
```javascript
// 切换告警日志和认证日志标签页
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const tab = btn.dataset.tab;
    Logs.switchTab(tab);  // 切换到对应标签页并加载数据
  });
});
```

**默认日期范围设置**:
```javascript
// 自动设置为最近7天
function setDefaultDates() {
  const today = new Date();
  const weekAgo = new Date(today);
  weekAgo.setDate(today.getDate() - 7);

  document.getElementById('alertStartDate').value =
    weekAgo.toISOString().split('T')[0];
  document.getElementById('alertEndDate').value =
    today.toISOString().split('T')[0];
}
```

### 3. 批量删除告警日志

```http
DELETE /api/v1/logs/alerts/batch
```

**功能**: 批量删除指定的告警日志记录

**请求体**:
```json
{
  "ids": [1, 2, 3, 5, 8]
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | array | 是 | 要删除的告警日志ID数组 |

**响应**:
```json
{
  "message": "成功删除 5 条告警日志"
}
```

**错误码**:
- `INVALID_REQUEST`: 请求参数错误
- `DELETE_FAILED`: 删除失败

**示例**:
```bash
curl -X DELETE http://localhost:8001/api/v1/logs/alerts/batch \
  -H "Content-Type: application/json" \
  -d '{"ids": [1, 2, 3, 5, 8]}'
```

### 4. 批量删除认证日志

```http
DELETE /api/v1/logs/auth/batch
```

**功能**: 批量删除指定的认证日志记录

**请求体**:
```json
{
  "ids": ["5475469a-4aeb-4d63-a4c0-3dc2e5be15ec", "a16a2b2b-678d-456d-b1bc-2d7a5430e958"]
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | array | 是 | 要删除的认证日志ID数组(UUID格式) |

**响应**:
```json
{
  "message": "成功删除 2 条认证日志"
}
```

**示例**:
```bash
curl -X DELETE http://localhost:8001/api/v1/logs/auth/batch \
  -H "Content-Type: application/json" \
  -d '{"ids": ["5475469a-4aeb-4d63-a4c0-3dc2e5be15ec"]}'
```

### 5. 清空所有认证日志

```http
DELETE /api/v1/logs/auth/clear
```

**功能**: 清空所有认证日志记录(包括challenges和sessions表)

**认证**: 无需认证

**重要说明**:
- ⚠️ **危险操作**: 此操作会删除所有认证历史记录，无法恢复
- 🗑️ **清空范围**: 同时清空 `challenges` 表和 `sessions` 表
- 📊 **用途**: 适用于开发环境清理测试数据或生产环境定期归档后清理

**响应**:
```json
{
  "message": "成功清空 challenges: 127 条, sessions: 53 条"
}
```

**示例**:
```bash
curl -X DELETE http://localhost:8001/api/v1/logs/auth/clear
```

**前端确认弹窗示例**:
```javascript
// 清空认证日志前需要二次确认
function clearAllAuthLogs() {
  if (confirm('⚠️ 确定要清空所有认证日志吗？此操作无法撤销！')) {
    if (confirm('⚠️ 最后确认：这将永久删除所有认证历史记录！')) {
      API.clearAllAuthLogs().then(result => {
        alert(result.message);
        loadAuthLogs(); // 刷新页面
      });
    }
  }
}
```

---

## 🔑 许可证管理接口 (`/api/v1/license`)

**认证要求**: 无需认证（专为Web管理界面设计）

**后端实现**: ✅ 已完成
- Handler: [api/handlers.go:1296-1305](api/handlers.go#L1296-L1305)
- Service: [internal/license/service.go:201-223](internal/license/service.go#L201-L223)
- 路由配置: [cmd/edge/main.go:352](cmd/edge/main.go#L352)

### 获取许可证信息

```http
GET /api/v1/license/info
```

**功能**: 获取当前系统的许可证状态信息

**查询参数**: 无

**响应（许可证未启用）**:
```json
{
  "enabled": false
}
```

**响应（许可证已启用 - 正常状态）**:
```json
{
  "enabled": true,
  "license_id": "LIC-2025-001",
  "mac_address": "00:15:5d:41:5b:ca",
  "max_devices": 100,
  "expires_at": "2026-01-01T00:00:00Z",
  "is_expired": false,
  "in_grace_period": false
}
```

**响应（许可证已过期 - 宽限期内）**:
```json
{
  "enabled": true,
  "license_id": "LIC-2025-001",
  "mac_address": "00:15:5d:41:5b:ca",
  "max_devices": 100,
  "expires_at": "2025-01-01T00:00:00Z",
  "is_expired": true,
  "in_grace_period": true
}
```

**响应（许可证已过期 - 超过宽限期）**:
```json
{
  "enabled": true,
  "license_id": "LIC-2025-001",
  "mac_address": "00:15:5d:41:5b:ca",
  "max_devices": 100,
  "expires_at": "2025-01-01T00:00:00Z",
  "is_expired": true,
  "in_grace_period": false
}
```

**响应参数说明**:
| 参数 | 类型 | 说明 |
|------|------|------|
| enabled | boolean | 许可证验证是否启用 |
| license_id | string | 许可证ID（仅在enabled=true时返回） |
| mac_address | string | 许可证绑定的MAC地址 |
| max_devices | integer | 许可证允许的最大设备数 |
| expires_at | datetime | 许可证过期时间（ISO 8601格式） |
| is_expired | boolean | 许可证是否已过期 |
| in_grace_period | boolean | 是否在宽限期内（默认72小时） |

**使用示例**:

```bash
# 查询许可证状态
curl http://localhost:8001/api/v1/license/info
```

**前端集成示例**:

```javascript
// 加载许可证信息
const licenseInfo = await API.getLicenseInfo();

if (!licenseInfo.enabled) {
    console.log('许可证验证未启用（开发模式）');
    return;
}

// 检查许可证状态
if (licenseInfo.is_expired && !licenseInfo.in_grace_period) {
    alert(`⚠️ 许可证已过期！过期时间: ${licenseInfo.expires_at}`);
} else if (licenseInfo.in_grace_period) {
    alert(`⚠️ 许可证已过期但在宽限期内，请尽快续期！`);
} else {
    const expiresAt = new Date(licenseInfo.expires_at);
    const daysRemaining = Math.ceil((expiresAt - new Date()) / (1000 * 60 * 60 * 24));
    console.log(`许可证有效，剩余 ${daysRemaining} 天`);
}
```

**Web界面展示**:

系统会在侧边栏底部自动显示许可证状态：

- ✅ **正常状态**（绿色图标）: 许可证有效，剩余XX天
- ⚠️ **警告状态**（黄色图标）: 许可证即将过期（少于30天）或在宽限期内
- ❌ **过期状态**（红色图标）: 许可证已过期且超过宽限期

**注意事项**:

1. 许可证验证在认证入口（`/api/v1/auth/challenge`）执行
2. 许可证校验失败时返回HTTP 403状态码
3. 设备数量限制在设备注册时检查
4. MAC地址绑定防止许可证在多台设备间共享
5. 宽限期默认72小时，可在配置文件中修改

---

## 📝 错误处理

### 通用错误格式
```json
{
  "error": "ERROR_CODE",
  "message": "错误描述信息"
}
```

### 常见错误码

#### 通用错误码
- `INVALID_REQUEST`: 请求参数错误 (400)
- `DEVICE_NOT_FOUND`: 设备不存在 (404)
- `QUERY_FAILED`: 查询失败 (500)
- `COLLECT_FAILED`: 数据采集失败 (500)

#### 认证错误码
- `AUTH_001`: 缺少认证令牌 (401)
- `AUTH_002`: 认证令牌无效或已过期 (401)
- `AUTH_FAILED`: 零知识证明验证失败 (401)

#### 许可证错误码（SPA单包授权）
- `LICENSE_001`: 许可证校验失败 (403)
  - 许可证文件不存在或无法读取
  - 许可证签名验证失败
  - 许可证MAC地址不匹配
  - 许可证已过期且超过宽限期（默认72小时）
  - 设备数量超过许可证限制

**说明**: 许可证错误发生在认证入口（`/api/v1/auth/challenge`），网关设备会收到拒绝响应但无法感知是许可证问题（对客户端透明）。

---

## 📊 性能指标

- **认证响应时间**: < 500ms
- **数据采集响应时间**: < 200ms (HTTP) / < 50ms (MQTT)
- **并发设备支持**: ≥ 100台
- **API限流**: 500次/分钟 (全局)
- **MQTT吞吐量**: > 10,000 msg/s
- **WebSocket并发**: 支持多个客户端同时连接
- **数据库**: SQLite3 (支持高并发读写)

---

## ⚙️ 系统配置

### 配置文件结构

系统配置文件位于 `configs/config.yaml`,包含以下模块:

#### 1. HTTP服务器配置

```yaml
server:
  host: "0.0.0.0"
  port: 8001
  mode: "release"  # debug, release, test
```

#### 2. MQTT配置 (新增)

```yaml
mqtt:
  enabled: true                              # 是否启用MQTT
  broker_address: "tcp://127.0.0.1:1883"    # Broker地址
  client_id: "edge-server-subscriber"        # 客户端ID
  username: "edge-server"                    # 用户名(管理员)
  password: "edge-server-password"           # 密码
  qos: 1                                     # QoS等级
  keep_alive: 60                             # 心跳间隔(秒)
  clean_session: true                        # 清除会话
  reconnect_interval: 5s                     # 重连间隔
  max_reconnect_attempts: 10                 # 最大重连次数
```

**MQTT订阅Topic**:
- `sensors/#` - 所有传感器数据
- `devices/+/status` - 所有设备状态
- `alerts/#` - 所有告警
- `devices/+/heartbeat` - 所有心跳

**数据流向**:
```
网关发布MQTT消息 → MQTT Broker → Edge订阅器 → 数据库 → WebSocket推送到前端
```

#### 3. 零知识认证配置

```yaml
auth:
  challenge_ttl: 60s          # 挑战有效期
  session_ttl: 86400s         # 会话有效期(24小时)
  max_retry: 3                # 最大重试次数
  zkp:
    circuit_path: "./internal/zkp/keys"
    proving_scheme: "groth16"
    verifying_key_path: "./auth_verifying.key"
```

#### 4. 设备管理配置

```yaml
device:
  heartbeat_interval: 30s     # 心跳间隔
  offline_timeout: 300s       # 离线超时(5分钟)
  max_devices: 100            # 最大设备数
  supported_sensors:          # 支持的传感器类型(7种)
    - co2
    - co
    - smoke
    - liquid_level
    - conductivity
    - temperature
    - flow
```

#### 5. 数据采集配置

```yaml
data:
  collect_interval: 60s       # 采集间隔
  sync_interval: 300s         # 云端同步间隔
  retention_days: 90          # 本地数据保留天数
  batch_size: 100            # 批量大小
  buffer_size: 10000         # 缓冲区大小
```

#### 6. 告警配置

```yaml
alert:
  enabled: true
  thresholds:
    co2_max: 5000.0           # CO2浓度上限 ppm
    co_max: 50.0              # CO浓度上限 ppm
    smoke_max: 1000.0         # 烟雾浓度上限 ppm
    liquid_level_min: 100.0   # 液位下限 mm
    liquid_level_max: 900.0   # 液位上限 mm
    conductivity_min: 0.5     # 电导率下限 mS/cm
    conductivity_max: 10.0    # 电导率上限 mS/cm
    temperature_min: -10.0    # 温度下限 °C
    temperature_max: 60.0     # 温度上限 °C
    flow_min: 0.5            # 流速下限 L/min
    flow_max: 100.0          # 流速上限 L/min
```

#### 7. 许可证配置 (SPA单包授权)

```yaml
license:
  enabled: false                            # 是否启用许可证验证
  path: "./configs/license.lic"             # 许可证文件路径
  pubkey_path: "./configs/vendor_pubkey.pem" # 厂商公钥路径
  grace_period: 72h                         # 过期宽限期(默认72小时)
```

#### 8. 云端同步配置

```yaml
cloud:
  enabled: false
  endpoint: "https://cloud.example.com/api/v1"
  api_key: "your_api_key"
  timeout: 30s
  retry_count: 3
  retry_interval: 5s
```

### 配置说明

**MQTT启用与否的影响**:
- `mqtt.enabled: true` - 系统同时支持HTTP和MQTT双通道数据接收
- `mqtt.enabled: false` - 系统仅支持HTTP API数据接收

**数据接收优先级**:
- MQTT和HTTP数据存储到相同的数据库表
- Web管理界面查询时不区分数据来源
- WebSocket实时推送优先推送MQTT数据(低延迟)

---

## 🔄 版本信息

**当前版本**: v1.0.0
**API版本**: v1
**更新日期**: 2025-10-28

### 版本历史
- **v1.0.0** (2025-10-28):
  - ✅ 完整的设备认证、数据采集、告警管理功能
  - ✅ MQTT双通道数据接收
  - ✅ WebSocket实时推送
  - ✅ 日志批量管理功能

### 技术栈

**后端**:
- Go 1.24+
- Gin HTTP框架
- Gnark ZKP (Groth16)
- Eclipse Paho MQTT客户端
- Gorilla WebSocket
- SQLite3 (CGO)

**前端**:
- 原生JavaScript (无框架)
- Chart.js (图表)
- WebSocket API

**协议**:
- HTTPS REST API
- MQTT v3.1.1 (QoS 1)
- WebSocket

**认证**:
- 零知识证明 (ZKP)
- JWT Token (HS256)
- RSA-2048 (许可证签名)
