-- 授予cloud_user对licenses表的完整权限
-- 需要以postgres超级用户执行此脚本

-- 授予cloud_user对licenses表的所有权限
GRANT ALL PRIVILEGES ON TABLE licenses TO cloud_user;

-- 授予cloud_user对licenses表序列的权限（如果有）
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cloud_user;

-- 授予cloud_user创建表的权限（如果需要）
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO cloud_user;

-- 将licenses表的所有权转移给cloud_user（可选，如果希望cloud_user成为所有者）
-- ALTER TABLE licenses OWNER TO cloud_user;

