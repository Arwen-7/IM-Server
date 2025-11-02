# IM Server 架构设计

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                         │
│  (iOS SDK / Android SDK / Web SDK)                         │
└─────────────────────────────────────────────────────────────┘
                            │
                    WebSocket / TCP
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Transport Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  WebSocket   │  │   TCP        │  │   HTTP       │      │
│  │  Server      │  │   Server     │  │   API        │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                     Connection Manager                      │
│  - 连接管理   - 用户绑定   - 消息路由                      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Handler Layer                          │
│  - 消息处理   - 协议解析   - 业务分发                      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   User   │  │ Message  │  │  Friend  │  │  Group   │   │
│  │ Service  │  │ Service  │  │ Service  │  │ Service  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                         │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │   PostgreSQL     │         │      Redis       │         │
│  │   / MySQL        │         │     Cache        │         │
│  └──────────────────┘         └──────────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## 核心模块

### 1. Transport Layer (传输层)

负责网络连接和数据传输：

- **WebSocket Server**: 实时双向通信
- **TCP Server**: 长连接支持（待实现）
- **HTTP Server**: RESTful API（待实现）

**特点**:
- 支持多种传输协议
- 自动心跳检测
- 断线重连机制
- 消息缓冲队列

### 2. Connection Manager (连接管理器)

管理所有客户端连接：

- 连接的创建、维护、销毁
- 用户与连接的绑定关系
- 多端登录管理（踢人逻辑）
- 在线状态维护

**数据结构**:
```go
type ConnectionManager struct {
    connections map[string]Connection  // connID -> Connection
    userConns   map[string]string      // userID -> connID
}
```

### 3. Handler Layer (处理层)

处理各种业务消息：

- Protocol Buffer 解析
- 命令分发路由
- 业务逻辑调用
- 响应消息构建

**主要Handler**:
- AuthHandler: 认证处理
- MessageHandler: 消息处理
- SyncHandler: 同步处理
- TypingHandler: 输入状态处理

### 4. Service Layer (业务层)

实现核心业务逻辑：

#### UserService
- 用户注册/登录
- Token 验证
- 用户信息管理

#### MessageService
- 消息存储
- 消息查询
- 序列号生成
- 消息同步
- 消息撤回

#### ConversationService
- 会话管理
- 未读计数
- 最后消息更新

#### FriendService (待完善)
- 好友关系管理
- 好友请求处理

#### GroupService (待完善)
- 群组管理
- 成员管理

### 5. Repository Layer (数据层)

数据持久化和缓存：

#### Database
- 用户数据
- 消息数据
- 会话数据
- 关系数据

#### Redis Cache
- 会话缓存
- 在线状态
- 用户连接映射
- 临时数据

## 数据模型

### 核心表结构

#### users (用户表)
```sql
- id: 用户ID
- username: 用户名
- nickname: 昵称
- password: 密码(bcrypt)
- avatar: 头像
- status: 状态
- created_at, updated_at
```

#### messages (消息表)
```sql
- id: 消息ID
- client_msg_id: 客户端消息ID
- conversation_id: 会话ID
- sender_id: 发送者ID
- receiver_id: 接收者ID
- group_id: 群组ID
- message_type: 消息类型
- content: 消息内容
- seq: 序列号
- status: 状态
- send_time, server_time
- created_at, updated_at
```

#### conversations (会话表)
```sql
- id: 会话ID
- type: 类型(1:单聊,2:群聊)
- user_id: 用户ID
- target_id: 目标ID
- last_message: 最后消息
- unread_count: 未读数
- status: 状态
- created_at, updated_at
```

## 消息流程

### 发送消息流程

```
1. 客户端发送消息
   ├─ WebSocket 连接
   ├─ 序列化 Protocol Buffer
   └─ 发送 CMD_SEND_MSG_REQ

2. 服务器接收
   ├─ 反序列化消息
   ├─ 验证用户权限
   ├─ 生成消息ID和序列号
   └─ 存储到数据库

3. 服务器响应发送者
   ├─ 构建 CMD_SEND_MSG_RSP
   ├─ 序列化响应
   └─ 发送给发送者

4. 服务器推送接收者
   ├─ 查找接收者连接
   ├─ 构建 CMD_PUSH_MSG
   ├─ 序列化推送消息
   └─ 发送给接收者

5. 接收者确认
   ├─ 接收推送消息
   ├─ 发送 CMD_MSG_ACK
   └─ 更新消息状态
```

### 消息同步流程

```
1. 客户端请求同步
   ├─ 发送 CMD_SYNC_REQ
   └─ 携带本地 max_seq

2. 服务器查询
   ├─ 查询 seq > max_seq 的消息
   ├─ 限制数量（分页）
   └─ 查询服务器 max_seq

3. 服务器响应
   ├─ 返回消息列表
   ├─ 返回 server_max_seq
   └─ 返回 has_more 标志

4. 客户端处理
   ├─ 保存新消息
   ├─ 更新本地 max_seq
   └─ 如果 has_more，继续同步
```

## 协议设计

### Protocol Buffer 优势

1. **高效**: 比 JSON 更小更快
2. **强类型**: 类型安全
3. **跨平台**: 多语言支持
4. **向后兼容**: 协议可扩展

### 消息封装

```
WebSocketMessage
├─ command: 命令类型
├─ sequence: 序列号
├─ body: 具体消息
└─ timestamp: 时间戳
```

## 扩展性设计

### 水平扩展

1. **多实例部署**
   - 使用负载均衡
   - Redis 共享状态
   - 消息队列通信

2. **数据库分片**
   - 按用户ID分片
   - 读写分离

3. **Redis 集群**
   - 主从复制
   - 哨兵模式
   - 集群模式

### 消息队列集成

```
Server A                 MQ                  Server B
   │                     │                      │
   │─── Publish Msg ────>│                      │
   │                     │───── Subscribe ─────>│
   │                     │                      │
```

## 性能优化

### 连接优化
- 连接池复用
- 心跳保活
- 快速失败重连

### 消息优化
- 批量处理
- 消息压缩
- 预读缓存

### 数据库优化
- 索引优化
- 查询优化
- 连接池管理

### 缓存优化
- 热点数据缓存
- 缓存预热
- 缓存更新策略

## 安全设计

### 认证机制
- JWT Token 认证
- Token 过期管理
- 刷新Token机制

### 数据安全
- 密码 bcrypt 加密
- 敏感数据脱敏
- SQL 注入防护

### 传输安全
- WSS (WebSocket Secure)
- TLS 加密
- 证书验证

### 限流防护
- 请求频率限制
- IP 黑名单
- 异常检测

## 监控和运维

### 日志系统
- 结构化日志
- 日志级别分类
- 日志轮转

### 监控指标
- 连接数
- 消息吞吐量
- 响应时间
- 错误率

### 告警机制
- 服务异常告警
- 性能告警
- 容量告警

## 未来规划

1. **功能增强**
   - 群组管理完善
   - 好友系统完善
   - 文件传输
   - 音视频通话

2. **性能提升**
   - 消息队列集成
   - 分布式部署
   - 缓存优化

3. **运维工具**
   - 管理后台
   - 监控面板
   - 数据分析

4. **安全加固**
   - 消息加密
   - 审计日志
   - 合规支持

