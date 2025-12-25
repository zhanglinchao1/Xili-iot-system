# Data Model: Cloud端储能柜集群管理系统

**Feature**: Cloud端储能柜集群管理系统  
**Date**: 2025-11-04  
**Phase**: 1 - Design & Contracts

## Database Schema

### PostgreSQL (关系型数据)

#### 1. Energy Storage Cabinet (储能柜)

**Table**: `cabinets`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| cabinet_id | VARCHAR(50) | PRIMARY KEY | 储能柜唯一标识，也作为Edge端标识 |
| location | VARCHAR(255) | NOT NULL | 部署地点 |
| type | VARCHAR(100) | NOT NULL | 储能柜类型（如"锂电池储能柜"） |
| capacity | DECIMAL(10,2) | NOT NULL | 容量(kWh) |
| mac_address | VARCHAR(17) | NOT NULL, UNIQUE | Edge端设备MAC地址 |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'offline' | 状态: online/offline/maintenance/fault |
| customer_name | VARCHAR(255) | | 客户名称 |
| install_date | DATE | | 安装日期 |
| device_count | INTEGER | DEFAULT 0 | 绑定的设备数量 |
| online_device_count | INTEGER | DEFAULT 0 | 在线设备数量 |
| health_score | DECIMAL(5,2) | DEFAULT 0 | 健康评分(0-100) |
| last_sync_at | TIMESTAMP | | 最后同步时间 |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 更新时间 |

**Indexes**:
- PRIMARY KEY: cabinet_id
- UNIQUE INDEX: mac_address
- INDEX: status, location, customer_name

**Validation Rules**:
- cabinet_id: 必须匹配格式 `^[A-Z0-9-]+$`
- mac_address: 必须匹配MAC地址格式 `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`
- status: 必须是枚举值之一
- capacity: 必须 > 0
- health_score: 必须在 0-100 范围内

#### 2. Sensor Device (传感器设备)

**Table**: `sensor_devices`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| device_id | VARCHAR(50) | PRIMARY KEY | 设备唯一标识 |
| cabinet_id | VARCHAR(50) | NOT NULL, FK(cabinets.cabinet_id) | 所属储能柜 |
| sensor_type | VARCHAR(50) | NOT NULL | 传感器类型: co2/co/smoke/liquid_level/conductivity/temperature/flow |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'offline' | 状态: online/offline/disabled/fault |
| model | VARCHAR(100) | | 设备型号 |
| manufacturer | VARCHAR(100) | | 制造商 |
| firmware_ver | VARCHAR(50) | | 固件版本 |
| last_seen_at | TIMESTAMP | | 最后在线时间 |
| last_synced_at | TIMESTAMP | | 最后同步时间 |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 更新时间 |

**Indexes**:
- PRIMARY KEY: device_id
- INDEX: cabinet_id, sensor_type, status

**Validation Rules**:
- sensor_type: 必须是7种类型之一
- status: 必须是枚举值之一
- cabinet_id: 必须存在于cabinets表

**State Transitions**:
- offline → online: 收到心跳或数据
- online → offline: 超过5分钟无心跳
- online → fault: 检测到异常
- fault → online: 异常恢复

#### 3. License (许可证)

**Table**: `licenses`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| license_id | VARCHAR(50) | PRIMARY KEY | 许可证唯一标识 |
| cabinet_id | VARCHAR(50) | NOT NULL, UNIQUE, FK(cabinets.cabinet_id) | 关联的储能柜 |
| mac_address | VARCHAR(17) | NOT NULL | 绑定的MAC地址 |
| max_devices | INTEGER | DEFAULT -1 | 最大设备数限制(-1=无限制) |
| expires_at | TIMESTAMP | NOT NULL | 过期时间 |
| grace_period | INTEGER | DEFAULT 72 | 宽限期(小时) |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'active' | 状态: active/suspended/expired/revoked |
| permissions | JSONB | NOT NULL | 权限列表（数组） |
| customer_name | VARCHAR(255) | | 客户名称 |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 更新时间 |
| revoked_at | TIMESTAMP | | 吊销时间 |
| revoked_reason | TEXT | | 吊销原因 |

