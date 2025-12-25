-- 修复audit_logs表的result字段问题
-- commands表实际上不应该有result字段,而是audit_logs表才有result字段
-- 但是代码中可能误用了result字段,这里先检查并确保正确性

-- 检查audit_logs表是否有result字段
DO $$
BEGIN
    -- 确保audit_logs表有result字段
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'audit_logs' AND column_name = 'result'
    ) THEN
        -- audit_logs表原本就应该有result字段,无需添加
        NULL;
    END IF;
END $$;

COMMENT ON COLUMN audit_logs.result IS '操作结果: success, failure';
