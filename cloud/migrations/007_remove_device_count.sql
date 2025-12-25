-- 迁移脚本: 删除cabinets表的device_count列
-- 版本: 007
-- 日期: 2025-11-07
-- 说明: 单柜系统不需要device_count字段，移除该列以简化数据模型

-- 删除device_count列（如果存在）
ALTER TABLE cabinets DROP COLUMN IF EXISTS device_count;

-- 验证迁移
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'cabinets' 
        AND column_name = 'device_count'
    ) THEN
        RAISE EXCEPTION 'device_count列删除失败';
    ELSE
        RAISE NOTICE 'device_count列已成功删除';
    END IF;
END $$;

