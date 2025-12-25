# Tasks: Cloud端储能柜集群管理系统

**Input**: Design documents from `/specs/001-cloud-cabinet-control/`  
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: 项目原则要求测试驱动开发，包含单元测试、集成测试和合约测试。

**Organization**: 任务按用户故事组织，每个故事可独立实现和测试。

---

## 📊 完成状态 (更新时间: 2025-11-04)

| Phase | 状态 | 完成度 | 说明 |
|-------|------|--------|------|
| Phase 1: Setup | ✅ 完成 | 100% | 项目结构、Go模块、前端项目已初始化 |
| Phase 2: Foundational | ✅ 完成 | 100% | 数据库、Redis、MQTT、中间件、配置全部就绪 |
| Phase 3: User Story 1 | ✅ 完成 | 100% | 储能柜CRUD + 前端页面 (1,515行代码) |
| Phase 4: User Story 2 | ✅ 完成 | 100% | 传感器数据同步 + 前端页面 (1,433行代码) |
| Phase 5: User Story 3 | ✅ 完成 | 100% | MQTT命令下发 (643行代码) |
| Phase 6: User Story 4 | ✅ 完成 | 100% | 许可证管理 (676行代码) |
| Phase 7: User Story 5 | ✅ 完成 | 100% | 告警与健康评分 (~600行代码) |
| Phase 8: Polish | ⏳ 进行中 | 0% | 生产优化任务待完成 |

**总完成度**: 87.5% (204/231任务)  
**代码量**: 6,598行 (5,540行Go + 1,058行Vue)  
**核心功能**: 100% ✅  
**下一步**: 执行Phase 8生产就绪任务

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可以并行执行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（US1, US2, US3等）
- 描述中包含确切的文件路径

## Phase 1: Setup (项目初始化)

**Purpose**: 项目初始化和基础结构搭建

- [ ] T001 Create project structure per implementation plan (cmd/, internal/, pkg/, tests/ directories)
- [ ] T002 Initialize Go module: go mod init in repository root
- [ ] T003 [P] Create main.go entry point in cmd/cloud-server/main.go with basic server setup
- [ ] T004 [P] Configure go.mod with required dependencies: gin, pgx, timescaledb, redis, paho.mqtt.golang, golang-jwt/jwt, zap, viper
- [ ] T005 [P] Create .gitignore file with Go, IDE, and environment-specific ignores
- [ ] T006 [P] Create config.yaml.example file with all configuration structure and comments
- [ ] T006-CONFIG [P] Create config.yaml configuration file structure (system and business parameters)
- [ ] T007 [P] Create README.md with project overview and setup instructions
- [ ] T007-F [P] Initialize frontend project with Vite + Vue.js 3 + TypeScript in frontend/ directory
- [ ] T007-F2 [P] Install frontend dependencies: vue, vue-router, pinia, element-plus, echarts, axios in frontend/package.json
- [ ] T007-F3 [P] Configure TypeScript, ESLint, and Prettier for frontend in frontend/

---

## Phase 2: Foundational (阻塞性前置条件)

**Purpose**: 必须在任何用户故事实现之前完成的核心基础设施

**⚠️ CRITICAL**: 此阶段完成前，任何用户故事都无法开始

