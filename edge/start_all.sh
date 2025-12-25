#!/bin/bash
# Edge设备管理系统 - 完整启动脚本
# 整合前后端服务启动，支持进程检查和管理

# 支持参数
FORCE_RESTART=false
FORCE_REBUILD=false

# 配置参数
BACKEND_PORT=8001
FRONTEND_PORT=8000
BACKEND_BINARY="./edge"
BACKEND_CONFIG="./configs/config.yaml"
FRONTEND_DIR="./web"

echo "================================"
echo "Edge设备管理系统 - 完整启动"
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

# 函数：终止进程
kill_process() {
    local pid=$1
    local service_name=$2
    
    if [ ! -z "$pid" ]; then
        echo "正在终止 $service_name 进程 (PID: $pid)..."
        kill $pid
        sleep 2
        
        # 检查进程是否被成功终止
        if kill -0 $pid 2>/dev/null; then
            echo "强制终止进程..."
            kill -9 $pid
            sleep 1
        fi
        
        if kill -0 $pid 2>/dev/null; then
            echo "✗ 无法终止 $service_name 进程"
            return 1
        else
            echo "✓ $service_name 进程已终止"
            return 0
        fi
    fi
    return 0
}

# 函数：处理端口占用
handle_port_conflict() {
    local port=$1
    local service_name=$2
    
    if check_port $port; then
        echo "⚠️  端口 $port 已被占用 ($service_name)"
        echo ""
        echo "当前占用端口 $port 的进程："
        lsof -i :$port
        echo ""
        
        local pid=$(get_port_pid $port)
        
        if [[ "$FORCE_RESTART" == "true" ]]; then
            echo "强制重启模式：自动终止现有进程"
            kill_process $pid $service_name
            return $?
        else
            read -p "是否要终止现有 $service_name 进程并重新启动？(y/n) " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill_process $pid $service_name
                return $?
            else
                echo "跳过 $service_name 启动"
                return 1
            fi
        fi
    else
        echo "✓ 端口 $port 可用 ($service_name)"
        return 0
    fi
}

# 函数：清理所有残留进程（只清理本项目相关进程）
cleanup_all_processes() {
    echo "================================"
    echo "清理残留进程"
    echo "================================"

    # 1. 清理所有名为 "./edge" 的进程（后端服务）
    local edge_pids=$(pgrep -f "^\./edge" 2>/dev/null)
    if [ ! -z "$edge_pids" ]; then
        echo "发现残留的edge后端进程: $edge_pids"
        pkill -9 -f "^\./edge" 2>/dev/null
        sleep 1
        echo "✓ edge后端进程已清理"
    else
        echo "✓ 没有发现edge后端残留进程"
    fi

    # 2. 清理本目录下的前端服务进程（python3 -m http.server 8000）
    local frontend_pids=$(ps aux | grep "python3 -m http.server $FRONTEND_PORT" | grep -v grep | awk '{print $2}')
    if [ ! -z "$frontend_pids" ]; then
        echo "发现残留的前端服务进程: $frontend_pids"
        echo "$frontend_pids" | xargs kill -9 2>/dev/null
        sleep 1
        echo "✓ 前端服务进程已清理"
    else
        echo "✓ 没有发现前端服务残留进程"
    fi

    # 3. 如果端口仍被占用，尝试释放（最后的保险措施）
    local backend_pid=$(lsof -t -i :$BACKEND_PORT 2>/dev/null | head -1)
    if [ ! -z "$backend_pid" ]; then
        echo "⚠️  后端端口 $BACKEND_PORT 仍被进程 $backend_pid 占用，强制释放"
        kill -9 $backend_pid 2>/dev/null
        sleep 1

        # 再次检查
        if lsof -i :$BACKEND_PORT > /dev/null 2>&1; then
            echo "✗ 后端端口 $BACKEND_PORT 无法释放（可能需要sudo权限）"
        else
            echo "✓ 后端端口 $BACKEND_PORT 已释放"
        fi
    else
        echo "✓ 后端端口 $BACKEND_PORT 未被占用"
    fi

    local frontend_pid=$(lsof -t -i :$FRONTEND_PORT 2>/dev/null | head -1)
    if [ ! -z "$frontend_pid" ]; then
        echo "⚠️  前端端口 $FRONTEND_PORT 仍被进程 $frontend_pid 占用，强制释放"
        kill -9 $frontend_pid 2>/dev/null
        sleep 1

        # 再次检查
        if lsof -i :$FRONTEND_PORT > /dev/null 2>&1; then
            echo "✗ 前端端口 $FRONTEND_PORT 无法释放"
        else
            echo "✓ 前端端口 $FRONTEND_PORT 已释放"
        fi
    else
        echo "✓ 前端端口 $FRONTEND_PORT 未被占用"
    fi

    echo ""
}

