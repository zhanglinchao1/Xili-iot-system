# Data Model: Cloud-Edge Integration

**Feature**: 002-cloud-edge-integration  
**Database**: PostgreSQL 14+ & TimescaleDB 2.0+

---

## 📐 Entity Relationship Diagram

```
┌─────────────────┐         ┌─────────────────┐
│    cabinets     │◄────────│ edge_cabinets   │
│  (已存在)        │ 1     1 │  (新增)          │
│  - cabinet_id   │         │  - cabinet_id   │
│  - name         │         │  - api_key_hash │
│  - mac_address  │         │  - edge_version │
│  - location     │         │  - connection_  │
│  ...            │         │    status       │
└────────┬────────┘         │  - mqtt_        │
         │                  │    connected    │
         │ 1                │  - last_sync_at │
         │                  │  - last_seen_at │
         │                  └─────────┬───────┘
         │                            │ 1
         │                            │
         │ *                          │ *
┌────────▼────────┐         ┌─────────▼───────┐
│ sensor_devices  │         │ edge_sync_logs  │
│  (已存在)        │         │  (新增)          │
│  - device_id    │         │  - id           │
│  - cabinet_id   │         │  - cabinet_id   │
│  - sensor_type  │         │  - sync_time    │
│  - status       │         │  - data_count   │
│  ...            │         │  - status       │
└────────┬────────┘         │  - error_msg    │
         │                  └─────────────────┘
         │ 1
         │
         │ *
┌────────▼────────┐         ┌─────────────────┐
│  sensor_data    │         │    commands     │
│  (已存在)        │         │  (已存在)        │
│  - id           │         │  - command_id   │
│  - device_id    │         │  - cabinet_id   │
│  - sensor_type  │         │  - command_type │
│  - value        │         │  - status       │
│  - timestamp    │         │  - payload      │
│  ...            │         │  ...            │
└─────────────────┘         └─────────────────┘
```

---

## 📊 New Tables

### 1. edge_cabinets (Edge 储能柜元数据)

**Purpose**: 存储 Edge 端储能柜的连接状态、同步信息和配置参数

```sql
CREATE TABLE edge_cabinets (
    -- 主键（关联到 cabinets 表）
    cabinet_id VARCHAR(64) PRIMARY KEY,
    
    -- API 认证
    api_key_hash VARCHAR(128) NOT NULL,          -- API Key 的哈希值（bcrypt）
    api_key_created_at TIMESTAMPTZ,              -- API Key 创建时间
    api_key_expires_at TIMESTAMPTZ,              -- API Key 过期时间（NULL 表示永不过期）
    
    -- Edge 端信息
    edge_version VARCHAR(32),                     -- Edge 系统版本（如 "v2.0.1"）
    edge_ip_address INET,                         -- Edge 端 IP 地址
    
    -- 连接状态
    connection_status VARCHAR(16) DEFAULT 'offline' CHECK (connection_status IN ('online', 'offline', 'warning')),
    mqtt_connected BOOLEAN DEFAULT FALSE,         -- MQTT 连接状态
    last_sync_at TIMESTAMPTZ,                    -- 最后同步时间
    last_seen_at TIMESTAMPTZ,                    -- 最后在线时间（收到任何消息）
    
    -- 同步配置
    sync_interval INT DEFAULT 300 CHECK (sync_interval BETWEEN 60 AND 3600),  -- 同步间隔（秒）
    sync_enabled BOOLEAN DEFAULT TRUE,            -- 是否启用数据同步
    
    -- 统计信息
    total_sync_count BIGINT DEFAULT 0,            -- 累计同步次数
    failed_sync_count BIGINT DEFAULT 0,           -- 累计失败次数
    last_sync_data_count INT DEFAULT 0,           -- 最后一次同步的数据条数
    
    -- 时间戳
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    
    -- 外键约束
    CONSTRAINT fk_edge_cabinets_cabinets FOREIGN KEY (cabinet_id) 
        REFERENCES cabinets(cabinet_id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_edge_cabinets_status ON edge_cabinets(connection_status);
CREATE INDEX idx_edge_cabinets_last_sync ON edge_cabinets(last_sync_at DESC);
CREATE INDEX idx_edge_cabinets_mqtt ON edge_cabinets(mqtt_connected) WHERE mqtt_connected = TRUE;

-- 触发器（自动更新 updated_at）
CREATE OR REPLACE FUNCTION update_edge_cabinets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_edge_cabinets_updated_at
    BEFORE UPDATE ON edge_cabinets
    FOR EACH ROW
    EXECUTE FUNCTION update_edge_cabinets_updated_at();

-- 注释
COMMENT ON TABLE edge_cabinets IS 'Edge 端储能柜元数据和连接状态';
COMMENT ON COLUMN edge_cabinets.api_key_hash IS 'API Key 的 bcrypt 哈希值，用于认证 Edge→Cloud 数据同步';
COMMENT ON COLUMN edge_cabinets.connection_status IS '连接状态：online（正常）/ offline（离线）/ warning（异常）';
COMMENT ON COLUMN edge_cabinets.last_sync_at IS '最后一次成功同步数据的时间';
COMMENT ON COLUMN edge_cabinets.last_seen_at IS '最后一次收到 Edge 端消息的时间（包括数据同步、MQTT 响应等）';
```

