#!/bin/bash
# =============================================================================
# Cloud端 Docker 重建脚本
# 功能: 删除旧容器 -> 重新构建镜像 -> 创建并启动新容器
# 使用方法: ./docker-rebuild.sh [--no-cache] [--backend-only] [--frontend-only]
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
DOCKER_COMPOSE_FILE="$PROJECT_DIR/docker compose.yml"

# 容器名称
BACKEND_CONTAINER="cloud-backend"
FRONTEND_CONTAINER="cloud-frontend"
POSTGRES_CONTAINER="cloud-postgres"
REDIS_CONTAINER="cloud-redis"
MQTT_CONTAINER="cloud-mqtt"

# 解析参数
NO_CACHE=""
BUILD_BACKEND=true
BUILD_FRONTEND=true
REBUILD_BASE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-cache)
            NO_CACHE="--no-cache"
            ;;
        --backend-only)
            BUILD_FRONTEND=false
            ;;
        --frontend-only)
            BUILD_BACKEND=false
            ;;
        --rebuild-base)
            REBUILD_BASE=true
            ;;
        --help|-h)
            echo "使用方法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --no-cache       不使用缓存重新构建镜像"
            echo "  --backend-only   仅重建后端服务"
            echo "  --frontend-only  仅重建前端服务"
            echo "  --rebuild-base   同时重建基础服务(postgres/redis/mqtt)"
            echo "  --help, -h       显示帮助信息"
            exit 0
            ;;
        *)
            echo -e "${RED}未知参数: $1${NC}"
            exit 1
            ;;
    esac
    shift
done

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Cloud端 Docker 重建脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 Docker 是否运行
echo -e "${YELLOW}[1/6] 检查 Docker 服务...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}错误: Docker 服务未运行，请先启动 Docker${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker 服务正常${NC}"

# 检查 BuildKit/buildx 是否可用
USE_BUILDKIT=0
if docker buildx version > /dev/null 2>&1; then
    USE_BUILDKIT=1
    echo -e "${GREEN}✓ BuildKit/buildx 可用，将使用加速构建${NC}"
else
    echo -e "${YELLOW}⚠️  BuildKit/buildx 不可用，将使用传统构建方式${NC}"
fi

echo ""

# 进入项目目录
cd "$PROJECT_DIR"

# 设置 Go 代理环境变量（加速构建，使用多个备选源）
export GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,https://goproxy.qihoo.net,direct
export GOSUMDB=sum.golang.google.cn
export GONOPROXY=
export GONOSUMDB=
export GOPRIVATE=

# 提示：Dockerfile中已配置使用镜像代理加速拉取基础镜像
# 镜像代理地址: nn7fm5xf4cl9hj.xuanyuan.run

# 停止并删除旧容器
echo -e "${YELLOW}[2/6] 停止并删除旧容器...${NC}"

# 删除后端容器
if [ "$BUILD_BACKEND" = true ]; then
    if docker ps -a --format '{{.Names}}' | grep -q "^${BACKEND_CONTAINER}$"; then
        echo "正在停止并删除 ${BACKEND_CONTAINER}..."
        docker stop "${BACKEND_CONTAINER}" 2>/dev/null || true
        docker rm "${BACKEND_CONTAINER}" 2>/dev/null || true
        echo -e "${GREEN}✓ ${BACKEND_CONTAINER} 已删除${NC}"
    else
        echo -e "${GREEN}✓ 没有找到 ${BACKEND_CONTAINER}，跳过${NC}"
    fi
fi

# 删除前端容器
if [ "$BUILD_FRONTEND" = true ]; then
    if docker ps -a --format '{{.Names}}' | grep -q "^${FRONTEND_CONTAINER}$"; then
        echo "正在停止并删除 ${FRONTEND_CONTAINER}..."
        docker stop "${FRONTEND_CONTAINER}" 2>/dev/null || true
        docker rm "${FRONTEND_CONTAINER}" 2>/dev/null || true
        echo -e "${GREEN}✓ ${FRONTEND_CONTAINER} 已删除${NC}"
    else
        echo -e "${GREEN}✓ 没有找到 ${FRONTEND_CONTAINER}，跳过${NC}"
    fi
fi

# 如果需要重建基础服务
if [ "$REBUILD_BASE" = true ]; then
    echo "正在停止基础服务..."
    docker compose stop postgres redis mqtt 2>/dev/null || true
fi

echo ""

# 启动基础服务
echo -e "${YELLOW}[3/6] 确保基础服务运行...${NC}"
docker compose up -d postgres redis mqtt
echo -e "${GREEN}✓ 基础服务已启动${NC}"

# 等待基础服务就绪
echo "等待基础服务就绪..."
sleep 5

# 检查基础服务健康状态
echo -e "${YELLOW}检查基础服务状态...${NC}"
docker compose ps postgres redis mqtt
echo ""

# 检测容器内网络连接（前后端共享）
echo "检测容器内网络连接..."
# 优化：基础服务已正常运行，说明Docker网络正常，使用快速检测
# 如果alpine镜像存在，使用ping快速检测（最多2秒）；否则直接使用默认网络模式
USE_HOST_NETWORK=false
if docker images alpine:latest --format "{{.Repository}}:{{.Tag}}" 2>/dev/null | grep -q "alpine:latest"; then
    # 使用ping检测，超时1秒，总超时2秒
    if timeout 2 docker run --rm --network bridge alpine:latest sh -c "timeout 1 ping -c 1 -W 1 8.8.8.8 >/dev/null 2>&1" 2>/dev/null; then
        echo -e "${GREEN}✓ 容器网络连接正常${NC}"
    else
        echo -e "${YELLOW}⚠️  容器网络连接异常，将使用host网络模式构建${NC}"
        USE_HOST_NETWORK=true
    fi
