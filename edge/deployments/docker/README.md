# Edge 系统 Docker 容器化部署

## 概述

本方案实现 **无源码容器化部署**，只需要编译后的二进制文件和配置文件。

### ✅ 优势

1. **源码保护**：Docker 镜像中不包含任何 Go 源代码
2. **镜像小巧**：基于 Alpine Linux，最终镜像约 50-80MB
3. **部署简单**：一键构建、一键部署
4. **跨平台**：可在任意支持 Docker 的 Linux 系统运行
5. **安全性高**：使用非 root 用户运行

---

## 构建镜像

### 前置条件

- Go 1.24+（仅用于编译）
- Docker 20.10+
- Docker Compose 2.0+

### 快速构建

```bash
# 在 Edge 项目根目录执行
cd /home/zhang/XiLi/Edge

# 执行构建脚本
./deployments/docker/build.sh
```

### 构建过程

脚本会自动完成：

1. ✅ 编译 Go 二进制文件（Linux amd64）
2. ✅ 准备 Docker 构建上下文
3. ✅ 复制配置文件和证书
4. ✅ 构建 Docker 镜像
5. ✅ 清理临时文件

### 构建输出

```
✅ 镜像构建成功: edge-system:latest
```

查看镜像：
```bash
docker images | grep edge-system
```

---

## 部署运行

### 方式 1: 使用 Docker Compose（推荐）

```bash
# 进入部署目录
cd deployments/docker

# 启动容器
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止容器
docker-compose down
```

### 方式 2: 使用 Docker 命令

```bash
docker run -d \
  --name edge-system \
  --network host \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -e TZ=Asia/Shanghai \
  edge-system:latest
```

---

## 目录结构

```
deployments/docker/
├── README.md                    # 本文档
├── Dockerfile.production        # 生产环境 Dockerfile（无源码）
├── docker-compose.yml           # Docker Compose 配置
├── build.sh                     # 自动构建脚本
└── entrypoint.sh                # 容器启动脚本

项目根目录需要准备:
├── configs/
│   ├── config.yaml              # Edge 配置文件
│   ├── mosquitto_tls.conf       # Mosquitto 配置
│   └── certs/                   # TLS 证书目录
├── data/                        # 数据目录（数据库）
├── logs/                        # 日志目录
└── auth_verifying.key           # ZKP 验证密钥
```

---

## 数据持久化

以下目录会挂载到宿主机，确保数据不丢失：

| 容器路径 | 宿主机路径 | 说明 | 权限 |
|---------|-----------|------|------|
| `/app/configs` | `./configs` | 配置文件 | 只读 |
| `/app/data` | `./data` | SQLite 数据库 | 读写 |
| `/app/logs` | `./logs` | 日志文件 | 读写 |
| `/app/configs/certs` | `./configs/certs` | TLS 证书 | 只读 |
| `/app/auth_verifying.key` | `./auth_verifying.key` | ZKP 密钥 | 只读 |

---

## 端口映射

| 容器端口 | 用途 | 外部访问 |
|---------|------|---------|
| 8001 | HTTP API | http://宿主机IP:8001 |
| 8000 | Web 前端 | http://宿主机IP:8000 |
| 8883 | MQTT TLS | mqtt://宿主机IP:8883 |
| 9090 | Prometheus Metrics | http://宿主机IP:9090/metrics |

**注意**：默认使用 `network_mode: host`，容器直接使用宿主机网络，方便与本地 orangepi 通信。

---

## 配置说明

### 修改配置文件

容器启动后，可以修改宿主机的配置文件：

```bash
# 编辑配置
vi configs/config.yaml

# 重启容器生效
docker-compose restart
```

### 环境变量

在 `docker-compose.yml` 中可配置：

```yaml
environment:
  - TZ=Asia/Shanghai      # 时区
  - LOG_LEVEL=info        # 日志级别（可选）
```

---

## 健康检查

容器内置健康检查：

```bash
# 查看健康状态
docker inspect --format='{{.State.Health.Status}}' edge-system

# 预期输出: healthy
```

健康检查配置：
- 检查间隔：30秒
- 超时时间：10秒
- 重试次数：3次
- 启动等待：10秒

---

## 日志管理

### 查看实时日志

```bash
# 容器日志
docker-compose logs -f

# Edge 应用日志
tail -f logs/edge.log

# Mosquitto 日志
tail -f logs/mosquitto/mosquitto_tls.log
```

### 日志轮转

Docker 日志配置（已在 `docker-compose.yml` 中设置）：
- 单文件最大：10MB
- 保留文件数：3个

---

## 镜像迁移

### 导出镜像（在开发机器）

```bash
docker save -o edge-system.tar edge-system:latest
```

