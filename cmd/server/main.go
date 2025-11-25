package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/arwen/im-server/internal/handler"
	"github.com/arwen/im-server/internal/repository"
	"github.com/arwen/im-server/internal/service"
	"github.com/arwen/im-server/internal/transport"
	"github.com/arwen/im-server/pkg/logger"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "config/config.yaml", "config file path")
)

func main() {
	flag.Parse()

	// 加载配置
	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(config.Logger.Level, config.Logger.Format, config.Logger.Output, config.Logger.Console); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting IM Server", zap.String("version", "1.0.0"))

	// 初始化数据库
	dbConfig := &repository.DatabaseConfig{
		Type:            config.Database.Type,
		Host:            config.Database.Host,
		Port:            config.Database.Port,
		User:            config.Database.User,
		Password:        config.Database.Password,
		DBName:          config.Database.DBName,
		MaxOpenConns:    config.Database.MaxOpenConns,
		MaxIdleConns:    config.Database.MaxIdleConns,
		ConnMaxLifetime: config.Database.ConnMaxLifetime,
	}
	if err := repository.InitDatabase(dbConfig); err != nil {
		logger.Fatal("Failed to init database", zap.Error(err))
	}
	logger.Info("Database initialized")

	// 初始化Redis
	redisConfig := &repository.RedisConfig{
		Host:     config.Redis.Host,
		Port:     config.Redis.Port,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		PoolSize: config.Redis.PoolSize,
	}
	if err := repository.InitRedis(redisConfig); err != nil {
		logger.Fatal("Failed to init redis", zap.Error(err))
	}
	logger.Info("Redis initialized")

	// 创建服务
	userService := service.NewUserService(config.Auth.JWTSecret)
	messageService := service.NewMessageService()
	conversationService := service.NewConversationService()

	// 创建连接管理器
	connManager := transport.NewConnectionManager()

	// 创建消息处理器
	messageHandler := handler.NewMessageHandler(
		connManager,
		userService,
		messageService,
		conversationService,
	)

	// 创建TCP服务器（默认传输协议）
	tcpServer := transport.NewTCPServer(connManager, messageHandler)

	// 启动TCP服务器
	tcpAddr := fmt.Sprintf(":%d", config.Server.TCPPort)
	go func() {
		logger.Info("TCP server starting", zap.String("addr", tcpAddr))
		if err := tcpServer.Start(tcpAddr); err != nil {
			logger.Fatal("Failed to start TCP server", zap.Error(err))
		}
	}()

	// 创建WebSocket服务器
	wsServer := transport.NewWebSocketServer(connManager, messageHandler)

	// 启动WebSocket服务器
	wsAddr := fmt.Sprintf(":%d", config.Server.WSPort)
	go func() {
		logger.Info("WebSocket server starting", zap.String("addr", wsAddr))
		if err := wsServer.Start(wsAddr); err != nil {
			logger.Fatal("Failed to start WebSocket server", zap.Error(err))
		}
	}()

	// 启动HTTP API服务器
	httpHandler := handler.NewHTTPHandler(userService, messageService, conversationService)
	httpAddr := fmt.Sprintf(":%d", config.Server.HTTPPort)
	go func() {
		mux := http.NewServeMux()
		httpHandler.RegisterRoutes(mux)
		
		logger.Info("HTTP API server starting", zap.String("addr", httpAddr))
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			logger.Fatal("Failed to start HTTP API server", zap.Error(err))
		}
	}()

	logger.Info("IM Server started successfully",
		zap.Int("tcp_port", config.Server.TCPPort),
		zap.Int("ws_port", config.Server.WSPort),
		zap.Int("http_port", config.Server.HTTPPort))

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// TODO: 优雅关闭

	logger.Info("Server stopped")
}