# 判断二进制是否需要重新编译
needs_rebuild_backend() {
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        echo "强制重新编译后端服务..."
        return 0
    fi

    if [ ! -x "$BACKEND_BINARY" ]; then
        echo "未检测到现有二进制，准备首次编译..."
        return 0
    fi

    local tracked_dirs=("cmd" "internal" "api" "pkg")
    local changed_file=""

    for dir in "${tracked_dirs[@]}"; do
        if [ -d "$dir" ]; then
            changed_file=$(find "$dir" -type f -name "*.go" -newer "$BACKEND_BINARY" -print -quit)
            if [ -n "$changed_file" ]; then
                echo "检测到较新的源码: $changed_file"
                return 0
            fi
        fi
    done

    for file in go.mod go.sum; do
        if [ -f "$file" ] && [ "$file" -nt "$BACKEND_BINARY" ]; then
            echo "依赖文件更新: $file"
            return 0
        fi
    done

    return 1
}

# 函数：编译后端服务（强制重新编译）
build_backend() {
    echo "================================"
    echo "编译后端服务"
    echo "================================"

    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        echo "✗ 未找到 Go 编译器"
        echo ""
        echo "请安装 Go 1.18+ 版本："
        echo "  Ubuntu/Debian: sudo apt install golang-go"
        echo "  或访问: https://golang.org/dl/"
        return 1
    fi

    # 显示 Go 版本
    local go_version=$(go version)
    echo "Go 版本: $go_version"
    echo ""

    # 检查主程序文件是否存在
    if [ ! -f "cmd/edge/main.go" ]; then
        echo "✗ 主程序文件不存在: cmd/edge/main.go"
        return 1
    fi

    # 总是强制重新编译（确保使用最新代码）
    echo "强制重新编译后端服务（确保使用最新代码）..."

    # if ! needs_rebuild_backend; then
    #     echo "✓ 检测到现有二进制已是最新，跳过编译"
    #     echo "  提示: 使用 --rebuild 可强制重新编译"
    #     echo ""
    #     return 0
    # fi

    echo "正在编译后端服务..."
    echo "命令: CGO_ENABLED=1 go build -o edge cmd/edge/main.go"
    echo ""

    if CGO_ENABLED=1 go build -o edge cmd/edge/main.go; then
        echo ""
        echo "✓ 后端编译成功"

        # 显示二进制文件信息
        if [ -f "$BACKEND_BINARY" ]; then
            local file_size=$(ls -lh "$BACKEND_BINARY" | awk '{print $5}')
            local file_time=$(stat -c "%y" "$BACKEND_BINARY" | cut -d'.' -f1)
            echo "  文件大小: $file_size"
            echo "  编译时间: $file_time"
        fi

        echo ""
        return 0
    else
        echo ""
        echo "✗ 后端编译失败"
        echo ""
        echo "常见问题排查："
        echo "  1. 检查 Go 版本是否 >= 1.18"
        echo "  2. 运行: go mod tidy"
        echo "  3. 检查 SQLite 依赖: apt install build-essential"
        return 1
    fi
}

