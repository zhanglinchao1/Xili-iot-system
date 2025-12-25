-- 策略分发日志表
CREATE TABLE IF NOT EXISTS policy_distribution_logs (
    id SERIAL PRIMARY KEY,
    policy_id TEXT NOT NULL,
    cabinet_id TEXT NOT NULL,
    operation_type TEXT NOT NULL,  -- distribute, broadcast, sync
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, success, failed
    operator_id INTEGER,
    operator_name TEXT,
    error_message TEXT,
    distributed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP
);

-- 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_distribution_policy
    ON policy_distribution_logs(policy_id);

CREATE INDEX IF NOT EXISTS idx_distribution_cabinet
    ON policy_distribution_logs(cabinet_id);

CREATE INDEX IF NOT EXISTS idx_distribution_status
    ON policy_distribution_logs(status);

CREATE INDEX IF NOT EXISTS idx_distribution_time
    ON policy_distribution_logs(distributed_at DESC);

-- 注释
COMMENT ON TABLE policy_distribution_logs IS '策略分发日志表，记录每次策略分发操作';
COMMENT ON COLUMN policy_distribution_logs.operation_type IS '操作类型: distribute(指定分发), broadcast(广播), sync(全量同步)';
COMMENT ON COLUMN policy_distribution_logs.status IS '分发状态: pending(待确认), success(已应用), failed(失败)';
COMMENT ON COLUMN policy_distribution_logs.acknowledged_at IS 'Edge端确认应用策略的时间';