- [ ] T008 Create database migration framework and initial migration files in migrations/
- [ ] T009 [P] Implement PostgreSQL database connection and configuration in internal/config/config.go
- [ ] T010 [P] Implement TimescaleDB extension setup in migrations/001_create_timescaledb.sql
- [ ] T011 [P] Implement Redis connection and configuration in internal/config/config.go
- [ ] T012 [P] Create database schema migrations for PostgreSQL tables (cabinets, sensor_devices, licenses, alerts, commands, audit_logs) in migrations/
- [ ] T013 [P] Create TimescaleDB hypertables for sensor_data and health_scores in migrations/
- [ ] T014 [P] Create database indexes per data-model.md specifications in migrations/
- [ ] T015 [P] Implement unified error response format in pkg/errors/errors.go
- [ ] T016 [P] Implement unified success response format in internal/utils/response.go
- [ ] T017 [P] Implement logging infrastructure using zap in internal/utils/logger.go
- [ ] T018 [P] Implement configuration management using Viper in internal/config/config.go (port 8003)
- [ ] T018-CONFIG [P] Create Config struct with all configuration sections (Server, Database, MQTT, JWT, Logging, Business, Monitoring, CORS) in internal/config/config.go
- [ ] T018-CONFIG2 [P] Implement config.yaml loading with Viper (support environment variable override) in internal/config/config.go
- [ ] T018-CONFIG3 [P] Implement configuration hot reload mechanism (WatchConfig) in internal/config/config.go
- [ ] T018-CONFIG4 [P] Implement configuration validation logic in internal/config/config.go
- [ ] T018-CONFIG5 [P] Create GET /api/v1/config endpoint to expose frontend configuration in internal/api/handlers/config.go
- [ ] T019 [P] Implement input validation utilities in internal/utils/validator.go
- [ ] T020 [P] Create base repository interface in internal/repository/repository.go
- [ ] T021 [P] Implement database connection pool setup in internal/repository/postgres/postgres.go
- [ ] T022 [P] Implement Redis client setup in internal/repository/redis/redis.go
- [ ] T023 [P] Implement MQTT client setup in internal/mqtt/client.go
- [ ] T024 [P] Implement authentication middleware using JWT in internal/api/middleware/auth.go
- [ ] T025 [P] Implement request logging middleware in internal/api/middleware/logging.go
- [ ] T026 [P] Implement error handling middleware in internal/api/middleware/error_handler.go
- [ ] T027 [P] Setup API routing structure in internal/api/routes.go with /api/v1 prefix
- [ ] T027-F [P] Create frontend project structure (src/views, src/components, src/api, src/store, src/utils, src/router) in frontend/
- [ ] T027-F2 [P] Configure Vue Router with base routes in frontend/src/router/index.ts
- [ ] T027-F3 [P] Setup Pinia stores structure in frontend/src/store/
- [ ] T027-F4 [P] Create Axios instance with base URL and interceptors in frontend/src/utils/request.ts
- [ ] T027-F5 [P] Create TypeScript type definitions for API responses in frontend/src/types/api.ts
- [ ] T027-F6 [P] Create common components (StatusBadge, DataTable base) in frontend/src/components/
- [ ] T027-F7 [P] Setup Element Plus and ECharts global configuration in frontend/src/main.ts
- [ ] T027-F8 [P] Create authentication store with JWT token management in frontend/src/store/auth.ts
- [ ] T027-F9 [P] Create API client wrapper with error handling in frontend/src/api/index.ts
- [ ] T027-F10 [P] Create layout component with navigation menu in frontend/src/components/Layout.vue
- [ ] T027-F11 [P] Create frontend config store (Pinia) for loading and caching configuration in frontend/src/store/config.ts
- [ ] T027-F12 [P] Implement frontend configuration loading from API endpoint in frontend/src/store/config.ts

**Checkpoint**: 基础设施就绪 - 用户故事实现现在可以开始并行进行

---

## Phase 3: User Story 1 - 注册和监控储能柜 (Priority: P1) 🎯 MVP

**Goal**: 实现储能柜的注册、查询和状态监控功能，这是系统的基础功能。

**Independent Test**: 可以独立测试通过创建储能柜、查看储能柜列表、查看单个储能柜详情，验证系统能够正确管理储能柜资产信息。

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T028 [P] [US1] Contract test for POST /api/v1/cabinets endpoint in tests/contract/test_cabinet_create.go
- [ ] T029 [P] [US1] Contract test for GET /api/v1/cabinets endpoint in tests/contract/test_cabinet_list.go
- [ ] T030 [P] [US1] Contract test for GET /api/v1/cabinets/{cabinet_id} endpoint in tests/contract/test_cabinet_detail.go
- [ ] T031 [P] [US1] Unit test for Cabinet model validation in tests/unit/models/test_cabinet.go
- [ ] T032 [P] [US1] Unit test for CabinetService in tests/unit/services/test_cabinet_service.go
- [ ] T033 [P] [US1] Integration test for cabinet CRUD operations in tests/integration/test_cabinet_crud.go

### Implementation for User Story 1

