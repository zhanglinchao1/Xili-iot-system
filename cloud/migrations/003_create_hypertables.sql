-- 创建TimescaleDB时序数据表（Hypertables）

-- 传感器数据表（时序数据）
CREATE TABLE IF NOT EXISTS sensor_data (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    cabinet_id VARCHAR(50) NOT NULL,
    device_id VARCHAR(50) NOT NULL,
    sensor_type VARCHAR(50) NOT NULL,
    value DECIMAL(15, 6) NOT NULL,
    unit VARCHAR(20),
    quality DECIMAL(5, 2) DEFAULT 100.00,
    raw_value JSONB,
    CONSTRAINT valid_quality CHECK (quality >= 0 AND quality <= 100)
);

COMMENT ON TABLE sensor_data IS '传感器时序数据表';
COMMENT ON COLUMN sensor_data.quality IS '数据质量指标 (0-100)';
COMMENT ON COLUMN sensor_data.raw_value IS '原始数据，JSON格式';

-- 将传感器数据表转换为Hypertable（按时间分区）
SELECT create_hypertable('sensor_data', 'time', if_not_exists => TRUE);

-- 创建复合索引
CREATE INDEX IF NOT EXISTS idx_sensor_data_cabinet_time 
    ON sensor_data (cabinet_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_data_device_time 
    ON sensor_data (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_data_type_time 
    ON sensor_data (sensor_type, time DESC);

-- 健康评分历史表（时序数据）
CREATE TABLE IF NOT EXISTS health_scores (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    cabinet_id VARCHAR(50) NOT NULL,
    score DECIMAL(5, 2) NOT NULL,
    online_rate DECIMAL(5, 2),
    data_quality DECIMAL(5, 2),
    alert_severity_score DECIMAL(5, 2),
    sensor_normalcy DECIMAL(5, 2),
    details JSONB,
    CONSTRAINT valid_score CHECK (score >= 0 AND score <= 100)
);

COMMENT ON TABLE health_scores IS '健康评分历史表';
COMMENT ON COLUMN health_scores.score IS '综合健康评分 (0-100)';
COMMENT ON COLUMN health_scores.online_rate IS '设备在线率得分';
COMMENT ON COLUMN health_scores.data_quality IS '数据质量得分';
COMMENT ON COLUMN health_scores.alert_severity_score IS '告警严重度得分';
COMMENT ON COLUMN health_scores.sensor_normalcy IS '传感器正常率得分';

-- 将健康评分表转换为Hypertable
SELECT create_hypertable('health_scores', 'time', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_health_scores_cabinet_time 
    ON health_scores (cabinet_id, time DESC);

-- 设置数据保留策略（可选）
-- 保留最近90天的传感器数据
-- SELECT add_retention_policy('sensor_data', INTERVAL '90 days', if_not_exists => TRUE);

-- 保留最近365天的健康评分数据
-- SELECT add_retention_policy('health_scores', INTERVAL '365 days', if_not_exists => TRUE);

