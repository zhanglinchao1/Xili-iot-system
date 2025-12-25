# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XiLi is a **three-tier IoT energy storage cabinet monitoring system** with Zero-Knowledge Proof (ZKP) authentication. The repository contains three interconnected projects:

- **OrangePi (Gateway)**: IoT gateway collecting sensor data from physical devices via RS485/Modbus
- **Edge (Local Server)**: Middleware server providing ZKP authentication, local data storage, and cloud synchronization
- **Cloud (Platform)**: Central cloud platform for multi-cabinet management with web dashboard

**Key Technologies**: Go 1.24+, Gnark ZKP (Groth16), MQTT, PostgreSQL (Cloud), SQLite (Edge), Vue 3 + TypeScript (Cloud frontend), Vanilla JS (Edge frontend)

## System Architecture

```
Hardware Sensors (7 types)
  ↓ RS485/Modbus Serial
OrangePi Gateway (client2)
  ↓ MQTT TLS (port 8883) + ZKP Auth
Edge Server (github.com/edge/storage-cabinet)
  ↓ HTTP + MQTT TLS (port 8884)
Cloud Platform (cloud-system)
  ↓ Vue Dashboard
Web Users
```

### Data Flow

1. **Sensor Collection**: OrangePi reads 7 sensor types (temperature, CO, CO2, smoke, liquid_level, conductivity, flow) via serial ports
2. **Local Transmission**: OrangePi publishes MQTT messages to Edge broker (98% bandwidth reduction via thresholds)
3. **Authentication**: Each device uses ZKP (Groth16) for initial auth, then JWT for MQTT
4. **Edge Processing**: Edge stores in SQLite, detects threshold violations, generates alerts
5. **Cloud Sync**: Edge syncs data to Cloud every 5 minutes via HTTP API (batch size 1000)
6. **Command Dispatch**: Cloud sends commands to Edge via MQTT for remote configuration

## Build & Run Commands

### OrangePi (Gateway)

```bash
cd orangepi

# Build
go build -o orangepi_gateway main.go

# Cross-compile for ARM64
GOOS=linux GOARCH=arm64 go build -o orangepi_gateway main.go

# Run
./orangepi_gateway

# Docker deployment
docker-compose up -d
docker-compose logs -f
```