**Indexes**:
- PRIMARY KEY: license_id
- UNIQUE INDEX: cabinet_id
- INDEX: status, expires_at, mac_address

**Validation Rules**:
- cabinet_id 和 mac_address 必须匹配
- expires_at 必须 > created_at
- permissions 必须是有效的JSON数组
- status: 必须是枚举值之一

**State Transitions**:
- active → expired: 过期时间到达
- active → suspended: 管理员暂停
- active → revoked: 管理员吊销
- suspended → active: 管理员恢复
- expired → active: 管理员续期

#### 4. Alert (告警)

**Table**: `alerts`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| alert_id | BIGSERIAL | PRIMARY KEY | 告警唯一标识 |
| cabinet_id | VARCHAR(50) | NOT NULL, FK(cabinets.cabinet_id) | 关联的储能柜 |
| device_id | VARCHAR(50) | FK(sensor_devices.device_id) | 关联的设备 |
| alert_type | VARCHAR(50) | NOT NULL | 告警类型（12种之一） |
| severity | VARCHAR(20) | NOT NULL | 严重程度: info/warning/error/critical |
| message | TEXT | NOT NULL | 告警消息 |
| value | DECIMAL(10,2) | | 触发值 |
| threshold | DECIMAL(10,2) | | 阈值 |
| timestamp | TIMESTAMP | NOT NULL, DEFAULT NOW() | 告警时间 |
| resolved | BOOLEAN | NOT NULL, DEFAULT FALSE | 是否已解决 |
| resolved_at | TIMESTAMP | | 解决时间 |
| resolved_by | VARCHAR(100) | | 解决操作员 |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |

**Indexes**:
- PRIMARY KEY: alert_id
- INDEX: cabinet_id, device_id, severity, resolved, timestamp

**Validation Rules**:
- alert_type: 必须是12种类型之一
- severity: 必须是枚举值之一
- cabinet_id: 必须存在于cabinets表

**State Transitions**:
- resolved: false → true: 管理员标记为已解决

#### 5. Command (指令)

**Table**: `commands`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| command_id | VARCHAR(50) | PRIMARY KEY | 指令唯一标识 |
| cabinet_id | VARCHAR(50) | NOT NULL, FK(cabinets.cabinet_id) | 目标储能柜 |
| command_type | VARCHAR(50) | NOT NULL | 指令类型 |
| parameters | JSONB | NOT NULL | 指令参数 |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'pending' | 状态: pending/sent/acknowledged/failed/timeout |
| timeout | INTEGER | DEFAULT 30 | 超时时间(秒) |
| retry | BOOLEAN | DEFAULT FALSE | 是否允许重试 |
| sent_at | TIMESTAMP | | 发送时间 |
| acknowledged_at | TIMESTAMP | | 确认时间 |
| result | JSONB | | 执行结果 |
| error_message | TEXT | | 错误消息 |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | 更新时间 |

**Indexes**:
- PRIMARY KEY: command_id
- INDEX: cabinet_id, command_type, status, created_at

**Validation Rules**:
- command_type: 必须是有效的指令类型
- status: 必须是枚举值之一
- timeout: 必须 > 0

**State Transitions**:
- pending → sent: 发送到MQTT
- sent → acknowledged: 收到Edge端确认
- sent → failed: 执行失败
- sent → timeout: 超时

#### 6. Audit Log (审计日志)

**Table**: `audit_logs`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| log_id | BIGSERIAL | PRIMARY KEY | 日志唯一标识 |
| operator | VARCHAR(100) | NOT NULL | 操作员 |
| action | VARCHAR(100) | NOT NULL | 操作类型 |
| resource_type | VARCHAR(50) | NOT NULL | 资源类型 |
| resource_id | VARCHAR(50) | | 资源ID |
| result | VARCHAR(20) | NOT NULL | 结果: success/failure |
| details | JSONB | | 详细信息 |
| ip_address | VARCHAR(45) | | IP地址 |
| timestamp | TIMESTAMP | NOT NULL, DEFAULT NOW() | 操作时间 |