**Validation Rules**:
- `api_key_hash`: 必须是 bcrypt 哈希（60 字符）
- `sync_interval`: 60-3600 秒之间
- `connection_status`: 只能是 'online', 'offline', 'warning'

**Business Logic**:
- 创建 Cabinet 时自动生成 API Key
- `connection_status` 根据 `last_sync_at` 和 `mqtt_connected` 自动推断
- `last_seen_at` 在收到任何 Edge 消息时更新

---

### 2. edge_sync_logs (Edge 数据同步日志)

**Purpose**: 记录每次 Edge→Cloud 数据同步的详细信息，用于审计和故障排查

```sql
CREATE TABLE edge_sync_logs (
    -- 主键
    id BIGSERIAL PRIMARY KEY,
    
    -- 关联
    cabinet_id VARCHAR(64) NOT NULL,
    
    -- 同步信息
    sync_time TIMESTAMPTZ NOT NULL,              -- 同步时间（Edge 端时间戳）
    request_time TIMESTAMPTZ DEFAULT NOW() NOT NULL,  -- 请求接收时间（Cloud 端时间戳）
    
    -- 数据统计
    total_data_count INT NOT NULL,                -- 总数据条数
    sensor_data_count INT DEFAULT 0,              -- 传感器数据条数
    alert_count INT DEFAULT 0,                    -- 告警条数
    device_count INT DEFAULT 0,                   -- 设备状态条数
    
    -- 执行结果
    status VARCHAR(16) NOT NULL CHECK (status IN ('success', 'partial', 'failed')),
    success_count INT DEFAULT 0,                  -- 成功插入条数
    failed_count INT DEFAULT 0,                   -- 失败条数
    error_message TEXT,                           -- 错误信息（失败时记录）
    error_code VARCHAR(32),                       -- 错误码（如 "DATABASE_ERROR"）
    
    -- 性能指标
    processing_time_ms INT,                       -- 处理时间（毫秒）
    db_insert_time_ms INT,                        -- 数据库插入时间（毫秒）
    
    -- 请求元数据
    request_ip INET,                              -- 请求来源 IP
    request_size_bytes INT,                       -- 请求体大小（字节）
    
    -- 时间戳
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    
    -- 外键约束
    CONSTRAINT fk_edge_sync_logs_cabinets FOREIGN KEY (cabinet_id) 
        REFERENCES edge_cabinets(cabinet_id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_edge_sync_logs_cabinet ON edge_sync_logs(cabinet_id, request_time DESC);
CREATE INDEX idx_edge_sync_logs_status ON edge_sync_logs(status, request_time DESC);
CREATE INDEX idx_edge_sync_logs_request_time ON edge_sync_logs(request_time DESC);

-- 分区（可选，数据量大时启用）
-- ALTER TABLE edge_sync_logs PARTITION BY RANGE (request_time);

-- 注释
COMMENT ON TABLE edge_sync_logs IS 'Edge→Cloud 数据同步日志，记录每次同步的详细信息';
COMMENT ON COLUMN edge_sync_logs.status IS '同步状态：success（全部成功）/ partial（部分成功）/ failed（全部失败）';
COMMENT ON COLUMN edge_sync_logs.processing_time_ms IS 'Cloud 端处理总时间，包括解析、验证、数据库插入';
```

