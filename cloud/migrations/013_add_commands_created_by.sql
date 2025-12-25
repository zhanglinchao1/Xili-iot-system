-- 添加commands表的created_by字段
-- 用于记录命令创建者

ALTER TABLE commands ADD COLUMN IF NOT EXISTS created_by VARCHAR(100);

COMMENT ON COLUMN commands.created_by IS '命令创建者（用户ID或系统标识）';
