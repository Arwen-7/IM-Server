# IM Server API 文档

## 协议概述

IM Server 使用 WebSocket 进行实时通信，消息格式基于 Protocol Buffers。

### 连接地址

```
ws://your-server:8081/ws
```

## 消息格式

所有消息使用 `WebSocketMessage` 封装：

```protobuf
message WebSocketMessage {
    CommandType command = 1;     // 命令类型
    uint32 sequence = 2;         // 序列号（用于请求响应匹配）
    bytes body = 3;              // 消息体（具体消息的序列化数据）
    int64 timestamp = 4;         // 时间戳（毫秒）
}
```

## 命令类型

### 认证相关

#### 1. 认证请求 (CMD_AUTH_REQ = 100)

**请求**:
```protobuf
message AuthRequest {
    string user_id = 1;
    string token = 2;          // JWT Token
    string platform = 3;       // iOS/Android/Web
}
```

**响应**:
```protobuf
message AuthResponse {
    ErrorCode error_code = 1;
    string error_msg = 2;
    int64 max_seq = 3;         // 用户当前最大序列号
}
```

**示例**:
```json
{
  "command": "CMD_AUTH_REQ",
  "sequence": 1,
  "body": {
    "user_id": "user123",
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "platform": "iOS"
  }
}
```

### 心跳相关

#### 2. 心跳请求 (CMD_HEARTBEAT_REQ = 5)

**请求**:
```protobuf
message HeartbeatRequest {
    int64 client_time = 1;
}
```

**响应**:
```protobuf
message HeartbeatResponse {
    int64 server_time = 1;
}
```

### 消息相关

#### 3. 发送消息 (CMD_SEND_MSG_REQ = 200)

**请求**:
```protobuf
message SendMessageRequest {
    string client_msg_id = 1;      // 客户端消息ID（去重用）
    string conversation_id = 2;     // 会话ID
    string sender_id = 3;           // 发送者ID
    string receiver_id = 4;         // 接收者ID（单聊）
    string group_id = 5;            // 群组ID（群聊）
    int32 message_type = 6;         // 消息类型
    bytes content = 7;              // 消息内容
    int64 send_time = 8;            // 发送时间
    map<string, string> extra = 10; // 扩展字段
}
```

**消息类型**:
- 1: 文本消息
- 2: 图片消息
- 3: 语音消息
- 4: 视频消息
- 5: 文件消息

**响应**:
```protobuf
message SendMessageResponse {
    ErrorCode error_code = 1;
    string error_msg = 2;
    string message_id = 3;         // 服务器生成的消息ID
    int64 seq = 4;                 // 消息序列号
    int64 server_time = 5;         // 服务器时间
}
```

#### 4. 接收消息推送 (CMD_PUSH_MSG = 202)

**服务器推送**:
```protobuf
message PushMessage {
    string message_id = 1;
    string client_msg_id = 2;
    string conversation_id = 3;
    string sender_id = 4;
    string receiver_id = 5;
    string group_id = 6;
    int32 message_type = 7;
    bytes content = 8;
    int64 send_time = 9;
    int64 server_time = 10;
    int64 seq = 11;
    map<string, string> extra = 15;
}
```

#### 5. 消息确认 (CMD_MSG_ACK = 203)

**请求**:
```protobuf
message MessageAck {
    string message_id = 1;
    int64 seq = 2;
}
```

### 同步相关

#### 6. 消息同步 (CMD_SYNC_REQ = 300)

**请求**:
```protobuf
message SyncRequest {
    int64 min_seq = 1;             // 客户端最小seq
    int64 max_seq = 2;             // 客户端最大seq
    int32 limit = 3;               // 每次拉取数量
}
```

**响应**:
```protobuf
message SyncResponse {
    ErrorCode error_code = 1;
    string error_msg = 2;
    repeated PushMessage messages = 3;
    int64 server_max_seq = 4;      // 服务器当前最大seq
    bool has_more = 5;             // 是否还有更多消息
}
```

### 已读回执

#### 7. 已读回执 (CMD_READ_RECEIPT_REQ = 500)

**请求**:
```protobuf
message ReadReceiptRequest {
    repeated string message_ids = 1;
    string conversation_id = 2;
}
```

**响应**:
```protobuf
message ReadReceiptResponse {
    ErrorCode error_code = 1;
    string error_msg = 2;
}
```

