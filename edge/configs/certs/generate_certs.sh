#!/bin/bash

# Edge端 TLS证书生成脚本
# 生成CA证书、服务器证书和客户端证书

CERTS_DIR="/home/zhang/XiLi/Edge/configs/certs"
cd "$CERTS_DIR"

echo "开始生成TLS证书..."

# 1. 生成CA密钥和证书
echo "1. 生成CA证书..."
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
    -out ca.crt \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EdgeSystem/OU=IoT/CN=MQTT_CA"

# 2. 生成服务器密钥
echo "2. 生成服务器密钥..."
openssl genrsa -out server.key 2048

# 3. 生成服务器证书签名请求
echo "3. 生成服务器CSR..."
openssl req -new -key server.key -out server.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EdgeSystem/OU=MQTT/CN=localhost"

# 4. 创建服务器证书扩展配置
cat > server_ext.cnf << EOF
subjectAltName = DNS:localhost,DNS:*.local,IP:127.0.0.1,IP:0.0.0.0
EOF

# 5. 签名服务器证书
echo "4. 签名服务器证书..."
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -days 3650 \
    -sha256 -extfile server_ext.cnf

# 6. 生成客户端密钥（可选）
echo "5. 生成客户端证书..."
openssl genrsa -out client.key 2048

# 7. 生成客户端证书签名请求
openssl req -new -key client.key -out client.csr \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=EdgeSystem/OU=Gateway/CN=edge-client"

# 8. 签名客户端证书
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -days 3650 -sha256

# 清理临时文件
rm -f server.csr client.csr server_ext.cnf ca.srl

echo "证书生成完成！"
echo "生成的文件："
ls -lh ca.* server.* client.*

# 设置权限
chmod 644 *.crt
chmod 600 *.key

echo ""
echo "证书位置: $CERTS_DIR"
echo "CA证书: ca.crt"
echo "服务器证书: server.crt, server.key"
echo "客户端证书: client.crt, client.key (可选使用)"
