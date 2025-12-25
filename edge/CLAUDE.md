# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Edge System is a **Zero-Knowledge Proof (ZKP) based authentication gateway** for energy storage cabinet sensor management. It authenticates IoT devices using Groth16 ZKP circuits (via Consensys Gnark), manages sensor data collection over RS485/Modbus, and provides a web-based management interface.

**Key Technologies**: Go 1.24+, Gnark ZKP, Gin HTTP framework, SQLite, MQTT (Paho), vanilla JavaScript frontend

**系统角色**: 本项目为Edge服务端,作为中间系统与cloud端和网关端连接,承担数据中转、设备管理、认证验证等任务

## Build & Run Commands

### Development

```bash
# Build the backend (CGO required for SQLite)
CGO_ENABLED=1 go build -o edge cmd/edge/main.go

# Run with configuration
./edge -config ./configs/config.yaml

# Run database migration
./edge -config ./configs/config.yaml --migrate

# Show version
./edge --version
```

### Production Deployment

```bash
# Start all services (backend + frontend web server)
./start_all.sh

# Start with force restart
./start_all.sh --force

# Stop all services
./stop_all.sh
```

### Frontend Development

The web UI is vanilla JavaScript (no build step required):

```bash
# Frontend is served by Python's HTTP server (via start_all.sh)
# Or manually:
cd web && python3 -m http.server 8000
```

**Important**: Frontend files use cache-busting query parameters (`?v=timestamp`). When editing CSS/JS, update the version timestamp in `index.html`.

### Testing

```bash
# Go module operations
go mod tidy
go mod download

# No automated test suite currently exists
# Manual testing via shell scripts in project root
```

## Architecture

### System Flow

```
IoT Device (Sensor)
  → ZKP Challenge Request (/api/v1/auth/challenge)
  → ZKP Proof Generation (client-side, Gnark)
  → ZKP Proof Verification (/api/v1/auth/verify)
  → JWT Token Issued
  → Authenticated Data Collection (/api/v1/data/collect)
```

### Key Components

1. **ZKP Authentication** (`internal/zkp/`)
   - Uses Gnark Groth16 proving system
   - `Verifier`: Production verifier using pre-generated `auth_verifying.key` from Trusted Setup
   - Circuit definition in `circuits/auth_circuit.go`
   - **Critical**: Server loads verifying key at startup, never generates keys (no Trusted Setup on server)
   - ZKP verification is CPU-intensive; only happens once per session
   - **Important**: `auth_verifying.key` (460 bytes) must exist in project root, generated from Trusted Setup by development team

2. **License System (SPA)** (`internal/license/`)
   - Single Package Authorization using RSA-2048 signed JWT
   - Binds license to MAC address (`ip link show` output parsing)
   - 72-hour grace period for expired licenses
   - **License check only at auth entry point** to minimize performance impact
   - Can be disabled via `license.enabled: false` in config

3. **Device Manager** (`internal/device/`)
   - Manages sensors: CO2, CO, smoke, liquid_level, conductivity, temperature, flow
   - Heartbeat monitoring (30s interval)
   - Auto-offline detection (5min timeout)

4. **Data Collector** (`internal/collector/`)
   - RS485 Modbus RTU protocol support
   - Batch collection with configurable intervals
   - Automatic threshold-based alerting
   - Statistics aggregation (1h/24h/7d/30d windows)

5. **Storage Layer** (`internal/storage/`)
   - SQLite with WAL mode for concurrent reads
   - Schema auto-migration on startup (`--migrate` flag)
   - Tables: devices, challenges, sessions, sensor_data, alerts, cabinets

