# IM Server

一个高性能、可扩展的即时通讯服务器，使用 Go 语言开发，支持 WebSocket 和 TCP 协议。

## 功能特性

### 核心功能
- ✅ **实时消息** - 支持文本、图片、语音、视频等多种消息类型
- ✅ **单聊/群聊** - 支持一对一聊天和群组聊天
- ✅ **离线消息** - 自动存储和同步离线消息
- ✅ **消息回执** - 已读/未读状态跟踪
- ✅ **消息撤回** - 支持消息撤回功能
- ✅ **输入状态** - 实时显示对方输入状态
- ✅ **在线状态** - 用户在线/离线状态管理

### 技术特性
- 🚀 **高性能** - 基于 Goroutine 的高并发处理
- 🔒 **安全认证** - JWT Token 认证机制
- 📦 **消息队列** - 支持消息缓冲和批量处理
- 💾 **数据持久化** - PostgreSQL/MySQL 数据库支持
- ⚡ **Redis 缓存** - 会话和在线状态缓存
- 🔄 **协议兼容** - 与 IM iOS SDK 完全兼容
- 🐳 **容器化部署** - Docker 和 Docker Compose 支持

## 项目结构

```
IM-Server/
├── api/                    # API定义
│   └── proto/             # Protocol Buffer定义
├── cmd/                   # 主程序入口
│   └── server/           # 服务器主程序
├── config/               # 配置文件
├── deployments/          # 部署配置
├── internal/             # 内部包
│   ├── cache/           # 缓存层
│   ├── handler/         # 消息处理器
│   ├── middleware/      # 中间件
│   ├── model/           # 数据模型
│   ├── protocol/        # 协议实现
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   └── transport/       # 传输层（WebSocket/TCP）
├── pkg/                 # 公共包
│   ├── crypto/         # 加密工具
│   ├── logger/         # 日志工具
│   └── utils/          # 通用工具
├── scripts/            # 脚本文件
├── Dockerfile          # Docker镜像配置
├── docker-compose.yaml # Docker Compose配置
├── Makefile           # 编译和运行脚本
└── README.md          # 项目文档
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- PostgreSQL 15+ 或 MySQL 8+
- Redis 7+
- Docker 和 Docker Compose（可选）

### 本地开发

1. **克隆项目**
```bash
cd IM-Server
```

2. **安装依赖**
```bash
make deps
```

3. **配置数据库**
```bash
# 创建PostgreSQL数据库
createdb im_db

# 或者使用Docker启动数据库
docker-compose up -d postgres redis
```

4. **配置文件**

编辑 `config/config.yaml` 文件，修改数据库和Redis连接信息：

```yaml
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "imserver"
  password: "imserver123"
  dbname: "im_db"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
```

5. **编译和运行**
```bash
# 编译
make build

# 运行
make run

# 或者直接运行
./bin/im-server -config=config/config.yaml
```

### Docker 部署

使用 Docker Compose 一键部署：

```bash
# 启动所有服务（包括PostgreSQL、Redis和IM Server）
docker-compose up -d

# 查看日志
docker-compose logs -f im-server

# 停止服务
docker-compose down
```

## API 文档

### WebSocket 连接

**端点**: `ws://localhost:8081/ws`

**连接流程**:
1. 建立 WebSocket 连接
2. 发送认证请求（CMD_AUTH_REQ）
3. 收到认证响应（CMD_AUTH_RSP）
4. 开始收发消息

### 协议格式

#### WebSocket 消息格式

使用 Protocol Buffer 定义的 `WebSocketMessage`:

```protobuf
message WebSocketMessage {
    CommandType command = 1;     // 命令类型
    uint32 sequence = 2;         // 序列号
    bytes body = 3;              // 消息体
    int64 timestamp = 4;         // 时间戳
}
```

#### 主要命令类型

| 命令 | 说明 |
|------|------|
| CMD_AUTH_REQ (100) | 认证请求 |
| CMD_AUTH_RSP (101) | 认证响应 |
| CMD_HEARTBEAT_REQ (5) | 心跳请求 |
| CMD_HEARTBEAT_RSP (6) | 心跳响应 |
| CMD_SEND_MSG_REQ (200) | 发送消息请求 |
| CMD_SEND_MSG_RSP (201) | 发送消息响应 |
| CMD_PUSH_MSG (202) | 推送消息 |
| CMD_MSG_ACK (203) | 消息确认 |
| CMD_SYNC_REQ (300) | 同步消息请求 |
| CMD_SYNC_RSP (301) | 同步消息响应 |

### 使用示例

#### 1. 认证