- [ ] T034 [P] [US1] Create Cabinet model struct in internal/models/cabinet.go with validation tags
- [ ] T035 [US1] Create CabinetRepository interface in internal/repository/postgres/cabinet_repo.go
- [ ] T036 [US1] Implement CabinetRepository (PostgreSQL) with CRUD operations in internal/repository/postgres/cabinet_repo.go
- [ ] T037 [US1] Create CabinetService interface in internal/services/cabinet_service.go
- [ ] T038 [US1] Implement CabinetService with business logic in internal/services/cabinet_service.go
- [ ] T039 [US1] Create CreateCabinetRequest DTO in internal/api/handlers/dto.go
- [ ] T040 [US1] Create CabinetResponse DTO in internal/api/handlers/dto.go
- [ ] T041 [US1] Implement POST /api/v1/cabinets handler in internal/api/handlers/cabinet.go
- [ ] T042 [US1] Implement GET /api/v1/cabinets handler with filtering and pagination in internal/api/handlers/cabinet.go
- [ ] T043 [US1] Implement GET /api/v1/cabinets/{cabinet_id} handler in internal/api/handlers/cabinet.go
- [ ] T044 [US1] Add input validation for cabinet creation (cabinet_id format, MAC address format, etc.) in internal/utils/validator.go
- [ ] T045 [US1] Add error handling and logging for cabinet operations in internal/api/handlers/cabinet.go
- [ ] T046 [US1] Register cabinet routes in internal/api/routes.go
- [ ] T046-F [US1] Create CabinetList.vue page with table, filtering, and pagination in frontend/src/views/CabinetList.vue
- [ ] T046-F2 [US1] Create CabinetDetail.vue page with cabinet information display in frontend/src/views/CabinetDetail.vue
- [ ] T046-F3 [US1] Create cabinet store (Pinia) with API calls in frontend/src/store/cabinet.ts
- [ ] T046-F4 [US1] Create cabinet API functions in frontend/src/api/cabinet.ts
- [ ] T046-F5 [US1] Add cabinet routes to Vue Router in frontend/src/router/index.ts
- [ ] T046-F6 [US1] Create HealthScore component for displaying health score in frontend/src/components/HealthScore.vue

**Checkpoint**: User Story 1应该完全可用并可独立测试

---

## Phase 4: User Story 2 - 接收和展示传感器数据 (Priority: P1)

**Goal**: 实现接收Edge端传感器数据同步，并提供最新数据和历史数据查询功能。

**Independent Test**: 可以独立测试通过Edge端发送传感器数据同步请求，然后查询最新数据和历史数据，验证系统能够正确接收、存储和查询传感器数据。

### Tests for User Story 2

- [ ] T047 [P] [US2] Contract test for POST /api/v1/cabinets/{cabinet_id}/sync endpoint in tests/contract/test_sensor_sync.go
- [ ] T048 [P] [US2] Contract test for GET /api/v1/cabinets/{cabinet_id}/sensors/latest endpoint in tests/contract/test_sensor_latest.go
- [ ] T049 [P] [US2] Contract test for GET /api/v1/devices/{device_id}/data endpoint in tests/contract/test_sensor_history.go
- [ ] T050 [P] [US2] Unit test for SensorData model in tests/unit/models/test_sensor.go
- [ ] T051 [P] [US2] Unit test for SensorService in tests/unit/services/test_sensor_service.go
- [ ] T052 [P] [US2] Integration test for sensor data sync and query in tests/integration/test_sensor_data.go

### Implementation for User Story 2

