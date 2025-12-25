#!/bin/bash
# Cloud端服务测试脚本

echo "=========================================="
echo "Cloud端服务测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 测试容器状态
echo "1. 检查容器状态..."
docker-compose ps
echo ""

# 2. 测试前端访问
echo "2. 测试前端访问 (http://localhost:8002)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8002/)
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 前端访问正常 (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}✗ 前端访问失败 (HTTP $HTTP_CODE)${NC}"
fi
echo ""

# 3. 测试后端健康检查
echo "3. 测试后端健康检查 (http://localhost:8003/health)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8003/health)
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 后端健康检查正常 (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}✗ 后端健康检查失败 (HTTP $HTTP_CODE)${NC}"
fi
echo ""

# 4. 测试后端API配置接口
echo "4. 测试后端配置API (http://localhost:8003/api/v1/config)..."
CONFIG_RESPONSE=$(curl -s http://localhost:8003/api/v1/config)
if echo "$CONFIG_RESPONSE" | grep -q "tencent_map_key"; then
    echo -e "${GREEN}✓ 后端配置API正常${NC}"
    echo "腾讯地图Key: $(echo "$CONFIG_RESPONSE" | grep -o '"tencent_map_key":"[^"]*"' | cut -d'"' -f4)"
else
    echo -e "${RED}✗ 后端配置API返回异常${NC}"
    echo "响应: $CONFIG_RESPONSE"
fi
echo ""

# 5. 检查数据库连接
echo "5. 检查后端数据库连接..."
if docker logs cloud-backend --tail 50 | grep -q "PostgreSQL connection established.*postgres:5432"; then
    echo -e "${GREEN}✓ PostgreSQL连接正确 (postgres:5432)${NC}"
else
    echo -e "${YELLOW}⚠ PostgreSQL连接可能不正确${NC}"
    docker logs cloud-backend --tail 50 | grep "PostgreSQL"
fi
echo ""

# 6. 检查Redis连接
echo "6. 检查后端Redis连接..."
if docker logs cloud-backend --tail 50 | grep -q "Redis connection established.*redis:6379"; then
    echo -e "${GREEN}✓ Redis连接正确 (redis:6379)${NC}"
else
    echo -e "${YELLOW}⚠ Redis连接可能不正确${NC}"
    docker logs cloud-backend --tail 50 | grep "Redis"
fi
echo ""

# 7. 检查MQTT连接
echo "7. 检查后端MQTT连接..."
if docker logs cloud-backend --tail 50 | grep -q "MQTT connected.*tcp://mqtt:1883"; then
    echo -e "${GREEN}✓ MQTT连接正确 (tcp://mqtt:1883)${NC}"
else
    echo -e "${YELLOW}⚠ MQTT连接可能不正确${NC}"
    docker logs cloud-backend --tail 50 | grep "MQTT"
fi
echo ""

# 8. 检查容器网络
echo "8. 检查容器网络配置..."
BACKEND_NETWORK=$(docker inspect cloud-backend --format '{{range $net, $config := .NetworkSettings.Networks}}{{$net}}{{end}}')
FRONTEND_NETWORK=$(docker inspect cloud-frontend --format '{{range $net, $config := .NetworkSettings.Networks}}{{$net}}{{end}}')

if [ "$BACKEND_NETWORK" = "cloud-network" ] && [ "$FRONTEND_NETWORK" = "cloud-network" ]; then
    echo -e "${GREEN}✓ 容器网络配置正确 (cloud-network)${NC}"
else
    echo -e "${YELLOW}⚠ 容器网络配置可能不正确${NC}"
    echo "  Backend: $BACKEND_NETWORK"
    echo "  Frontend: $FRONTEND_NETWORK"
fi
echo ""

# 9. 检查Docker Compose项目
echo "9. 检查Docker Compose项目归属..."
BACKEND_PROJECT=$(docker inspect cloud-backend --format '{{index .Config.Labels "com.docker.compose.project"}}')
FRONTEND_PROJECT=$(docker inspect cloud-frontend --format '{{index .Config.Labels "com.docker.compose.project"}}')

if [ "$BACKEND_PROJECT" = "cloud" ] && [ "$FRONTEND_PROJECT" = "cloud" ]; then
    echo -e "${GREEN}✓ 容器正确归属于Cloud项目${NC}"
else
    echo -e "${YELLOW}⚠ 容器可能不在Cloud项目中${NC}"
    echo "  Backend项目: $BACKEND_PROJECT"
    echo "  Frontend项目: $FRONTEND_PROJECT"
fi
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
