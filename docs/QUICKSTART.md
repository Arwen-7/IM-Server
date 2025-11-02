# 快速开始指南

## 5分钟快速体验

### 方式一：Docker Compose（推荐）

1. **启动服务**
```bash
cd IM-Server
docker-compose up -d
```

2. **查看日志**
```bash
docker-compose logs -f im-server
```

3. **测试连接**
```bash
# 使用 wscat 测试
npm install -g wscat
wscat -c ws://localhost:8081/ws
```

### 方式二：本地运行

1. **准备环境**
```bash
# 启动 PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_USER=imserver \
  -e POSTGRES_PASSWORD=imserver123 \
  -e POSTGRES_DB=im_db \
  -p 5432:5432 \
  postgres:15-alpine

# 启动 Redis
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine
```

2. **配置和运行**
```bash
# 安装依赖
make deps

# 编译
make build

# 运行
make run
```

## 完整开发流程

### 1. 创建用户

由于服务器刚启动没有用户，先创建一个测试用户：

```go
// 可以通过 HTTP API 或直接插入数据库
// 临时方案：直接插入数据库

// 密码: "password123"
// bcrypt hash: $2a$10$N9qo8uLOickgx2ZMRZoMye1IVI1nYYQHs.VVZxVz.cYaO3xqfPh7u

INSERT INTO users (id, username, nickname, password, status, created_at, updated_at)
VALUES ('user_001', 'testuser', 'Test User', '$2a$10$N9qo8uLOickgx2ZMRZoMye1IVI1nYYQHs.VVZxVz.cYaO3xqfPh7u', 1, NOW(), NOW());
```

### 2. 生成 Token

使用提供的工具生成 JWT Token：

```go
package main

import (
    "fmt"
    "github.com/arwen/im-server/pkg/crypto"
)

func main() {
    token, err := crypto.GenerateToken(
        "user_001",              // userID
        "iOS",                   // platform
        "your-secret-key",       // jwtSecret (与配置文件一致)
        720,                     // expireHours (30天)
    )
    if err != nil {
        panic(err)
    }
    fmt.Println("Token:", token)
}
```

### 3. 连接服务器

使用 iOS SDK 连接：

```swift
import IMSDK

// 配置
let config = IMConfig()
config.serverURL = "ws://localhost:8081/ws"

// 创建客户端
let client = IMClient(config: config)

// 连接
client.connect(userID: "user_001", token: "your_jwt_token") { result in
    switch result {
    case .success:
        print("连接成功")
    case .failure(let error):
        print("连接失败: \(error)")
    }
}
```

### 4. 发送消息

```swift
let message = IMMessage()
message.conversationID = "conv_123"
message.receiverID = "user_002"
message.type = .text
message.content = IMTextContent(text: "Hello!")

client.sendMessage(message) { result in
    switch result {
    case .success(let msg):
        print("发送成功: \(msg.messageID)")
    case .failure(let error):
        print("发送失败: \(error)")
    }
}
```

### 5. 接收消息

```swift
client.onReceiveMessage = { message in
    print("收到消息: \(message.content)")
}
```

## 常用操作

### 查看服务状态

```bash
# Docker 方式
docker-compose ps

# 查看日志
docker-compose logs -f

# 查看资源使用
docker stats im-server
```

### 数据库操作

```bash
# 连接数据库
docker exec -it im-postgres psql -U imserver -d im_db

# 查询用户
SELECT * FROM users;

# 查询消息
SELECT * FROM messages ORDER BY seq DESC LIMIT 10;

# 查看连接数
SELECT count(*) FROM online_status WHERE status = 1;
```

### Redis 操作

```bash
# 连接 Redis
docker exec -it im-redis redis-cli

# 查看会话
KEYS session:*

# 查看用户连接
KEYS user_conn:*

# 监控命令
MONITOR
```

### 停止服务

```bash
# Docker 方式
docker-compose down

# 保留数据
docker-compose down --volumes  # 删除数据卷
```

## 性能测试

### 压力测试工具

```bash
# 使用 ws-bench
npm install -g ws-bench
ws-bench -c 100 -m 1000 ws://localhost:8081/ws
```

### 监控指标

```bash
# 连接数
SELECT COUNT(*) FROM online_status WHERE status = 1;

# 消息统计
SELECT 
    DATE(created_at) as date,
    COUNT(*) as message_count
FROM messages
GROUP BY DATE(created_at);

# 活跃用户
SELECT COUNT(DISTINCT user_id) 
FROM messages 
WHERE created_at > NOW() - INTERVAL '1 day';
```

## 故障排查

### 1. 无法连接数据库

```bash
# 检查数据库是否运行
docker ps | grep postgres

# 测试连接
psql -h localhost -U imserver -d im_db

# 查看日志
docker logs im-postgres
```

### 2. Redis 连接失败

```bash
# 检查 Redis
docker ps | grep redis

# 测试连接
redis-cli -h localhost -p 6379 ping

# 查看日志
docker logs im-redis
```

### 3. WebSocket 连接失败

```bash
# 检查端口
netstat -an | grep 8081

# 测试连接
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: test" \
  http://localhost:8081/ws
```

### 4. 认证失败

- 检查 Token 是否有效
- 检查用户是否存在
- 检查 JWT Secret 配置
- 查看服务器日志

## 下一步

- 阅读 [API 文档](API.md) 了解详细接口
- 查看 [README](../README.md) 了解完整功能
- 探索源代码了解实现细节
- 根据需求定制开发

## 获取帮助

- GitHub Issues: 提交问题
- 查看日志: `logs/im-server.log`
- 开启 Debug 模式: 修改配置 `logger.level = "debug"`

