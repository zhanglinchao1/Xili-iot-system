#!/bin/bash
# Cloud端 Docker 启动脚本

set -e

echo "=========================================="
echo "Cloud端 Docker 服务启动脚本"
echo "=========================================="

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "错误: Docker 服务未运行，请先启动 Docker"
    exit 1
fi

# 设置 Go 代理环境变量（用于构建）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

# 启动基础服务（PostgreSQL, Redis, MQTT）
echo ""
echo "1. 启动基础服务..."
docker-compose up -d postgres redis mqtt

# 等待基础服务就绪
echo ""
echo "2. 等待基础服务就绪..."
sleep 5

# 检查服务状态
echo ""
echo "3. 检查服务状态..."
docker-compose ps

# 构建后端服务
echo ""
echo "4. 构建后端服务（这可能需要较长时间，请耐心等待）..."
echo "   提示: Go 模块下载可能需要几分钟时间"
DOCKER_BUILDKIT=1 docker-compose build backend

# 启动后端服务
echo ""
echo "5. 启动后端服务..."
docker-compose up -d backend

# 等待后端服务就绪
echo ""
echo "6. 等待后端服务就绪..."
sleep 10

# 构建并启动前端服务
echo ""
echo "7. 构建并启动前端服务..."
docker-compose build frontend
docker-compose up -d frontend

# 最终状态检查
echo ""
echo "=========================================="
echo "所有服务启动完成！"
echo "=========================================="
docker-compose ps

echo ""
echo "服务访问地址:"
echo "  - 前端: http://localhost:5173"
echo "  - 后端API: http://localhost:8003/api/v1"
echo "  - 健康检查: http://localhost:8003/health"
echo ""
echo "数据库端口: 5433 (避免与系统 PostgreSQL 冲突)"
echo "Redis端口: 6379"
echo "MQTT端口: 1883 (非TLS), 8883 (TLS), 9001 (WebSocket)"
echo ""

