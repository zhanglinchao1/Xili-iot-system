-- 016_create_abac_tables.sql
-- 创建ABAC访问控制相关表

-- 1. 访问策略表
CREATE TABLE IF NOT EXISTS access_policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    subject_type TEXT NOT NULL, -- user, cabinet, device
    conditions JSONB NOT NULL,   -- 策略条件(JSON格式)
    permissions JSONB NOT NULL,  -- 权限列表(JSON数组)
    priority INTEGER DEFAULT 50,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加注释
COMMENT ON TABLE access_policies IS 'ABAC访问策略表';
COMMENT ON COLUMN access_policies.id IS '策略唯一标识';
COMMENT ON COLUMN access_policies.subject_type IS '主体类型: user/cabinet/device';
COMMENT ON COLUMN access_policies.conditions IS '策略条件列表(JSONB格式)';
COMMENT ON COLUMN access_policies.permissions IS '授予的权限列表(JSONB格式)';
COMMENT ON COLUMN access_policies.priority IS '优先级,数值越大优先级越高';

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_policies_subject_type ON access_policies(subject_type);
CREATE INDEX IF NOT EXISTS idx_policies_enabled ON access_policies(enabled);
CREATE INDEX IF NOT EXISTS idx_policies_priority ON access_policies(priority DESC);

-- 2. 访问日志表
CREATE TABLE IF NOT EXISTS access_logs (
    id SERIAL PRIMARY KEY,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    allowed BOOLEAN NOT NULL,
    policy_id TEXT,
    trust_score FLOAT,
    ip_address TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    attributes JSONB
);

-- 添加注释
COMMENT ON TABLE access_logs IS 'ABAC访问日志表,用于审计';
COMMENT ON COLUMN access_logs.subject_type IS '主体类型: user/cabinet/device';
COMMENT ON COLUMN access_logs.subject_id IS '主体ID';
COMMENT ON COLUMN access_logs.resource IS '访问的资源路径';
COMMENT ON COLUMN access_logs.action IS 'HTTP方法: GET/POST/PUT/DELETE';
COMMENT ON COLUMN access_logs.allowed IS '是否允许访问';
COMMENT ON COLUMN access_logs.policy_id IS '匹配的策略ID';
COMMENT ON COLUMN access_logs.trust_score IS '信任度分数(0-100)';
COMMENT ON COLUMN access_logs.attributes IS '主体属性快照(JSONB格式)';

-- 索引优化(用于日志查询和审计)
CREATE INDEX IF NOT EXISTS idx_logs_subject ON access_logs(subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON access_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_resource ON access_logs(resource);
CREATE INDEX IF NOT EXISTS idx_logs_allowed ON access_logs(allowed);
CREATE INDEX IF NOT EXISTS idx_logs_policy_id ON access_logs(policy_id);

-- 3. 插入预定义策略
INSERT INTO access_policies (id, name, description, subject_type, conditions, permissions, priority, enabled)
VALUES
    -- 策略1: 管理员完全访问
    (
        'policy_admin_full',
        '管理员完全访问',
        '管理员拥有所有权限',
        'user',
        '[
            {"attribute": "role", "operator": "eq", "value": "admin"},
            {"attribute": "status", "operator": "eq", "value": "active"}
        ]'::jsonb,
        '["*"]'::jsonb,
        100,
        true
    ),
    -- 策略2: 普通用户只读访问
    (
        'policy_user_readonly',
        '普通用户只读访问',
        '普通用户只能读取数据',
        'user',
        '[
            {"attribute": "role", "operator": "eq", "value": "user"},
            {"attribute": "status", "operator": "eq", "value": "active"}
        ]'::jsonb,
        '["read:cabinets", "read:sensors", "read:alerts", "read:devices", "read:licenses"]'::jsonb,
        50,
        true
    ),
    -- 策略3: 已激活储能柜数据同步
    (
        'policy_cabinet_sync',
        '已激活储能柜数据同步',
        '健康且已激活的储能柜可以同步数据',
        'cabinet',
        '[
            {"attribute": "activation_status", "operator": "eq", "value": "activated"},
            {"attribute": "status", "operator": "in", "value": ["online", "maintenance"]},
            {"attribute": "trust_score", "operator": "gte", "value": 30}
        ]'::jsonb,
        '["write:sensor_data", "write:alerts", "write:vulnerability", "write:sync", "read:commands", "write:heartbeat", "*"]'::jsonb,
        80,
        true
    ),
    -- 策略4: 低信任度储能柜受限访问
    (
        'policy_cabinet_limited',
        '低信任度储能柜受限访问',
        '信任度较低的储能柜仅可上传传感器数据',
        'cabinet',
        '[
            {"attribute": "activation_status", "operator": "eq", "value": "activated"},
            {"attribute": "trust_score", "operator": "lt", "value": 30}
        ]'::jsonb,
        '["write:sensor_data"]'::jsonb,
        60,
        true
    ),
    -- 策略5: 高质量传感器完全数据上传
    (
        'policy_device_high_quality',
        '高质量传感器完全数据上传',
        '数据质量高的传感器可以上传所有数据',
        'device',
        '[
            {"attribute": "status", "operator": "eq", "value": "active"},
            {"attribute": "quality", "operator": "gte", "value": 80}
        ]'::jsonb,
        '["write:sensor_data", "trigger:alert"]'::jsonb,
        70,
        true
    )
ON CONFLICT (id) DO NOTHING;

-- 成功提示
DO $$
BEGIN
    RAISE NOTICE '✅ ABAC表创建成功: access_policies, access_logs';
    RAISE NOTICE '✅ 已插入5条预定义策略';
END $$;
