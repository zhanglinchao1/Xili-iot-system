#!/bin/bash
# Cloud端储能柜管理系统 - 一键启动脚本
# 功能：启动后端和前端服务，实时监控日志

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

# 日志目录
LOG_DIR="$PROJECT_ROOT/logs"
mkdir -p "$LOG_DIR"

# PID文件
BACKEND_PID_FILE="$LOG_DIR/backend.pid"
FRONTEND_PID_FILE="$LOG_DIR/frontend.pid"

# 日志文件
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
COMBINED_LOG="$LOG_DIR/combined.log"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC}          $1${PURPLE}║${NC}"
    echo -e "${PURPLE}╚═══════════════════════════════════════════════════════════════╝${NC}"
}

# 清理函数（在脚本退出时执行）
cleanup() {
    echo ""
    print_info "接收到退出信号，正在优雅关闭服务..."
    
    # 停止后端
    if [ -f "$BACKEND_PID_FILE" ]; then
        BACKEND_PID=$(cat "$BACKEND_PID_FILE")
        if kill -0 "$BACKEND_PID" 2>/dev/null; then
            print_info "停止后端服务 (PID: $BACKEND_PID)..."
            kill -TERM "$BACKEND_PID" 2>/dev/null || true
            sleep 2
            kill -9 "$BACKEND_PID" 2>/dev/null || true
        fi
        rm -f "$BACKEND_PID_FILE"
    fi
    
    # 停止前端
    if [ -f "$FRONTEND_PID_FILE" ]; then
        FRONTEND_PID=$(cat "$FRONTEND_PID_FILE")
        if kill -0 "$FRONTEND_PID" 2>/dev/null; then
            print_info "停止前端服务 (PID: $FRONTEND_PID)..."
            kill -TERM "$FRONTEND_PID" 2>/dev/null || true
            sleep 2
            kill -9 "$FRONTEND_PID" 2>/dev/null || true
        fi
        rm -f "$FRONTEND_PID_FILE"
    fi
    
    print_success "所有服务已停止"
    exit 0
}

# 注册退出信号处理
trap cleanup SIGINT SIGTERM

# 检查依赖服务
check_dependencies() {
    print_header "检查系统依赖"
    
    local all_ok=true
    
    # 检查PostgreSQL
    print_info "检查PostgreSQL..."
    if pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
        print_success "PostgreSQL: 运行中 ✓"
    else
        print_error "PostgreSQL: 未运行 ✗"
        all_ok=false
    fi
    
    # 检查MQTT Broker (8883 TLS端口)
    print_info "检查MQTT Broker (TLS)..."
    if nc -z localhost 8883 >/dev/null 2>&1; then
        print_success "MQTT Broker (8883): 运行中 ✓"
    else
        print_warning "MQTT Broker (8883): 未运行 (需要Edge端mosquitto服务)"
    fi
    
    # 检查Redis（可选）
    print_info "检查Redis..."
    if nc -z localhost 6379 >/dev/null 2>&1; then
        print_success "Redis: 运行中 ✓"
    else
        print_warning "Redis: 未运行 (可选服务，系统将继续运行)"
    fi
    
    # 检查Go环境
    print_info "检查Go环境..."
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version | awk '{print $3}')
        print_success "Go: $GO_VERSION ✓"
    else
        print_error "Go: 未安装 ✗"
        all_ok=false
    fi
    
    # 检查Node.js环境
    print_info "检查Node.js环境..."
    if command -v node >/dev/null 2>&1; then
        NODE_VERSION=$(node --version)
        print_success "Node.js: $NODE_VERSION ✓"
    else
        print_error "Node.js: 未安装 ✗"
        all_ok=false
    fi
    
    # 检查npm
    if command -v npm >/dev/null 2>&1; then
        NPM_VERSION=$(npm --version)
        print_success "npm: v$NPM_VERSION ✓"
    else
        print_error "npm: 未安装 ✗"
        all_ok=false
    fi
    
    echo ""
    
    if [ "$all_ok" = false ]; then
        print_error "存在必需的依赖未满足，请先安装缺失的依赖"
        exit 1
    fi
    
    print_success "所有必需依赖检查通过"
    echo ""
}

