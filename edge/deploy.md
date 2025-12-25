

# Edge系统部署和启动指南

本文档提供Edge系统（储能柜边缘认证网关）的完整部署和启动教程，适合部署新手。

## ⚡ 开发环境快速启动

如果你已经安装了Go 1.24+和SQLite，可以直接运行以下命令快速启动项目：

```bash
# 1. 进入项目目录
cd /home/uestc/Edge

# 2. 确保使用正确的Go版本
export PATH=/usr/local/go/bin:$PATH
go version  # 应该显示 go1.24.9

# 3. 下载依赖
go mod download && go mod tidy

# 4. 创建必要目录
mkdir -p data logs bin

# 5. 编译项目
go build -o bin/edge cmd/edge/main.go

# 6. 初始化数据库
./bin/edge -migrate -config ./configs/config.yaml

# 7. 启动服务（前台运行）
./bin/edge -config ./configs/config.yaml

# 8. 在新终端测试接口（注意：如果配置了代理需要使用--noproxy）
curl --noproxy '*' http://localhost:8001/health
# 或者
NO_PROXY=localhost curl http://localhost:8001/health
```

**开发常用命令**：
```bash
# 运行快速API测试（推荐）
./test_api.sh

# 运行所有测试
go test ./...

# 运行集成测试
go run test_modules.go

# 代码质量检查
python3 test_code_check.py

# 格式化代码
go fmt ./...

# 查看日志
tail -f logs/edge.log
```

---

## 📋 部署前准备

### 系统要求
- **操作系统**: Linux (推荐Ubuntu 22.04或CentOS 7+)
- **CPU**: 1核心以上
- **内存**: 512MB以上
- **存储**: 10GB以上
- **网络**: 100Mbps以上

### 需要安装的软件
1. Go语言环境 (1.24或更高版本，**必须使用1.24+**)
2. SQLite3数据库
3. Docker和Docker Compose (可选，用于容器化部署)
4. Git (用于下载代码)

---

## 🚀 方式一：本地直接部署（推荐新手）

### 步骤1: 安装Go语言环境

#### Ubuntu/Debian系统:
```bash
# 更新系统包
sudo apt update

# 下载Go 1.24.9
cd /tmp
wget https://go.dev/dl/go1.24.9.linux-amd64.tar.gz

# 解压到/usr/local
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.9.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
# 应该显示: go version go1.24.9 linux/amd64
```

#### CentOS/RHEL系统:
```bash
# 下载Go 1.24.9
cd /tmp
wget https://go.dev/dl/go1.24.9.linux-amd64.tar.gz

# 解压到/usr/local
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.9.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bash_profile
echo 'export GOPATH=$HOME/go' >> ~/.bash_profile
source ~/.bash_profile

# 验证安装
go version
# 应该显示: go version go1.24.9 linux/amd64
```

### 步骤2: 安装SQLite3

#### Ubuntu/Debian:
```bash
sudo apt install -y sqlite3 libsqlite3-dev
```

#### CentOS/RHEL:
```bash
sudo yum install -y sqlite sqlite-devel
```

#### 验证安装:
```bash
sqlite3 --version
# 应该显示版本号，例如: 3.37.2
```

### 步骤3: 下载项目代码

```bash
# 进入工作目录
cd /home/uestc

# 如果已有Edge目录，先备份
# mv Edge Edge.backup.$(date +%Y%m%d)

# 克隆或复制项目代码到Edge目录
# 假设代码已经在/home/uestc/Edge目录
cd /home/uestc/Edge

# 查看项目结构
ls -la
```

### 步骤4: 配置Go模块代理（加速下载）

```bash
# 设置Go模块代理（使用国内镜像）
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

# 验证设置
go env | grep GOPROXY
```

### 步骤5: 下载项目依赖

```bash
# 确保在项目根目录
cd /home/uestc/Edge

# 下载所有依赖包
go mod download

# 整理依赖
go mod tidy

# 这个过程可能需要几分钟，请耐心等待
```

### 步骤6: 创建必要的目录

```bash
# 创建数据目录
mkdir -p data

# 创建日志目录
mkdir -p logs

# 创建配置目录（如果不存在）
mkdir -p configs

# 设置权限
chmod 755 data logs configs
```

### 步骤7: 配置系统参数

