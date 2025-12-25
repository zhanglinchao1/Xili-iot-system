-- 创建数据库索引

-- 储能柜表索引
CREATE INDEX IF NOT EXISTS idx_cabinets_mac_address ON cabinets(mac_address);
CREATE INDEX IF NOT EXISTS idx_cabinets_status ON cabinets(status);
CREATE INDEX IF NOT EXISTS idx_cabinets_last_sync_at ON cabinets(last_sync_at);
CREATE INDEX IF NOT EXISTS idx_cabinets_health_score ON cabinets(health_score);

-- 传感器设备表索引
CREATE INDEX IF NOT EXISTS idx_sensor_devices_cabinet_id ON sensor_devices(cabinet_id);
CREATE INDEX IF NOT EXISTS idx_sensor_devices_sensor_type ON sensor_devices(sensor_type);
CREATE INDEX IF NOT EXISTS idx_sensor_devices_status ON sensor_devices(status);

-- 许可证表索引
CREATE INDEX IF NOT EXISTS idx_licenses_cabinet_id ON licenses(cabinet_id);
CREATE INDEX IF NOT EXISTS idx_licenses_mac_address ON licenses(mac_address);
CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status);
CREATE INDEX IF NOT EXISTS idx_licenses_expires_at ON licenses(expires_at);

-- 告警表索引
CREATE INDEX IF NOT EXISTS idx_alerts_cabinet_id ON alerts(cabinet_id);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_cabinet_created ON alerts(cabinet_id, created_at DESC);

-- 命令表索引
CREATE INDEX IF NOT EXISTS idx_commands_cabinet_id ON commands(cabinet_id);
CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status);
CREATE INDEX IF NOT EXISTS idx_commands_created_at ON commands(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commands_cabinet_status ON commands(cabinet_id, status);

-- 审计日志表索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_result ON audit_logs(action, result);

-- JSONB索引（使用GIN索引）
CREATE INDEX IF NOT EXISTS idx_licenses_permissions ON licenses USING GIN(permissions);
CREATE INDEX IF NOT EXISTS idx_alerts_details ON alerts USING GIN(details);
CREATE INDEX IF NOT EXISTS idx_commands_payload ON commands USING GIN(payload);
CREATE INDEX IF NOT EXISTS idx_commands_response ON commands USING GIN(response);
CREATE INDEX IF NOT EXISTS idx_audit_logs_details ON audit_logs USING GIN(details);

