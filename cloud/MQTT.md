# BMS MQTT 数据获取接口（Cloud/Edge 版）

> 本文全部依据 `mqtt.md` 的架构与数据，描述 Edge → Cloud 传感器数据上行、Cloud → Edge 指令下行的双 Broker（TLS-only）链路。

## 文档信息
- 版本：v2.0（与 mqtt.md 同步）
- 更新日期：2025-11-28
- 适用：Edge ↔ Cloud 传感器数据与控制命令

## 架构与端口

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              储能柜现场（本地网络）                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌───────────────────┐         发布数据           ┌───────────────────────┐  │
│  │ orangepi-gateway  │ ─────────────────────────► │  Edge端 Mosquitto     │  │
│  │                   │  ssl://127.0.0.1:8883      │  Broker #1 (TLS 8883) │  │
│  │ • 读取传感器数据   │                            │                       │  │
│  │ • ZKP获取Token     │  Topics:                  │  • 接收本地传感器数据   │  │
│  │ • MQTT发布数据     │  - sensors/#              │  • 存储到本地SQLite    │  │
│  │                   │  - devices/+/status       │  • 桥接转发到Cloud    │  │
│  └───────────────────┘  - alerts/#               └──────────┬────────────┘  │
│                          - devices/+/heartbeat              │               │
│                                                              │               │
└──────────────────────────────────────────────────────────────┼───────────────┘
                                                               │
                                                               │ MQTT 桥接 (TLS)
                                                               │ 地址: ssl://<CLOUD_IP>:8884
                                                               │
                        ┌──────────────────────────────────────┴───────────────────┐
                        │                                                          │
                        │              上行 (Edge → Cloud)                         │
                        │  • sensors/#                                             │
                        │  • devices/+/status                                      │
                        │  • devices/+/heartbeat                                   │
                        │  • alerts/#                                              │
                        │                                                          │
                        │              下行 (Cloud → Edge)                         │
                        │  • commands/#                                            │
                        │  • control/#                                             │
                        │  • config/#                                              │
                        │  • cloud/cabinet/+/policy/#                              │
                        ▼                                                          │
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              云端服务器 (Cloud Server)                            │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌───────────────────────┐                       ┌───────────────────────────┐  │
│  │  Cloud端 Mosquitto    │ ◄───────────────────► │     Cloud 后端服务         │  │
│  │  Broker #2 (TLS 8884) │   订阅桥接过来的数据    │     (端口8003)             │  │
│  │                       │                       │  • 存储到PostgreSQL       │  │
│  │  • 接收Edge桥接数据    │   ssl://mqtt:8884     │  • 提供API给前端           │  │
│  │  • 转发命令到Edge     │   (容器内部)            │  • 管理多个储能柜           │  │
│  └───────────────────────┘                       └───────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**端口与角色**
- Edge Broker (#1)：TLS `8883`（仅TLS，无明文端口），本地采集与桥接上行。
- Cloud Broker (#2)：TLS `8884`（仅TLS，无明文端口），既接收 Edge 桥接上行，也承载 Cloud → Edge 指令。
- 所有 MQTT 通信均使用 TLS 加密。

## 组件与连接信息

| 组件 | Broker 地址 | 端口 | 角色 |
|------|-------------|------|------|
| orangepi-gateway | `ssl://127.0.0.1:8883` | 8883 | 发布者（连接本地 Edge Broker） |
| Edge 服务 | `ssl://127.0.0.1:8883` | 8883 | 订阅者 + Broker 宿主 |
| Edge Mosquitto | 桥接到 `ssl://<CLOUD_IP>:8884` | 8884 | 数据中转（TLS 桥接） |
| Cloud 服务 | `ssl://mqtt:8884`（容器内） / `ssl://<CLOUD_IP>:8884` | 8884 | 订阅者、命令发布者 |

## Topic 设计

**上行（Edge → Cloud，经桥接）**
- `sensors/#`（主通道，100ms-1s 级）
- `devices/+/status`
- `devices/+/heartbeat`
- `alerts/#`

**下行（Cloud → Edge，经 Cloud Broker）**
- `commands/#`（指令）
- `control/#`（控制）
- `config/#`（配置）
- `cloud/cabinet/+/policy/#`（策略分发）

**常用格式**
- 传感器：`sensors/{device_id}/{sensor_type}`
- 状态：`devices/{device_id}/status`
- 心跳：`devices/{device_id}/heartbeat`
- 告警：`alerts/{device_id}/{severity}`

## 数据格式（与 mqtt.md 一致）

```json
{
  "device_id": "temperature_humidity",
  "sensor_type": "temperature",
  "value": 24.5,
  "unit": "°C",
  "timestamp": "2025-11-28T10:30:00Z",
  "quality": 100
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| device_id | string | 设备唯一标识 |
| sensor_type | string | 传感器类型 (`temperature`, `humidity`, `co2`, `co`, `smoke`, `liquid_level`, `conductivity`, `flow`) |
| value | float64 | 传感器数值 |
| unit | string | 测量单位 |
| timestamp | string | ISO 8601 时间戳 |
| quality | int | 数据质量 (0-100) |

## 快速测试

**Edge 侧订阅（TLS 8883）**
```bash
mosquitto_sub -h 127.0.0.1 -p 8883 --cafile /path/to/ca.crt --insecure -t "sensors/#" -v
```

**Cloud 侧订阅（TLS 8884）**
```bash
mosquitto_sub -h <CLOUD_IP> -p 8884 --cafile /path/to/ca.crt --insecure -t "sensors/#" -v
```

## 证书与安全（单向 TLS）
- Edge Broker：使用本地 CA（`ca.crt`），OrangePi 发布可用 `client.crt/key`，Edge Broker 验证。
- Cloud Broker：使用 Cloud CA（`ca_cloud.crt`），Edge 桥接使用该 CA 验证，单向 TLS，不校验客户端证书（测试场景可 `--insecure`）。

## 参考 Go 客户端片段（订阅 sensors/#）

```go
opts := mqtt.NewClientOptions().
    AddBroker("ssl://127.0.0.1:8883").
    SetClientID("bms-client-001")
opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
opts.SetOnConnectHandler(func(c mqtt.Client) {
    c.Subscribe("sensors/#", 1, handler)
})
```

> 数据落地后由 Cloud 后端写入 PostgreSQL，并通过 WebSocket 推送至前端卡片。
