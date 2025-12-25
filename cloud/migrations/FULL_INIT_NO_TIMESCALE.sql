-- ===============================================
-- Cloud端储能柜管理系统 - 数据库初始化脚本(无TimescaleDB版本)
-- 用于测试和不支持TimescaleDB的环境
-- 生成时间: 2025-11-23
-- ===============================================

-- ===============================================
-- 第一部分: 扩展创建
-- ===============================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ===============================================
-- 第二部分: 核心表创建
-- ===============================================

-- 储能柜表
CREATE TABLE IF NOT EXISTS cabinets (
    cabinet_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    location VARCHAR(500),
    latitude DECIMAL(10, 6),
    longitude DECIMAL(11, 6),
    address VARCHAR(500),
    mac_address VARCHAR(17) UNIQUE NOT NULL,

    -- 状态字段
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    activation_status VARCHAR(20) DEFAULT 'pending',

    -- 激活相关字段
    registration_token VARCHAR(500),
    token_expires_at TIMESTAMP WITH TIME ZONE,
    api_key VARCHAR(100),
    api_secret_hash VARCHAR(255),
    activated BOOLEAN DEFAULT FALSE,
    activated_at TIMESTAMP WITH TIME ZONE,

    -- 设备信息
    ip_address VARCHAR(45),
    device_model VARCHAR(100),
    notes TEXT,

    -- 脆弱性评分字段(替代health_score)
    latest_vulnerability_score FLOAT DEFAULT 0,
    latest_risk_level TEXT DEFAULT 'unknown',
    vulnerability_updated_at TIMESTAMP,

    -- 时间戳
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT valid_cabinet_status CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance')),
    CONSTRAINT valid_activation_status CHECK (activation_status IN ('pending', 'activated')),
    CONSTRAINT valid_latitude CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT valid_longitude CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

-- 其他表定义与FULL_INIT.sql相同,只是移除了TimescaleDB相关的create_hypertable调用
-- 为简化测试,仅创建关键表

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    email VARCHAR(128),
    role VARCHAR(32) DEFAULT 'user',
    status VARCHAR(16) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_login_at TIMESTAMPTZ,

    CONSTRAINT valid_user_role CHECK (role IN ('admin', 'user', 'viewer')),
    CONSTRAINT valid_user_status CHECK (status IN ('active', 'disabled'))
);

-- 插入默认管理员用户(密码: admin)
INSERT INTO users (username, password_hash, email, role, status)
VALUES ('admin', '$2a$10$Bz2EG1pvt1eLSacMeLTk2euuNqXuiNFI2Ec2aMlK7vp67WlqKEzr2', 'admin@example.com', 'admin', 'active')
ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash;

-- 验证表创建
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Database Initialization Complete (No TimescaleDB)!';
    RAISE NOTICE 'Tables created: cabinets, users';
    RAISE NOTICE 'Default admin user: admin / admin';
    RAISE NOTICE '========================================';
END $$;
