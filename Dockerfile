# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git make protobuf-dev

# 安装 protoc-gen-go
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 设置工作目录
WORKDIR /build

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 生成 protobuf 代码
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go_opt=Mapi/proto/im_protocol.proto=github.com/arwen/im-server/internal/protocol \
    api/proto/im_protocol.proto && \
    mv api/proto/im_protocol.pb.go internal/protocol/

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o im-server cmd/server/main.go cmd/server/config.go

# 运行阶段
FROM alpine:latest

# 安装ca证书
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/im-server .
COPY --from=builder /build/config ./config

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8080 8081 8082

# 运行
CMD ["./im-server", "-config=config/config.yaml"]