编辑配置文件：
```bash
# 使用你喜欢的编辑器编辑配置文件
nano configs/config.yaml
# 或者
vim configs/config.yaml
```

**重要配置项说明**：
```yaml
server:
  host: "0.0.0.0"      # 监听所有网卡
  port: 8001           # HTTP端口，可以改成其他端口
  mode: "release"      # 生产模式

database:
  path: "./data/edge.db"  # 数据库文件路径

cloud:
  enabled: false       # 初次部署建议设为false，稍后再启用
  endpoint: ""         # 云端地址
  api_key: ""         # API密钥
```

### 步骤8: 编译项目

```bash
# 在项目根目录执行编译
cd /home/uestc/Edge

# 编译生成可执行文件
go build -o edge ./cmd/edge/main.go

# 查看生成的文件
ls -lh edge
# 应该看到一个edge可执行文件，大小约20-30MB

# 添加执行权限
chmod +x edge
```

### 步骤9: 初始化数据库

```bash
# 执行数据库迁移（创建表结构）
./edge -migrate -config ./configs/config.yaml

# 如果看到"数据库迁移完成"，说明成功
```

### 步骤10: 启动系统（前台测试）

```bash
# 前台启动，用于测试
./edge -config ./configs/config.yaml

# 如果看到以下信息，说明启动成功：
# [INFO] SQLite storage initialized
# [INFO] Simple ZKP verifier initialized (for testing)
# [INFO] Device manager started
# [INFO] Data collector started
# [INFO] HTTP服务器启动 address=0.0.0.0:8001
```

### 步骤11: 测试系统是否正常

**打开新的终端窗口**，执行测试：

```bash
# 如果系统配置了代理，需要绕过localhost的代理
# 方法1: 使用--noproxy参数（推荐）
curl --noproxy '*' http://localhost:8001/health

# 方法2: 临时设置NO_PROXY环境变量
NO_PROXY=localhost,127.0.0.1 curl http://localhost:8001/health

# 应该返回：
# {"status":"ok","timestamp":1760505924,"service":"edge-system"}

# 测试就绪检查
curl --noproxy '*' http://localhost:8001/ready

# 应该返回服务状态信息
```

**如果您经常需要访问localhost，建议永久配置no_proxy**：
```bash
# 编辑配置文件
nano ~/.bashrc

# 在文件末尾添加（如果已有http_proxy配置）：
export no_proxy="localhost,127.0.0.1,::1"
export NO_PROXY="localhost,127.0.0.1,::1"

# 保存后重新加载
source ~/.bashrc
```

**使用快速测试脚本**：
```bash
# 项目提供了一键测试脚本，可以快速验证所有接口
./test_api.sh

# 脚本会自动测试：
# ✓ 健康检查接口
# ✓ 就绪检查接口
# ✓ 服务进程状态
# ✓ 端口监听状态
# ✓ 最新日志信息
```

### 步骤12: 后台运行（正式部署）

如果测试成功，按`Ctrl+C`停止前台进程，然后使用以下方式后台运行：

#### 方法A: 使用nohup（简单）
```bash
# 后台启动
nohup ./edge -config ./configs/config.yaml > logs/edge.log 2>&1 &

# 查看进程
ps aux | grep edge

# 查看日志
tail -f logs/edge.log

# 停止服务
pkill edge
```

#### 方法B: 使用systemd（推荐）
创建系统服务文件：
```bash
# 创建服务文件
sudo nano /etc/systemd/system/edge.service
```

写入以下内容：
```ini
[Unit]
Description=Edge System - Storage Cabinet Edge Gateway
After=network.target

[Service]
Type=simple
User=uestc
Group=uestc
WorkingDirectory=/home/uestc/Edge
ExecStart=/home/uestc/Edge/edge -config /home/uestc/Edge/configs/config.yaml
Restart=always
RestartSec=10
StandardOutput=append:/home/uestc/Edge/logs/edge.log
StandardError=append:/home/uestc/Edge/logs/edge_error.log

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
# 重新加载systemd配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start edge

# 查看状态
sudo systemctl status edge

# 设置开机自启动
sudo systemctl enable edge

# 查看日志
sudo journalctl -u edge -f

# 停止服务
sudo systemctl stop edge

# 重启服务
sudo systemctl restart edge
```

---

## 🐳 方式二：Docker容器部署（推荐生产环境）

### 步骤1: 安装Docker