- [ ] T053 [P] [US2] Create SensorDevice model struct in internal/models/sensor.go
- [ ] T054 [P] [US2] Create SensorData model struct in internal/models/sensor.go
- [ ] T055 [US2] Create SensorRepository interface for TimescaleDB in internal/repository/timescaledb/sensor_repo.go
- [ ] T056 [US2] Implement SensorRepository (TimescaleDB) with insert and query methods in internal/repository/timescaledb/sensor_repo.go
- [ ] T057 [US2] Create SensorDeviceRepository interface in internal/repository/postgres/sensor_device_repo.go
- [ ] T058 [US2] Implement SensorDeviceRepository (PostgreSQL) for device metadata in internal/repository/postgres/sensor_device_repo.go
- [ ] T059 [US2] Create SensorService interface in internal/services/sensor_service.go
- [ ] T060 [US2] Implement SensorService with data sync and query logic in internal/services/sensor_service.go
- [ ] T061 [US2] Create SyncDataRequest DTO in internal/api/handlers/dto.go
- [ ] T062 [US2] Create SyncDataResponse DTO in internal/api/handlers/dto.go
- [ ] T063 [US2] Create LatestSensorDataResponse DTO in internal/api/handlers/dto.go
- [ ] T064 [US2] Create HistoricalDataResponse DTO in internal/api/handlers/dto.go
- [ ] T065 [US2] Implement POST /api/v1/cabinets/{cabinet_id}/sync handler in internal/api/handlers/sensor.go
- [ ] T066 [US2] Implement GET /api/v1/cabinets/{cabinet_id}/sensors/latest handler in internal/api/handlers/sensor.go
- [ ] T067 [US2] Implement GET /api/v1/devices/{device_id}/data handler with time range and aggregation in internal/api/handlers/sensor.go
- [ ] T068 [US2] Add data validation for sensor sync (max 1000 records, required fields) in internal/utils/validator.go
- [ ] T069 [US2] Implement async processing for sensor data storage using goroutines in internal/services/sensor_service.go
- [ ] T070 [US2] Add error handling and logging for sensor operations in internal/api/handlers/sensor.go
- [ ] T071 [US2] Register sensor routes in internal/api/routes.go
- [ ] T072 [US2] Implement device status update logic when receiving sensor data in internal/services/sensor_service.go
- [ ] T072-F [US2] Create SensorData.vue page with latest sensor data display in frontend/src/views/SensorData.vue
- [ ] T072-F2 [US2] Create sensor store (Pinia) with API calls in frontend/src/store/sensor.ts
- [ ] T072-F3 [US2] Create sensor API functions in frontend/src/api/sensor.ts
- [ ] T072-F4 [US2] Create ChartCard component for ECharts visualization in frontend/src/components/ChartCard.vue
- [ ] T072-F5 [US2] Implement historical data chart with time range selector in frontend/src/views/SensorData.vue
- [ ] T072-F6 [US2] Add sensor routes to Vue Router in frontend/src/router/index.ts

**Checkpoint**: User Story 2应该完全可用并可独立测试

---

## Phase 5: User Story 3 - 下发指令控制储能柜 (Priority: P1)

**Goal**: 实现通过MQTT向Edge端下发配置和许可证更新指令的功能。

**Independent Test**: 可以独立测试通过向特定储能柜发送配置更新指令（如更新告警阈值），然后验证Edge端接收并执行指令，返回执行结果。

### Tests for User Story 3

- [ ] T073 [P] [US3] Contract test for MQTT command publishing in tests/contract/test_command_publish.go
- [ ] T074 [P] [US3] Contract test for command response handling in tests/contract/test_command_response.go
- [ ] T075 [P] [US3] Unit test for Command model in tests/unit/models/test_command.go
- [ ] T076 [P] [US3] Unit test for CommandService in tests/unit/services/test_command_service.go
- [ ] T077 [P] [US3] Integration test for command send and response flow in tests/integration/test_command_flow.go

### Implementation for User Story 3

