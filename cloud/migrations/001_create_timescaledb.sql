-- 创建TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 验证扩展安装
SELECT default_version, installed_version 
FROM pg_available_extensions 
WHERE name = 'timescaledb';

