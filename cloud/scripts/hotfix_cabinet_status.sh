#!/bin/bash
# Hotfix脚本：修复Cabinet状态约束问题
# 问题：数据库约束不包含'active'状态，导致同步后状态无法更新
# 使用方法：./scripts/hotfix_cabinet_status.sh

set -e

echo "========================================="
echo "Cabinet状态约束Hotfix脚本"
echo "========================================="
echo ""

# 检查是否在cloud目录
if [ ! -f "go.mod" ] || ! grep -q "cloud-system" go.mod; then
    echo "❌ 错误：请在cloud项目根目录下执行此脚本"
    exit 1
fi

# 数据库连接信息（从config.yaml或环境变量读取）
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

# 检查PostgreSQL连接
echo "🔍 检查数据库连接..."
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\q" 2>/dev/null; then
    echo "❌ 错误：无法连接到数据库"
    echo "请检查数据库配置或设置环境变量："
    echo "  export DB_HOST=your_host"
    echo "  export DB_PORT=your_port"
    echo "  export DB_USER=your_user"
    echo "  export DB_PASSWORD=your_password"
    exit 1
fi
echo "✅ 数据库连接成功"
echo ""

# 检查当前约束
echo "🔍 检查当前status约束..."
CURRENT_CONSTRAINT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
    "SELECT consrc FROM pg_constraint WHERE conrelid = 'cabinets'::regclass AND conname = 'valid_status';" 2>/dev/null || echo "")

if [ -z "$CURRENT_CONSTRAINT" ]; then
    echo "⚠️  警告：未找到valid_status约束"
else
    echo "当前约束: $CURRENT_CONSTRAINT"
fi
echo ""

# 检查是否需要修复
if echo "$CURRENT_CONSTRAINT" | grep -q "'active'"; then
    echo "✅ 约束已包含'active'状态，无需修复"
    echo ""
    echo "📊 当前Cabinet状态统计："
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
        "SELECT status, COUNT(*) as count FROM cabinets GROUP BY status ORDER BY count DESC;"
    exit 0
fi

echo "⚠️  检测到约束需要修复（缺少'active'状态）"
echo ""

# 确认执行
read -p "是否继续执行修复？(y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 用户取消操作"
    exit 0
fi

echo ""
echo "🔧 开始执行修复..."
echo ""

# 执行修复SQL
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<'EOF'
-- 1. 删除旧约束
ALTER TABLE cabinets DROP CONSTRAINT IF EXISTS valid_status;
\echo '✓ 已删除旧的status约束'

-- 2. 添加新约束
ALTER TABLE cabinets ADD CONSTRAINT valid_status
    CHECK (status IN ('pending', 'active', 'inactive', 'offline', 'maintenance'));
\echo '✓ 已添加新的status约束（包含active和inactive）'

-- 3. 查询需要更新的Cabinet
SELECT cabinet_id, status, last_sync_at, activation_status
FROM cabinets
WHERE status = 'offline'
  AND activation_status = 'activated'
  AND last_sync_at IS NOT NULL
  AND last_sync_at > NOW() - INTERVAL '10 minutes';

\echo ''
\echo '以上Cabinet将被更新为active状态'
\echo ''

-- 4. 更新状态
UPDATE cabinets
SET status = 'active', updated_at = NOW()
WHERE status = 'offline'
  AND activation_status = 'activated'
  AND last_sync_at IS NOT NULL
  AND last_sync_at > NOW() - INTERVAL '10 minutes';

\echo ''
\echo '✓ 状态更新完成'
\echo ''

-- 5. 显示修复后的统计
\echo '📊 修复后Cabinet状态统计：'
SELECT status, COUNT(*) as count
FROM cabinets
GROUP BY status
ORDER BY count DESC;

\echo ''
\echo '✅ 修复完成！'
EOF

echo ""
echo "========================================="
echo "修复成功完成！"
echo "========================================="
echo ""
echo "📝 后续步骤："
echo "  1. 检查前端储能柜管理页面，确认状态显示正确"
echo "  2. 等待Edge端下一次同步（5分钟内），验证状态能正常更新为'active'"
echo "  3. 如需回滚，执行："
echo "     ALTER TABLE cabinets DROP CONSTRAINT valid_status;"
echo "     ALTER TABLE cabinets ADD CONSTRAINT valid_status"
echo "       CHECK (status IN ('pending', 'online', 'offline', 'maintenance', 'error'));"
echo ""