**Important Files**:
- `device_config.json` - Sensor device registry (auto-generated)
- `auth_proving.key` - ZKP proving key (238KB, must match Edge's verifying key)
- `config/certs/` - TLS certificates for MQTT

### Edge (Local Server)

```bash
cd edge

# Build backend (CGO required for SQLite)
CGO_ENABLED=1 go build -o edge cmd/edge/main.go

# Database migration
./edge -config ./configs/config.yaml --migrate

# Production deployment (backend + frontend)
./start_all.sh

# Force restart
./start_all.sh --force

# Stop all
./stop_all.sh

# View logs
tail -f logs/edge.log
tail -f logs/backend.log
```

**Ports**:
- 8001: Backend API
- 8000: Frontend web UI
- 8883: MQTT broker (TLS)

**Important Files**:
- `configs/config.yaml` - Main configuration
- `auth_verifying.key` - ZKP verifying key (460 bytes, must match OrangePi's proving key)
- `configs/license.lic` - License file (if license system enabled)
- `data/edge.db` - SQLite database

### Cloud (Platform)

```bash
cd cloud

# Build backend
go build -o cloud cmd/cloud-server/main.go

# Build frontend
cd frontend
npm install
npm run build
cd ..

# Production deployment (recommended: Docker)
docker-compose up -d

# Start all services (backend + frontend + DB + Redis + MQTT)
./start_all.sh

# Check deployment health
./check_deployment.sh

# View status
./status.sh

# Stop all
./stop_all.sh
```

**Ports**:
- 8003: Backend API
- 8002: Frontend (Vue app)
- 5433: PostgreSQL (external)
- 6379: Redis
- 8884: MQTT broker (TLS)

**Important Files**:
- `config.yaml` - Local development config
- `config.docker.yaml` - Docker environment config
- `docker-compose.yml` - Full stack orchestration
- `configs/certs/` - TLS certificates

## Configuration Files

### Edge Configuration (`edge/configs/config.yaml`)

Key settings:
- `server.port: 8001` - Backend API port
- `auth.zkp.verifying_key_path` - ZKP verifying key location
- `cloud.endpoint` - Cloud API endpoint (http://150.158.102.52:8003/api/v1)
- `cloud.api_key` - Authentication for Cloud API
- `cloud.sync_interval: 5m` - Data sync frequency
- `mqtt.broker_address` - Cloud MQTT broker (ssl://150.158.102.52:8884)
- `license.enabled` - Enable/disable license system

### Cloud Configuration (`cloud/config.yaml`)

Key settings:
- `server.port: 8003` - Backend API port
- `database.postgres` - PostgreSQL connection settings
- `database.redis` - Redis connection settings
- `mqtt.broker` - Cloud MQTT broker for commands (ssl://localhost:8884)
- `edge_mqtt.enabled` - Enable Edge data subscription
- `business.sync.interval: 5m` - Sync settings
- `business.map.tencent_map_key` - Tencent Maps API key

## Inter-Project Communication

### OrangePi → Edge

**Protocol**: MQTT (primary) + HTTP (legacy)

**MQTT Broker**: Edge local Mosquitto (ssl://0.0.0.0:8883)

**Topics** (OrangePi publishes):
```
sensors/{device_id}/{sensor_type}      # Sensor readings
devices/{device_id}/status             # Device status changes
alerts/{device_id}/{severity}          # Alert messages
devices/{device_id}/heartbeat          # Heartbeat (30s interval)
```

**Authentication Flow**:
1. Device requests ZKP challenge: `POST /api/v1/auth/challenge`
2. Device generates proof using `auth_proving.key`
3. Device submits proof: `POST /api/v1/auth/verify` → receives JWT token
4. Device uses JWT as MQTT password for publishing

### Edge → Cloud

**Protocol**: HTTP (data sync) + MQTT (commands)

**HTTP API** (Edge → Cloud):
```
POST /api/v1/cabinets/{cabinet_id}/sync       # Bulk sensor data (every 5min)
POST /api/v1/cabinets/{cabinet_id}/heartbeat  # Cabinet heartbeat
GET  /api/v1/cabinets/{cabinet_id}/license    # License validation
```

**Authentication**: API Key in `cloud.api_key` config

**MQTT Broker**: Cloud Mosquitto (ssl://150.158.102.52:8884)

**Topics** (Cloud → Edge):
```
cloud/commands/{cabinet_id}/config           # Config update command
cloud/commands/{cabinet_id}/license/update   # License update
cloud/commands/{cabinet_id}/license/revoke   # License revocation
```

**Topics** (Edge → Cloud):
```
cloud/responses/{cabinet_id}/config          # Config response
cloud/responses/{cabinet_id}/license/update  # License update response
```

## Key Components by Project

### OrangePi Components

- **ZKP Client** (`zkp/`) - Gnark Groth16 proof generation
- **MQTT Publisher** (`mqtt/`) - Sensor data publishing with auto-reconnect
- **Serial Manager** (`modbus/`) - RS485/Modbus communication for 7 sensors
- **Device Registry** (`config/`) - Device configuration and JWT token management

**Serial Ports**: `/dev/ttyCH9344USB0` through `/dev/ttyCH9344USB6` (CH9344 driver required)

### Edge Components

See `edge/CLAUDE.md` for detailed Edge architecture. Key highlights:

- **ZKP Verifier** (`internal/zkp/`) - Gnark Groth16 verification using `auth_verifying.key`
- **License System** (`internal/license/`) - SPA (Single Package Authorization) with RSA-2048
- **MQTT Subscriber** (`internal/mqtt/`) - Receives OrangePi sensor data
- **Cloud Sync Service** (`internal/cloud/`) - HTTP client for bulk data sync
- **Device Manager** (`internal/device/`) - Device lifecycle, heartbeat monitoring
- **Collector** (`internal/collector/`) - Threshold detection, alert generation, statistics
- **Web UI** (`web/`) - Vanilla JavaScript SPA with Chart.js

### Cloud Components

**Backend**:
- **MQTT Broker** (`internal/mqtt/`) - Command dispatch and Edge data subscription
- **Cabinet Manager** (`internal/cabinet/`) - Multi-cabinet cluster management
- **ABAC System** (`internal/abac/`) - Attribute-Based Access Control with trust scoring
- **Health Scoring** (`internal/health/`) - Cabinet health algorithm
- **License Manager** (`internal/license/`) - License distribution and validation
- **Sync Receiver** (`internal/sync/`) - Handles bulk data from Edge servers

**Frontend** (`cloud/frontend/`):
- Vue 3 + TypeScript + Vite build system
- Element Plus UI components
- ECharts for data visualization
- Pinia for state management
- Tencent Maps integration for cabinet locations

## Critical Notes

### Zero-Knowledge Proof (ZKP) Keys

**CRITICAL**: The ZKP proving and verifying keys must be from the same Trusted Setup ceremony.

- **Proving Key** (`orangepi/auth_proving.key`, 238KB): Used by OrangePi for generating proofs
- **Verifying Key** (`edge/auth_verifying.key`, 460 bytes): Used by Edge for verifying proofs
- **Security Principle**: Edge server NEVER generates keys, only verifies proofs
- **Mismatch Symptoms**: All authentication fails if keys don't match

Generate keys using: `orangepi/cmd/zkp_setup/`

### MQTT Broker Architecture

**Two Independent Brokers**:

1. **Edge Broker** (port 8883 TLS):
   - Purpose: OrangePi → Edge sensor data
   - High-frequency traffic (optimized with thresholds)
   - Authentication: ZKP + JWT
   - TLS: Mutual TLS (client cert required)

2. **Cloud Broker** (port 8884 TLS):
   - Purpose: Cloud ↔ Edge commands/responses
   - Low-frequency control traffic
   - Authentication: API key or anonymous
   - TLS: Single-sided (no client cert verification)

### License System (Edge Only)

- **Type**: Single Package Authorization (SPA)
- **Cryptography**: RSA-2048 signed JWT
- **Binding**: MAC address of Edge device
- **Grace Period**: 72 hours after expiration
- **Generation**: Use `edge/tools/license-gen/license-gen` tool

When `license.enabled: true`:
- License file must exist at `edge/configs/license.lic`
- Public key must exist at `edge/configs/vendor_pubkey.pem`
- If validation fails, all auth endpoints return 403

### Database Architecture

- **Edge**: SQLite with WAL mode (embedded, lightweight, concurrent reads)
- **Cloud**: PostgreSQL + TimescaleDB (time-series optimization, multi-cabinet scale, hypertables)

**Schema Auto-Migration**: Both projects auto-migrate schemas on startup with `--migrate` flag

### TLS Certificate Chain

Root CA: XiLi-CA
- Cloud MQTT Server Cert (150.158.102.52:8884)
- Edge MQTT Server Cert (local:8883)

Certificate locations:
- Cloud: `cloud/configs/certs/ca.crt`
- Edge: `edge/configs/certs/` (ca.crt, server.crt, client.crt)
- OrangePi: `orangepi/config/certs/` (ca.crt, client.crt)

See `tls.md` for certificate management guide.

## Sensor Types

All projects support these 7 sensor types:

| Type | Unit | Description | Typical Range |
|------|------|-------------|---------------|
| `temperature` | °C | Temperature | -10 to 60°C |
| `co` | ppm | Carbon Monoxide | 0 to 50 ppm |
| `co2` | ppm | Carbon Dioxide | 0 to 5000 ppm |
| `smoke` | AD value | Smoke detection | 0 to 1000 |
| `liquid_level` | mm | Liquid level | 0 to 900 mm |
| `conductivity` | mS/cm | Electrical conductivity | 0 to 10 mS/cm |
| `flow` | L/min | Flow rate | 0 to 100 L/min |

## Common Pitfalls

1. **CGO Required for Edge**: SQLite driver needs `CGO_ENABLED=1` or build fails
2. **Port Conflicts**: Ensure ports 8000-8003, 8883-8884, 5433, 6379 are available
3. **ZKP Key Mismatch**: OrangePi `auth_proving.key` and Edge `auth_verifying.key` must be from same Trusted Setup
4. **MAC Address Binding**: License is bound to Edge device MAC - VM/Docker needs careful handling
5. **Frontend Cache**: Edge web UI uses `?v=timestamp` cache-busting - update timestamps in `index.html` when editing CSS/JS
6. **MQTT TLS**: Both brokers use TLS - ensure certificates are valid and CA cert is present
7. **Cloud PostgreSQL Init**: First startup may be slow due to TimescaleDB extension installation
8. **Serial Port Permissions**: OrangePi needs `/dev/ttyCH9344USB*` access - add user to `dialout` group

## Testing and Troubleshooting

### OrangePi

```bash
# Test MQTT connection
cd orangepi/cmd/mqtt_test
go run main.go

# Check serial ports
ls -l /dev/ttyCH9344USB*

# View device registry
cat device_config.json

# Test ZKP authentication
# (generates proof and verifies with Edge)
```

### Edge

```bash
# Test MQTT dataflow
mosquitto_sub -h localhost -p 1883 -t 'sensors/#' -v

# Check database
sqlite3 data/edge.db "SELECT COUNT(*) FROM sensor_data;"

# Verify ZKP key
ls -lh auth_verifying.key  # Should be 460 bytes

# Test Cloud sync
curl http://localhost:8001/api/v1/status
```

### Cloud

```bash
# Check deployment health
./check_deployment.sh

# Test MQTT dataflow
./check_mqtt_dataflow.sh

# Service integration tests
./scripts/test_cloud_services.sh

# Check PostgreSQL
docker exec -it cloud-postgres psql -U cloud_user -d cloud_system -c "\dt"

# Check Redis
docker exec -it cloud-redis redis-cli PING
```

## Documentation References

### OrangePi
- `LINUX/README.md` - CH9344 serial driver installation

### Edge
- `edge/CLAUDE.md` - Comprehensive Edge architecture guide
- `edge/ALL_API.md` - Complete API specification (1300+ lines)
- `edge/GNARK_ZKP_COMPLETE_GUIDE.md` - ZKP implementation details
- `edge/ZKP_VERIFICATION_FIX.md` - ZKP key mismatch troubleshooting
- `edge/MQTT_INTEGRATION_COMPLETE.md` - MQTT integration report
- `edge/SPA.md` - License system design
- `edge/web/README.md` - Frontend development guide

### Cloud
- `cloud/MQTT.md` - Cloud MQTT architecture
- `cloud/DOCKER.md` - Docker deployment guide
- `cloud/Edge_ALL_API.md` - Edge API reference (for Cloud sync)
- `cloud/EDGE_CLIENT_CONFIG.md` - Edge client configuration

### Root
- `tls.md` - TLS certificate architecture and management

## Module Paths

- **OrangePi**: `client2`
- **Edge**: `github.com/edge/storage-cabinet`
- **Cloud**: `cloud-system`

When adding imports:
```go
// Edge example
import "github.com/edge/storage-cabinet/internal/device"

// Cloud example
import "cloud-system/internal/cabinet"
```

## Environment Context

**Current Deployment**: Edge and OrangePi are deployed on the same physical system. Cloud is deployed separately on a cloud server. When modifying Cloud code, changes will be synced to the cloud environment for you.

**Network Architecture**:
- OrangePi ↔ Edge: Local network (127.0.0.1 or 172.18.2.214)
- Edge ↔ Cloud: Internet (150.158.102.52)

## Logging

All projects use `go.uber.org/zap` for structured logging:

```go
logger.Info("Operation successful",
    zap.String("device_id", deviceID),
    zap.Int("count", count),
)
```

**Log Locations**:
- Edge: `logs/edge.log`, `logs/backend.log`, `logs/frontend.log`
- Cloud: `logs/cloud.log` (or Docker logs)
- OrangePi: `orangepi.log` (if run with nohup)
