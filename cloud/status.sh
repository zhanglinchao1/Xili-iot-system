#!/bin/bash
# Cloud端储能柜管理系统 - 查看服务状态脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$PROJECT_ROOT/logs"

# PID文件
BACKEND_PID_FILE="$LOG_DIR/backend.pid"
FRONTEND_PID_FILE="$LOG_DIR/frontend.pid"

print_header() {
    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}          Cloud端储能柜管理系统 - 服务状态                    ${CYAN}║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

check_service() {
    local service_name=$1
    local pid_file=$2
    local port=$3
    local url=$4
    
    echo -e "${CYAN}${service_name}:${NC}"
    
    if [ -f "$pid_file" ]; then
        PID=$(cat "$pid_file")
        if kill -0 "$PID" 2>/dev/null; then
            echo -e "  状态: ${GREEN}✓ 运行中${NC}"
            echo -e "  PID:  $PID"
            
            # 检查端口
            if lsof -i:$port >/dev/null 2>&1; then
                echo -e "  端口: ${GREEN}✓ $port (监听中)${NC}"
            else
                echo -e "  端口: ${RED}✗ $port (未监听)${NC}"
            fi
            
            # 检查HTTP访问
            if [ ! -z "$url" ]; then
                if curl -s "$url" >/dev/null 2>&1; then
                    echo -e "  访问: ${GREEN}✓ $url${NC}"
                else
                    echo -e "  访问: ${YELLOW}⚠ $url (无响应)${NC}"
                fi
            fi
        else
            echo -e "  状态: ${RED}✗ 未运行 (PID文件存在但进程不存在)${NC}"
        fi
    else
        echo -e "  状态: ${RED}✗ 未运行${NC}"
        
        # 检查端口是否被其他进程占用
        if lsof -i:$port >/dev/null 2>&1; then
            echo -e "  端口: ${YELLOW}⚠ $port (被其他进程占用)${NC}"
        fi
    fi
    
    echo ""
}

check_dependencies() {
    echo -e "${CYAN}依赖服务:${NC}"
    
    # PostgreSQL
    if pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
        echo -e "  PostgreSQL: ${GREEN}✓ 运行中${NC}"
    else
        echo -e "  PostgreSQL: ${RED}✗ 未运行${NC}"
    fi
    
    # MQTT (TLS on Edge端)
    if nc -z localhost 8883 >/dev/null 2>&1; then
        echo -e "  MQTT (TLS): ${GREEN}✓ 运行中 (Edge端:8883)${NC}"
    else
        echo -e "  MQTT (TLS): ${YELLOW}⚠ 未运行 (需要Edge端)${NC}"
    fi
    
    # Redis
    if nc -z localhost 6379 >/dev/null 2>&1; then
        echo -e "  Redis:      ${GREEN}✓ 运行中${NC}"
    else
        echo -e "  Redis:      ${YELLOW}⚠ 未运行 (可选)${NC}"
    fi
    
    echo ""
}

show_recent_logs() {
    local service=$1
    local log_file=$2
    local lines=${3:-5}
    
    if [ -f "$log_file" ]; then
        echo -e "${CYAN}${service} 最近日志 (最后${lines}行):${NC}"
        echo -e "${YELLOW}────────────────────────────────────────${NC}"
        tail -n "$lines" "$log_file" 2>/dev/null | sed 's/^/  /'
        echo -e "${YELLOW}────────────────────────────────────────${NC}"
        echo ""
    fi
}

main() {
    print_header
    
    # 检查服务状态
    check_service "后端服务" "$BACKEND_PID_FILE" "8003" "http://localhost:8003/health"
    check_service "前端服务" "$FRONTEND_PID_FILE" "8002" "http://localhost:8002"
    
    # 检查依赖服务
    check_dependencies
    
    # 显示访问信息
    echo -e "${CYAN}访问地址:${NC}"
    echo -e "  前端: ${GREEN}http://localhost:8002${NC}"
    echo -e "  后端: ${GREEN}http://localhost:8003${NC}"
    echo -e "  API:  ${GREEN}http://localhost:8003/api/v1${NC}"
    echo ""
    
    echo -e "${CYAN}测试账号:${NC}"
    echo -e "  用户名: ${GREEN}admin${NC}"
    echo -e "  密码:   ${GREEN}admin${NC}"
    echo ""
    
    # 显示日志位置
    if [ -d "$LOG_DIR" ]; then
        echo -e "${CYAN}日志文件:${NC}"
        echo -e "  后端: $LOG_DIR/backend.log"
        echo -e "  前端: $LOG_DIR/frontend.log"
        echo ""
        
        # 显示最近的日志
        if [ "$1" = "-v" ] || [ "$1" = "--verbose" ]; then
            show_recent_logs "后端" "$LOG_DIR/backend.log" 10
            show_recent_logs "前端" "$LOG_DIR/frontend.log" 10
        fi
    fi
    
    echo -e "${CYAN}提示:${NC}"
    echo -e "  启动服务: ${GREEN}./start_all.sh${NC}"
    echo -e "  停止服务: ${GREEN}./stop_all.sh${NC}"
    echo -e "  查看日志: ${GREEN}./status.sh -v${NC}"
    echo ""
}

main "$@"