6. **MQTT Integration** (`internal/mqtt/`)
   - **新增功能** (2025-10-27): 支持通过 MQTT 接收网关数据,与 HTTP API 并行工作
   - MQTT Broker: Mosquitto (tcp://172.18.2.214:1883)
   - 订阅 4 类 Topic: `sensors/#`, `devices/+/status`, `alerts/#`, `devices/+/heartbeat`
   - 自动重连机制(指数退避)
   - 与 HTTP 数据统一存储到相同数据库表
   - **优势**: 带宽节省 98%,实时性提升,支持设备离线即时检测(遗嘱消息)
   - 配置: `configs/config.yaml` 中 `mqtt.enabled: true` 启用

### Frontend Architecture

**Single-Page Application** pattern with no frameworks:

- `web/js/api.js`: Centralized fetch API wrapper
- `web/js/app.js`: Router and page lifecycle manager
- `web/js/devices.js`: Device management module
- `web/js/statistics.js`: Chart.js-based statistics with 7 sensor charts
- `web/js/alerts.js`: Alert management with severity badges
- `web/js/realtime.js`: Live sensor data with auto-refresh
- `web/js/logs.js`: Alert/auth log viewer with date filtering

**State Management**: Each module maintains its own state object (e.g., `Devices.filters`, `Logs.activeTab`)

## Critical Configuration

### License System

When `license.enabled: true` in config:
- License file must exist at `configs/license.lic`
- Public key must exist at `configs/vendor_pubkey.pem`
- If license check fails, **all auth endpoints return 403**
- Use license generator tool: `tools/license-gen/license-gen`

```bash
# Generate license
cd tools/license-gen
go run main.go \
  --mac "00:15:5d:41:5b:ca" \
  --devices 50 \
  --expire "2026-01-01" \
  --output ../../configs/license.lic
```

### Database

SQLite at `data/edge.db` (auto-created). For fresh start:

```bash
rm data/edge.db
./edge -config ./configs/config.yaml --migrate
```

## API Documentation

Complete API reference in `ALL_API.md` (1300+ lines). Key patterns:

- **No authentication required**: Device management, alerts, statistics, logs (designed for internal network)
- **JWT required**: Data collection endpoint (`/api/v1/data/collect`)
- **ZKP flow**: Challenge → Verify → JWT → Collect

### Example: Device Authentication Flow

```bash
# 1. Get challenge
curl -X POST http://localhost:8001/api/v1/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"device_id": "CO2_SENSOR_001"}'

# 2. Generate proof (client-side with Gnark)
# ... ZKP proof generation happens here ...

# 3. Verify proof and get JWT
curl -X POST http://localhost:8001/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"device_id": "CO2_SENSOR_001", "challenge_id": "...", "proof": "..."}'

# 4. Use JWT for data collection
curl -X POST http://localhost:8001/api/v1/data/collect \
  -H "Authorization: Bearer <jwt-token>" \
  -d '{"readings": [...]}'
```

## Code Conventions

### Error Handling Pattern

```go
// API handlers return structured JSON errors
c.JSON(http.StatusBadRequest, gin.H{
    "error": "ERROR_CODE",
    "message": "用户友好的错误描述"
})
```

### Logging

Uses `zap` structured logging:

```go
logger.Info("操作成功",
    zap.String("device_id", deviceID),
    zap.Int("count", count),
)
```

Logs go to `logs/edge.log` (main), `logs/backend.log`, `logs/frontend.log`

### Database Queries

Direct SQL (no ORM):

```go
rows, err := db.Query("SELECT id, name FROM devices WHERE status = ?", status)
defer rows.Close()
```

## Common Pitfalls

1. **CGO Required**: SQLite driver needs `CGO_ENABLED=1`. Build fails without it.

2. **Port Conflicts**: Backend (8001) and frontend (8000) must both be free. Use `start_all.sh` which auto-detects conflicts.

3. **ZKP Verifying Key Missing**: `auth_verifying.key` must exist in project root. Server initialization fails without it. This file is generated by development team via Trusted Setup (see `ZKP_VERIFICATION_FIX.md`). **Never generate keys on the server!**

4. **License MAC Mismatch**: License is bound to specific MAC address. VM/Docker environments need careful MAC handling.

5. **Frontend Cache**: CSS/JS changes not visible? Update `?v=` timestamp in `<link>`/`<script>` tags in `index.html`.

6. **CORS Issues**: Frontend must be served via HTTP server (not `file://`). Use Python or the integrated server in `start_all.sh`.

7. **ZKP Key Mismatch**: Network gateway's `auth_proving.key` and server's `auth_verifying.key` must be from the same Trusted Setup. Mismatched keys cause all ZKP verifications to fail.

## Project-Specific Context

### Sensor Types

System supports 7 sensor types (defined in `pkg/models/data.go`):
- `co2`: CO2 concentration (ppm)
- `co`: CO concentration (ppm)
- `smoke`: Smoke detection (0-100)
- `liquid_level`: Liquid level (0-100%)
- `conductivity`: Electrical conductivity (μS/cm)
- `temperature`: Temperature (°C)
- `flow`: Flow rate (L/min)

### Alert Severity Levels

Auto-calculated based on threshold excess:
- `critical`: >200% over threshold
- `high`: 100-200% over threshold
- `medium`: 50-100% over threshold
- `low`: <50% over threshold

### Cabinet System

Devices can be grouped into `cabinets` (storage units):
- Cabinet ID in format `CABINET_A1`, `CABINET_B2`
- Used for physical organization and cloud sync grouping
- API: `/api/v1/cabinets` for cabinet management

## Documentation References

- `ALL_API.md`: Complete API specification with examples
- `SPA.md`: License system (Single Package Authorization) design doc
- `GATEWAY_ARCHITECTURE_GUIDE.md`: System architecture deep dive
- `GNARK_ZKP_COMPLETE_GUIDE.md`: Zero-knowledge proof implementation details
- `ZKP完整流程说明.md`: ZKP complete flow explanation (Chinese)
- `ZKP_VERIFICATION_FIX.md`: ZKP verification bug fix report (2025-10-26)
- `REALTIME_SENSOR_MONITOR.md`: Real-time monitoring feature guide
- `STATISTICS_ALERT_FEATURE_PRD.md`: Statistics & alerting product requirements
- `MQTT_INTEGRATION_COMPLETE.md`: MQTT integration complete report (2025-10-27)
- `EDGE_MQTT_INTEGRATION_PROMPT.md`: MQTT integration requirements
- `web/README.md`: Frontend development guide

## Environment Variables

None currently used. All configuration via `configs/config.yaml`.

## Go Module Path

Module name: `github.com/edge/storage-cabinet`

When adding imports:
```go
import "github.com/edge/storage-cabinet/internal/device"
```

## Data Flow Architecture

### HTTP vs MQTT Data Ingestion

系统支持**双通道数据接收**:

1. **HTTP 通道** (传统方式):
   - 网关通过 ZKP 认证后获取 JWT token
   - 使用 `POST /api/v1/data/collect` 上传传感器数据
   - 适用场景: 需要严格认证的场景

2. **MQTT 通道** (推荐方式, 2025-10-27 新增):
   - 网关发布消息到 MQTT Broker
   - Edge 服务端订阅对应 Topic 接收数据
   - JWT 认证在 Broker 层完成(mosquitto-go-auth 插件)
   - 适用场景: 高频数据上传,带宽受限环境

**重要**: 两种通道接收的数据**统一存储**到相同的数据库表,Web 管理界面可以查看所有来源的数据

### Data Flow Diagram

```
┌─────────────┐
│   Gateway   │
│  (网关端)    │
└──────┬──────┘
       │
       ├──────── HTTP (POST /api/v1/data/collect) ────┐
       │                                               │
       └──────── MQTT (Publish to sensors/*)  ────────┤
                         │                            │
                         ▼                            ▼
                 ┌───────────────┐          ┌─────────────────┐
                 │ MQTT Broker   │          │  HTTP Server    │
                 │ (Mosquitto)   │          │  (Gin)          │
                 └───────┬───────┘          └────────┬────────┘
                         │                           │
                         ▼                           ▼
                 ┌────────────────────────────────────────┐
                 │   MQTT Subscriber / HTTP Handler      │
                 │   (internal/mqtt  / api/handlers)     │
                 └────────────────┬───────────────────────┘
                                  │
                                  ▼
                 ┌──────────────────────────────────────┐
                 │  Collector Service (统一数据处理)    │
                 │  - SaveSensorData() 数据通道         │
                 │  - 批量存储 (每100条或10秒)          │
                 │  - 阈值检测 → 告警生成              │
                 └────────────────┬─────────────────────┘
                                  │
                                  ▼
                 ┌──────────────────────────────────────┐
                 │       SQLite Database                │
                 │  - sensor_data (HTTP + MQTT 数据)    │
                 │  - alerts                            │
                 │  - devices                           │
                 └────────────────┬─────────────────────┘
                                  │
                                  ▼
                 ┌──────────────────────────────────────┐
                 │   Web Management Interface           │
                 │   Cloud Sync Service                 │
                 └──────────────────────────────────────┘
```

## MQTT Configuration

### Enable MQTT Support

在 `configs/config.yaml` 中配置:

```yaml
mqtt:
  enabled: true                              # 启用 MQTT
  broker_address: "tcp://172.18.2.214:1883"
  client_id: "edge-server-subscriber"
  username: "edge-server"                    # 管理员账号
  password: "edge-server-password"
  qos: 1
  keep_alive: 60
  clean_session: true
  reconnect_interval: 5s
  max_reconnect_attempts: 10
```

### MQTT Topics

Edge 服务端订阅以下 Topic:

| Topic Pattern | 说明 | 消息格式 |
|--------------|------|---------|
| `sensors/#` | 传感器数据 | `{"device_id":"...","sensor_type":"co2","value":420.5,"unit":"ppm","quality":100,"timestamp":"..."}` |
| `devices/+/status` | 设备状态变更 | `{"device_id":"...","status":"online","timestamp":"..."}` |
| `alerts/#` | 告警消息 | `{"device_id":"...","alert_type":"threshold_exceeded","severity":"high","message":"...","value":1200,"threshold":1000}` |
| `devices/+/heartbeat` | 设备心跳 | `{"device_id":"...","timestamp":"..."}` |

### MQTT Troubleshooting

**问题: MQTT 连接失败**

```bash
# 检查 Broker 是否运行
ss -tuln | grep 1883

# 测试连接
mosquitto_sub -h 172.18.2.214 -p 1883 -t test -u edge-server -P edge-server-password

# 查看 Edge 日志
tail -f logs/edge.log | grep MQTT
```

**问题: 消息未保存到数据库**

检查数据通道是否满:
```bash
# 日志中搜索
grep "data channel full" logs/edge.log
```

## Development Workflow

### 启动服务(包含 MQTT)

```bash
# 方式1: 使用启动脚本
./start_all.sh

# 方式2: 手动启动
CGO_ENABLED=1 go build -o edge cmd/edge/main.go
./edge -config ./configs/config.yaml
```

### 停止服务

```bash
# 使用停止脚本
./stop_all.sh

# 或手动停止
pkill -f './edge'
```

### 查看日志

```bash
# 主日志
tail -f logs/edge.log

# 后端日志
tail -f logs/backend.log

# 前端日志
tail -f logs/frontend.log
```
- 本项目为Edge服务端，与cloud端和网关端连接，本项目作为中间系统，承担中转等任务。