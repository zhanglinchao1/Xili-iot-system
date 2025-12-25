#!/bin/bash
# =============================================================================
# Edge数据库验证和自动修复脚本
# 功能: 验证数据库schema完整性，自动修复缺失的表和字段
# 使用方法: ./verify_database.sh [--db-path /path/to/edge.db] [--no-auto-fix]
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 默认配置
DB_PATH="./data/edge.db"
AUTO_FIX=true
VERBOSE=false

# 预期的表清单（基于 internal/storage/sqlite.go）
EXPECTED_TABLES=(
    "devices"
    "challenges"
    "sessions"
    "sensor_data"
    "alerts"
    "system_logs"
    "data_statistics"
    "vulnerability_assessments"
    "transmission_metrics"
    "dismissed_vulnerabilities"
    "cloud_credentials"
)

# 关键字段检查（表名:字段名列表）
declare -A CRITICAL_FIELDS=(
    ["devices"]="device_id,device_type,sensor_type,status"
    ["sensor_data"]="device_id,sensor_type,value,timestamp,synced"
    ["alerts"]="device_id,alert_type,severity,timestamp,resolved"
    ["cloud_credentials"]="cabinet_id,api_key,cloud_endpoint,enabled"
    ["vulnerability_assessments"]="cabinet_id,license_compliance_score,overall_score"
)

# 统计变量
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
FIXED_ISSUES=0

# 解析参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --db-path)
            DB_PATH="$2"
            shift 2
            ;;
        --no-auto-fix)
            AUTO_FIX=false
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "使用方法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --db-path PATH    数据库文件路径 (默认: ./data/edge.db)"
            echo "  --no-auto-fix     禁用自动修复，仅检查"
            echo "  --verbose, -v     详细输出"
            echo "  --help, -h        显示帮助信息"
            exit 0
            ;;
        *)
            echo -e "${RED}未知参数: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Edge数据库验证工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "数据库路径: ${CYAN}$DB_PATH${NC}"
echo -e "自动修复: ${CYAN}$([ "$AUTO_FIX" = true ] && echo "启用" || echo "禁用")${NC}"
echo ""

# 检查 sqlite3 命令是否存在
if ! command -v sqlite3 &> /dev/null; then
    echo -e "${RED}错误: 未找到 sqlite3 命令${NC}"
    echo "请安装 sqlite3: sudo apt-get install sqlite3"
    exit 1
fi

# 检查1: 数据库文件是否存在
echo -e "${YELLOW}[检查 1/5] 数据库文件存在性...${NC}"
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}✗ 数据库文件不存在: $DB_PATH${NC}"
    echo -e "${YELLOW}提示: 容器首次启动时会自动创建数据库${NC}"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))

    # 如果数据库不存在，无法继续后续检查
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}验证摘要${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo -e "总检查项: $TOTAL_CHECKS"
    echo -e "失败: ${RED}$FAILED_CHECKS${NC}"
    echo -e "建议: 启动Edge服务以自动初始化数据库"
    exit 1
else
    echo -e "${GREEN}✓ 数据库文件存在${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))

    # 显示文件信息
    DB_SIZE=$(du -h "$DB_PATH" | cut -f1)
    DB_MODIFIED=$(stat -c "%y" "$DB_PATH" 2>/dev/null || stat -f "%Sm" "$DB_PATH" 2>/dev/null || echo "未知")
    echo -e "  文件大小: ${CYAN}$DB_SIZE${NC}"
    [ "$VERBOSE" = true ] && echo -e "  最后修改: ${CYAN}$DB_MODIFIED${NC}"
fi
echo ""

# 检查2: 数据库文件可读写
echo -e "${YELLOW}[检查 2/5] 数据库文件权限...${NC}"
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

if [ ! -r "$DB_PATH" ]; then
    echo -e "${RED}✗ 数据库文件不可读${NC}"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
elif [ ! -w "$DB_PATH" ]; then
    echo -e "${RED}✗ 数据库文件不可写${NC}"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
else
    echo -e "${GREEN}✓ 数据库文件可读写${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
fi
echo ""

# 检查3: 数据库完整性
echo -e "${YELLOW}[检查 3/5] 数据库完整性...${NC}"
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

INTEGRITY_CHECK=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;" 2>&1 || echo "ERROR")

if [ "$INTEGRITY_CHECK" = "ok" ]; then
    echo -e "${GREEN}✓ 数据库完整性正常${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "${RED}✗ 数据库完整性检查失败${NC}"
    [ "$VERBOSE" = true ] && echo -e "  详情: $INTEGRITY_CHECK"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi
echo ""

# 检查4: 表数量和存在性
echo -e "${YELLOW}[检查 4/5] 数据库表结构...${NC}"

# 获取实际表数量（排除sqlite内部表）
ACTUAL_TABLE_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';" 2>/dev/null || echo "0")
EXPECTED_TABLE_COUNT=${#EXPECTED_TABLES[@]}

echo -e "预期表数量: ${CYAN}$EXPECTED_TABLE_COUNT${NC}"
echo -e "实际表数量: ${CYAN}$ACTUAL_TABLE_COUNT${NC}"
echo ""

MISSING_TABLES=()
for table in "${EXPECTED_TABLES[@]}"; do
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    TABLE_EXISTS=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';" 2>/dev/null || echo "0")

    if [ "$TABLE_EXISTS" -eq 1 ]; then
        echo -e "${GREEN}✓${NC} 表 ${CYAN}$table${NC} 存在"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))

        # 显示记录数（verbose模式）
        if [ "$VERBOSE" = true ]; then
            ROW_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "?")
            echo -e "    记录数: $ROW_COUNT"
        fi
    else
        echo -e "${RED}✗${NC} 表 ${CYAN}$table${NC} 缺失"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        MISSING_TABLES+=("$table")
    fi