```go
// 构造认证请求
authReq := &protocol.AuthRequest{
    UserId:   "user123",
    Token:    "your_jwt_token",
    Platform: "iOS",
}

// 序列化并发送
body, _ := proto.Marshal(authReq)
wsMsg := &protocol.WebSocketMessage{
    Command:   protocol.CommandType_CMD_AUTH_REQ,
    Sequence:  1,
    Body:      body,
    Timestamp: time.Now().UnixMilli(),
}
data, _ := proto.Marshal(wsMsg)
conn.WriteMessage(websocket.BinaryMessage, data)
```

#### 2. 发送消息

```go
// 构造发送消息请求
sendReq := &protocol.SendMessageRequest{
    ClientMsgId:    "client_msg_123",
    ConversationId: "conv_456",
    SenderId:       "user123",
    ReceiverId:     "user456",
    MessageType:    1, // 文本消息
    Content:        []byte(`{"text":"Hello!"}`),
    SendTime:       time.Now().UnixMilli(),
}

// 序列化并发送
body, _ := proto.Marshal(sendReq)
wsMsg := &protocol.WebSocketMessage{
    Command:   protocol.CommandType_CMD_SEND_MSG_REQ,
    Sequence:  2,
    Body:      body,
    Timestamp: time.Now().UnixMilli(),
}
data, _ := proto.Marshal(wsMsg)
conn.WriteMessage(websocket.BinaryMessage, data)
```

## 配置说明

### 服务器配置

```yaml
server:
  name: "IM-Server"
  mode: "debug"          # debug | release
  http_port: 8080        # HTTP API端口
  ws_port: 8081          # WebSocket端口
  tcp_port: 8082         # TCP端口
```

### 数据库配置

```yaml
database:
  type: "postgres"       # postgres | mysql
  host: "localhost"
  port: 5432
  user: "imserver"
  password: "imserver123"
  dbname: "im_db"
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 60
```

### Redis 配置

```yaml
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 100
```

### 认证配置

```yaml
auth:
  jwt_secret: "your-secret-key-change-in-production"
  token_expire_hours: 720  # 30天
```

## 开发指南

### 编译命令

```bash
# 编译二进制文件
make build

# 运行服务器
make run

# 清理编译产物
make clean

# 生成protobuf代码
make proto

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint
```

### 生成 Protocol Buffer 代码

```bash
# 需要先安装protoc
make proto
```

### 添加新功能

1. 在 `internal/model/` 添加数据模型
2. 在 `internal/repository/` 添加数据访问层
3. 在 `internal/service/` 添加业务逻辑
4. 在 `internal/handler/` 添加消息处理器
5. 在 `api/proto/` 添加协议定义（如需要）

## 性能优化

### 推荐配置

- **CPU**: 4核心或更多
- **内存**: 8GB或更多
- **并发连接**: 单实例支持10,000+并发连接
- **消息吞吐**: 100,000+ 消息/秒

### 优化建议

1. 使用 Redis 集群提高缓存性能
2. 使用数据库主从复制和读写分离
3. 使用负载均衡部署多个服务器实例
4. 启用消息批量处理减少数据库IO
5. 合理配置连接池大小

## 监控和日志

### 日志配置

```yaml
logger:
  level: "debug"         # debug | info | warn | error
  format: "console"      # console | json
  output: "logs/im-server.log"
  console: true
```

### 查看日志

```bash
# 实时查看日志
make logs

# 或者
tail -f logs/im-server.log
```

## 安全建议

1. **生产环境务必修改**:
   - JWT密钥 (`auth.jwt_secret`)
   - 数据库密码
   - Redis密码

2. **启用HTTPS/WSS**:
   - 使用反向代理（如Nginx）配置SSL证书

3. **限流配置**:
   - 启用限流防止恶意请求

4. **防火墙规则**:
   - 只开放必要的端口

## 与 iOS SDK 配合使用

此服务器完全兼容 IM-iOS-SDK，可以直接配合使用：

```swift
// iOS 端配置
let config = IMConfig()
config.serverURL = "ws://your-server:8081/ws"

let client = IMClient(config: config)
client.connect(userID: "user123", token: "your_token")
```

## 常见问题

### Q: 如何生成 JWT Token？

A: 可以使用提供的工具函数：

```go
import "github.com/arwen/im-server/pkg/crypto"

token, err := crypto.GenerateToken(userID, platform, jwtSecret, expireHours)
```

### Q: 如何扩展到多个服务器实例？

A: 使用 Redis 作为消息队列，实现跨服务器消息推送。后续版本会提供完整的集群方案。

### Q: 支持哪些数据库？

A: 目前支持 PostgreSQL 和 MySQL。推荐使用 PostgreSQL 以获得更好的性能。

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 许可证

MIT License

## 联系方式

- GitHub Issues: 提交问题和建议
- 邮箱: [your-email@example.com]

## 致谢

感谢所有贡献者和开源社区的支持！

