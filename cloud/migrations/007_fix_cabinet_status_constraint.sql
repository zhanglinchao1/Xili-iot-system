-- Hotfix: 修复Cabinet状态约束，添加'active'和'inactive'状态
-- 问题：006迁移文件的status约束缺少'active'状态，导致UpdateLastSyncTime()更新失败
-- 影响：Edge端同步后Cabinet状态无法从'offline'变为'active'，前端显示"离线未同步"
-- 创建时间：2025-12-18

-- 1. 删除旧的status约束
ALTER TABLE cabinets DROP CONSTRAINT IF EXISTS valid_status;

-- 2. 添加新的status约束（包含active和inactive）
ALTER TABLE cabinets ADD CONSTRAINT valid_status
    CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance'));

-- 3. 更新当前所有'offline'且最近有同步的Cabinet为'active'
-- （如果last_sync_at在最近10分钟内，说明Edge端正在同步）
UPDATE cabinets
SET status = 'active', updated_at = NOW()
WHERE status = 'offline'
  AND activation_status = 'activated'
  AND last_sync_at IS NOT NULL
  AND last_sync_at > NOW() - INTERVAL '10 minutes';

-- 4. 添加注释
COMMENT ON CONSTRAINT valid_status ON cabinets IS '储能柜状态约束: pending-待激活, active-激活且同步中, inactive-已停用, offline-离线, maintenance-维护中';

-- 5. 输出修复结果
DO $$
DECLARE
    updated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO updated_count
    FROM cabinets
    WHERE status = 'active' AND activation_status = 'activated';

    RAISE NOTICE '修复完成！已将 % 个储能柜状态更新为 active', updated_count;
END $$;
