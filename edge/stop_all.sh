#!/bin/bash
# Edge设备管理系统 - 服务停止脚本
# 一键关闭前后端所有服务

# 支持强制停止模式
FORCE_KILL=false
if [[ "$1" == "--force" ]] || [[ "$1" == "-f" ]]; then
    FORCE_KILL=true
fi

# 配置参数
BACKEND_PORT=8001
FRONTEND_PORT=8000
MQTT_PORT=8883
BACKEND_PROCESS_NAME="edge"
FRONTEND_PROCESS_PATTERN="python.*http.server.*${FRONTEND_PORT}"
MQTT_PROCESS_PATTERN="mosquitto.*mosquitto_tls.conf"

echo "================================"
echo "Edge设备管理系统 - 服务停止"
echo "================================"
echo ""

# 函数：检查端口是否被占用
check_port() {
    local port=$1
    lsof -i :$port > /dev/null 2>&1
}

# 函数：获取占用端口的PID
get_port_pid() {
    local port=$1
    lsof -t -i :$port 2>/dev/null
}

# 函数：获取进程信息
get_process_info() {
    local pid=$1
    if [ ! -z "$pid" ] && kill -0 $pid 2>/dev/null; then
        ps -p $pid -o pid,ppid,cmd --no-headers 2>/dev/null
    fi
}

# 函数：优雅停止进程
graceful_stop() {
    local pid=$1
    local service_name=$2
    local timeout=${3:-10}
    
    if [ -z "$pid" ]; then
        return 0
    fi
    
    if ! kill -0 $pid 2>/dev/null; then
        echo "✓ $service_name 进程已经停止"
        return 0
    fi
    
    echo "正在优雅停止 $service_name 进程 (PID: $pid)..."
    
    # 发送 SIGTERM 信号
    kill -TERM $pid 2>/dev/null
    
    # 等待进程优雅退出
    local count=0
    while [ $count -lt $timeout ] && kill -0 $pid 2>/dev/null; do
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    echo ""
    
    # 检查进程是否已经停止
    if kill -0 $pid 2>/dev/null; then
        return 1  # 进程仍在运行
    else
        echo "✓ $service_name 进程已优雅停止"
        return 0
    fi
}

# 函数：强制停止进程
force_stop() {
    local pid=$1
    local service_name=$2
    
    if [ -z "$pid" ]; then
        return 0
    fi
    
    if ! kill -0 $pid 2>/dev/null; then
        echo "✓ $service_name 进程已经停止"
        return 0
    fi
    
    echo "强制停止 $service_name 进程 (PID: $pid)..."
    
    # 发送 SIGKILL 信号
    kill -9 $pid 2>/dev/null
    sleep 1
    
    # 检查进程是否已经停止
    if kill -0 $pid 2>/dev/null; then
        echo "✗ 无法停止 $service_name 进程"
        return 1
    else
        echo "✓ $service_name 进程已强制停止"
        return 0
    fi
}

# 函数：停止服务
stop_service() {
    local port=$1
    local service_name=$2
    local process_pattern=$3
    
    echo "================================"
    echo "停止 $service_name"
    echo "================================"
    
    # 检查端口是否被占用
    if ! check_port $port; then
        echo "✓ $service_name 未运行 (端口 $port 未被占用)"
        echo ""
        return 0
    fi
    
    # 获取占用端口的进程信息
    local pid=$(get_port_pid $port)
    if [ -z "$pid" ]; then
        echo "✓ $service_name 未运行"
        echo ""
        return 0
    fi
    
    echo "发现 $service_name 进程:"
    echo "端口: $port"
    echo "PID: $pid"
    
    # 显示进程详细信息
    local process_info=$(get_process_info $pid)
    if [ ! -z "$process_info" ]; then
        echo "进程信息: $process_info"
    fi
    echo ""
    
    # 根据模式选择停止方式
    if [[ "$FORCE_KILL" == "true" ]]; then
        echo "强制停止模式"
        force_stop $pid "$service_name"
    else
        # 尝试优雅停止
        if ! graceful_stop $pid "$service_name" 10; then
            echo ""
            echo "优雅停止失败，是否强制停止？"
            read -p "输入 y 强制停止，其他键跳过: " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                force_stop $pid "$service_name"
            else
                echo "跳过强制停止 $service_name"
            fi
        fi
    fi
    
    echo ""
}

# 函数：停止所有相关进程（按进程名模式）
stop_processes_by_pattern() {
    local pattern=$1
    local service_name=$2
    
    echo "查找 $service_name 相关进程..."
    
    # 使用 pgrep 查找匹配的进程
    local pids=$(pgrep -f "$pattern" 2>/dev/null)
    
    if [ -z "$pids" ]; then
        echo "✓ 未发现 $service_name 相关进程"
        return 0
    fi
    
    echo "发现 $service_name 进程: $pids"
    
    # 逐个停止进程
    for pid in $pids; do
        if kill -0 $pid 2>/dev/null; then
            local process_info=$(get_process_info $pid)
            echo "停止进程 PID: $pid"
            if [ ! -z "$process_info" ]; then
                echo "  $process_info"
            fi
            
            if [[ "$FORCE_KILL" == "true" ]]; then
                force_stop $pid "$service_name"
            else
                if ! graceful_stop $pid "$service_name" 5; then
                    force_stop $pid "$service_name"
                fi
            fi
        fi
    done
}