else
    # 如果alpine镜像不存在，跳过检测，使用默认网络模式（避免拉取镜像耗时）
    # 基础服务已正常运行，说明网络正常
    echo -e "${GREEN}✓ 使用默认网络模式（基础服务已正常运行）${NC}"
fi
echo ""

# 构建后端镜像
if [ "$BUILD_BACKEND" = true ]; then
    echo -e "${YELLOW}[4/6] 构建后端镜像...${NC}"
    if [ -n "$NO_CACHE" ]; then
        echo "使用 --no-cache 模式构建..."
    fi
    
    # 根据网络情况选择构建方式
    if [ "$USE_HOST_NETWORK" = true ]; then
        # 使用host网络模式构建（容器可以使用主机网络）
        echo "使用host网络模式构建..."
        if [ "$USE_BUILDKIT" -eq 1 ]; then
            DOCKER_BUILDKIT=1 docker build --network=host $NO_CACHE -t cloud-backend:latest -f Dockerfile.backend .
        else
            DOCKER_BUILDKIT=0 docker build --network=host $NO_CACHE -t cloud-backend:latest -f Dockerfile.backend .
        fi
    else
        # 使用默认网络模式
        if [ "$USE_BUILDKIT" -eq 1 ]; then
            DOCKER_BUILDKIT=1 docker compose build $NO_CACHE backend
        else
            DOCKER_BUILDKIT=0 docker compose build $NO_CACHE backend
        fi
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 后端镜像构建成功${NC}"
    else
        echo -e "${RED}✗ 后端镜像构建失败${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}[4/6] 跳过后端构建${NC}"
fi
echo ""

# 构建前端镜像
if [ "$BUILD_FRONTEND" = true ]; then
    echo -e "${YELLOW}[5/6] 构建前端镜像...${NC}"
    if [ -n "$NO_CACHE" ]; then
        echo "使用 --no-cache 模式构建..."
    fi
    
    # 使用与后端相同的网络模式
    if [ "$USE_HOST_NETWORK" = true ]; then
        echo "使用host网络模式构建前端..."
        if [ "$USE_BUILDKIT" -eq 1 ]; then
            DOCKER_BUILDKIT=1 docker build --network=host $NO_CACHE -t cloud-frontend:latest -f Dockerfile.frontend .
        else
            DOCKER_BUILDKIT=0 docker build --network=host $NO_CACHE -t cloud-frontend:latest -f Dockerfile.frontend .
        fi
    else
        # 使用默认网络模式
        if [ "$USE_BUILDKIT" -eq 1 ]; then
            DOCKER_BUILDKIT=1 docker compose build $NO_CACHE frontend
        else
            DOCKER_BUILDKIT=0 docker compose build $NO_CACHE frontend
        fi
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 前端镜像构建成功${NC}"
    else
        echo -e "${RED}✗ 前端镜像构建失败${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}[5/6] 跳过前端构建${NC}"
fi
echo ""

# 启动所有服务
echo -e "${YELLOW}[6/6] 启动服务...${NC}"
docker compose up -d
echo -e "${GREEN}✓ 所有服务已启动${NC}"
echo ""

# 等待服务完全启动
echo "等待服务就绪..."
sleep 5

# 显示容器状态
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}容器状态${NC}"
echo -e "${BLUE}========================================${NC}"
docker compose ps

# 健康检查
echo ""
echo -e "${YELLOW}执行后端健康检查...${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0
HEALTH_URL="http://localhost:8003/health"

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    
    if curl -s --max-time 5 "$HEALTH_URL" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 后端健康检查通过 (尝试 $ATTEMPT/$MAX_ATTEMPTS)${NC}"
        break
    fi
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${YELLOW}⚠️  健康检查超时，服务可能仍在启动中${NC}"
        echo "请查看日志: docker logs ${BACKEND_CONTAINER}"
    else
        echo "等待后端服务就绪... ($ATTEMPT/$MAX_ATTEMPTS)"
        sleep 2
    fi
done

# 显示后端日志（最后10行）
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}后端最新日志 (最后10行)${NC}"
echo -e "${BLUE}========================================${NC}"
docker logs --tail 10 "${BACKEND_CONTAINER}" 2>&1 || true

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}🎉 Cloud Docker 重建完成!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "服务访问地址:"
echo "  - 前端界面: http://localhost:8002"
echo "  - 后端API: http://localhost:8003/api/v1"
echo "  - 健康检查: http://localhost:8003/health"
echo ""
echo "数据库端口:"
echo "  - PostgreSQL: 5433 (避免与系统冲突)"
echo "  - Redis: 6379"
echo ""
echo "MQTT端口:"
echo "  - TLS: 8884 (仅TLS，无明文端口)"
echo "  - WebSocket: 9001"
echo ""
echo "常用命令:"
echo "  查看所有日志: docker compose logs -f"
echo "  查看后端日志: docker logs -f ${BACKEND_CONTAINER}"
echo "  查看前端日志: docker logs -f ${FRONTEND_CONTAINER}"
echo "  进入后端容器: docker exec -it ${BACKEND_CONTAINER} sh"
echo "  停止所有服务: docker compose down"
echo ""

