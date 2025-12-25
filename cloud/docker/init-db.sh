#!/bin/bash
# PostgreSQL数据库初始化脚本
# 在PostgreSQL容器首次启动时执行

set -e

echo "=========================================="
echo "Cloud System Database Initialization"
echo "=========================================="

# 等待PostgreSQL服务就绪
until pg_isready -U postgres; do
  echo "Waiting for PostgreSQL to be ready..."
  sleep 1
done

echo "PostgreSQL is ready!"

# 创建应用数据库用户（如果不存在）
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 创建cloud_user用户（如果不存在）
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'cloud_user') THEN
            CREATE USER cloud_user WITH PASSWORD 'cloud123456';
            RAISE NOTICE 'User cloud_user created';
        ELSE
            RAISE NOTICE 'User cloud_user already exists';
        END IF;
    END
    \$\$;

    -- 授予cloud_user必要的权限
    GRANT CONNECT ON DATABASE cloud_system TO cloud_user;
    GRANT USAGE ON SCHEMA public TO cloud_user;
    GRANT CREATE ON SCHEMA public TO cloud_user;
    
    -- 授予cloud_user对现有表的权限
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cloud_user;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cloud_user;
    
    -- 设置默认权限，使cloud_user对新创建的表也有权限
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO cloud_user;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO cloud_user;
    
    -- 创建TimescaleDB扩展（如果可用）
    CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
EOSQL

echo "=========================================="
echo "Database initialization completed!"
echo "=========================================="
echo "Database: $POSTGRES_DB"
echo "User: cloud_user"
echo "Password: cloud123456"
echo "=========================================="

