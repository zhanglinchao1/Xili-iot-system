-- 添加储能柜位置坐标字段
-- 支持地图展示和位置搜索功能

-- 添加纬度字段（范围：-90到90）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'cabinets' 
      AND column_name = 'latitude'
  ) THEN
    ALTER TABLE cabinets ADD COLUMN latitude DECIMAL(10, 6);
  END IF;
END $$;

-- 添加经度字段（范围：-180到180）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_schema = 'public' 
      AND table_name = 'cabinets' 
      AND column_name = 'longitude'
  ) THEN
    ALTER TABLE cabinets ADD COLUMN longitude DECIMAL(10, 6);
  END IF;
END $$;

-- 添加索引以支持位置查询
CREATE INDEX IF NOT EXISTS idx_cabinets_latitude ON cabinets(latitude) WHERE latitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cabinets_longitude ON cabinets(longitude) WHERE longitude IS NOT NULL;

-- 添加复合索引用于地理位置查询
CREATE INDEX IF NOT EXISTS idx_cabinets_location ON cabinets(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- 添加约束：纬度范围
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint 
    WHERE conname = 'valid_latitude' 
      AND conrelid = 'cabinets'::regclass
  ) THEN
    ALTER TABLE cabinets ADD CONSTRAINT valid_latitude 
      CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90));
  END IF;
END $$;

-- 添加约束：经度范围
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint 
    WHERE conname = 'valid_longitude' 
      AND conrelid = 'cabinets'::regclass
  ) THEN
    ALTER TABLE cabinets ADD CONSTRAINT valid_longitude 
      CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180));
  END IF;
END $$;

-- 添加注释
COMMENT ON COLUMN cabinets.latitude IS '储能柜纬度坐标（-90到90）';
COMMENT ON COLUMN cabinets.longitude IS '储能柜经度坐标（-180到180）';