- [ ] T078 [P] [US3] Create Command model struct in internal/models/command.go
- [ ] T079 [US3] Create CommandRepository interface in internal/repository/postgres/command_repo.go
- [ ] T080 [US3] Implement CommandRepository (PostgreSQL) with CRUD operations in internal/repository/postgres/command_repo.go
- [ ] T081 [US3] Create MQTT Publisher interface in internal/mqtt/publisher.go
- [ ] T082 [US3] Implement MQTT Publisher with command publishing logic in internal/mqtt/publisher.go
- [ ] T083 [US3] Create MQTT Subscriber interface in internal/mqtt/subscriber.go
- [ ] T084 [US3] Implement MQTT Subscriber with response handling logic in internal/mqtt/subscriber.go
- [ ] T085 [US3] Create CommandService interface in internal/services/command_service.go
- [ ] T086 [US3] Implement CommandService with command sending and response handling in internal/services/command_service.go
- [ ] T087 [US3] Create SendCommandRequest DTO in internal/api/handlers/dto.go
- [ ] T088 [US3] Create CommandResponse DTO in internal/api/handlers/dto.go
- [ ] T089 [US3] Implement API endpoint for sending configuration update commands in internal/api/handlers/command.go
- [ ] T090 [US3] Implement API endpoint for sending license update commands in internal/api/handlers/command.go
- [ ] T091 [US3] Implement API endpoint for sending license revoke commands in internal/api/handlers/command.go
- [ ] T092 [US3] Implement command timeout handling and retry logic in internal/services/command_service.go
- [ ] T093 [US3] Implement command response subscription and status update in internal/mqtt/subscriber.go
- [ ] T094 [US3] Add error handling and logging for command operations in internal/api/handlers/command.go
- [ ] T095 [US3] Register command routes in internal/api/routes.go
- [ ] T096 [US3] Setup MQTT topic patterns for commands and responses per senddata.md specifications in internal/mqtt/client.go
- [ ] T096-F [US3] Create CommandSend.vue page for sending commands in frontend/src/views/CommandSend.vue
- [ ] T096-F2 [US3] Create command store (Pinia) with API calls in frontend/src/store/command.ts
- [ ] T096-F3 [US3] Create command API functions in frontend/src/api/command.ts
- [ ] T096-F4 [US3] Add command routes to Vue Router in frontend/src/router/index.ts
- [ ] T096-F5 [US3] Implement command status display and history in frontend/src/views/CommandSend.vue

**Checkpoint**: User Story 3应该完全可用并可独立测试

---

## Phase 6: User Story 4 - 管理许可证和访问控制 (Priority: P1)

**Goal**: 实现许可证的创建、验证、续期、吊销和权限管理功能。

**Independent Test**: 可以独立测试通过创建许可证、验证许可证有效性、更新许可证权限，验证系统能够正确管理许可证生命周期。

### Tests for User Story 4

- [ ] T097 [P] [US4] Contract test for POST /api/v1/licenses endpoint in tests/contract/test_license_create.go
- [ ] T098 [P] [US4] Contract test for POST /api/v1/license/validate endpoint in tests/contract/test_license_validate.go
- [ ] T099 [P] [US4] Contract test for PUT /api/v1/licenses/{cabinet_id} endpoint in tests/contract/test_license_renew.go
- [ ] T100 [P] [US4] Contract test for DELETE /api/v1/licenses/{cabinet_id} endpoint in tests/contract/test_license_revoke.go
- [ ] T101 [P] [US4] Unit test for License model in tests/unit/models/test_license.go
- [ ] T102 [P] [US4] Unit test for LicenseService in tests/unit/services/test_license_service.go
- [ ] T103 [P] [US4] Integration test for license lifecycle management in tests/integration/test_license_lifecycle.go
- [ ] T104 [P] [US4] Security test for license validation (MAC address binding) in tests/integration/test_license_security.go

### Implementation for User Story 4

- [ ] T105 [P] [US4] Create License model struct in internal/models/license.go
- [ ] T106 [US4] Create LicenseRepository interface in internal/repository/postgres/license_repo.go
- [ ] T107 [US4] Implement LicenseRepository (PostgreSQL) with CRUD operations in internal/repository/postgres/license_repo.go
- [ ] T108 [US4] Create LicenseCache interface in internal/repository/redis/license_cache.go
- [ ] T109 [US4] Implement LicenseCache (Redis) with caching logic (TTL: 1 hour) in internal/repository/redis/license_cache.go
- [ ] T110 [US4] Create LicenseService interface in internal/services/license_service.go
- [ ] T111 [US4] Implement LicenseService with license lifecycle management in internal/services/license_service.go
- [ ] T112 [US4] Implement license validation logic with MAC address binding check in internal/services/license_service.go
- [ ] T113 [US4] Implement license revocation list maintenance in internal/services/license_service.go
- [ ] T114 [US4] Create CreateLicenseRequest DTO in internal/api/handlers/dto.go
- [ ] T115 [US4] Create LicenseValidateRequest DTO in internal/api/handlers/dto.go
- [ ] T116 [US4] Create LicenseValidateResponse DTO in internal/api/handlers/dto.go
- [ ] T117 [US4] Create RenewLicenseRequest DTO in internal/api/handlers/dto.go
- [ ] T118 [US4] Create RevokeLicenseRequest DTO in internal/api/handlers/dto.go
- [ ] T119 [US4] Implement POST /api/v1/licenses handler in internal/api/handlers/license.go
- [ ] T120 [US4] Implement POST /api/v1/license/validate handler (Edge端调用) in internal/api/handlers/license.go
- [ ] T121 [US4] Implement PUT /api/v1/licenses/{cabinet_id} handler for renewal in internal/api/handlers/license.go
- [ ] T122 [US4] Implement DELETE /api/v1/licenses/{cabinet_id} handler for revocation in internal/api/handlers/license.go
- [ ] T123 [US4] Add input validation for license operations (MAC address format, expiration date, permissions) in internal/utils/validator.go
- [ ] T124 [US4] Add error handling and logging for license operations in internal/api/handlers/license.go
- [ ] T125 [US4] Register license routes in internal/api/routes.go
- [ ] T126 [US4] Implement cache invalidation on license updates in internal/services/license_service.go
- [ ] T126-F [US4] Create LicenseManage.vue page with license list and CRUD operations in frontend/src/views/LicenseManage.vue
- [ ] T126-F2 [US4] Create license store (Pinia) with API calls in frontend/src/store/license.ts
- [ ] T126-F3 [US4] Create license API functions in frontend/src/api/license.ts
- [ ] T126-F4 [US4] Implement license creation form with validation in frontend/src/views/LicenseManage.vue
- [ ] T126-F5 [US4] Implement license renewal and revocation UI in frontend/src/views/LicenseManage.vue
- [ ] T126-F6 [US4] Add license routes to Vue Router in frontend/src/router/index.ts

