# 多阶段构建 Dockerfile

# 第一阶段：编译
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装必要的工具
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dnsfailover-agent .

# 第二阶段：运行
FROM alpine:latest

# 安装 ca-certificates（HTTPS 需要）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/dnsfailover-agent .

# 创建日志目录
RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

# 切换到非 root 用户
USER appuser

# 暴露端口（如果需要）
# EXPOSE 8080

# 设置入口点
ENTRYPOINT ["./dnsfailover-agent"]

# 默认命令
CMD ["monitor", "start"]
