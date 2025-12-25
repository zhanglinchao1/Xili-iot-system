# Edge系统Docker镜像
# 多阶段构建，减小镜像大小

# 构建阶段
FROM nn7fm5xf4cl9hj.xuanyuan.run/golang:1.24-alpine AS builder

# 配置 Go 代理（使用国内镜像加速）
ENV GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on

# 设置工作目录
WORKDIR /app

# 配置 Alpine 镜像源（使用中科大镜像HTTP协议，避免TLS问题）
RUN sed -i 's/https:\/\/dl-cdn.alpinelinux.org/http:\/\/mirrors.ustc.edu.cn/g' /etc/apk/repositories || true

# 安装必要的包（先更新索引，然后安装）
RUN apk update && \
    apk add --no-cache git gcc musl-dev sqlite-dev && \
    rm -rf /var/cache/apk/*

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖（使用Go代理加速）
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
# 添加 _LARGEFILE64_SOURCE 以支持Alpine Linux的大文件API
RUN CGO_ENABLED=1 GOOS=linux CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -a -installsuffix cgo -o edge ./cmd/edge

# 运行阶段
FROM nn7fm5xf4cl9hj.xuanyuan.run/alpine:latest

# 配置 Alpine 镜像源（使用中科大镜像HTTP协议，避免TLS问题）
RUN sed -i 's/https:\/\/dl-cdn.alpinelinux.org/http:\/\/mirrors.ustc.edu.cn/g' /etc/apk/repositories || true

# 安装运行时依赖（先更新索引）
RUN apk update && \
    apk --no-cache add ca-certificates sqlite tzdata wget && \
    rm -rf /var/cache/apk/*

# 设置时区
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# 创建非root用户
RUN addgroup -g 1000 edge && \
    adduser -D -s /bin/sh -u 1000 -G edge edge

# 创建必要的目录
RUN mkdir -p /app/data /app/logs /app/configs && \
    chown -R edge:edge /app

# 切换到非root用户
USER edge

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder --chown=edge:edge /app/edge .

# 复制配置文件
COPY --chown=edge:edge configs/config.yaml ./configs/

# 复制ZKP验证密钥
COPY --chown=edge:edge auth_verifying.key ./

# 复制web前端文件
COPY --chown=edge:edge web ./web

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 设置环境变量
ENV GIN_MODE=release
ENV EDGE_CONFIG_PATH=/app/configs/config.yaml

# 启动命令
CMD ["./edge", "-config", "/app/configs/config.yaml"]