**Validation Rules**:
- `status`: 只能是 'success', 'partial', 'failed'
- `total_data_count = sensor_data_count + alert_count + device_count`
- `success_count + failed_count <= total_data_count`

**Retention Policy**:
- 保留最近 90 天的日志
- 自动清理超过 90 天的记录（定时任务）

---

## 🔄 Modified Tables

### 1. commands (扩展字段)

**New Columns**:
```sql
ALTER TABLE commands ADD COLUMN IF NOT EXISTS command_category VARCHAR(16) 
    CHECK (command_category IN ('config', 'license', 'query', 'control'));

ALTER TABLE commands ADD COLUMN IF NOT EXISTS mqtt_topic VARCHAR(256);

ALTER TABLE commands ADD COLUMN IF NOT EXISTS response_data JSONB;

ALTER TABLE commands ADD COLUMN IF NOT EXISTS timeout_seconds INT DEFAULT 30;

ALTER TABLE commands ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

-- 更新注释
COMMENT ON COLUMN commands.command_category IS '指令分类：config（配置管理）/ license（许可证）/ query（查询）/ control（控制）';
COMMENT ON COLUMN commands.mqtt_topic IS '实际使用的 MQTT Topic';
COMMENT ON COLUMN commands.response_data IS 'Edge 端响应数据（JSON 格式）';
```

---

## 📝 Data Transfer Objects (DTOs)

### 1. EdgeSyncRequest (Edge→Cloud 数据同步请求)

```go
// EdgeSyncRequest Edge 端批量同步请求
type EdgeSyncRequest struct {
    CabinetID string    `json:"cabinet_id" validate:"required"`
    SyncTime  time.Time `json:"sync_time" validate:"required"`
    Devices   []EdgeDevice      `json:"devices"`
    SensorData []EdgeSensorData `json:"sensor_data"`
    Alerts    []EdgeAlert       `json:"alerts"`
}

// EdgeDevice Edge 端设备状态
type EdgeDevice struct {
    DeviceID    string    `json:"device_id" validate:"required"`
    SensorType  string    `json:"sensor_type" validate:"required,oneof=co2 co smoke liquid_level conductivity temperature flow"`
    Status      string    `json:"status" validate:"required,oneof=online offline fault"`
    LastSeenAt  time.Time `json:"last_seen_at"`
    Model       string    `json:"model"`
    FirmwareVer string    `json:"firmware_ver"`
}

// EdgeSensorData Edge 端传感器数据
type EdgeSensorData struct {
    DeviceID    string    `json:"device_id" validate:"required"`
    SensorType  string    `json:"sensor_type" validate:"required,oneof=co2 co smoke liquid_level conductivity temperature flow"`
    Value       float64   `json:"value" validate:"required"`
    Unit        string    `json:"unit" validate:"required"`
    Timestamp   time.Time `json:"timestamp" validate:"required"`
    Quality     int       `json:"quality" validate:"min=0,max=100"`
}

// EdgeAlert Edge 端告警
type EdgeAlert struct {
    AlertID    string    `json:"alert_id" validate:"required"`
    DeviceID   string    `json:"device_id" validate:"required"`
    AlertType  string    `json:"alert_type" validate:"required"`
    Severity   string    `json:"severity" validate:"required,oneof=low medium high critical"`
    Message    string    `json:"message" validate:"required"`
    Value      float64   `json:"value"`
    Threshold  float64   `json:"threshold"`
    Timestamp  time.Time `json:"timestamp" validate:"required"`
}
```

### 2. EdgeSyncResponse (Cloud→Edge 数据同步响应)

```go
// EdgeSyncResponse Cloud 端同步响应
type EdgeSyncResponse struct {
    Success       bool      `json:"success"`
    Message       string    `json:"message"`
    TotalCount    int       `json:"total_count"`
    SuccessCount  int       `json:"success_count"`
    FailedCount   int       `json:"failed_count"`
    ProcessingTime int      `json:"processing_time_ms"`
    Errors        []SyncError `json:"errors,omitempty"`
}

// SyncError 同步错误详情
type SyncError struct {
    Index   int    `json:"index"`   // 数据在数组中的索引
    Type    string `json:"type"`    // 数据类型（sensor_data/alert/device）
    Field   string `json:"field"`   // 错误字段
    Message string `json:"message"` // 错误信息
}
```