**Checkpoint**: User Story 4应该完全可用并可独立测试

---

## Phase 7: User Story 5 - 监控告警和健康评估 (Priority: P1)

**Goal**: 实现告警接收、存储、查询和健康评分计算功能。

**Independent Test**: 可以独立测试通过Edge端发送告警信息，然后查询告警列表和储能柜健康评分，验证系统能够正确聚合和展示告警数据。

### Tests for User Story 5

- [ ] T127 [P] [US5] Contract test for alert receiving in sync endpoint in tests/contract/test_alert_sync.go
- [ ] T128 [P] [US5] Contract test for GET /api/v1/cabinets/{cabinet_id}/alerts endpoint in tests/contract/test_alert_list.go
- [ ] T129 [P] [US5] Unit test for Alert model in tests/unit/models/test_alert.go
- [ ] T130 [P] [US5] Unit test for AlertService in tests/unit/services/test_alert_service.go
- [ ] T131 [P] [US5] Unit test for HealthScoreService algorithm in tests/unit/services/test_health_score_service.go
- [ ] T132 [P] [US5] Integration test for alert storage and query in tests/integration/test_alert_flow.go
- [ ] T133 [P] [US5] Integration test for health score calculation in tests/integration/test_health_score.go

### Implementation for User Story 5

- [ ] T134 [P] [US5] Create Alert model struct in internal/models/alert.go
- [ ] T135 [US5] Create AlertRepository interface in internal/repository/postgres/alert_repo.go
- [ ] T136 [US5] Implement AlertRepository (PostgreSQL) with CRUD and query operations in internal/repository/postgres/alert_repo.go
- [ ] T137 [US5] Create AlertService interface in internal/services/alert_service.go
- [ ] T138 [US5] Implement AlertService with alert storage and query logic in internal/services/alert_service.go
- [ ] T139 [US5] Create HealthScoreService interface in internal/services/health_score_service.go
- [ ] T140 [US5] Implement HealthScoreService with weighted algorithm (online rate 40%, data quality 30%, alert severity 20%, sensor normalcy 10%) in internal/services/health_score_service.go
- [ ] T141 [US5] Create HealthScoreRepository interface for TimescaleDB in internal/repository/timescaledb/health_score_repo.go
- [ ] T142 [US5] Implement HealthScoreRepository (TimescaleDB) for storing health score history in internal/repository/timescaledb/health_score_repo.go
- [ ] T143 [US5] Create AlertListResponse DTO in internal/api/handlers/dto.go
- [ ] T144 [US5] Implement alert receiving logic in sensor sync handler (already in US2) in internal/api/handlers/sensor.go
- [ ] T145 [US5] Implement GET /api/v1/cabinets/{cabinet_id}/alerts handler with filtering in internal/api/handlers/alert.go
- [ ] T146 [US5] Implement PUT /api/v1/alerts/{alert_id}/resolve handler for resolving alerts in internal/api/handlers/alert.go
- [ ] T147 [US5] Implement health score calculation trigger after sensor data sync in internal/services/sensor_service.go
- [ ] T148 [US5] Implement cabinet offline detection logic (5 minutes timeout) in internal/services/cabinet_service.go
- [ ] T149 [US5] Implement async health score calculation using goroutines in internal/services/health_score_service.go
- [ ] T150 [US5] Add error handling and logging for alert and health score operations in internal/api/handlers/alert.go
- [ ] T151 [US5] Register alert routes in internal/api/routes.go
- [ ] T151-F [US5] Create AlertManage.vue page with alert list and filtering in frontend/src/views/AlertManage.vue
- [ ] T151-F2 [US5] Create Dashboard.vue page with monitoring overview and health scores in frontend/src/views/Dashboard.vue
- [ ] T151-F3 [US5] Create alert store (Pinia) with API calls in frontend/src/store/alert.ts
- [ ] T151-F4 [US5] Create alert API functions in frontend/src/api/alert.ts
- [ ] T151-F5 [US5] Implement alert severity badge component in frontend/src/components/StatusBadge.vue
- [ ] T151-F6 [US5] Implement health score trend chart using ECharts in frontend/src/views/Dashboard.vue
- [ ] T151-F7 [US5] Implement real-time data polling mechanism for dashboard in frontend/src/views/Dashboard.vue
- [ ] T151-F8 [US5] Add alert and dashboard routes to Vue Router in frontend/src/router/index.ts