# 函数：启动Mosquitto TLS Broker
start_mqtt_tls_broker() {
    echo "================================"
    echo "启动Mosquitto TLS Broker"
    echo "================================"

    local MQTT_TLS_PORT=8883
    local MQTT_TLS_CONFIG="./configs/mosquitto_tls.conf"

    mkdir -p logs/mosquitto logs/mosquitto/data

    # 检查配置文件是否存在
    if [ ! -f "$MQTT_TLS_CONFIG" ]; then
        echo "✗ Mosquitto TLS配置文件不存在: $MQTT_TLS_CONFIG"
        return 1
    fi

    # 检查8883端口是否已被占用
    if lsof -i :$MQTT_TLS_PORT > /dev/null 2>&1; then
        local mqtt_tls_pid=$(lsof -t -i :$MQTT_TLS_PORT 2>/dev/null | head -1)
        echo "✓ Mosquitto TLS已在运行 (PID: $mqtt_tls_pid, 端口: $MQTT_TLS_PORT)"
        return 0
    fi

    # 启动Mosquitto TLS
    echo "正在启动Mosquitto TLS Broker..."
    nohup mosquitto -c "$MQTT_TLS_CONFIG" > logs/mosquitto/mosquitto_tls.log 2>&1 &
    local mqtt_tls_pid=$!

    # 等待启动（减少等待时间）
    sleep 1

    # 检查是否成功启动
    if lsof -i :$MQTT_TLS_PORT > /dev/null 2>&1; then
        echo "✓ Mosquitto TLS启动成功 (PID: $mqtt_tls_pid, 端口: $MQTT_TLS_PORT)"
        return 0
    else
        echo "✗ Mosquitto TLS启动失败"
        echo "  请检查日志: logs/mosquitto/mosquitto_tls.log"
        return 1
    fi
}

# 函数：检查MQTT Broker是否运行
check_mqtt_broker() {
    echo "检查MQTT Broker状态..."

    # 仅检查8883端口（TLS）- 系统只使用TLS加密通信
    if lsof -i :8883 > /dev/null 2>&1; then
        echo "✓ MQTT Broker (TLS)正在运行 (端口 8883)"
        local mqtt_tls_pid=$(lsof -t -i :8883 2>/dev/null | head -1)
        if [ ! -z "$mqtt_tls_pid" ]; then
            local mqtt_tls_info=$(ps -p $mqtt_tls_pid -o comm= 2>/dev/null)
            echo "  进程: $mqtt_tls_info (PID: $mqtt_tls_pid)"
        fi
        echo ""
        return 0
    else
        echo "⚠️  MQTT Broker (TLS)未运行 (端口 8883)"
        echo ""
        echo "尝试启动Mosquitto TLS Broker..."
        if start_mqtt_tls_broker; then
            echo ""
            return 0
        else
            echo ""
            echo "❌ 无法启动MQTT TLS Broker"
            echo ""
            echo "Edge后端需要MQTT TLS Broker才能接收传感器数据。"
            echo "请检查配置文件: ./configs/mosquitto_tls.conf"
            echo ""

            if [[ "$FORCE_RESTART" == "true" ]]; then
                echo "⚠️  强制启动模式：继续启动后端（但MQTT功能将不可用）"
                return 0
            else
                read -p "是否继续启动后端服务？(y/n) " -n 1 -r
                echo ""
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    return 0
                else
                    echo "已取消启动"
                    return 1
                fi
            fi
        fi
    fi
}

# 函数：启动后端服务
start_backend() {
    echo "================================"
    echo "启动后端服务"
    echo "================================"

    # 先编译后端服务
    if ! build_backend; then
        echo "编译失败，无法启动后端服务"
        return 1
    fi

    # 检查配置文件是否存在
    if [ ! -f "$BACKEND_CONFIG" ]; then
        echo "✗ 配置文件不存在: $BACKEND_CONFIG"
        return 1
    fi
    
    # 检查MQTT Broker状态（可选但推荐）
    if ! check_mqtt_broker; then
        return 1
    fi
    
    echo ""

    # 处理端口冲突
    if ! handle_port_conflict $BACKEND_PORT "后端服务"; then
        return 1
    fi

    echo "正在启动后端服务..."
    echo "命令: $BACKEND_BINARY -config $BACKEND_CONFIG"
    echo ""

    # ⚠️ 关键优化：清理端口并立即启动服务（不给IDE重新占用的机会）
    local max_attempts=3
    local attempt=1
    local backend_pid=""

    while [ $attempt -le $max_attempts ]; do
        echo "尝试启动后端服务 (第 $attempt 次)..."

        # 清理端口
        local port_pid=$(lsof -t -i :$BACKEND_PORT 2>/dev/null | head -1)
        if [ ! -z "$port_pid" ]; then
            echo "  清理占用端口的进程 (PID: $port_pid)..."
            kill -9 $port_pid 2>/dev/null
        fi

        # 立即启动（不sleep，争取时间）
        nohup $BACKEND_BINARY -config $BACKEND_CONFIG > logs/backend.log 2>&1 &
        backend_pid=$!

        # 短暂等待确认启动（减少等待时间）
        sleep 1

        # 检查是否成功启动
        if check_port $BACKEND_PORT; then
            echo "✓ 后端服务成功占用端口"
            break
        else
            echo "✗ 启动失败，端口未被占用"
            attempt=$((attempt + 1))
        fi
    done

    if [ $attempt -gt $max_attempts ]; then
        echo "✗ 经过 $max_attempts 次尝试，后端服务启动失败"
        return 1
    fi
    
    # 等待服务启动（减少等待时间，使用轮询检查）
    echo "等待后端服务启动..."
    local health_check_attempts=0
    local max_health_attempts=10
    
    while [ $health_check_attempts -lt $max_health_attempts ]; do
        sleep 0.5
        health_check_attempts=$((health_check_attempts + 1))
        
        # 检查端口和健康状态
        if check_port $BACKEND_PORT; then
            # 健康检查（设置5秒超时）
            if curl -s --max-time 5 --noproxy '*' http://localhost:$BACKEND_PORT/health > /dev/null 2>&1; then
                echo "✓ 后端服务启动成功 (PID: $backend_pid)"
                echo "✓ 健康检查通过"
                break
            fi
        fi
        
        if [ $health_check_attempts -eq $max_health_attempts ]; then
            if check_port $BACKEND_PORT; then
                echo "⚠️  后端服务已启动但健康检查超时"
            else
                echo "✗ 后端服务启动失败"
                return 1
            fi
        fi
    done
    
    echo ""
    return 0
}

