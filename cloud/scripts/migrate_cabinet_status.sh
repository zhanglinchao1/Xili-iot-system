#!/bin/bash
# Cabinet状态迁移脚本
# 将旧状态值('online', 'error')迁移到新状态值('active', 'offline')

set -e

echo "========================================="
echo "Cabinet状态值迁移脚本"
echo "========================================="
echo ""

# 检查是否在cloud目录
if [ ! -f "go.mod" ] || ! grep -q "cloud-system" go.mod; then
    echo "❌ 错误：请在cloud项目根目录下执行此脚本"
    exit 1
fi

# 数据库连接信息
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-cloud_user}"
DB_NAME="${DB_NAME:-cloud_system}"
DB_PASSWORD="${DB_PASSWORD:-cloud123456}"

echo "📋 当前配置："
echo "  数据库主机: $DB_HOST"
echo "  数据库端口: $DB_PORT"
echo "  数据库名称: $DB_NAME"
echo "  数据库用户: $DB_USER"
echo ""

# 检查数据库连接
echo "🔍 检查数据库连接..."
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\q" 2>/dev/null; then
    echo "❌ 错误：无法连接到数据库"
    exit 1
fi
echo "✅ 数据库连接成功"
echo ""

# 检查是否需要迁移
echo "🔍 检查是否需要迁移..."
ONLINE_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
    "SELECT COUNT(*) FROM cabinets WHERE status = 'online';" 2>/dev/null || echo "0")
ERROR_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
    "SELECT COUNT(*) FROM cabinets WHERE status = 'error';" 2>/dev/null || echo "0")

# 去除空格
ONLINE_COUNT=$(echo $ONLINE_COUNT | tr -d ' ')
ERROR_COUNT=$(echo $ERROR_COUNT | tr -d ' ')

echo "发现旧状态值:"
echo "  'online' 状态: $ONLINE_COUNT 个"
echo "  'error' 状态: $ERROR_COUNT 个"
echo ""

if [ "$ONLINE_COUNT" = "0" ] && [ "$ERROR_COUNT" = "0" ]; then
    echo "✅ 无需迁移，所有Cabinet已使用新状态值"
    echo ""
    echo "📊 当前状态分布:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
        "SELECT status, COUNT(*) as count FROM cabinets GROUP BY status ORDER BY count DESC;"
    exit 0
fi

# 确认执行
echo "⚠️  即将执行状态值迁移:"
echo "  • 'online' → 'active' (${ONLINE_COUNT}个)"
echo "  • 'error' → 'offline' (${ERROR_COUNT}个)"
echo ""
read -p "是否继续？(y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 用户取消操作"
    exit 0
fi

echo ""
echo "🔧 开始执行迁移..."
echo ""

# 执行迁移SQL
if [ -f "migrations/008_migrate_cabinet_status_values.sql" ]; then
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/008_migrate_cabinet_status_values.sql
else
    # 如果migration文件不存在，直接执行SQL
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<'EOF'
-- 迁移'online'到'active'
UPDATE cabinets
SET status = 'active', updated_at = NOW()
WHERE status = 'online';

-- 迁移'error'到'offline'
UPDATE cabinets
SET status = 'offline', updated_at = NOW()
WHERE status = 'error';

-- 显示结果
SELECT status, COUNT(*) as count
FROM cabinets
GROUP BY status
ORDER BY count DESC;
EOF
fi

echo ""
echo "========================================="
echo "✅ 迁移成功完成！"
echo "========================================="
echo ""
echo "📝 后续步骤："
echo "  1. 检查前端储能柜管理页面，确认状态显示正确"
echo "  2. 如需回滚，执行："
echo "     UPDATE cabinets SET status = 'online' WHERE status = 'active';"
echo "     UPDATE cabinets SET status = 'error' WHERE status = 'offline';"
echo ""
