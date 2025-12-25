#!/bin/bash
# Cloud 端云服务器部署检查脚本
# 在云服务器上运行此脚本，诊断部署问题

echo "=========================================="
echo "Cloud端部署状态检查"
echo "=========================================="
echo ""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 检查后端进程
echo "1. 检查后端进程..."
if ps aux | grep -v grep | grep cloud-server > /dev/null; then
    echo -e "   ${GREEN}✓${NC} 后端进程运行中"
    ps aux | grep -v grep | grep cloud-server | head -1
else
    echo -e "   ${RED}✗${NC} 后端进程未运行"
    echo "   提示: 启动后端服务"
    echo "   cd /path/to/Cloud && nohup ./cloud-server > /var/log/cloud-server.log 2>&1 &"
fi
echo ""

# 2. 检查端口监听
echo "2. 检查端口监听状态..."
if command -v ss >/dev/null 2>&1; then
    listen_result=$(ss -tlnp | grep :8003)
    if [ ! -z "$listen_result" ]; then
        echo -e "   ${GREEN}✓${NC} 端口8003已监听"
        echo "$listen_result"
        
        # 检查监听地址
        if echo "$listen_result" | grep -q "0.0.0.0:8003"; then
            echo -e "   ${GREEN}✓${NC} 监听地址正确 (0.0.0.0:8003)"
        elif echo "$listen_result" | grep -q "127.0.0.1:8003"; then
            echo -e "   ${YELLOW}⚠${NC}  监听地址错误 (127.0.0.1:8003)"
            echo "   提示: 修改 config.yaml 中的 server.host 为 0.0.0.0"
        fi
    else
        echo -e "   ${RED}✗${NC} 端口8003未监听"
    fi
else
    echo "   ⚠ ss命令不可用，尝试使用netstat..."
    if command -v netstat >/dev/null 2>&1; then
        netstat -tlnp | grep :8003
    else
        echo "   ${RED}✗${NC} netstat和ss命令都不可用"
    fi
fi
echo ""

# 3. 本地连接测试
echo "3. 本地连接测试..."
if command -v curl >/dev/null 2>&1; then
    # 测试 localhost
    echo "   测试 localhost:8003..."
    if curl -s --max-time 5 http://localhost:8003/health > /dev/null 2>&1; then
        response=$(curl -s --max-time 5 http://localhost:8003/health)
        echo -e "   ${GREEN}✓${NC} localhost连接成功"
        echo "   响应: $response"
    else
        echo -e "   ${RED}✗${NC} localhost连接失败"
    fi
    
    # 测试 0.0.0.0
    echo "   测试 0.0.0.0:8003..."
    if curl -s --max-time 5 http://0.0.0.0:8003/health > /dev/null 2>&1; then
        echo -e "   ${GREEN}✓${NC} 0.0.0.0连接成功"
    else
        echo -e "   ${RED}✗${NC} 0.0.0.0连接失败"
    fi
else
    echo "   ${RED}✗${NC} curl命令不可用"
fi
echo ""

# 4. 防火墙检查
echo "4. 防火墙检查..."
if command -v ufw >/dev/null 2>&1; then
    echo "   检查 ufw 状态..."
    ufw_status=$(sudo ufw status 2>/dev/null)
    if echo "$ufw_status" | grep -q "8003"; then
        echo -e "   ${GREEN}✓${NC} ufw已开放8003端口"
    elif echo "$ufw_status" | grep -q "inactive"; then
        echo -e "   ${YELLOW}⚠${NC}  ufw未启用"
    else
        echo -e "   ${YELLOW}⚠${NC}  ufw未开放8003端口"
        echo "   提示: sudo ufw allow 8003"
    fi
elif command -v firewall-cmd >/dev/null 2>&1; then
    echo "   检查 firewalld 状态..."
    if sudo firewall-cmd --list-ports 2>/dev/null | grep -q "8003"; then
        echo -e "   ${GREEN}✓${NC} firewalld已开放8003端口"
    else
        echo -e "   ${YELLOW}⚠${NC}  firewalld未开放8003端口"
        echo "   提示: sudo firewall-cmd --add-port=8003/tcp --permanent && sudo firewall-cmd --reload"
    fi
else
    echo "   ${YELLOW}⚠${NC}  未检测到防火墙管理工具（ufw/firewalld）"
fi
echo ""

# 5. Nginx检查
echo "5. Nginx检查..."
if command -v nginx >/dev/null 2>&1; then
    if ps aux | grep -v grep | grep nginx > /dev/null; then
        echo -e "   ${GREEN}✓${NC} Nginx运行中"
        
        # 检查配置中是否有health端点
        if sudo nginx -T 2>/dev/null | grep -A 5 "location /health" | grep -q "return 200"; then
            echo -e "   ${RED}✗${NC} Nginx配置错误：/health端点直接返回文本"
            echo "   提示: 修改Nginx配置，将/health转发到后端：proxy_pass http://127.0.0.1:8003/health;"
        elif sudo nginx -T 2>/dev/null | grep -A 5 "location /health" | grep -q "proxy_pass"; then
            echo -e "   ${GREEN}✓${NC} Nginx配置正确：/health转发到后端"
        fi
    else
        echo "   ${YELLOW}⚠${NC}  Nginx未运行（可能不使用Nginx）"
    fi
else
    echo "   ${YELLOW}⚠${NC}  Nginx未安装（可能直接暴露后端端口）"
fi
echo ""

# 6. 配置文件检查
echo "6. 配置文件检查..."
config_files=(
    "/opt/Cloud/config.yaml"
    "/www/wwwroot/Cloud/config.yaml"
    "/root/Cloud/config.yaml"
    "./config.yaml"
)

found_config=false
for config_file in "${config_files[@]}"; do
    if [ -f "$config_file" ]; then
        found_config=true
        echo -e "   ${GREEN}✓${NC} 找到配置文件: $config_file"
        
        # 检查host配置
        if grep -q "host: 0.0.0.0" "$config_file"; then
            echo -e "   ${GREEN}✓${NC} server.host配置正确 (0.0.0.0)"
        elif grep -q "host: localhost" "$config_file" || grep -q "host: 127.0.0.1" "$config_file"; then
            echo -e "   ${RED}✗${NC} server.host配置错误（应为0.0.0.0）"
        fi
        
        # 检查CORS配置
        if grep -A 5 "cors:" "$config_file" | grep -q 'allow_origins:' && grep -A 5 "allow_origins:" "$config_file" | grep -q '"\*"'; then
            echo -e "   ${GREEN}✓${NC} CORS配置正确（允许所有来源）"
        else
            echo -e "   ${YELLOW}⚠${NC}  CORS可能未配置为允许所有来源"
        fi
        break
    fi
done

if [ "$found_config" = false ]; then
    echo -e "   ${YELLOW}⚠${NC}  未找到配置文件"
fi
echo ""

# 7. 总结和建议
echo "=========================================="
echo "检查总结"
echo "=========================================="
echo ""
echo "需要确保："
echo "1. 后端进程运行中"
echo "2. 监听地址为 0.0.0.0:8003（不是 127.0.0.1）"
echo "3. 本地可以访问 http://localhost:8003/health"
echo "4. 防火墙/安全组开放 8003 端口"
echo "5. Nginx配置正确（如果使用）"
echo ""
echo "如果以上都正常，请从Edge端测试："
echo "  curl http://$(hostname -I | awk '{print $1}'):8003/health"
echo ""

