# IM 即时通讯服务器

[English](README.md) | 简体中文

一个使用 Go 语言开发的高性能、可扩展的即时通讯服务器。

## 主要特性

- 🚀 高性能并发处理
- 💬 支持单聊和群聊
- 📱 多平台支持（iOS、Android、Web）
- 🔐 JWT 安全认证
- 💾 消息持久化存储
- ⚡ Redis 缓存加速
- 🔄 离线消息同步
- ✅ 消息已读回执
- 🐳 Docker 容器化部署

## 快速开始

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 本地开发

```bash
# 安装依赖
make deps

# 编译
make build

# 运行
make run
```

## 详细文档

请参考 [README.md](README.md) 获取完整文档。

## 技术栈

- **语言**: Go 1.21+
- **数据库**: PostgreSQL 15+ / MySQL 8+
- **缓存**: Redis 7+
- **协议**: WebSocket + Protocol Buffers
- **部署**: Docker + Docker Compose

## 项目状态

✅ 核心功能完成
🚧 持续优化中

## 许可证

MIT License