### 传输到目标机器

```bash
# 方式1: scp
scp edge-system.tar user@target-machine:/path/to/

# 方式2: U盘
cp edge-system.tar /media/usb/
```

### 导入镜像（在目标机器）

```bash
docker load -i edge-system.tar
```

### 验证导入

```bash
docker images | grep edge-system
```

---

## 生产部署流程

### 1. 准备部署包

在开发机器上：

```bash
# 构建镜像
./deployments/docker/build.sh

# 导出镜像
docker save -o edge-system.tar edge-system:latest

# 打包配置文件
tar czf edge-configs.tar.gz configs/ auth_verifying.key deployments/docker/docker-compose.yml
```

### 2. 上传到生产服务器

```bash
scp edge-system.tar user@production-server:/opt/edge/
scp edge-configs.tar.gz user@production-server:/opt/edge/
```

### 3. 在生产服务器部署

```bash
# 解压配置
cd /opt/edge
tar xzf edge-configs.tar.gz

# 导入镜像
docker load -i edge-system.tar

# 创建数据目录
mkdir -p data logs logs/mosquitto/data

# 启动服务
cd deployments/docker
docker-compose up -d

# 查看状态
docker-compose ps
docker-compose logs
```

---

## 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker-compose logs

# 检查配置文件
cat configs/config.yaml

# 手动进入容器调试
docker run -it --rm \
  -v $(pwd)/configs:/app/configs \
  edge-system:latest /bin/sh
```

### 权限问题

```bash
# 确保数据目录权限正确
chown -R 1000:1000 data/ logs/
```

### 网络问题

```bash
# 检查端口占用
netstat -tuln | grep -E "8001|8883"

# 检查容器网络
docker network inspect bridge
```

### MQTT 连接失败

```bash
# 进入容器检查
docker exec -it edge-system sh

# 测试 MQTT
mosquitto_sub -h 127.0.0.1 -p 8883 -t test -v
```

---

## 安全最佳实践

### 1. 使用非 root 用户

容器内已配置 `edge` 用户（UID 1000），不使用 root 运行。

### 2. 只读挂载敏感文件

配置文件和证书使用只读挂载（`:ro`）：

```yaml
volumes:
  - ./configs:/app/configs:ro
  - ./configs/certs:/app/configs/certs:ro
```

### 3. 限制资源使用

在 `docker-compose.yml` 中添加资源限制：

```yaml
services:
  edge:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 512M
```

### 4. 使用私有镜像仓库

```bash
# 标记镜像
docker tag edge-system:latest your-registry.com/edge-system:1.0.0

# 推送到私有仓库
docker push your-registry.com/edge-system:1.0.0
```

---

## 监控和维护

### Prometheus 监控

如果启用了 metrics：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'edge'
    static_configs:
      - targets: ['edge-system:9090']
```

### 容器状态监控

```bash
# 查看容器资源使用
docker stats edge-system

# 查看容器事件
docker events --filter container=edge-system
```

### 定期备份

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d)
BACKUP_DIR="/backup/edge"

# 备份数据库
docker exec edge-system sqlite3 /app/data/edge.db ".backup /app/data/edge_backup.db"
docker cp edge-system:/app/data/edge_backup.db $BACKUP_DIR/edge_$DATE.db

# 备份配置
tar czf $BACKUP_DIR/configs_$DATE.tar.gz configs/
```

---

## 常见问题

### Q1: 镜像太大怎么办？

A: 已使用 Alpine Linux 作为基础镜像，并编译时使用 `-ldflags="-s -w"` 去除调试信息，镜像已经很小。

### Q2: 如何更新版本？

A: 重新执行 `build.sh`，然后：
```bash
docker-compose down
docker-compose up -d
```

### Q3: 如何查看容器内文件？

A:
```bash
docker exec -it edge-system ls -la /app
```

### Q4: 数据库损坏怎么办？

A: 使用备份恢复：
```bash
docker cp edge_backup.db edge-system:/app/data/edge.db
docker-compose restart
```

---

## 总结

### ✅ 容器化的优势

1. **源码保护**：只包含二进制文件，源码完全保密
2. **部署简单**：一键构建、一键部署
3. **环境隔离**：不影响宿主机环境
4. **易于迁移**：导出镜像可在任意机器运行
5. **便于管理**：统一的 Docker 工具链

### 📦 部署文件清单

交付给客户的部署包只需包含：

1. `edge-system.tar` - Docker 镜像（约 50-80MB）
2. `docker-compose.yml` - 容器编排配置
3. `configs/` - 配置文件目录
4. `auth_verifying.key` - ZKP 验证密钥
5. `README.md` - 部署说明文档

**完全不需要源代码！** 🎉
