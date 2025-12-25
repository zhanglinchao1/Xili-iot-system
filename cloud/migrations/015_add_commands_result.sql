-- 添加commands表的result字段
-- 用于存储Edge端返回的命令执行结果

ALTER TABLE commands ADD COLUMN IF NOT EXISTS result TEXT;

COMMENT ON COLUMN commands.result IS 'Edge端返回的命令执行结果';
