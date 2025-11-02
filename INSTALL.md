# IM Server 安装指南

## macOS 本地开发环境配置

### 步骤 1: 安装 Homebrew（如已安装可跳过）

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 步骤 2: 安装 Go

```bash
# 安装 Go 1.21+
brew install go

# 验证安装
go version

# 配置环境变量（添加到 ~/.zshrc）
echo 'export GOPATH=$HOME/go' >> ~/.zshrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.zshrc
source ~/.zshrc
```

### 步骤 3: 安装 Protocol Buffers 编译器

```bash
# 安装 protobuf
brew install protobuf

# 验证安装
protoc --version
```

### 步骤 4: 安装 Go Protocol Buffers 插件

```bash
# 安装 protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 验证安装（确保 $GOPATH/bin 在 PATH 中）
which protoc-gen-go
```

### 步骤 5: 安装数据库和 Redis

```bash
# 使用 Docker 运行（推荐）
docker run -d --name postgres \
  -e POSTGRES_USER=imserver \
  -e POSTGRES_PASSWORD=imserver123 \
  -e POSTGRES_DB=im_db \
  -p 5432:5432 \
  postgres:15-alpine

docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 或者使用 Homebrew 安装
# brew install postgresql@15
# brew install redis
# brew services start postgresql@15
# brew services start redis
```

### 步骤 6: 克隆项目并配置

```bash
# 进入项目目录
cd /Users/arwen/Project/IM/IM-Server

# 生成 Protocol Buffer 代码
make proto

# 下载 Go 依赖
make deps

# 配置数据库连接（如需修改）
# 编辑 config/config.yaml
```

### 步骤 7: 编译和运行

```bash
# 编译项目
make build

# 运行服务器
make run

# 或者直接运行
./bin/im-server -config=config/config.yaml
```

## 验证安装

### 检查服务状态

```bash
# 检查数据库
psql -h localhost -U imserver -d im_db -c "SELECT version();"

# 检查 Redis
redis-cli ping

# 检查 IM Server（打开另一个终端）
curl http://localhost:8080/health  # 如果实现了健康检查接口
```

### 测试 WebSocket 连接

```bash
# 安装 wscat
npm install -g wscat

# 连接测试
wscat -c ws://localhost:8081/ws
```

## 常见问题

### Q: protoc-gen-go: command not found

A: 确保 `$GOPATH/bin` 在 PATH 中：
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Q: 无法连接数据库

A: 检查数据库是否运行：
```bash
docker ps  # 查看容器状态
# 或
brew services list  # 查看服务状态
```

### Q: 端口被占用

A: 修改 `config/config.yaml` 中的端口配置

### Q: Go 依赖下载失败

A: 配置 Go 代理：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

## 开发工具推荐

- **IDE**: VS Code + Go 扩展
- **数据库客户端**: TablePlus, DBeaver
- **API 测试**: Postman, Insomnia
- **WebSocket 测试**: wscat, Postman

## 下一步

- 查看 [README.md](README.md) 了解项目功能
- 查看 [docs/API.md](docs/API.md) 了解 API 接口
- 查看 [docs/QUICKSTART.md](docs/QUICKSTART.md) 快速开始

## 获取帮助

遇到问题？
- 查看日志: `logs/im-server.log`
- 开启 Debug: 修改 `config/config.yaml` 中 `logger.level = "debug"`
- 提交 Issue: [GitHub Issues]

