-- Migration: 迁移Cabinet状态值到新的状态体系
-- 目的: 将旧状态值('online', 'error')迁移到新状态值('active', 'offline'/'inactive')
-- 创建时间: 2025-12-18

-- 1. 首先显示当前状态分布
DO $$
DECLARE
    online_count INTEGER;
    error_count INTEGER;
    total_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO online_count FROM cabinets WHERE status = 'online';
    SELECT COUNT(*) INTO error_count FROM cabinets WHERE status = 'error';
    SELECT COUNT(*) INTO total_count FROM cabinets;

    RAISE NOTICE '========================================';
    RAISE NOTICE '迁移前Cabinet状态统计:';
    RAISE NOTICE '  总数: %', total_count;
    RAISE NOTICE '  online状态: %', online_count;
    RAISE NOTICE '  error状态: %', error_count;
    RAISE NOTICE '========================================';
END $$;

-- 2. 将'online'状态迁移为'active'
-- 逻辑: 'online'表示在线且正常工作，对应新的'active'状态
UPDATE cabinets
SET status = 'active', updated_at = NOW()
WHERE status = 'online';

-- 3. 将'error'状态迁移为'offline'
-- 逻辑: 'error'表示故障，将其视为离线状态
-- 如果需要更细粒度的处理，可以根据其他条件判断是'offline'还是'inactive'
UPDATE cabinets
SET status = 'offline', updated_at = NOW()
WHERE status = 'error';

-- 4. 显示迁移后的状态分布
DO $$
DECLARE
    pending_count INTEGER;
    active_count INTEGER;
    inactive_count INTEGER;
    offline_count INTEGER;
    maintenance_count INTEGER;
    other_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO pending_count FROM cabinets WHERE status = 'pending';
    SELECT COUNT(*) INTO active_count FROM cabinets WHERE status = 'active';
    SELECT COUNT(*) INTO inactive_count FROM cabinets WHERE status = 'inactive';
    SELECT COUNT(*) INTO offline_count FROM cabinets WHERE status = 'offline';
    SELECT COUNT(*) INTO maintenance_count FROM cabinets WHERE status = 'maintenance';
    SELECT COUNT(*) INTO other_count FROM cabinets WHERE status NOT IN ('pending', 'active', 'inactive', 'offline', 'maintenance');

    RAISE NOTICE '========================================';
    RAISE NOTICE '迁移后Cabinet状态统计:';
    RAISE NOTICE '  pending(待激活): %', pending_count;
    RAISE NOTICE '  active(激活且同步中): %', active_count;
    RAISE NOTICE '  inactive(已停用): %', inactive_count;
    RAISE NOTICE '  offline(离线): %', offline_count;
    RAISE NOTICE '  maintenance(维护中): %', maintenance_count;
    RAISE NOTICE '  其他状态(异常): %', other_count;
    RAISE NOTICE '========================================';

    IF other_count > 0 THEN
        RAISE WARNING '⚠️  检测到 % 个Cabinet使用了未定义的状态值！', other_count;
    END IF;
END $$;

-- 5. 添加注释
COMMENT ON COLUMN cabinets.status IS '储能柜状态: pending-待激活, active-激活且同步中, inactive-已停用, offline-离线, maintenance-维护中';
COMMENT ON CONSTRAINT valid_status ON cabinets IS '储能柜状态约束: 只允许 pending, active, inactive, offline, maintenance 五种状态';

-- 6. 验证约束是否正确
DO $$
DECLARE
    constraint_def TEXT;
BEGIN
    SELECT consrc INTO constraint_def
    FROM pg_constraint
    WHERE conrelid = 'cabinets'::regclass
      AND conname = 'valid_status';

    IF constraint_def IS NULL THEN
        RAISE EXCEPTION '❌ 未找到valid_status约束，请先执行007_fix_cabinet_status_constraint.sql';
    END IF;

    IF constraint_def NOT LIKE '%active%' THEN
        RAISE EXCEPTION '❌ valid_status约束不包含active状态，请先执行007_fix_cabinet_status_constraint.sql';
    END IF;

    RAISE NOTICE '✅ 约束验证通过: %', constraint_def;
END $$;