# 函数：启动前端服务
start_frontend() {
    echo "================================"
    echo "启动前端服务"
    echo "================================"
    
    # 检查前端目录是否存在
    if [ ! -d "$FRONTEND_DIR" ]; then
        echo "✗ 前端目录不存在: $FRONTEND_DIR"
        return 1
    fi
    
    # 检查前端入口文件
    if [ ! -f "$FRONTEND_DIR/index.html" ]; then
        echo "✗ 前端入口文件不存在: $FRONTEND_DIR/index.html"
        return 1
    fi
    
    # 处理端口冲突
    if ! handle_port_conflict $FRONTEND_PORT "前端服务"; then
        return 1
    fi
    
    # 进入前端目录
    cd "$FRONTEND_DIR" || {
        echo "✗ 无法进入前端目录: $FRONTEND_DIR"
        return 1
    }
    
    # 检查Python是否可用
    if command -v python3 &> /dev/null; then
        echo "正在启动前端服务..."
        echo "使用 Python3 HTTP服务器"
        echo ""
        
        # 启动前端服务（后台运行）
        nohup python3 -m http.server $FRONTEND_PORT > ../logs/frontend.log 2>&1 &
        local frontend_pid=$!
        
        # 返回项目根目录
        cd ..
        
        # 等待服务启动（使用轮询检查，更快响应）
        echo "等待前端服务启动..."
        local frontend_check_attempts=0
        local max_frontend_attempts=6
        
        while [ $frontend_check_attempts -lt $max_frontend_attempts ]; do
            sleep 0.5
            frontend_check_attempts=$((frontend_check_attempts + 1))
            
            if check_port $FRONTEND_PORT; then
                # 简单的HTTP检查（设置3秒超时）
                if curl -s --max-time 3 -o /dev/null -w "%{http_code}" http://localhost:$FRONTEND_PORT | grep -q "200"; then
                    echo "✓ 前端服务启动成功 (PID: $frontend_pid)"
                    break
                fi
            fi
            
            if [ $frontend_check_attempts -eq $max_frontend_attempts ]; then
                if check_port $FRONTEND_PORT; then
                    echo "⚠️  前端服务已启动但HTTP检查超时"
                else
                    echo "✗ 前端服务启动失败"
                    return 1
                fi
            fi
        done
        
    elif command -v python &> /dev/null; then
        echo "正在启动前端服务..."
        echo "使用 Python2 HTTP服务器"
        echo ""
        
        # 启动前端服务（后台运行）
        nohup python -m SimpleHTTPServer $FRONTEND_PORT > ../logs/frontend.log 2>&1 &
        local frontend_pid=$!
        
        # 返回项目根目录
        cd ..
        
        # 等待服务启动（使用轮询检查）
        local frontend_check_attempts=0
        local max_frontend_attempts=6
        
        while [ $frontend_check_attempts -lt $max_frontend_attempts ]; do
            sleep 0.5
            frontend_check_attempts=$((frontend_check_attempts + 1))
            
            if check_port $FRONTEND_PORT; then
                echo "✓ 前端服务启动成功 (PID: $frontend_pid)"
                break
            fi
            
            if [ $frontend_check_attempts -eq $max_frontend_attempts ]; then
                echo "✗ 前端服务启动失败"
                return 1
            fi
        done
    else
        echo "✗ 未找到Python，无法启动HTTP服务器"
        echo ""
        echo "请安装Python或使用其他方式访问："
        echo "  1. 安装Python: sudo apt install python3"
        echo "  2. 使用Node.js: npm install -g serve && serve -p $FRONTEND_PORT"
        echo "  3. 直接在浏览器打开: file://$(pwd)/index.html"
        cd ..
        return 1
    fi
    
    echo ""
    return 0
}