#### Ubuntu系统:
```bash
# 更新apt包索引
sudo apt update

# 安装必要的包
sudo apt install -y apt-transport-https ca-certificates curl software-properties-common

# 添加Docker官方GPG密钥
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# 添加Docker仓库
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 更新apt包索引
sudo apt update

# 安装Docker
sudo apt install -y docker-ce docker-ce-cli containerd.io

# 安装Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 将当前用户添加到docker组
sudo usermod -aG docker $USER

# 重新登录或执行
newgrp docker

# 验证安装
docker --version
docker-compose --version
```

#### CentOS系统:
```bash
# 安装必要的包
sudo yum install -y yum-utils

# 添加Docker仓库
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# 安装Docker
sudo yum install -y docker-ce docker-ce-cli containerd.io

# 启动Docker
sudo systemctl start docker
sudo systemctl enable docker

# 安装Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 将当前用户添加到docker组
sudo usermod -aG docker $USER

# 验证安装
docker --version
docker-compose --version
```

### 步骤2: 准备Docker部署

```bash
# 进入项目目录
cd /home/uestc/Edge

# 创建数据持久化目录
mkdir -p deployments/data
mkdir -p deployments/logs
mkdir -p deployments/configs

# 复制配置文件
cp configs/config.yaml deployments/configs/

# 设置环境变量（可选）
export CLOUD_API_KEY="your-api-key-here"
export CLOUD_ENDPOINT="https://cloud.example.com/api/v1"
```

### 步骤3: 构建Docker镜像

```bash
# 构建镜像
docker build -f deployments/Dockerfile -t edge-system:latest .

# 查看镜像
docker images | grep edge-system

# 应该看到新构建的镜像
```

### 步骤4: 使用Docker Compose启动

编辑docker-compose配置（如果需要）：
```bash
cd deployments
nano docker-compose.yaml
```

启动服务：
```bash
# 启动所有服务（后台运行）
docker-compose up -d

# 查看运行状态
docker-compose ps

# 查看日志
docker-compose logs -f edge

# 查看所有服务日志
docker-compose logs -f
```

### 步骤5: 验证Docker部署

```bash
# 测试健康检查
curl http://localhost:8001/health

# 查看容器状态
docker ps

# 进入容器内部（如果需要）
docker exec -it edge-system sh

# 退出容器
exit
```

### 步骤6: Docker服务管理

```bash
# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f edge

# 更新服务（修改代码后）
docker-compose down
docker build -f Dockerfile -t edge-system:latest ..
docker-compose up -d

# 清理旧镜像
docker system prune -a
```

---

## 🔧 常见问题排查

### 问题1: 编译失败 - "package xxx is not in GOROOT" 或版本不匹配

**原因**: Go版本太低或不兼容
**解决**:
```bash
# 升级Go到1.24或更高版本（必须1.24+）
# 参考步骤1重新安装Go 1.24.9
cd /tmp
wget https://go.dev/dl/go1.24.9.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.9.linux-amd64.tar.gz
export PATH=/usr/local/go/bin:$PATH
go version  # 验证版本
```

### 问题2: 端口被占用

**错误信息**: `bind: address already in use`
**解决**:
```bash
# 查看占用8001端口的进程
sudo lsof -i :8001

# 或者
sudo netstat -tlnp | grep 8001

# 杀死占用进程
sudo kill -9 <PID>

# 或者修改配置文件中的端口号
```

### 问题3: 数据库权限错误

**错误信息**: `unable to open database file`
**解决**:
```bash
# 确保数据目录存在且有写权限
mkdir -p data
chmod 755 data

# 检查磁盘空间
df -h
```

### 问题4: 无法访问API或curl没有返回

**常见原因**：
1. 系统配置了HTTP代理，导致curl无法访问localhost
2. 服务未启动
3. 端口被占用或防火墙阻止

**解决步骤**:
```bash
# 1. 检查是否有代理配置
echo $http_proxy $https_proxy $no_proxy

# 如果有代理，使用--noproxy参数
curl --noproxy '*' http://localhost:8001/health

# 或者永久配置no_proxy（推荐）
echo 'export no_proxy="localhost,127.0.0.1,::1"' >> ~/.bashrc
echo 'export NO_PROXY="localhost,127.0.0.1,::1"' >> ~/.bashrc
source ~/.bashrc

# 2. 检查服务是否运行
ps aux | grep edge
# 或
sudo systemctl status edge

# 3. 检查端口监听
sudo netstat -tlnp | grep 8001
# 或
ss -tlnp | grep 8001

# 4. 检查防火墙
sudo ufw status
# 如果启用了防火墙，需要开放端口
sudo ufw allow 8001

# 5. 检查日志
tail -f logs/edge.log
```

