#!/bin/bash

# MQTT数据流检查脚本
# 检查传感器数据是否通过MQTT broker传到Cloud并显示在卡片中

echo "=========================================="
echo "MQTT传感器数据流检查"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 检查配置文件
echo "1. 检查配置文件..."
if grep -q "edge_mqtt:" /home/zhang/XiLi/Cloud/config.yaml; then
    echo -e "${GREEN}✓${NC} edge_mqtt配置存在"
    if grep -q "enabled: true" /home/zhang/XiLi/Cloud/config.yaml; then
        echo -e "${GREEN}✓${NC} Edge MQTT订阅已启用"
        EDGE_BROKER=$(grep -A 1 "edge_mqtt:" /home/zhang/XiLi/Cloud/config.yaml | grep "broker:" | awk '{print $2}')
        echo "   Broker地址: $EDGE_BROKER"
    else
        echo -e "${RED}✗${NC} Edge MQTT订阅未启用"
        echo "   请在config.yaml中设置 edge_mqtt.enabled: true"
    fi
else
    echo -e "${RED}✗${NC} edge_mqtt配置不存在"
    echo "   请添加edge_mqtt配置到config.yaml"
fi
echo ""

# 2. 检查MQTT broker是否运行
echo "2. 检查MQTT broker连接..."
if command -v mosquitto_pub &> /dev/null; then
    echo "   使用mosquitto工具测试连接..."
    # 这里可以添加实际的连接测试
    echo -e "${YELLOW}⚠${NC} 请手动检查MQTT broker是否运行在 $EDGE_BROKER"
else
    echo -e "${YELLOW}⚠${NC} mosquitto工具未安装，无法测试MQTT连接"
fi
echo ""

# 3. 检查数据库中的传感器数据
echo "3. 检查数据库中的传感器数据..."
PGPASSWORD=cloud123456 psql -h localhost -U cloud_user -d cloud_system -c "
SELECT 
    COUNT(*) as total_records,
    COUNT(DISTINCT device_id) as unique_devices,
    COUNT(DISTINCT sensor_type) as unique_sensor_types,
    MAX(time) as latest_timestamp
FROM sensor_data;
" 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} 数据库查询成功"
else
    echo -e "${RED}✗${NC} 数据库查询失败"
    echo "   请检查PostgreSQL连接和数据库配置"
fi
echo ""

# 4. 检查传感器设备
echo "4. 检查传感器设备..."
PGPASSWORD=cloud123456 psql -h localhost -U cloud_user -d cloud_system -c "
SELECT 
    cabinet_id,
    COUNT(*) as device_count,
    STRING_AGG(DISTINCT sensor_type, ', ') as sensor_types
FROM sensor_devices
GROUP BY cabinet_id;
" 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} 传感器设备查询成功"
else
    echo -e "${RED}✗${NC} 传感器设备查询失败"
fi
echo ""

# 5. 检查最新传感器数据（按设备）
echo "5. 检查最新传感器数据（每个设备的最新一条）..."
PGPASSWORD=cloud123456 psql -h localhost -U cloud_user -d cloud_system -c "
SELECT 
    sd.device_id,
    s.sensor_type,
    s.name,
    s.unit,
    sd.value,
    sd.quality,
    sd.time as timestamp
FROM sensor_data sd
INNER JOIN sensor_devices s ON sd.device_id = s.device_id
INNER JOIN LATERAL (
    SELECT device_id, MAX(time) as max_time
    FROM sensor_data
    WHERE device_id = sd.device_id
    GROUP BY device_id
) latest ON sd.device_id = latest.device_id AND sd.time = latest.max_time
ORDER BY s.sensor_type, sd.device_id
LIMIT 10;
" 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} 最新传感器数据查询成功"
else
    echo -e "${RED}✗${NC} 最新传感器数据查询失败"
fi
echo ""

# 6. 检查Cloud服务日志（如果存在）
echo "6. 检查Cloud服务日志..."
LOG_FILE="/home/zhang/XiLi/Cloud/logs/cloud-server.log"
if [ -f "$LOG_FILE" ]; then
    echo "   最近的MQTT相关日志："
    tail -n 20 "$LOG_FILE" | grep -i "mqtt\|sensor\|edge" | tail -n 5
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} 日志文件存在"
    else
        echo -e "${YELLOW}⚠${NC} 未找到MQTT相关日志"
    fi
else
    echo -e "${YELLOW}⚠${NC} 日志文件不存在: $LOG_FILE"
fi
echo ""

# 7. 检查API端点
echo "7. 检查API端点..."
API_URL="http://localhost:8003/api/v1"
echo "   测试健康检查端点..."
if curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} API服务运行中"
    echo "   API地址: $API_URL"
else
    echo -e "${RED}✗${NC} API服务未运行或无法访问"
    echo "   请确保Cloud服务已启动"
fi
echo ""

# 总结
echo "=========================================="
echo "检查完成"
echo "=========================================="
echo ""
echo "下一步操作："
echo "1. 如果edge_mqtt未启用，请编辑config.yaml并设置enabled: true"
echo "2. 确保Edge端MQTT broker正在运行并发布数据到 sensors/# 主题"
echo "3. 确保Cloud服务已重启以加载新配置"
echo "4. 检查前端页面是否显示传感器卡片"
echo "5. 如果数据未显示，检查浏览器控制台和网络请求"
