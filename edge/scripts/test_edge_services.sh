#!/bin/bash
# Edge端服务测试脚本

echo "=========================================="
echo "Edge端服务测试"
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

# 2. 测试后端访问
echo "2. 测试Edge后端访问 (http://localhost:8001)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8001/health)
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 后端健康检查正常 (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}✗ 后端健康检查失败 (HTTP $HTTP_CODE)${NC}"
fi
echo ""

# 3. 测试前端访问
echo "3. 测试Edge前端访问 (http://localhost:8000)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/)
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 前端访问正常 (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}✗ 前端访问失败 (HTTP $HTTP_CODE)${NC}"
fi
echo ""

# 4. 测试设备列表API
echo "4. 测试设备列表API (http://localhost:8001/api/v1/devices)..."
DEVICES_RESPONSE=$(curl -s http://localhost:8001/api/v1/devices)
DEVICE_COUNT=$(echo "$DEVICES_RESPONSE" | grep -o '"device_id"' | wc -l)
if [ "$DEVICE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ 设备列表API正常 (发现 $DEVICE_COUNT 个设备)${NC}"
else
    echo -e "${YELLOW}⚠ 设备列表API返回0个设备${NC}"
fi
echo ""

# 5. 检查MQTT Broker
echo "5. 检查MQTT Broker (端口 8883)..."
if docker logs edge-system --tail 100 | grep -q "Mosquitto.*已就绪"; then
    echo -e "${GREEN}✓ MQTT Broker已就绪${NC}"
elif docker logs edge-system --tail 100 | grep -q "MQTT.*连接"; then
    echo -e "${GREEN}✓ MQTT连接正常${NC}"
else
    echo -e "${YELLOW}⚠ 无法确认MQTT Broker状态${NC}"
fi
echo ""

# 6. 检查容器网络
echo "6. 检查容器网络配置..."
EDGE_NETWORK=$(docker inspect edge-system --format '{{range $net, $config := .NetworkSettings.Networks}}{{$net}}{{end}}')

if [ "$EDGE_NETWORK" = "edge-network" ]; then
    echo -e "${GREEN}✓ 容器网络配置正确 (edge-network)${NC}"
else
    echo -e "${YELLOW}⚠ 容器网络可能不正确: $EDGE_NETWORK${NC}"
fi
echo ""

# 7. 检查Docker Compose项目
echo "7. 检查Docker Compose项目归属..."
EDGE_PROJECT=$(docker inspect edge-system --format '{{index .Config.Labels "com.docker.compose.project"}}')

if [ "$EDGE_PROJECT" = "edge" ]; then
    echo -e "${GREEN}✓ 容器正确归属于Edge项目${NC}"
else
    echo -e "${YELLOW}⚠ 容器项目归属: $EDGE_PROJECT${NC}"
fi
echo ""

# 8. 检查端口映射
echo "8. 检查端口映射..."
PORTS=$(docker port edge-system)
if echo "$PORTS" | grep -q "8001" && echo "$PORTS" | grep -q "8000" && echo "$PORTS" | grep -q "8883"; then
    echo -e "${GREEN}✓ 端口映射正确${NC}"
    echo "$PORTS" | grep -E "(8000|8001|8883)"
else
    echo -e "${RED}✗ 端口映射可能不完整${NC}"
    echo "$PORTS"
fi
echo ""

# 9. 检查最近日志中的错误
echo "9. 检查最近日志中的错误..."
ERROR_COUNT=$(docker logs edge-system --tail 100 2>&1 | grep -i "error" | grep -v "error_count\":0" | wc -l)
if [ "$ERROR_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✓ 无错误日志${NC}"
else
    echo -e "${YELLOW}⚠ 发现 $ERROR_COUNT 条错误日志${NC}"
    docker logs edge-system --tail 100 2>&1 | grep -i "error" | grep -v "error_count\":0" | tail -5
fi
echo ""

# 10. Cloud同步状态检查
echo "10. 检查Cloud同步状态..."
if docker logs edge-system --tail 50 | grep -q "connection refused"; then
    echo -e "${YELLOW}⚠ Cloud同步连接被拒绝 (预期行为,Cloud可能在不同机器)${NC}"
    echo "   Edge配置的Cloud地址: $(grep 'endpoint:' /home/zhang/XiLi/Edge/configs/config.yaml | awk '{print $2}')"
elif docker logs edge-system --tail 50 | grep -q "同步成功"; then
    echo -e "${GREEN}✓ Cloud同步正常${NC}"
else
    echo -e "${YELLOW}⚠ 无法确认Cloud同步状态${NC}"
fi
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