# 函数：显示服务状态
show_status() {
    echo "================================"
    echo "服务状态"
    echo "================================"
    
    # 检查后端服务
    if check_port $BACKEND_PORT; then
        local backend_pid=$(get_port_pid $BACKEND_PORT)
        echo "✓ 后端服务: 运行中 (PID: $backend_pid, 端口: $BACKEND_PORT)"
    else
        echo "✗ 后端服务: 未运行"
    fi
    
    # 检查前端服务
    if check_port $FRONTEND_PORT; then
        local frontend_pid=$(get_port_pid $FRONTEND_PORT)
        echo "✓ 前端服务: 运行中 (PID: $frontend_pid, 端口: $FRONTEND_PORT)"
    else
        echo "✗ 前端服务: 未运行"
    fi
    
    echo ""
    echo "访问地址："
    echo "  前端界面: http://localhost:$FRONTEND_PORT"
    echo "  后端API:  http://localhost:$BACKEND_PORT"
    echo ""
    echo "日志文件："
    echo "  后端日志: logs/backend.log"
    echo "  前端日志: logs/frontend.log"
    echo ""
    echo "💡 温馨提示："
    echo "  访问前端时请使用 Ctrl+Shift+R 强制刷新浏览器缓存"
    echo "  确保加载最新版本的 JavaScript 和 CSS 文件"
    echo ""
}

# 主执行流程
main() {
    # 确保日志目录存在
    mkdir -p logs

    # 第一步：清理所有残留进程（确保干净的启动环境）
    cleanup_all_processes

    # 第二步：启动后端服务（会强制重新编译）
    if ! start_backend; then
        echo "后端服务启动失败，继续尝试启动前端服务..."
    fi

    # 第三步：启动前端服务
    if ! start_frontend; then
        echo "前端服务启动失败"
    fi
    
    # 显示最终状态
    show_status
    
    # 检查是否至少有一个服务启动成功
    if check_port $BACKEND_PORT || check_port $FRONTEND_PORT; then
        echo "🎉 Edge系统启动完成！"
        
        if check_port $BACKEND_PORT && check_port $FRONTEND_PORT; then
            echo ""
            echo "💡 提示："
            echo "  - 使用 Ctrl+C 停止当前脚本（不会停止后台服务）"
            echo "  - 要停止所有服务，请运行: pkill -f 'edge|python.*http.server.*$FRONTEND_PORT'"
            echo "  - 查看实时日志: tail -f logs/backend.log 或 tail -f logs/frontend.log"
        fi
        
        return 0
    else
        echo "❌ 所有服务启动失败"
        return 1
    fi
}

# 脚本使用说明
show_help() {
    echo "使用方法："
    echo "  $0              # 交互模式启动"
    echo "  $0 --force      # 强制重启模式（自动终止占用端口的进程）"
    echo "  $0 -f           # 强制重启模式（简写）"
    echo "  $0 --rebuild    # 强制重新编译后端二进制"
    echo "  $0 -r           # 强制重新编译后端二进制（简写）"
    echo "  $0 --help       # 显示帮助信息"
    echo ""
    echo "服务端口："
    echo "  后端服务: $BACKEND_PORT"
    echo "  前端服务: $FRONTEND_PORT"
    echo "  MQTT TLS: 8883 (已弃用1883明文端口)"
}

# 处理命令行参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h)
            show_help
            exit 0
            ;;
        --force|-f)
            FORCE_RESTART=true
            ;;
        --rebuild|-r)
            FORCE_REBUILD=true
            ;;
        *)
            echo "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
    shift
done

# 执行主程序
main
exit $?
