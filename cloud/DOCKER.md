# Docker 部署指南

本文档说明如何使用Docker和Docker Compose部署Cloud端储能柜管理系统。

## 📋 目录

- [前提条件](#前提条件)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [配置说明](#配置说明)
- [构建和运行](#构建和运行)
- [生产环境部署](#生产环境部署)
- [故障排查](#故障排查)
- [常见问题](#常见问题)

## 前提条件

### 必需软件

1. **Docker** (版本 20.10+)
   ```bash
   docker --version
   ```

2. **Docker Compose** (版本 1.29+)
   ```bash
   docker-compose --version
   ```

### 系统要求

- **CPU**: 至少 2 核
- **内存**: 至少 4GB RAM
- **磁盘**: 至少 10GB 可用空间
- **操作系统**: Linux, macOS, 或 Windows (WSL2)

## 快速开始

### 1. 克隆项目

```bash
cd /home/zhang/XiLi/Cloud
```

### 2. 配置环境

复制并编辑配置文件：

```bash
cp config.yaml config.docker.yaml
```

修改 `config.docker.yaml` 中的数据库和MQTT连接地址：

```yaml
database:
  postgres:
    host: postgres  # Docker服务名
    port: 5432
    user: cloud_user
    password: cloud123456
    dbname: cloudsystem

mqtt:
  broker: tcp://mqtt:1883  # Docker服务名

redis:
  host: redis  # Docker服务名
  port: 6379
```

### 3. 启动所有服务

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看日志
docker-compose logs -f

# 查看服务状态
docker-compose ps
```

### 4. 访问应用

- **前端**: http://localhost:5173
- **后端API**: http://localhost:8003
- **健康检查**: http://localhost:8003/health

### 5. 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

## 项目结构

```
Cloud/
├── Dockerfile.backend          # 后端Dockerfile
├── Dockerfile.frontend         # 前端Dockerfile
├── docker-compose.yml          # Docker Compose配置（开发环境）
├── docker-compose.prod.yml     # Docker Compose配置（生产环境）
├── .dockerignore               # Docker忽略文件
├── docker/
│   ├── nginx.conf              # Nginx配置
│   └── mosquitto.conf          # MQTT Broker配置
└── config.yaml                 # 应用配置文件
```

## 配置说明

### 环境变量

后端服务支持以下环境变量（会覆盖config.yaml中的配置）：

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `CLOUD_CONFIG_PATH` | 配置文件路径 | `/app/config.yaml` |
| `DB_HOST` | 数据库主机 | `postgres` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户 | `cloud_user` |
| `DB_PASSWORD` | 数据库密码 | `cloud123456` |
| `DB_NAME` | 数据库名 | `cloudsystem` |
| `REDIS_HOST` | Redis主机 | `redis` |
| `REDIS_PORT` | Redis端口 | `6379` |
| `MQTT_BROKER` | MQTT Broker地址 | `tcp://mqtt:1883` |
| `SERVER_HOST` | 服务器监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 服务器端口 | `8003` |
| `SERVER_MODE` | 运行模式 | `release` |

### 数据卷

Docker Compose创建以下数据卷：

- `postgres_data`: PostgreSQL数据持久化
- `redis_data`: Redis数据持久化
- `mqtt_data`: MQTT数据持久化
- `mqtt_logs`: MQTT日志
- `backend_logs`: 后端应用日志

### 网络

所有服务连接到 `cloud-network` 网络，可以通过服务名互相访问。

## 构建和运行

### 单独构建镜像

```bash
# 构建后端镜像
docker build -f Dockerfile.backend -t cloud-backend:latest .

# 构建前端镜像
docker build -f Dockerfile.frontend -t cloud-frontend:latest .
```

### 运行单个服务

```bash
# 运行PostgreSQL
docker-compose up -d postgres

# 运行Redis
docker-compose up -d redis

# 运行MQTT
docker-compose up -d mqtt

# 运行后端（需要先启动依赖服务）
docker-compose up -d backend

# 运行前端（需要先启动后端）
docker-compose up -d frontend
```

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres

# 查看最近100行日志
docker-compose logs --tail=100 backend
```

### 重启服务

```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart backend
```

### 更新服务

```bash
# 重新构建并启动
docker-compose up -d --build

# 仅重新构建特定服务
docker-compose build backend
docker-compose up -d backend
```

## 生产环境部署

### 1. 使用生产配置

```bash
# 使用生产环境配置
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

### 2. 安全配置

#### 数据库安全

1. **修改默认密码**：
   ```yaml
   # docker-compose.yml
   postgres:
     environment:
       POSTGRES_PASSWORD: <强密码>
   ```

2. **限制网络访问**：
   ```yaml
   postgres:
     ports: []  # 移除端口映射，仅内部访问
   ```

#### MQTT安全

1. **启用认证**：
   编辑 `docker/mosquitto.conf`：
   ```conf
   allow_anonymous false
   password_file /mosquitto/config/passwd
   ```

2. **创建密码文件**：
   ```bash
   docker exec -it cloud-mqtt mosquitto_passwd -c /mosquitto/config/passwd username
   ```

#### 前端安全

1. **配置HTTPS**：
   修改 `docker/nginx.conf` 添加SSL配置

2. **限制CORS**：
   在 `config.yaml` 中配置正确的CORS源

### 3. 数据备份

#### PostgreSQL备份

```bash
# 备份数据库
docker exec cloud-postgres pg_dump -U postgres cloudsystem > backup.sql

# 恢复数据库
docker exec -i cloud-postgres psql -U postgres cloudsystem < backup.sql
```

#### 数据卷备份

```bash
# 备份所有数据卷
docker run --rm -v cloud-postgres-data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz /data
```

### 4. 监控和日志

#### 查看资源使用

```bash
# 查看容器资源使用
docker stats

# 查看特定容器
docker stats cloud-backend
```

#### 日志管理

```bash
# 查看日志大小
docker-compose logs --no-log-prefix backend | wc -l

# 清理日志（谨慎使用）
docker-compose down
docker volume rm cloud-backend-logs
```

### 5. 高可用部署

对于生产环境，建议：

1. **使用Docker Swarm或Kubernetes**
2. **配置负载均衡**
3. **设置数据库主从复制**
4. **配置Redis Sentinel**
5. **使用外部MQTT集群**

## 故障排查

### 1. 服务无法启动

```bash
# 查看服务状态
docker-compose ps

# 查看详细日志
docker-compose logs backend

# 检查容器健康状态
docker inspect cloud-backend | grep Health -A 10
```

### 2. 数据库连接失败

```bash
# 检查PostgreSQL是否运行
docker-compose ps postgres

# 测试数据库连接
docker exec -it cloud-postgres psql -U postgres -d cloudsystem

# 查看数据库日志
docker-compose logs postgres
```

### 3. 前端无法访问后端

```bash
# 检查网络连接
docker network inspect cloud-network

# 测试后端API
curl http://localhost:8003/health

# 检查nginx配置
docker exec -it cloud-frontend cat /etc/nginx/conf.d/default.conf
```

### 4. MQTT连接问题

```bash
# 检查MQTT服务
docker-compose ps mqtt

# 测试MQTT连接
docker exec -it cloud-mqtt mosquitto_sub -h localhost -t 'test' -v

# 查看MQTT日志
docker-compose logs mqtt
```

### 5. 清理和重置

```bash
# 停止所有服务
docker-compose down

# 删除所有容器和数据卷（谨慎使用）
docker-compose down -v

# 清理未使用的镜像
docker image prune -a

# 清理未使用的数据卷
docker volume prune
```

## 常见问题

### Q1: 如何修改端口？

A: 编辑 `docker-compose.yml` 中的 `ports` 配置：

```yaml
backend:
  ports:
    - "8003:8003"  # 修改左侧端口号
```

### Q2: 如何添加新的环境变量？

A: 在 `docker-compose.yml` 的 `environment` 部分添加：

```yaml
backend:
  environment:
    - NEW_VAR=value
```

### Q3: 数据会丢失吗？

A: 不会。数据存储在Docker数据卷中，即使删除容器也不会丢失数据。只有执行 `docker-compose down -v` 才会删除数据卷。

### Q4: 如何更新应用？

A: 

```bash
# 1. 拉取最新代码
git pull

# 2. 重新构建并启动
docker-compose up -d --build

# 3. 查看日志确认更新成功
docker-compose logs -f backend
```

### Q5: 如何查看数据库内容？

A:

```bash
# 进入PostgreSQL容器
docker exec -it cloud-postgres psql -U postgres -d cloudsystem

# 或者使用外部工具连接
# 主机: localhost
# 端口: 5432
# 用户: postgres
# 密码: postgres123
# 数据库: cloudsystem
```

### Q6: 如何备份和恢复？

A: 参考[数据备份](#数据备份)章节。

### Q7: 性能优化建议？

A:

1. **使用生产配置**：`docker-compose.prod.yml`
2. **配置资源限制**：在 `docker-compose.prod.yml` 中设置CPU和内存限制
3. **启用数据库连接池**：在 `config.yaml` 中配置
4. **使用Redis缓存**：确保Redis服务正常运行
5. **配置日志轮转**：避免日志文件过大

## 技术支持

如遇到问题，请：

1. 查看日志：`docker-compose logs -f`
2. 检查服务状态：`docker-compose ps`
3. 查看本文档的[故障排查](#故障排查)章节
4. 提交Issue到项目仓库

---

**最后更新**: 2025-01-XX
**维护者**: Cloud开发团队

