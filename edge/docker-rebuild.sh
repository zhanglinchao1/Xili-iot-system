#!/bin/bash
# =============================================================================
# Edge端 Docker 重建脚本
# 功能: 删除旧容器 -> 重新构建镜像 -> 创建并启动新容器
# 使用方法: ./docker-rebuild.sh [--no-cache]
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
CONTAINER_NAME="edge-system"
IMAGE_NAME="edge-system:latest"
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
DOCKER_COMPOSE_FILE="$PROJECT_DIR/docker-compose.yml"

# 解析参数
NO_CACHE=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-cache)
            NO_CACHE="--no-cache"
            ;;
        --help|-h)
            echo "使用方法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --no-cache    不使用缓存重新构建镜像"
            echo "  --help, -h    显示帮助信息"
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
echo -e "${BLUE}Edge端 Docker 重建脚本${NC}"
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

# 检查端口冲突
echo -e "${YELLOW}[1.5/6] 检查端口冲突...${NC}"
if ss -tuln | grep -q ":8883 "; then
    MQTT_CONTAINER=$(docker ps --filter "publish=8883" --format "{{.Names}}" | head -1)
    if [ -n "$MQTT_CONTAINER" ] && [ "$MQTT_CONTAINER" != "edge-mqtt" ]; then
        echo -e "${YELLOW}⚠️  端口8883被其他容器占用: $MQTT_CONTAINER${NC}"
        echo "Edge的MQTT需要使用8883端口"
        echo ""
        read -p "是否停止 $MQTT_CONTAINER 并继续? (y/n) " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker stop "$MQTT_CONTAINER" 2>/dev/null || true
            echo -e "${GREEN}✓ 已停止 $MQTT_CONTAINER${NC}"
        else
            echo "已取消"
            exit 1
        fi
    else
        echo -e "${GREEN}✓ 端口8883可用或已被edge-mqtt使用${NC}"
    fi
else
    echo -e "${GREEN}✓ 端口8883可用${NC}"
fi
echo ""

# 进入项目目录
cd "$PROJECT_DIR"

# 停止并删除旧容器
echo -e "${YELLOW}[2/6] 停止并删除旧容器...${NC}"
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "正在停止容器 ${CONTAINER_NAME}..."
    docker stop "${CONTAINER_NAME}" 2>/dev/null || true
    echo "正在删除容器 ${CONTAINER_NAME}..."
    docker rm "${CONTAINER_NAME}" 2>/dev/null || true
    echo -e "${GREEN}✓ 旧容器已删除${NC}"
else
    echo -e "${GREEN}✓ 没有找到旧容器，跳过删除步骤${NC}"
fi
echo ""

# 删除旧镜像（可选，释放空间）
echo -e "${YELLOW}[3/6] 构建新镜像...${NC}"
if [ -n "$NO_CACHE" ]; then
    echo "使用 --no-cache 模式构建..."
fi

# 设置 Go 代理环境变量（加速构建，优先使用国内镜像）
export GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct
export GOSUMDB=sum.golang.google.cn

# 检查 Dockerfile 是否存在
if [ -f "$PROJECT_DIR/deployments/Dockerfile" ]; then
    DOCKERFILE_PATH="$PROJECT_DIR/deployments/Dockerfile"
elif [ -f "$PROJECT_DIR/Dockerfile" ]; then
    DOCKERFILE_PATH="$PROJECT_DIR/Dockerfile"
else
    echo -e "${RED}错误: 未找到 Dockerfile${NC}"
    exit 1
fi

echo "使用 Dockerfile: $DOCKERFILE_PATH"
echo "构建镜像: $IMAGE_NAME"
echo ""

# 构建镜像
if [ "$USE_BUILDKIT" -eq 1 ]; then
    DOCKER_BUILDKIT=1 docker build $NO_CACHE -t "$IMAGE_NAME" -f "$DOCKERFILE_PATH" "$PROJECT_DIR"
else
    DOCKER_BUILDKIT=0 docker build $NO_CACHE -t "$IMAGE_NAME" -f "$DOCKERFILE_PATH" "$PROJECT_DIR"
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 镜像构建成功${NC}"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi
echo ""

# 使用 docker compose 启动新容器
echo -e "${YELLOW}[4/6] 启动新容器...${NC}"
if [ -f "$DOCKER_COMPOSE_FILE" ]; then
    echo "使用 docker compose 启动容器..."
    docker compose -f "$DOCKER_COMPOSE_FILE" up -d
else
    echo -e "${RED}错误: 未找到 docker-compose.yml 文件${NC}"
    exit 1
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 容器启动成功${NC}"
else
    echo -e "${RED}✗ 容器启动失败${NC}"
    exit 1
fi
echo ""

# 等待容器完全启动
echo -e "${YELLOW}[5/7] 等待服务就绪...${NC}"
sleep 3

# 显示容器状态
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}容器状态${NC}"
echo -e "${BLUE}========================================${NC}"
docker ps --filter "name=${CONTAINER_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 数据库验证
echo ""
echo -e "${YELLOW}[6/7] 验证数据库初始化...${NC}"

# 等待数据库初始化（容器启动时自动执行）
echo "等待数据库初始化完成..."
sleep 5

# 检查验证脚本是否存在
VERIFY_SCRIPT="$PROJECT_DIR/scripts/verify_database.sh"
if [ -f "$VERIFY_SCRIPT" ]; then
    echo "运行数据库验证脚本..."
    echo ""

    if bash "$VERIFY_SCRIPT" --db-path "$PROJECT_DIR/data/edge.db"; then
        echo ""
        echo -e "${GREEN}✓ 数据库验证通过${NC}"
    else
        echo ""
        echo -e "${YELLOW}⚠️  数据库验证发现问题${NC}"
        echo -e "${YELLOW}提示: 容器已启动，数据库将在首次请求时完成初始化${NC}"
        echo ""
        read -p "是否继续? (y/n) " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}已取消${NC}"
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}⚠️  未找到数据库验证脚本: $VERIFY_SCRIPT${NC}"
    echo "跳过数据库验证步骤"
fi
echo ""

# 健康检查
echo -e "${YELLOW}[7/7] 执行服务健康检查...${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0
HEALTH_URL="http://localhost:8001/health"

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    
    if curl -s --max-time 5 "$HEALTH_URL" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 健康检查通过 (尝试 $ATTEMPT/$MAX_ATTEMPTS)${NC}"
        break
    fi
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${YELLOW}⚠️  健康检查超时，服务可能仍在启动中${NC}"
        echo "请查看日志: docker logs ${CONTAINER_NAME}"
    else
        echo "等待服务就绪... ($ATTEMPT/$MAX_ATTEMPTS)"
        sleep 2
    fi
done

# 显示日志（最后10行）
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}最新日志 (最后10行)${NC}"
echo -e "${BLUE}========================================${NC}"
docker logs --tail 10 "${CONTAINER_NAME}" 2>&1 || true

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}🎉 Edge Docker 重建完成!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "访问地址:"
echo "  - 后端API: http://localhost:8001"
echo "  - 健康检查: http://localhost:8001/health"
echo ""
echo "常用命令:"
echo "  查看日志: docker logs -f ${CONTAINER_NAME}"
echo "  进入容器: docker exec -it ${CONTAINER_NAME} sh"
echo "  停止容器: docker stop ${CONTAINER_NAME}"
echo ""