**推送**:
```protobuf
message ReadReceiptPush {
    repeated string message_ids = 1;
    string conversation_id = 2;
    string user_id = 3;
    int64 read_time = 4;
}
```

### 输入状态

#### 8. 输入状态 (CMD_TYPING_STATUS_REQ = 600)

**请求**:
```protobuf
message TypingStatusRequest {
    string conversation_id = 1;
    int32 status = 2;              // 0: 停止输入，1: 正在输入
}
```

**推送**:
```protobuf
message TypingStatusPush {
    string conversation_id = 1;
    string user_id = 2;
    int32 status = 3;
}
```

### 消息撤回

#### 9. 撤回消息 (CMD_REVOKE_MSG_REQ = 205)

**请求**:
```protobuf
message RevokeMessageRequest {
    string message_id = 1;
    string conversation_id = 2;
}
```

**响应**:
```protobuf
message RevokeMessageResponse {
    ErrorCode error_code = 1;
    string error_msg = 2;
}
```

**推送**:
```protobuf
message RevokeMessagePush {
    string message_id = 1;
    string conversation_id = 2;
    string revoked_by = 3;
    int64 revoked_time = 4;
}
```

## 错误码

```protobuf
enum ErrorCode {
    ERR_SUCCESS = 0;                   // 成功
    ERR_UNKNOWN = 1;                   // 未知错误
    ERR_INVALID_PARAM = 2;             // 参数错误
    ERR_AUTH_FAILED = 100;             // 认证失败
    ERR_TOKEN_EXPIRED = 101;           // Token过期
    ERR_PERMISSION_DENIED = 102;       // 权限不足
    ERR_USER_NOT_EXIST = 103;          // 用户不存在
    ERR_MESSAGE_TOO_LARGE = 200;       // 消息过大
    ERR_SEND_TOO_FAST = 201;           // 发送过快
    ERR_CONVERSATION_NOT_EXIST = 202;  // 会话不存在
}
```

## 使用流程

### 1. 建立连接和认证

```
Client                          Server
  |                               |
  |--- WebSocket Connect -------->|
  |<-- WebSocket Connected -------|
  |                               |
  |--- CMD_AUTH_REQ ------------->|
  |<-- CMD_AUTH_RSP --------------|
  |   (含 max_seq)                |
```

### 2. 消息同步

```
Client                          Server
  |                               |
  |--- CMD_SYNC_REQ ------------->|
  |   (max_seq, limit)            |
  |<-- CMD_SYNC_RSP --------------|
  |   (messages, has_more)        |
```

### 3. 发送和接收消息

```
Client A                Server                Client B
  |                       |                       |
  |--- CMD_SEND_MSG_REQ ->|                       |
  |<-- CMD_SEND_MSG_RSP --|                       |
  |                       |--- CMD_PUSH_MSG ----->|
  |                       |<-- CMD_MSG_ACK -------|
```

### 4. 心跳保活

```
Client                          Server
  |                               |
  |--- CMD_HEARTBEAT_REQ -------->|
  |<-- CMD_HEARTBEAT_RSP ---------|
  |    (每30秒)                   |
```

## 客户端实现建议

1. **连接管理**
   - 实现自动重连机制
   - 处理网络切换
   - 监听连接状态变化

2. **消息去重**
   - 使用 client_msg_id 去重
   - 检查消息 seq 连续性

3. **离线消息**
   - 认证后立即同步消息
   - 记录本地最大 seq
   - 定期检查消息缺失

4. **心跳机制**
   - 每30秒发送心跳
   - 超时后重连

5. **消息确认**
   - 收到消息后发送 ACK
   - 处理重复消息

## 测试工具

可以使用以下工具测试 WebSocket 连接：

- **在线工具**: [websocket.org/echo.html](http://www.websocket.org/echo.html)
- **命令行**: wscat (`npm install -g wscat`)
- **浏览器**: Chrome DevTools

### wscat 示例

```bash
# 连接服务器
wscat -c ws://localhost:8081/ws -b

# 发送二进制消息（需要先序列化为 protobuf）
```

## 注意事项

1. 所有时间戳使用毫秒（UnixMilli）
2. 消息体必须使用 Protocol Buffers 序列化
3. WebSocket 使用二进制消息（BinaryMessage）
4. sequence 用于请求响应匹配，客户端自行维护
5. 认证失败会断开连接
6. 心跳超时（90秒）会断开连接

