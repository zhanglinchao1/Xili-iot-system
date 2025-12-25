-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    email VARCHAR(128),
    role VARCHAR(32) DEFAULT 'user' CHECK (role IN ('admin', 'user', 'viewer')),
    status VARCHAR(16) DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_login_at TIMESTAMPTZ
);

-- 创建索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);

-- 插入默认admin账号（密码: admin）
-- 使用bcrypt哈希后的密码
INSERT INTO users (username, password_hash, email, role, status)
VALUES ('admin', '$2a$10$Bz2EG1pvt1eLSacMeLTk2euuNqXuiNFI2Ec2aMlK7vp67WlqKEzr2', 'admin@example.com', 'admin', 'active')
ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash;

-- 添加注释
COMMENT ON TABLE users IS '系统用户表';
COMMENT ON COLUMN users.password_hash IS 'bcrypt哈希后的密码';
COMMENT ON COLUMN users.role IS '用户角色：admin（管理员）/user（普通用户）/viewer（只读）';
COMMENT ON COLUMN users.status IS '用户状态：active（激活）/disabled（禁用）';

