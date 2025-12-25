# Quick Start Guide: Cloud端储能柜集群管理系统

**Feature**: Cloud端储能柜集群管理系统  
**Date**: 2025-11-04

## 概述

本指南帮助开发者快速搭建和运行Cloud端储能柜集群管理系统。

## 前置要求

- Go 1.21 或更高版本
- PostgreSQL 14+ (安装TimescaleDB扩展)
- Redis 6+
- MQTT Broker (如Mosquitto)
- Docker 和 Docker Compose (可选，用于本地开发)

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd Cloud
git checkout 001-cloud-cabinet-control
```

### 2. 配置文件设置

系统使用统一的 `config.yaml` 配置文件管理所有系统参数和业务参数。

**创建配置文件**：

```bash
# 复制配置示例文件
cp config.example.yaml config.yaml

# 编辑配置文件，修改数据库密码、JWT密钥等敏感信息
vim config.yaml
```

**config.yaml 示例结构**：

```yaml
# 服务器配置
server:
  port: 8003
  host: 0.0.0.0

# 数据库配置
database:
  postgres:
    host: localhost
    port: 5432
    user: cloud_user
    password: your_password  # 修改为实际密码
    dbname: cloud_system
  redis:
    host: localhost
    port: 6379
    password: ""

# MQTT配置
mqtt:
  broker: tcp://localhost:1883
  username: ""
  password: ""

# JWT配置
jwt:
  secret: your-secret-key-change-in-production  # 修改为安全的密钥
  expiry: 24h

# 日志配置
logging:
  level: info
  format: json

# 业务参数配置
business:
  health_score:
    weights:
      online_rate: 0.4
      data_quality: 0.3
      alert_severity: 0.2
      sensor_normalcy: 0.1
  frontend:
    api_base_url: http://localhost:8003/api/v1
    polling_interval: 5000
```

**环境变量覆盖**（可选）：

系统支持通过环境变量覆盖配置文件中的值：

```bash
# 环境变量命名规则: CLOUD_<配置路径>，使用下划线分隔
export CLOUD_SERVER_PORT=8080
export CLOUD_DATABASE_POSTGRES_PASSWORD=actual_password
export CLOUD_JWT_SECRET=actual_secret_key
```

**多环境配置**：

- 开发环境: `config.dev.yaml` 或 `config.yaml` + 环境变量
- 生产环境: `config.prod.yaml` 或通过环境变量覆盖

### 3. 数据库初始化

```bash
# 创建数据库
createdb cloud_system

# 安装TimescaleDB扩展
psql -d cloud_system -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"

# 运行迁移脚本（待实现）
# go run cmd/migrate/main.go up
```

### 4. 安装依赖

```bash
go mod download
```

### 5. 运行服务

```bash
go run cmd/cloud-server/main.go
```

服务将在 `http://localhost:8003` 启动。

## Docker Compose 快速启动

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: timescale/timescaledb:latest-pg14
    environment:
      POSTGRES_USER: cloud_user
      POSTGRES_PASSWORD: cloud_password
      POSTGRES_DB: cloud_system
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  mqtt:
    image: eclipse-mosquitto:2.0
    ports:
      - "1883:1883"
      - "9001:9001"
    volumes:
      - ./mosquitto.conf:/mosquitto/config/mosquitto.conf

volumes:
  postgres_data:
  redis_data:
```

启动服务：

```bash
docker-compose up -d
```

## API 使用示例

### 1. 创建储能柜

```bash
curl -X POST http://localhost:8080/api/v1/cabinets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "cabinet_id": "CABINET-001",
    "location": "北京-储能柜#1",
    "type": "锂电池储能柜",
    "capacity": 1000.0,
    "mac_address": "00:0c:29:3c:42:fe",
    "customer_name": "XX储能科技",
    "install_date": "2025-01-01"
  }'
```

### 2. 查询储能柜列表

```bash
curl -X GET "http://localhost:8080/api/v1/cabinets?page=1&page_size=20&status=online" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. Edge端数据同步

```bash
curl -X POST http://localhost:8080/api/v1/cabinets/CABINET-001/sync \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer EDGE_API_KEY" \
  -d '{
    "cabinet_id": "CABINET-001",
    "timestamp": "2025-11-04T10:30:00+08:00",
    "sensor_data": [
      {
        "device_id": "CO2_SENSOR_001",
        "sensor_type": "co2",
        "value": 520.5,
        "unit": "ppm",
        "timestamp": "2025-11-04T10:25:00+08:00",
        "quality": 100,
        "synced": false
      }
    ],
    "alerts": []
  }'
```

### 4. 创建许可证

```bash
curl -X POST http://localhost:8080/api/v1/licenses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "cabinet_id": "CABINET-001",
    "mac_address": "00:0c:29:3c:42:fe",
    "max_devices": -1,
    "expires_at": "2026-11-04T00:00:00+08:00",
    "permissions": ["auth", "collect", "alert", "statistics"]
  }'
```

### 5. Edge端许可证验证

```bash
curl -X POST http://localhost:8080/api/v1/license/validate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer EDGE_API_KEY" \
  -d '{
    "cabinet_id": "CABINET-001",
    "mac_address": "00:0c:29:3c:42:fe",
    "version": "2.0.1"
  }'
```

## 测试

### 运行单元测试

```bash
go test ./... -v
```

### 运行集成测试

```bash
go test ./tests/integration/... -v
```

### 检查代码覆盖率

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 开发工作流

1. **创建功能分支**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **编写测试**
   - 先编写单元测试
   - 确保测试失败

3. **实现功能**
   - 实现代码
   - 确保测试通过

4. **代码检查**
   ```bash
   go fmt ./...
   go vet ./...
   golangci-lint run
   ```

5. **提交代码**
   ```bash
   git add .
   git commit -m "feat: your feature description"
   ```

## 项目结构

```
cmd/cloud-server/        # 应用入口
internal/
  api/                   # API层
  models/                # 数据模型
  services/              # 业务逻辑
  repository/            # 数据访问
  mqtt/                  # MQTT客户端
  config/                # 配置管理
pkg/                     # 公共包
tests/                   # 测试代码
```

## 下一步

- 查看 [API文档](./contracts/openapi.yaml)
- 查看 [数据模型](./data-model.md)
- 查看 [研究文档](./research.md)
- 查看 [实施计划](./plan.md)

## 故障排查

### 数据库连接失败

检查：
- PostgreSQL服务是否运行
- 数据库配置是否正确
- TimescaleDB扩展是否安装

### Redis连接失败

检查：
- Redis服务是否运行
- Redis配置是否正确

### MQTT连接失败

检查：
- MQTT Broker是否运行
- MQTT配置是否正确
- 网络连接是否正常

## 支持

如有问题，请查看：
- 项目文档
- 项目原则文档
- 代码注释

