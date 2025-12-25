#!/bin/bash

# ============================================
# Edge 系统 Docker 镜像构建脚本
# 无需源码，只需编译好的二进制文件
# ============================================

set -e

echo "========================================"
echo "Edge 系统 Docker 镜像构建"
echo "========================================"
echo ""

# 切换到项目根目录
cd "$(dirname "$0")/../.."
PROJECT_ROOT=$(pwd)

echo "📁 项目目录: $PROJECT_ROOT"
echo ""

# ============================================
# 步骤 1: 编译 Go 二进制文件
# ============================================
echo "🔧 步骤 1: 编译 Go 二进制文件..."
echo ""

# 设置 CGO（SQLite 需要）
export CGO_ENABLED=1

# 编译为 Linux amd64 二进制文件
echo "正在编译 Edge 服务..."
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o edge \
    cmd/edge/main.go

if [ ! -f "edge" ]; then
    echo "❌ 编译失败: edge 二进制文件不存在"
    exit 1
fi

echo "✅ 编译成功: edge"
ls -lh edge
echo ""

# ============================================
# 步骤 2: 准备 Docker 构建上下文
# ============================================
echo "🔧 步骤 2: 准备 Docker 构建上下文..."
echo ""

# 创建临时构建目录
BUILD_DIR="deployments/docker/build_context"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# 复制二进制文件
cp edge "$BUILD_DIR/"

# 复制配置文件
mkdir -p "$BUILD_DIR/configs"
cp configs/config.yaml "$BUILD_DIR/configs/"
cp configs/mosquitto_tls_container.conf "$BUILD_DIR/configs/"

# 复制证书（如果存在）
if [ -d "configs/certs" ]; then
    cp -r configs/certs "$BUILD_DIR/configs/"
fi

# 复制前端文件
if [ -d "web" ]; then
    cp -r web "$BUILD_DIR/"
    echo "✅ 前端文件已复制"
fi

# 复制 ZKP 密钥
if [ -f "auth_verifying.key" ]; then
    cp auth_verifying.key "$BUILD_DIR/"
fi

# 复制许可证文件（如果存在）
if [ -f "configs/license.lic" ]; then
    cp configs/license.lic "$BUILD_DIR/configs/"
    echo "✅ 许可证文件已复制"
fi

if [ -f "configs/vendor_pubkey.pem" ]; then
    cp configs/vendor_pubkey.pem "$BUILD_DIR/configs/"
    echo "✅ 厂商公钥已复制"
fi

# 复制启动脚本
cp deployments/docker/entrypoint.sh "$BUILD_DIR/"

# 复制 Dockerfile
cp deployments/docker/Dockerfile.production "$BUILD_DIR/Dockerfile"

echo "✅ 构建上下文准备完成"
echo ""

# ============================================
# 步骤 3: 构建 Docker 镜像
# ============================================
echo "🔧 步骤 3: 构建 Docker 镜像..."
echo ""

IMAGE_NAME="edge-system"
IMAGE_TAG="latest"
IMAGE_FULL_NAME="$IMAGE_NAME:$IMAGE_TAG"

cd "$BUILD_DIR"

# 使用 --network=host 解决 Docker 容器内 DNS 问题
docker build \
    --network=host \
    --tag "$IMAGE_FULL_NAME" \
    --label "version=1.0.0" \
    --label "build-date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    .

cd "$PROJECT_ROOT"

echo ""
echo "✅ 镜像构建成功: $IMAGE_FULL_NAME"
echo ""

# ============================================
# 步骤 4: 清理
# ============================================
echo "🔧 步骤 4: 清理临时文件..."
echo ""

rm -rf "$BUILD_DIR"
rm -f edge  # 删除编译的二进制文件

echo "✅ 清理完成"
echo ""

# ============================================
# 步骤 5: 显示镜像信息
# ============================================
echo "========================================"
echo "📊 镜像信息"
echo "========================================"
docker images | grep "$IMAGE_NAME"
echo ""

echo "========================================"
echo "🎉 构建完成！"
echo "========================================"
echo ""
echo "🚀 下一步操作："
echo ""
echo "1. 启动容器:"
echo "   cd deployments/docker"
echo "   docker-compose up -d"
echo ""
echo "2. 查看日志:"
echo "   docker-compose logs -f"
echo ""
echo "3. 停止容器:"
echo "   docker-compose down"
echo ""
echo "4. 导出镜像（用于其他机器）:"
echo "   docker save -o edge-system.tar $IMAGE_FULL_NAME"
echo ""
echo "5. 导入镜像（在目标机器）:"
echo "   docker load -i edge-system.tar"
echo ""
