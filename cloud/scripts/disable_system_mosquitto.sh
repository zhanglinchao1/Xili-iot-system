#!/bin/bash
# 禁用系统级mosquitto服务，统一使用Edge端的8883 TLS服务

echo "================================"
echo "禁用系统级Mosquitto服务"
echo "================================"
echo ""

echo "系统现已统一使用Edge端的TLS mosquitto服务 (端口8883)"
echo "不再需要系统级的1883明文端口服务"
echo ""

echo "正在停止系统级mosquitto服务..."
if sudo systemctl stop mosquitto 2>/dev/null; then
    echo "✓ mosquitto服务已停止"
else
    echo "⚠️ 停止mosquitto服务失败或服务不存在"
fi

echo ""
echo "正在禁用系统级mosquitto服务(防止开机自启)..."
if sudo systemctl disable mosquitto 2>/dev/null; then
    echo "✓ mosquitto服务已禁用"
else
    echo "⚠️ 禁用mosquitto服务失败或服务不存在"
fi

echo ""
echo "检查mosquitto服务状态..."
systemctl status mosquitto --no-pager 2>/dev/null | head -3 || echo "服务已停止"

echo ""
echo "================================"
echo "完成"
echo "================================"
echo ""
echo "系统信息:"
echo "  旧端口(已停用): 1883 (明文,不安全)"
echo "  新端口(启用):   8883 (TLS加密)"
echo "  配置文件:       /home/zhang/XiLi/Edge/configs/mosquitto_tls.conf"
echo "  启动方式:       由Edge系统自动启动 (./start_all.sh)"
echo ""
