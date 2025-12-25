-- 添加edge_alert_id字段到alerts表
-- 用于存储Edge端告警的ID,方便回调解决告警

ALTER TABLE alerts ADD COLUMN IF NOT EXISTS edge_alert_id BIGINT;

COMMENT ON COLUMN alerts.edge_alert_id IS 'Edge端告警ID,用于命令下发时定位Edge端数据';

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_alerts_edge_alert_id ON alerts(edge_alert_id) WHERE edge_alert_id IS NOT NULL;
