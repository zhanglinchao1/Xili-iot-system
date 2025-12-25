#!/bin/bash
# 修复licenses表结构的脚本
# 使用postgres超级用户执行，解决权限问题

set -e

# 数据库配置（从config.yaml读取或使用环境变量）
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-cloud_system}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

# 迁移文件路径
MIGRATION_FILE="migrations/009_fix_license_schema.sql"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "执行数据库迁移: $MIGRATION_FILE"
echo "使用用户: $POSTGRES_USER"
echo "数据库: $DB_NAME@$DB_HOST:$DB_PORT"
echo ""

# 如果设置了密码环境变量，使用它
if [ -n "$POSTGRES_PASSWORD" ]; then
    export PGPASSWORD="$POSTGRES_PASSWORD"
fi

# 执行迁移
psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d "$DB_NAME" -f "$MIGRATION_FILE"

echo ""
echo "迁移完成！"

