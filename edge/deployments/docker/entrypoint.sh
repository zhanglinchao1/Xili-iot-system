#!/bin/sh

# ============================================
# Edge 系统容器启动脚本
# ============================================

set -e

echo "========================================"
echo "Edge 系统容器启动"
echo "========================================"

# 检查配置文件
if [ ! -f /app/configs/config.yaml ]; then
    echo "❌ 错误: 配置文件 /app/configs/config.yaml 不存在"
    exit 1
fi

echo "✅ 配置文件检查通过"

# 检查 ZKP 验证密钥
if [ ! -f /app/auth_verifying.key ]; then
    echo "⚠️  警告: ZKP 验证密钥 /app/auth_verifying.key 不存在"
fi

# 启动 Mosquitto MQTT Broker（后台）
if [ -f /app/configs/mosquitto_tls.conf ]; then
    echo "🔧 启动 Mosquitto MQTT Broker..."

    # 确保日志目录存在并有权限
    mkdir -p /app/logs/mosquitto/data
    chmod 755 /app/logs/mosquitto

    # 启动 Mosquitto
    mosquitto -c /app/configs/mosquitto_tls.conf -d
    MOSQUITTO_PID=$!
    echo "   Mosquitto 启动命令已执行 (PID: $MOSQUITTO_PID)"

    # 等待 Mosquitto 真正就绪 (监听 8883 TLS 端口)
    echo "⏳ 等待 Mosquitto TLS Broker 就绪..."
    MAX_WAIT=30
    COUNTER=0
    while [ $COUNTER -lt $MAX_WAIT ]; do
        # 检查进程是否存在
        if pgrep mosquitto > /dev/null; then
            # 检查端口是否被监听（使用 ss 或 netstat）
            if ss -tuln | grep -q ":8883"; then
                echo "✅ Mosquitto TLS Broker 已就绪 (端口 8883 已监听)"
                # 再等待 3 秒确保完全就绪并接受连接
                echo "   等待3秒确保 Mosquitto 完全就绪..."
                sleep 3
                break
            fi
        fi
        echo "  等待中... ($COUNTER/$MAX_WAIT)"
        COUNTER=$((COUNTER+1))
        sleep 1
    done

    if [ $COUNTER -eq $MAX_WAIT ]; then
        echo "❌ 警告: Mosquitto 未能在 ${MAX_WAIT} 秒内就绪"
        echo "📋 尝试查看 Mosquitto 日志:"
        tail -20 /app/logs/mosquitto/mosquitto_tls.log 2>/dev/null || echo "  (日志文件不存在)"
        echo "❌ Mosquitto 启动失败,容器将退出"
        exit 1
    fi
else
    echo "⚠️  警告: Mosquitto 配置文件不存在，跳过启动"
fi

# 启动前端服务器（后台）
if [ -d /app/web ]; then
    echo "🌐 启动前端服务器 (端口 8000)..."
    cd /app/web
    python3 -m http.server 8000 > /app/logs/frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo "   前端服务器已启动 (PID: $FRONTEND_PID)"
    cd /app
fi

# 启动 Edge 服务（前台运行,保持容器运行）
echo "🚀 启动 Edge 服务..."
exec /app/edge -config /app/configs/config.yaml