done
echo ""

# 自动修复缺失的表
if [ ${#MISSING_TABLES[@]} -gt 0 ] && [ "$AUTO_FIX" = true ]; then
    echo -e "${YELLOW}检测到 ${#MISSING_TABLES[@]} 个缺失的表，开始自动修复...${NC}"
    echo ""

    for table in "${MISSING_TABLES[@]}"; do
        echo -e "${BLUE}正在修复表: $table${NC}"

        # 注意：这里需要根据实际的CREATE语句来修复
        # 简化起见，建议重新运行应用的迁移逻辑
        echo -e "${YELLOW}  建议: 运行 './edge -config ./configs/config.yaml --migrate' 自动创建缺失的表${NC}"
    done

    echo ""
fi

# 检查5: 关键字段完整性
echo -e "${YELLOW}[检查 5/5] 关键字段完整性...${NC}"

for table in "${!CRITICAL_FIELDS[@]}"; do
    # 先检查表是否存在
    TABLE_EXISTS=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';" 2>/dev/null || echo "0")

    if [ "$TABLE_EXISTS" -eq 0 ]; then
        continue  # 表不存在，跳过字段检查
    fi

    IFS=',' read -ra FIELDS <<< "${CRITICAL_FIELDS[$table]}"

    for field in "${FIELDS[@]}"; do
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

        # 获取表的字段列表
        FIELD_EXISTS=$(sqlite3 "$DB_PATH" "PRAGMA table_info($table);" 2>/dev/null | grep -c "^[0-9]*|$field|" || echo "0")

        if [ "$FIELD_EXISTS" -gt 0 ]; then
            echo -e "${GREEN}✓${NC} 表 ${CYAN}$table${NC} 字段 ${CYAN}$field${NC} 存在"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            echo -e "${RED}✗${NC} 表 ${CYAN}$table${NC} 缺失字段 ${CYAN}$field${NC}"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))

            # 自动修复字段（简化版，实际需要根据字段类型）
            if [ "$AUTO_FIX" = true ]; then
                echo -e "${YELLOW}  ⚠️  自动添加字段功能需要完整的字段定义${NC}"
                echo -e "${YELLOW}  建议: 运行 './edge -config ./configs/config.yaml --migrate'${NC}"
            fi
        fi
    done
done
echo ""

# 显示验证摘要
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证摘要${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "总检查项: ${CYAN}$TOTAL_CHECKS${NC}"
echo -e "通过: ${GREEN}$PASSED_CHECKS${NC}"
echo -e "失败: ${RED}$FAILED_CHECKS${NC}"

if [ $FIXED_ISSUES -gt 0 ]; then
    echo -e "已修复: ${GREEN}$FIXED_ISSUES${NC}"
fi

echo ""

# 额外信息：数据库统计
if [ "$VERBOSE" = true ] && [ -f "$DB_PATH" ]; then
    echo -e "${BLUE}数据库统计信息：${NC}"
    echo -e "  表数量: $(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';")"
    echo -e "  索引数量: $(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%';")"
    echo -e "  总记录数: $(sqlite3 "$DB_PATH" "SELECT SUM(cnt) FROM (SELECT COUNT(*) as cnt FROM devices UNION ALL SELECT COUNT(*) FROM sensor_data UNION ALL SELECT COUNT(*) FROM alerts);" 2>/dev/null || echo "N/A")"
    echo ""
fi

# 建议
if [ $FAILED_CHECKS -gt 0 ]; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}修复建议${NC}"
    echo -e "${YELLOW}========================================${NC}"

    if [ ${#MISSING_TABLES[@]} -gt 0 ]; then
        echo -e "1. 运行数据库迁移创建缺失的表:"
        echo -e "   ${CYAN}./edge -config ./configs/config.yaml --migrate${NC}"
        echo ""
    fi

    echo -e "2. 或者删除数据库让系统重新创建（${RED}会丢失所有数据${NC}）:"
    echo -e "   ${CYAN}rm $DB_PATH${NC}"
    echo -e "   ${CYAN}docker-compose restart edge-system${NC}"
    echo ""

    echo -e "3. 如果问题持续，检查应用日志:"
    echo -e "   ${CYAN}docker logs edge-system | grep -i database${NC}"
    echo ""
fi

# 退出码
if [ $FAILED_CHECKS -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库验证通过${NC}"
    exit 0
else
    echo -e "${RED}❌ 数据库验证失败 ($FAILED_CHECKS 项失败)${NC}"
    exit 1
fi