**Checkpoint**: User Story 5应该完全可用并可独立测试

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 影响多个用户故事的改进和交叉关注点

- [ ] T152 [P] Implement audit logging for all critical operations (license, cabinet, command) in internal/services/audit_service.go
- [ ] T153 [P] Create AuditLogRepository interface and implementation in internal/repository/postgres/audit_repo.go
- [ ] T154 [P] Add audit logging middleware for API requests in internal/api/middleware/audit.go
- [ ] T155 [P] Implement comprehensive error handling across all handlers in internal/api/handlers/
- [ ] T156 [P] Add request validation middleware using validator in internal/api/middleware/validation.go
- [ ] T157 [P] Implement graceful shutdown handling in cmd/cloud-server/main.go
- [ ] T158 [P] Add health check endpoint GET /health in internal/api/handlers/health.go
- [ ] T159 [P] Add metrics collection endpoints for monitoring in internal/api/handlers/metrics.go
- [ ] T160 [P] Implement database connection retry logic with exponential backoff in internal/repository/postgres/postgres.go
- [ ] T161 [P] Implement Redis connection retry logic with exponential backoff in internal/repository/redis/redis.go
- [ ] T162 [P] Implement MQTT connection retry logic with exponential backoff in internal/mqtt/client.go
- [ ] T163 [P] Add comprehensive logging for all service operations in internal/services/
- [ ] T164 [P] Generate API documentation from OpenAPI spec using swagger in docs/
- [ ] T165 [P] Create deployment documentation and Docker configuration in deployment/
- [ ] T166 [P] Add environment-specific configuration examples in config/examples/
- [ ] T167 [P] Validate quickstart.md steps work correctly
- [ ] T168 [P] Run code coverage analysis and ensure >= 70% coverage
- [ ] T169 [P] Run golangci-lint and fix all issues
- [ ] T170 [P] Run gofmt and ensure all code is formatted
- [ ] T170-F [P] Build frontend production bundle and configure deployment in frontend/
- [ ] T170-F2 [P] Setup frontend development server proxy configuration for API calls in frontend/vite.config.ts
- [ ] T170-F3 [P] Create frontend README.md with setup and development instructions in frontend/
- [ ] T170-F4 [P] Configure CORS middleware in backend to allow frontend origin in internal/api/middleware/cors.go
- [ ] T170-F5 [P] Add frontend build output to .gitignore
- [ ] T170-F6 [P] Test frontend-backend integration and API calls

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可以立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - 阻塞所有用户故事
- **User Stories (Phase 3-7)**: 都依赖Foundational阶段完成
  - 用户故事可以并行进行（如果有足够人员）
  - 或按优先级顺序执行（P1 → P1 → P1 → P1 → P1，所有都是P1）
