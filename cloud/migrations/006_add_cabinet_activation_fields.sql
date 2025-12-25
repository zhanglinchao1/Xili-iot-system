-- 添加储能柜激活相关字段
-- 支持预注册+自动激活架构

-- 添加激活状态字段
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS activation_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS registration_token VARCHAR(500);
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS token_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS api_key VARCHAR(100);
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS api_secret_hash VARCHAR(255);
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS activated_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS device_model VARCHAR(100);
ALTER TABLE cabinets ADD COLUMN IF NOT EXISTS notes TEXT;

-- 修改status约束，添加pending、active、inactive状态
-- 新的状态体系：pending(待激活), active(激活且同步中), inactive(已停用), offline(离线), maintenance(维护中)
ALTER TABLE cabinets DROP CONSTRAINT IF EXISTS valid_status;
ALTER TABLE cabinets ADD CONSTRAINT valid_status
    CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance'));

-- 添加activation_status约束
ALTER TABLE cabinets ADD CONSTRAINT valid_activation_status
    CHECK (activation_status IN ('pending', 'activated'));

-- 创建唯一索引用于快速查找
CREATE UNIQUE INDEX IF NOT EXISTS idx_cabinets_api_key ON cabinets(api_key) WHERE api_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cabinets_registration_token ON cabinets(registration_token) WHERE registration_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cabinets_activation_status ON cabinets(activation_status);

-- 添加注释
COMMENT ON COLUMN cabinets.activation_status IS '激活状态: pending-待激活, activated-已激活';
COMMENT ON COLUMN cabinets.registration_token IS '注册Token，用于首次激活，24小时有效';
COMMENT ON COLUMN cabinets.token_expires_at IS 'Token过期时间';
COMMENT ON COLUMN cabinets.api_key IS 'Edge端API密钥（激活后生成）';
COMMENT ON COLUMN cabinets.api_secret_hash IS 'Edge端API密钥哈希值';
COMMENT ON COLUMN cabinets.activated_at IS '激活时间';
COMMENT ON COLUMN cabinets.ip_address IS 'Edge设备IP地址';
COMMENT ON COLUMN cabinets.device_model IS '设备型号';
COMMENT ON COLUMN cabinets.notes IS '备注信息';