# 函数：清理日志文件（可选）
cleanup_logs() {
    if [[ "$1" == "--clean-logs" ]]; then
        echo "================================"
        echo "清理日志文件"
        echo "================================"
        
        if [ -f "logs/backend.log" ]; then
            echo "清理后端日志: logs/backend.log"
            > logs/backend.log
        fi
        
        if [ -f "logs/frontend.log" ]; then
            echo "清理前端日志: logs/frontend.log"
            > logs/frontend.log
        fi
        
        echo "✓ 日志文件已清理"
        echo ""
    fi
}

# 函数：显示最终状态
show_final_status() {
    echo "================================"
    echo "最终状态检查"
    echo "================================"
    
    local all_stopped=true
    
    # 检查后端服务
    if check_port $BACKEND_PORT; then
        local backend_pid=$(get_port_pid $BACKEND_PORT)
        echo "⚠️  后端服务仍在运行 (PID: $backend_pid, 端口: $BACKEND_PORT)"
        all_stopped=false
    else
        echo "✓ 后端服务已停止"
    fi
    
    # 检查前端服务
    if check_port $FRONTEND_PORT; then
        local frontend_pid=$(get_port_pid $FRONTEND_PORT)
        echo "⚠️  前端服务仍在运行 (PID: $frontend_pid, 端口: $FRONTEND_PORT)"
        all_stopped=false
    else
        echo "✓ 前端服务已停止"
    fi

    # 检查MQTT Broker服务
    if check_port $MQTT_PORT; then
        local mqtt_pid=$(get_port_pid $MQTT_PORT)
        echo "⚠️  MQTT Broker仍在运行 (PID: $mqtt_pid, 端口: $MQTT_PORT)"
        all_stopped=false
    else
        echo "✓ MQTT Broker已停止"
    fi

    echo ""

    if [ "$all_stopped" = true ]; then
        echo "🎉 所有服务已成功停止！"
        return 0
    else
        echo "⚠️  部分服务仍在运行"
        echo ""
        echo "如需强制停止所有服务，请运行："
        echo "  $0 --force"
        echo ""
        echo "或手动停止："
        echo "  pkill -9 -f 'edge|python.*http.server.*$FRONTEND_PORT|mosquitto.*mosquitto_tls.conf'"
        return 1
    fi
}

# 主执行流程
main() {
    echo "开始停止 Edge 系统服务..."
    echo ""
    
    # 停止后端服务
    stop_service $BACKEND_PORT "后端服务" "$BACKEND_PROCESS_NAME"

    # 停止前端服务
    stop_service $FRONTEND_PORT "前端服务" "$FRONTEND_PROCESS_PATTERN"

    # 停止MQTT Broker服务
    stop_service $MQTT_PORT "MQTT Broker" "$MQTT_PROCESS_PATTERN"

    # 额外清理：按进程名模式停止可能遗漏的进程
    echo "================================"
    echo "额外清理检查"
    echo "================================"

    stop_processes_by_pattern "$BACKEND_PROCESS_NAME" "后端服务"
    stop_processes_by_pattern "$FRONTEND_PROCESS_PATTERN" "前端服务"
    stop_processes_by_pattern "$MQTT_PROCESS_PATTERN" "MQTT Broker"
    
    echo ""
    
    # 清理日志（如果指定）
    cleanup_logs "$2"
    
    # 显示最终状态
    show_final_status
}

# 脚本使用说明
show_help() {
    echo "Edge设备管理系统 - 服务停止脚本"
    echo ""
    echo "使用方法："
    echo "  $0                    # 交互模式停止（优雅停止，失败时询问是否强制停止）"
    echo "  $0 --force            # 强制停止模式（直接发送 SIGKILL 信号）"
    echo "  $0 -f                 # 强制停止模式（简写）"
    echo "  $0 --help             # 显示帮助信息"
    echo ""
    echo "额外选项："
    echo "  --clean-logs          # 停止服务后清理日志文件"
    echo ""
    echo "示例："
    echo "  $0                    # 优雅停止所有服务"
    echo "  $0 --force            # 强制停止所有服务"
    echo "  $0 --force --clean-logs  # 强制停止并清理日志"
    echo ""
    echo "服务信息："
    echo "  后端服务端口: $BACKEND_PORT"
    echo "  前端服务端口: $FRONTEND_PORT"
    echo "  MQTT Broker端口: $MQTT_PORT"
    echo ""
    echo "注意："
    echo "  - 优雅停止会先发送 SIGTERM 信号，等待进程自行退出"
    echo "  - 强制停止会直接发送 SIGKILL 信号"
    echo "  - 建议优先使用优雅停止方式"
}

# 处理命令行参数
if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
    show_help
    exit 0
fi

# 执行主程序
main "$@"
exit $?