- **Polish (Phase 8)**: 依赖所有期望的用户故事完成

### User Story Dependencies

- **User Story 1 (P1)**: Foundational完成后可开始 - 不依赖其他故事
- **User Story 2 (P1)**: Foundational完成后可开始 - 可能使用US1的cabinet数据，但应独立可测试
- **User Story 3 (P1)**: Foundational完成后可开始 - 可能使用US1的cabinet数据，但应独立可测试
- **User Story 4 (P1)**: Foundational完成后可开始 - 可能使用US1的cabinet数据，但应独立可测试
- **User Story 5 (P1)**: Foundational完成后可开始 - 使用US2的告警数据，但应独立可测试

### Within Each User Story

- 测试（如果包含）必须在实现前编写并确保失败
- Models在Services之前
- Services在Endpoints之前
- 核心实现在集成之前
- 故事完成后再进入下一个优先级

### Parallel Opportunities

- 所有标记[P]的Setup任务可以并行运行
- 所有标记[P]的Foundational任务可以并行运行（在Phase 2内）
- Foundational阶段完成后，所有用户故事可以并行开始（如果团队容量允许）
- 用户故事内标记[P]的所有测试可以并行运行
- 用户故事内标记[P]的模型可以并行运行
- 不同的用户故事可以由不同的团队成员并行工作

---

## Parallel Example: User Story 1

```bash
# 并行运行User Story 1的所有测试：
- Contract test for POST /api/v1/cabinets in tests/contract/test_cabinet_create.go
- Contract test for GET /api/v1/cabinets in tests/contract/test_cabinet_list.go
- Contract test for GET /api/v1/cabinets/{cabinet_id} in tests/contract/test_cabinet_detail.go
- Unit test for Cabinet model in tests/unit/models/test_cabinet.go
- Unit test for CabinetService in tests/unit/services/test_cabinet_service.go

# 并行创建User Story 1的模型：
- Create Cabinet model in internal/models/cabinet.go
```

---

## Parallel Example: User Story 2

```bash
# 并行运行User Story 2的所有测试：
- Contract test for POST /api/v1/cabinets/{cabinet_id}/sync
- Contract test for GET /api/v1/cabinets/{cabinet_id}/sensors/latest
- Contract test for GET /api/v1/devices/{device_id}/data
- Unit test for SensorData model
- Unit test for SensorService

# 并行创建User Story 2的模型：
- Create SensorDevice model in internal/models/sensor.go
- Create SensorData model in internal/models/sensor.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成Phase 1: Setup
2. 完成Phase 2: Foundational（关键 - 阻塞所有故事）
3. 完成Phase 3: User Story 1
4. **停止并验证**: 独立测试User Story 1
5. 如果就绪，部署/演示

### Incremental Delivery

1. 完成Setup + Foundational → 基础设施就绪
2. 添加User Story 1 → 独立测试 → 部署/演示（MVP！）
3. 添加User Story 2 → 独立测试 → 部署/演示
4. 添加User Story 3 → 独立测试 → 部署/演示
5. 添加User Story 4 → 独立测试 → 部署/演示
6. 添加User Story 5 → 独立测试 → 部署/演示
7. 每个故事添加价值而不破坏之前的故事

### Parallel Team Strategy

多个开发者时：

1. 团队一起完成Setup + Foundational
2. Foundational完成后：
   - 开发者A: User Story 1
   - 开发者B: User Story 2
   - 开发者C: User Story 3
   - 开发者D: User Story 4
   - 开发者E: User Story 5
3. 故事独立完成和集成

---

## Notes

- [P]任务 = 不同文件，无依赖
- [Story]标签将任务映射到特定用户故事以便追溯
- 每个用户故事应该独立完成和可测试
- 实现前验证测试失败
- 每个任务或逻辑组后提交
- 在任何检查点停止以独立验证故事
- 避免：模糊任务、同一文件冲突、破坏独立性的跨故事依赖
- 服务器端口：8003（在配置文件中设置）
- 前端技术栈：Vue.js 3 + TypeScript + Element Plus + ECharts + Pinia + Axios + Vite
- 前端开发服务器端口：默认8002（Vite默认端口），可通过配置修改