**Indexes**:
- PRIMARY KEY: log_id
- INDEX: operator, action, resource_type, timestamp

**Validation Rules**:
- result: 必须是 success 或 failure
- operator: 不能为空

### TimescaleDB (时序数据)

#### 1. Sensor Data (传感器数据)

**Table**: `sensor_data` (Hypertable)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| time | TIMESTAMP | NOT NULL | 时间戳（TimescaleDB要求） |
| device_id | VARCHAR(50) | NOT NULL | 设备ID |
| cabinet_id | VARCHAR(50) | NOT NULL | 储能柜ID |
| sensor_type | VARCHAR(50) | NOT NULL | 传感器类型 |
| value | DECIMAL(10,2) | NOT NULL | 数值 |
| unit | VARCHAR(20) | NOT NULL | 单位 |
| quality | INTEGER | NOT NULL, DEFAULT 100 | 质量指标(0-100) |
| synced | BOOLEAN | NOT NULL, DEFAULT FALSE | 是否已同步 |

**Indexes**:
- PRIMARY KEY: (time, device_id)
- INDEX: device_id, cabinet_id, sensor_type, time DESC

**Partitioning**:
- Hypertable按时间分区（1天）

**Retention Policy**:
- 保留90天数据（根据需求调整）

#### 2. Health Score History (健康评分历史)

**Table**: `health_scores` (Hypertable)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| time | TIMESTAMP | NOT NULL | 时间戳 |
| cabinet_id | VARCHAR(50) | NOT NULL | 储能柜ID |
| score | DECIMAL(5,2) | NOT NULL | 健康评分 |
| device_online_rate | DECIMAL(5,2) | | 设备在线率 |
| data_quality | DECIMAL(5,2) | | 数据质量 |
| alert_severity | DECIMAL(5,2) | | 告警严重度 |
| sensor_normalcy | DECIMAL(5,2) | | 传感器正常率 |

**Indexes**:
- PRIMARY KEY: (time, cabinet_id)
- INDEX: cabinet_id, time DESC

**Partitioning**:
- Hypertable按时间分区（7天）

## Entity Relationships

```
Energy Storage Cabinet (1) ──< (N) Sensor Device
Energy Storage Cabinet (1) ──< (1) License
Energy Storage Cabinet (1) ──< (N) Alert
Energy Storage Cabinet (1) ──< (N) Command
Energy Storage Cabinet (1) ──< (N) Sensor Data (时序)
Energy Storage Cabinet (1) ──< (N) Health Score (时序)
Sensor Device (1) ──< (N) Sensor Data (时序)
Sensor Device (1) ──< (N) Alert
```

## Data Validation Rules

### Cabinet ID
- 格式: `^[A-Z0-9-]+$`
- 长度: 3-50字符
- 唯一性: 全局唯一

### MAC Address
- 格式: `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`
- 唯一性: 全局唯一

### Sensor Type
- 枚举值: co2, co, smoke, liquid_level, conductivity, temperature, flow
- 必须匹配7种类型之一

### Alert Type
- 枚举值: co2_high, co_high, smoke_detected, liquid_level_low, liquid_level_high, conductivity_abnormal, temperature_high, temperature_low, flow_abnormal, device_offline, auth_failed, data_abnormal

### Status Values
- Cabinet: online, offline, maintenance, fault
- Device: online, offline, disabled, fault
- License: active, suspended, expired, revoked
- Command: pending, sent, acknowledged, failed, timeout
- Alert: resolved (boolean)

## Data Migration Strategy

### Initial Schema
1. 创建PostgreSQL数据库
2. 创建TimescaleDB扩展
3. 创建所有表结构
4. 创建索引
5. 创建Hypertables

### Migration Scripts
- 使用Go migrate或类似工具管理数据库迁移
- 每个迁移文件包含版本号和描述
- 支持回滚

## Data Retention Policy

### PostgreSQL
- 审计日志: 保留1年
- 告警: 保留6个月（已解决的）
- 指令: 保留3个月

### TimescaleDB
- 传感器数据: 保留90天
- 健康评分: 保留180天

超过保留期的数据自动删除（使用TimescaleDB的retention policy）。

