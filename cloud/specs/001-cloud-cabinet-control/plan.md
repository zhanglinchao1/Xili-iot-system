# Implementation Plan: Cloud端储能柜集群管理系统

**Branch**: `001-cloud-cabinet-control` | **Date**: 2025-11-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-cloud-cabinet-control/spec.md`

## Summary

构建Cloud端储能柜集群管理系统,用于管理和监控多个储能柜(Edge系统)。系统提供储能柜注册、传感器数据接收与查询、远程指令下发、许可证管理、告警监控和健康评分功能。采用Go + Gin构建RESTful API后端,使用PostgreSQL(关系型数据) + TimescaleDB(时序数据) + Redis(缓存) + MQTT(实时消息),前端使用Vue.js 3独立开发。

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- Gin (HTTP框架)
- pgx (PostgreSQL驱动)
- timescaledb (时序数据库扩展)
- go-redis (Redis客户端)
- paho.mqtt.golang (MQTT客户端)
- golang-jwt/jwt (JWT认证)
- zap (结构化日志)
- viper (配置管理)

**Storage**:
- PostgreSQL 15+ (关系型数据: cabinets, devices, licenses, alerts, commands, audit_logs)
- TimescaleDB (时序数据: sensor_data, health_scores)
- Redis 7+ (缓存: license cache, session cache)

**Testing**:
- Go标准testing包
- testify (断言和mock)
- Contract tests (API端点测试)
- Integration tests (数据库、MQTT集成测试)

**Target Platform**: Linux服务器 (Docker容器化部署)

**Project Type**: Web应用 (Backend + Frontend分离)

**Performance Goals**:
- 1000个储能柜并发连接(MQTT持久连接)
- 2000 QPS (API请求)
- 95%请求在2秒内响应
- 每5分钟处理1000个储能柜 × 1000条传感器数据

**Constraints**:
- API响应时间 p95 < 2秒
- License验证 p95 < 200ms (使用Redis缓存)
- 离线检测延迟 < 1分钟
- 健康评分计算 < 30秒

**Scale/Scope**:
- 支持管理1000+储能柜
- 每天处理约2.88亿条传感器数据 (1000柜 × 12次/小时 × 1000条 × 24小时)
- MVP阶段: 5个核心用户故事,约6000行Go代码 + 1000行Vue.js代码

## Constitution Check

*GATE: Must pass before implementation. Re-checked after design.*

根据 `.specify/memory/constitution.md` 验证:

| 原则 | 状态 | 验证 |
|------|------|------|
| **I. Test-First Development** | ✅ PASS | tasks.md包含完整的测试任务(T028-T133),先于实现任务 |
| **II. API Design Standards** | ✅ PASS | contracts/openapi.yaml定义统一响应格式,使用/api/v1版本化 |
| **III. Data Architecture** | ✅ PASS | data-model.md明确定义PostgreSQL(关系型) + TimescaleDB(时序) + Redis(缓存) |
| **IV. Security Requirements** | ✅ PASS | spec.md FR-017要求JWT认证,FR-018-019要求审计日志 |
| **V. Observability** | ✅ PASS | tasks.md包含zap日志、health check端点(T158)、metrics端点(T159) |
| **VI. Code Quality** | ✅ PASS | tasks.md包含golangci-lint(T169)、gofmt(T170)、70%覆盖率(T168) |
| **VII. Simplicity** | ✅ PASS | 技术栈精简(7个核心依赖),MVP优先,按用户故事增量交付 |

**Constitution合规性**: ✅ **所有原则通过**

## Project Structure

### Documentation (this feature)

```text
specs/001-cloud-cabinet-control/
├── plan.md              # 本文件 (技术实现计划)
├── spec.md              # 功能规格说明
├── research.md          # 技术调研文档
├── data-model.md        # 数据模型设计
├── quickstart.md        # 快速开始指南
├── contracts/           # API契约
│   └── openapi.yaml     # OpenAPI 3.0规范
└── tasks.md             # 实现任务列表 (231个任务)
```

### Source Code (repository root)

采用 **Option 2: Web应用架构** (Backend + Frontend分离)

```text
Cloud/
├── cmd/
│   └── cloud-server/
│       └── main.go                 # 服务器入口
├── internal/
│   ├── api/
│   │   ├── handlers/               # HTTP处理器
│   │   │   ├── cabinet.go
│   │   │   ├── sensor.go
│   │   │   ├── command.go
│   │   │   ├── license.go
│   │   │   ├── alert.go
│   │   │   ├── config.go
│   │   │   ├── health.go
│   │   │   ├── metrics.go
│   │   │   └── dto.go              # 数据传输对象
│   │   ├── middleware/             # 中间件
│   │   │   ├── auth.go
│   │   │   ├── logging.go
│   │   │   ├── error_handler.go
│   │   │   ├── validation.go
│   │   │   ├── audit.go
│   │   │   └── cors.go
│   │   └── routes.go               # 路由定义
│   ├── models/                     # 数据模型
│   │   ├── cabinet.go
│   │   ├── sensor.go
│   │   ├── license.go
│   │   ├── alert.go
│   │   └── command.go
│   ├── services/                   # 业务逻辑
│   │   ├── cabinet_service.go
│   │   ├── sensor_service.go
│   │   ├── command_service.go
│   │   ├── license_service.go
│   │   ├── alert_service.go
│   │   ├── health_score_service.go
│   │   └── audit_service.go
│   ├── repository/                 # 数据访问
│   │   ├── repository.go           # 接口定义
│   │   ├── postgres/
│   │   │   ├── postgres.go         # 连接池
│   │   │   ├── cabinet_repo.go
│   │   │   ├── sensor_device_repo.go
│   │   │   ├── license_repo.go
│   │   │   ├── alert_repo.go
│   │   │   ├── command_repo.go
│   │   │   └── audit_repo.go
│   │   ├── timescaledb/
│   │   │   ├── sensor_repo.go
│   │   │   └── health_score_repo.go
│   │   └── redis/
│   │       ├── redis.go            # 连接管理
│   │       └── license_cache.go
│   ├── mqtt/                       # MQTT客户端
│   │   ├── client.go
│   │   ├── publisher.go
│   │   └── subscriber.go
│   ├── config/                     # 配置管理
│   │   └── config.go
│   └── utils/                      # 工具函数
│       ├── logger.go
│       ├── validator.go
│       └── response.go
├── pkg/
│   └── errors/                     # 错误定义
│       └── errors.go
├── tests/
│   ├── contract/                   # 契约测试
│   │   ├── test_cabinet_create.go
│   │   ├── test_cabinet_list.go
│   │   ├── test_sensor_sync.go
│   │   └── ...
│   ├── integration/                # 集成测试
│   │   ├── test_cabinet_crud.go
│   │   ├── test_sensor_data.go
│   │   └── ...
│   └── unit/                       # 单元测试
│       ├── models/
│       └── services/
├── migrations/                     # 数据库迁移
│   ├── 001_create_timescaledb.sql
│   ├── 002_create_tables.sql
│   └── ...
├── frontend/                       # Vue.js前端 (独立项目)
│   ├── src/
│   │   ├── views/                  # 页面组件
│   │   │   ├── CabinetList.vue
│   │   │   ├── CabinetDetail.vue
│   │   │   ├── SensorData.vue
│   │   │   ├── CommandSend.vue
│   │   │   ├── LicenseManage.vue
│   │   │   ├── AlertManage.vue
│   │   │   └── Dashboard.vue
│   │   ├── components/             # 公共组件
│   │   │   ├── Layout.vue
│   │   │   ├── StatusBadge.vue
│   │   │   ├── HealthScore.vue
│   │   │   ├── DataTable.vue
│   │   │   └── ChartCard.vue
│   │   ├── api/                    # API调用
│   │   │   ├── index.ts
│   │   │   ├── cabinet.ts
│   │   │   ├── sensor.ts
│   │   │   ├── command.ts
│   │   │   ├── license.ts
│   │   │   └── alert.ts
│   │   ├── store/                  # Pinia状态管理
│   │   │   ├── auth.ts
│   │   │   ├── config.ts
│   │   │   ├── cabinet.ts
│   │   │   ├── sensor.ts
│   │   │   ├── command.ts
│   │   │   ├── license.ts
│   │   │   └── alert.ts
│   │   ├── router/                 # Vue Router
│   │   │   └── index.ts
│   │   ├── utils/                  # 工具函数
│   │   │   └── request.ts
│   │   ├── types/                  # TypeScript类型
│   │   │   └── api.ts
│   │   └── main.ts
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── config.yaml                     # 配置文件
├── config.example.yaml             # 配置示例
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

**Structure Decision**:

选择Web应用架构(Backend + Frontend分离),原因:

1. **清晰的关注点分离**: 后端专注API和业务逻辑,前端专注用户界面
2. **独立部署**: Backend和Frontend可独立部署和扩展
3. **技术栈专业化**: Go后端高性能,Vue.js前端现代化用户体验
4. **符合宪章**: constitution.md明确定义"Frontend: Vue.js 3 + TypeScript + Element Plus (独立交付)"
5. **团队协作**: 前后端可并行开发,提高效率

后端使用标准Go项目结构:
- `cmd/`: 可执行程序入口
- `internal/`: 私有应用代码(不可被外部导入)
- `pkg/`: 可被外部导入的公共库
- `tests/`: 测试代码(按测试类型分类)
- `migrations/`: 数据库版本控制

前端使用Vue.js 3标准结构:
- 采用Vite构建工具(快速开发体验)
- TypeScript类型安全
- Pinia状态管理(Vue 3官方推荐)
- Vue Router路由管理
- Element Plus UI组件库
- ECharts数据可视化

## Complexity Tracking

> **本项目无宪章违规,此节为空**

所有设计决策符合宪章原则,无需复杂性豁免。
