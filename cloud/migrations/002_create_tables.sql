-- Cloud端储能柜集群管理系统数据库迁移
-- 创建关系型数据表

-- 储能柜表
CREATE TABLE IF NOT EXISTS cabinets (
    cabinet_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(200),
    capacity_kwh DECIMAL(10, 2),
    mac_address VARCHAR(17) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    health_score DECIMAL(5, 2) DEFAULT 0.00,
    device_count INT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_status CHECK (status IN ('online', 'offline', 'maintenance', 'error')),
    CONSTRAINT valid_health_score CHECK (health_score >= 0 AND health_score <= 100)
);

COMMENT ON TABLE cabinets IS '储能柜资产表';
COMMENT ON COLUMN cabinets.cabinet_id IS '储能柜唯一标识';
COMMENT ON COLUMN cabinets.mac_address IS '储能柜MAC地址，用于设备绑定';
COMMENT ON COLUMN cabinets.health_score IS '健康评分 (0-100)';

-- 传感器设备表
CREATE TABLE IF NOT EXISTS sensor_devices (
    device_id VARCHAR(50) PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    sensor_type VARCHAR(50) NOT NULL,
    name VARCHAR(100),
    unit VARCHAR(20),
    min_value DECIMAL(15, 6),
    max_value DECIMAL(15, 6),
    status VARCHAR(20) DEFAULT 'active',
    last_reading_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_sensor_status CHECK (status IN ('active', 'inactive', 'error'))
);

COMMENT ON TABLE sensor_devices IS '传感器设备元数据表';
COMMENT ON COLUMN sensor_devices.sensor_type IS '传感器类型：voltage, current, temperature等';

-- 许可证表
CREATE TABLE IF NOT EXISTS licenses (
    license_id VARCHAR(50) PRIMARY KEY,
    cabinet_id VARCHAR(50) UNIQUE NOT NULL,
    mac_address VARCHAR(17) NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoke_reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    permissions JSONB DEFAULT '[]',
    max_devices INTEGER NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_license_status CHECK (status IN ('active', 'expired', 'revoked'))
);

COMMENT ON TABLE licenses IS '储能柜许可证表';
COMMENT ON COLUMN licenses.mac_address IS 'MAC地址绑定，防止设备克隆';
COMMENT ON COLUMN licenses.permissions IS '许可证权限配置，JSON格式';

-- 告警表
CREATE TABLE IF NOT EXISTS alerts (
    alert_id BIGSERIAL PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    edge_alert_id BIGINT,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_severity CHECK (severity IN ('info', 'warning', 'error', 'critical'))
);

COMMENT ON TABLE alerts IS '告警记录表';
COMMENT ON COLUMN alerts.severity IS '告警严重度：info, warning, error, critical';
COMMENT ON COLUMN alerts.details IS '告警详细信息，JSON格式';

-- 命令表
CREATE TABLE IF NOT EXISTS commands (
    command_id VARCHAR(50) PRIMARY KEY,
    cabinet_id VARCHAR(50) NOT NULL,
    command_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    response JSONB,
    sent_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_by VARCHAR(100) NOT NULL DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(cabinet_id) ON DELETE CASCADE,
    CONSTRAINT valid_command_status CHECK (status IN ('pending', 'sent', 'completed', 'failed', 'timeout'))
);

COMMENT ON TABLE commands IS '命令下发记录表';
COMMENT ON COLUMN commands.command_type IS '命令类型：config_update, license_update, license_revoke等';
COMMENT ON COLUMN commands.payload IS '命令负载，JSON格式';
COMMENT ON COLUMN commands.created_by IS '命令创建者（用户ID或系统标识）';

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    log_id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(50),
    result VARCHAR(20) NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_result CHECK (result IN ('success', 'failure'))
);

COMMENT ON TABLE audit_logs IS '审计日志表';
COMMENT ON COLUMN audit_logs.action IS '操作类型：create, update, delete等';
COMMENT ON COLUMN audit_logs.details IS '操作详细信息，JSON格式';

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加更新时间触发器
CREATE TRIGGER update_cabinets_updated_at BEFORE UPDATE ON cabinets 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sensor_devices_updated_at BEFORE UPDATE ON sensor_devices 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_licenses_updated_at BEFORE UPDATE ON licenses 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_commands_updated_at BEFORE UPDATE ON commands 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