### 问题5: Docker容器无法启动

**解决步骤**:
```bash
# 查看容器日志
docker-compose logs edge

# 检查容器状态
docker ps -a

# 重新构建
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

---

## 📊 性能优化建议

### 1. 调整数据库设置
```yaml
database:
  max_connections: 20      # 增加连接数
  max_idle_connections: 10 # 增加空闲连接
```

### 2. 调整日志级别
```yaml
log:
  level: "info"  # 生产环境使用info，调试使用debug
```

### 3. 启用监控
```yaml
monitoring:
  metrics_enabled: true
  metrics_port: 9090
```

访问 `http://localhost:9090/metrics` 查看指标

---

## 🔒 安全加固建议

### 1. 设置JWT密钥
```bash
# 生成随机密钥
export JWT_SECRET=$(openssl rand -hex 32)

# 添加到环境变量
echo "export JWT_SECRET=$JWT_SECRET" >> ~/.bashrc
```

### 2. 配置防火墙
```bash
# Ubuntu/Debian
sudo ufw enable
sudo ufw allow 8001/tcp
sudo ufw allow 9090/tcp  # 如果启用监控

# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=8001/tcp
sudo firewall-cmd --permanent --add-port=9090/tcp
sudo firewall-cmd --reload
```

### 3. 使用HTTPS（生产环境必须）
```bash
# 安装nginx作为反向代理
sudo apt install nginx

# 配置SSL证书（使用Let's Encrypt）
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
```

---

## 📝 日常运维

### 查看系统状态
```bash
# 查看服务状态
sudo systemctl status edge

# 查看最新日志
tail -f logs/edge.log

# 查看错误日志
tail -f logs/edge_error.log

# 查看系统资源
top
# 按'M'按内存排序，按'P'按CPU排序
```

### 数据库备份
```bash
# 创建备份目录
mkdir -p backups

# 备份数据库
sqlite3 data/edge.db ".backup backups/edge_backup_$(date +%Y%m%d_%H%M%S).db"

# 定期备份（添加到crontab）
crontab -e
# 添加：每天凌晨2点备份
0 2 * * * sqlite3 /home/uestc/Edge/data/edge.db ".backup /home/uestc/Edge/backups/edge_backup_\$(date +\%Y\%m\%d).db"
```

### 日志轮转
```bash
# 创建logrotate配置
sudo nano /etc/logrotate.d/edge

# 添加内容：
/home/uestc/Edge/logs/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 uestc uestc
}
```

### 更新系统
```bash
# 停止服务
sudo systemctl stop edge

# 备份当前版本
cp edge edge.backup

# 拉取新代码或复制新文件
git pull
# 或者复制新的代码文件

# 重新编译
go build -o edge ./cmd/edge/main.go

# 执行数据库迁移（如果有新的表结构）
./edge -migrate -config ./configs/config.yaml

# 启动服务
sudo systemctl start edge

# 检查状态
sudo systemctl status edge
```

---

## ✅ 部署检查清单

部署完成后，请检查以下项目：

- [ ] Go版本 >= 1.24 (**必须1.24+**)
- [ ] SQLite已安装
- [ ] 项目代码已下载
- [ ] 依赖包已下载完成
- [ ] 配置文件已修改
- [ ] 必要目录已创建（data, logs）
- [ ] 编译成功，生成edge可执行文件
- [ ] 数据库初始化成功
- [ ] 服务可以启动
- [ ] 健康检查接口正常
- [ ] 防火墙已配置
- [ ] 已设置开机自启动
- [ ] 已配置日志轮转
- [ ] 已设置数据库备份

---

## 📞 获取帮助

如果遇到问题：

1. **查看日志**: `tail -f logs/edge.log`
2. **查看文档**: 阅读 [readme.md](mdc:readme.md)
3. **检查配置**: 确认 [configs/config.yaml](mdc:configs/config.yaml) 正确
4. **查看测试**: 参考测试脚本 `test_device_api.sh`

---

**祝部署顺利！** 🎉