# 检查并停止已存在的服务
stop_existing_services() {
    print_info "检查并停止已存在的服务..."
    
    # 检查端口占用
    if lsof -i:8003 >/dev/null 2>&1; then
        print_warning "端口8003已被占用，正在停止..."
        lsof -ti:8003 | xargs -r kill -9 2>/dev/null || true
        sleep 1
    fi
    
    if lsof -i:8002 >/dev/null 2>&1; then
        print_warning "端口8002已被占用，正在停止..."
        lsof -ti:8002 | xargs -r kill -9 2>/dev/null || true
        sleep 1
    fi
    
    # 清理旧的PID文件
    rm -f "$BACKEND_PID_FILE" "$FRONTEND_PID_FILE"
    
    print_success "清理完成"
    echo ""
}

# 编译后端
build_backend() {
    print_header "编译后端服务"
    
    # 强制重新编译，删除旧的二进制文件
    if [ -f "$PROJECT_ROOT/bin/cloud-server" ]; then
        print_info "删除旧的二进制文件..."
        rm -f "$PROJECT_ROOT/bin/cloud-server"
    fi
    
    print_info "正在编译Go后端..."
    if go build -o "$PROJECT_ROOT/bin/cloud-server" "$PROJECT_ROOT/cmd/cloud-server/main.go"; then
        print_success "后端编译成功 ✓"
    else
        print_error "后端编译失败 ✗"
        exit 1
    fi
    echo ""
}

# 启动后端服务
start_backend() {
    print_header "启动后端服务"
    
    # 清空日志
    > "$BACKEND_LOG"
    
    print_info "启动后端服务..."
    nohup "$PROJECT_ROOT/bin/cloud-server" >> "$BACKEND_LOG" 2>&1 &
    BACKEND_PID=$!
    echo "$BACKEND_PID" > "$BACKEND_PID_FILE"
    
    print_info "等待后端服务启动..."
    sleep 3
    
    # 检查后端是否启动成功
    if kill -0 "$BACKEND_PID" 2>/dev/null; then
        # 测试健康检查端点
        if curl -s http://localhost:8003/health >/dev/null 2>&1; then
            print_success "后端服务启动成功 ✓"
            print_info "后端地址: ${CYAN}http://localhost:8003${NC}"
            print_info "后端PID: ${CYAN}$BACKEND_PID${NC}"
            print_info "后端日志: ${CYAN}$BACKEND_LOG${NC}"
        else
            print_warning "后端服务已启动，但健康检查失败（可能仍在初始化）"
        fi
    else
        print_error "后端服务启动失败 ✗"
        print_error "请查看日志: tail -f $BACKEND_LOG"
        exit 1
    fi
    echo ""
}

# 安装前端依赖
install_frontend_deps() {
    print_header "安装前端依赖"
    
    # 检查是否需要安装依赖
    if [ ! -d "$PROJECT_ROOT/frontend/node_modules" ]; then
        print_info "首次运行，正在安装前端依赖..."
        cd "$PROJECT_ROOT/frontend"
        npm install
        cd "$PROJECT_ROOT"
        print_success "前端依赖安装完成 ✓"
    else
        print_info "前端依赖已存在，跳过安装"
    fi
    echo ""
}