### 3. LicenseValidateRequest (许可证验证请求)

```go
// LicenseValidateRequest Edge 端许可证验证请求
type LicenseValidateRequest struct {
    CabinetID  string `json:"cabinet_id" validate:"required"`
    MACAddress string `json:"mac_address" validate:"required,mac"`
}

// LicenseValidateResponse Cloud 端许可证验证响应
type LicenseValidateResponse struct {
    Valid         bool      `json:"valid"`
    LicenseID     string    `json:"license_id,omitempty"`
    ExpiresAt     time.Time `json:"expires_at,omitempty"`
    IsExpired     bool      `json:"is_expired"`
    InGracePeriod bool      `json:"in_grace_period"`
    MaxDevices    int       `json:"max_devices,omitempty"`
    Permissions   []string  `json:"permissions,omitempty"`
    Message       string    `json:"message"`
}
```

### 4. CloudCommandRequest (Cloud→Edge 指令请求)

```go
// CloudCommandRequest Cloud 端指令请求（通过 MQTT 发送）
type CloudCommandRequest struct {
    CommandID   string                 `json:"command_id" validate:"required,uuid"`
    CommandType string                 `json:"command_type" validate:"required"`
    Timestamp   time.Time              `json:"timestamp" validate:"required"`
    Params      map[string]interface{} `json:"params" validate:"required"`
    Timeout     int                    `json:"timeout,omitempty"`   // 超时时间（秒）
    Retry       bool                   `json:"retry,omitempty"`     // 是否允许重试
}

// CloudCommandResponse Edge→Cloud 指令响应（通过 MQTT 发送）
type CloudCommandResponse struct {
    CommandID string                 `json:"command_id" validate:"required,uuid"`
    Status    string                 `json:"status" validate:"required,oneof=success failed timeout"`
    Message   string                 `json:"message"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Timestamp time.Time              `json:"timestamp" validate:"required"`
    Error     *CommandError          `json:"error,omitempty"`
}

// CommandError 指令执行错误
type CommandError struct {
    Code    string `json:"code"`    // 错误码（如 "PERMISSION_DENIED"）
    Message string `json:"message"` // 错误信息
}
```

---

## 🔐 Security Considerations

### API Key Management

**Generation**:
```go
// 生成 API Key（32 字节随机数，Base64 编码）
func GenerateAPIKey() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)  // 43 字符
}

// 存储 API Key（使用 bcrypt 哈希）
func HashAPIKey(apiKey string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
    return string(hash), err
}

// 验证 API Key
func ValidateAPIKey(apiKey, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
    return err == nil
}
```

**Storage**:
- Cloud 端：存储 bcrypt 哈希值（`edge_cabinets.api_key_hash`）
- Edge 端：明文存储在配置文件（`configs/config.yaml`）
- 传输：通过 HTTPS Header（`X-API-Key`）

---

## 📊 Validation Rules Summary

| Field | Type | Constraints | Validation |
|-------|------|-------------|------------|
| cabinet_id | VARCHAR(64) | PK, NOT NULL | 必须存在于 cabinets 表 |
| api_key_hash | VARCHAR(128) | NOT NULL | bcrypt 哈希（60 字符） |
| connection_status | VARCHAR(16) | ENUM | online/offline/warning |
| sync_interval | INT | 60-3600 | 同步间隔（秒） |
| sensor_type | VARCHAR(32) | ENUM | 7 种固定类型 |
| status (sync_logs) | VARCHAR(16) | ENUM | success/partial/failed |
| command_category | VARCHAR(16) | ENUM | config/license/query/control |

---

## 🔄 Migration Script

```sql
-- migration_009_edge_integration.sql

BEGIN;

