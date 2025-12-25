#!/bin/bash
# Cloud端储能柜管理系统 - 停止所有服务脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$PROJECT_ROOT/logs"

# PID文件
BACKEND_PID_FILE="$LOG_DIR/backend.pid"
FRONTEND_PID_FILE="$LOG_DIR/frontend.pid"

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

# 停止后端服务
stop_backend() {
    print_info "停止后端服务..."
    
    if [ -f "$BACKEND_PID_FILE" ]; then
        BACKEND_PID=$(cat "$BACKEND_PID_FILE")
        if kill -0 "$BACKEND_PID" 2>/dev/null; then
            kill -TERM "$BACKEND_PID" 2>/dev/null || true
            sleep 2
            if kill -0 "$BACKEND_PID" 2>/dev/null; then
                kill -9 "$BACKEND_PID" 2>/dev/null || true
            fi
            print_success "后端服务已停止"
        else
            print_warning "后端服务未运行"
        fi
        rm -f "$BACKEND_PID_FILE"
    else
        print_warning "未找到后端PID文件"
    fi
    
    # 额外检查端口占用
    if lsof -i:8003 >/dev/null 2>&1; then
        print_warning "端口8003仍被占用，强制停止..."
        lsof -ti:8003 | xargs -r kill -9 2>/dev/null || true
    fi
}

# 停止前端服务
stop_frontend() {
    print_info "停止前端服务..."
    
    if [ -f "$FRONTEND_PID_FILE" ]; then
        FRONTEND_PID=$(cat "$FRONTEND_PID_FILE")
        if kill -0 "$FRONTEND_PID" 2>/dev/null; then
            kill -TERM "$FRONTEND_PID" 2>/dev/null || true
            sleep 2
            if kill -0 "$FRONTEND_PID" 2>/dev/null; then
                kill -9 "$FRONTEND_PID" 2>/dev/null || true
            fi
            print_success "前端服务已停止"
        else
            print_warning "前端服务未运行"
        fi
        rm -f "$FRONTEND_PID_FILE"
    else
        print_warning "未找到前端PID文件"
    fi
    
    # 额外检查端口占用
    if lsof -i:8002 >/dev/null 2>&1; then
        print_warning "端口8002仍被占用，强制停止..."
        lsof -ti:8002 | xargs -r kill -9 2>/dev/null || true
    fi
}

# 主函数
main() {
    echo ""
    echo "═══════════════════════════════════════"
    echo "  Cloud端系统 - 停止所有服务"
    echo "═══════════════════════════════════════"
    echo ""
    
    stop_backend
    stop_frontend
    
    echo ""
    print_success "所有服务已停止"
    echo ""
}

main