# 启动前端服务（开发模式）
start_frontend() {
    print_header "启动前端服务"
    
    # 清空日志
    > "$FRONTEND_LOG"
    
    print_info "启动前端服务..."
    cd "$PROJECT_ROOT/frontend"
    nohup npm run dev >> "$FRONTEND_LOG" 2>&1 &
    FRONTEND_PID=$!
    echo "$FRONTEND_PID" > "$FRONTEND_PID_FILE"
    cd "$PROJECT_ROOT"
    
    print_info "等待前端服务启动..."
    sleep 5
    
    # 检查前端是否启动成功
    if kill -0 "$FRONTEND_PID" 2>/dev/null; then
        if curl -s http://localhost:8002 >/dev/null 2>&1; then
            print_success "前端服务启动成功 ✓"
            print_info "前端地址: ${CYAN}http://localhost:8002${NC}"
            print_info "前端PID: ${CYAN}$FRONTEND_PID${NC}"
            print_info "前端日志: ${CYAN}$FRONTEND_LOG${NC}"
        else
            print_warning "前端服务已启动，但访问失败（可能仍在编译）"
        fi
    else
        print_error "前端服务启动失败 ✗"
        print_error "请查看日志: tail -f $FRONTEND_LOG"
        exit 1
    fi
    echo ""
}

# 显示系统状态
show_status() {
    print_header "系统运行状态"
    
    echo -e "${CYAN}后端服务:${NC}"
    echo -e "  地址: http://localhost:8003"
    echo -e "  PID:  $(cat $BACKEND_PID_FILE 2>/dev/null || echo 'N/A')"
    echo -e "  API:  http://localhost:8003/api/v1"
    echo ""
    
    echo -e "${CYAN}前端服务:${NC}"
    echo -e "  地址: http://localhost:8002"
    echo -e "  PID:  $(cat $FRONTEND_PID_FILE 2>/dev/null || echo 'N/A')"
    echo ""
    
    echo -e "${CYAN}测试账号:${NC}"
    echo -e "  用户名: admin"
    echo -e "  密码:   admin"
    echo ""
    
    echo -e "${CYAN}日志文件:${NC}"
    echo -e "  后端: $BACKEND_LOG"
    echo -e "  前端: $FRONTEND_LOG"
    echo -e "  合并: $COMBINED_LOG"
    echo ""
}

# 实时监控日志
monitor_logs() {
    print_header "实时日志监控"
    
    print_info "按 ${CYAN}Ctrl+C${NC} 停止所有服务并退出"
    print_info "正在监控日志..."
    echo ""
    
    # 创建合并日志文件
    > "$COMBINED_LOG"
    
    # 使用tail -f监控两个日志文件
    tail -f "$BACKEND_LOG" "$FRONTEND_LOG" 2>/dev/null | while IFS= read -r line; do
        # 为不同服务的日志添加前缀
        if [[ "$line" == *"==> $BACKEND_LOG <=="* ]]; then
            echo -e "${GREEN}[BACKEND]${NC}"
        elif [[ "$line" == *"==> $FRONTEND_LOG <=="* ]]; then
            echo -e "${BLUE}[FRONTEND]${NC}"
        elif [[ ! -z "$line" ]]; then
            # 根据日志内容判断来源
            if [[ "$line" == *"\"level\""* ]] || [[ "$line" == *"cloud-server"* ]]; then
                echo -e "${GREEN}[BACKEND]${NC} $line"
            elif [[ "$line" == *"vite"* ]] || [[ "$line" == *"vue"* ]]; then
                echo -e "${BLUE}[FRONTEND]${NC} $line"
            else
                echo "$line"
            fi
        fi
        
        # 同时写入合并日志
        echo "$line" >> "$COMBINED_LOG"
    done
}

# 主函数
main() {
    clear
    
    print_header "Cloud端储能柜管理系统 - 启动脚本"
    echo ""
    
    # 1. 检查依赖
    check_dependencies
    
    # 2. 停止已存在的服务
    stop_existing_services
    
    # 3. 编译后端
    build_backend
    
    # 4. 启动后端
    start_backend
    
    # 5. 安装前端依赖（如需要）
    install_frontend_deps
    
    # 6. 启动前端（开发模式，自动重新编译）
    start_frontend
    
    # 7. 显示状态
    show_status
    
    # 8. 监控日志
    monitor_logs
}

# 执行主函数
main

