-- 手动修复 licenses 表结构
-- 使用DO块安全地添加字段，避免权限问题

-- 添加 license_id 字段（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'licenses' 
      AND column_name = 'license_id'
  ) THEN
    ALTER TABLE licenses ADD COLUMN license_id VARCHAR(50);
  END IF;
END $$;

-- 添加 max_devices 字段（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'licenses' 
      AND column_name = 'max_devices'
  ) THEN
    ALTER TABLE licenses ADD COLUMN max_devices INTEGER NOT NULL DEFAULT 0;
  END IF;
END $$;

-- 添加 created_by 字段（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'licenses' 
      AND column_name = 'created_by'
  ) THEN
    ALTER TABLE licenses ADD COLUMN created_by VARCHAR(100) NOT NULL DEFAULT 'system';
  END IF;
END $$;

-- 添加 revoke_reason 字段（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'licenses' 
      AND column_name = 'revoke_reason'
  ) THEN
    ALTER TABLE licenses ADD COLUMN revoke_reason TEXT;
  END IF;
END $$;

-- 更新 license_id 字段的值（如果为空）
UPDATE licenses
SET license_id = cabinet_id
WHERE (license_id IS NULL OR license_id = '');

-- 设置 license_id 为 NOT NULL（如果字段存在且可以为空）
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'licenses' 
      AND column_name = 'license_id'
      AND is_nullable = 'YES'
  ) THEN
    ALTER TABLE licenses ALTER COLUMN license_id SET NOT NULL;
  END IF;
END $$;

-- 添加唯一约束（如果不存在）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = 'public'
      AND tablename = 'licenses'
      AND indexname = 'licenses_license_id_key'
  ) THEN
    -- 检查约束是否已存在
    IF NOT EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'licenses_license_id_key'
        AND conrelid = 'licenses'::regclass
  ) THEN
    ALTER TABLE licenses ADD CONSTRAINT licenses_license_id_key UNIQUE (license_id);
    END IF;
  END IF;
END $$;