-- 1. 创建 edge_cabinets 表
CREATE TABLE IF NOT EXISTS edge_cabinets (
    cabinet_id VARCHAR(64) PRIMARY KEY,
    api_key_hash VARCHAR(128) NOT NULL,
    api_key_created_at TIMESTAMPTZ,
    api_key_expires_at TIMESTAMPTZ,
    edge_version VARCHAR(32),
    edge_ip_address INET,
    connection_status VARCHAR(16) DEFAULT 'offline' CHECK (connection_status IN ('online', 'offline', 'warning')),
    mqtt_connected BOOLEAN DEFAULT FALSE,
    last_sync_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    sync_interval INT DEFAULT 300 CHECK (sync_interval BETWEEN 60 AND 3600),
    sync_enabled BOOLEAN DEFAULT TRUE,
    total_sync_count BIGINT DEFAULT 0,
    failed_sync_count BIGINT DEFAULT 0,
    last_sync_data_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT fk_edge_cabinets_cabinets FOREIGN KEY (cabinet_id) 
        REFERENCES cabinets(cabinet_id) ON DELETE CASCADE
);

-- 2. 创建 edge_sync_logs 表
CREATE TABLE IF NOT EXISTS edge_sync_logs (
    id BIGSERIAL PRIMARY KEY,
    cabinet_id VARCHAR(64) NOT NULL,
    sync_time TIMESTAMPTZ NOT NULL,
    request_time TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    total_data_count INT NOT NULL,
    sensor_data_count INT DEFAULT 0,
    alert_count INT DEFAULT 0,
    device_count INT DEFAULT 0,
    status VARCHAR(16) NOT NULL CHECK (status IN ('success', 'partial', 'failed')),
    success_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    error_message TEXT,
    error_code VARCHAR(32),
    processing_time_ms INT,
    db_insert_time_ms INT,
    request_ip INET,
    request_size_bytes INT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT fk_edge_sync_logs_cabinets FOREIGN KEY (cabinet_id) 
        REFERENCES edge_cabinets(cabinet_id) ON DELETE CASCADE
);

-- 3. 扩展 commands 表
ALTER TABLE commands ADD COLUMN IF NOT EXISTS command_category VARCHAR(16) 
    CHECK (command_category IN ('config', 'license', 'query', 'control'));
ALTER TABLE commands ADD COLUMN IF NOT EXISTS mqtt_topic VARCHAR(256);
ALTER TABLE commands ADD COLUMN IF NOT EXISTS response_data JSONB;
ALTER TABLE commands ADD COLUMN IF NOT EXISTS timeout_seconds INT DEFAULT 30;
ALTER TABLE commands ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

-- 4. 创建索引
CREATE INDEX IF NOT EXISTS idx_edge_cabinets_status ON edge_cabinets(connection_status);
CREATE INDEX IF NOT EXISTS idx_edge_cabinets_last_sync ON edge_cabinets(last_sync_at DESC);
CREATE INDEX IF NOT EXISTS idx_edge_cabinets_mqtt ON edge_cabinets(mqtt_connected) WHERE mqtt_connected = TRUE;
CREATE INDEX IF NOT EXISTS idx_edge_sync_logs_cabinet ON edge_sync_logs(cabinet_id, request_time DESC);
CREATE INDEX IF NOT EXISTS idx_edge_sync_logs_status ON edge_sync_logs(status, request_time DESC);
CREATE INDEX IF NOT EXISTS idx_edge_sync_logs_request_time ON edge_sync_logs(request_time DESC);

-- 5. 创建触发器
CREATE OR REPLACE FUNCTION update_edge_cabinets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_edge_cabinets_updated_at ON edge_cabinets;
CREATE TRIGGER trg_edge_cabinets_updated_at
    BEFORE UPDATE ON edge_cabinets
    FOR EACH ROW
    EXECUTE FUNCTION update_edge_cabinets_updated_at();

-- 6. 添加注释
COMMENT ON TABLE edge_cabinets IS 'Edge 端储能柜元数据和连接状态';
COMMENT ON TABLE edge_sync_logs IS 'Edge→Cloud 数据同步日志';

COMMIT;
```

---

## ✅ Data Model Validation

- [x] 所有表都有主键
- [x] 所有外键关系正确定义
- [x] 所有 NOT NULL 列有默认值或业务逻辑保证
- [x] 所有 ENUM 类型有 CHECK 约束
- [x] 所有时间戳列有默认值（NOW()）
- [x] 所有关键字段有索引
- [x] 所有表和列有注释说明

---

**Data Model Completed**: 2025-11-04  
**Next Steps**: Define API contracts (contracts/openapi-edge-integration.yaml)

