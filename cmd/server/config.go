package main

import (
	"github.com/spf13/viper"
)

// Config 配置
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Logger     LoggerConfig     `mapstructure:"logger"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Message    MessageConfig    `mapstructure:"message"`
	Connection ConnectionConfig `mapstructure:"connection"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Name     string `mapstructure:"name"`
	Mode     string `mapstructure:"mode"`
	HTTPPort int    `mapstructure:"http_port"`
	WSPort   int    `mapstructure:"ws_port"`
	TCPPort  int    `mapstructure:"tcp_port"`
}

type DatabaseConfig struct {
	Type            string `mapstructure:"type"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type LoggerConfig struct {
	Level   string `mapstructure:"level"`
	Format  string `mapstructure:"format"`
	Output  string `mapstructure:"output"`
	Console bool   `mapstructure:"console"`
}

type AuthConfig struct {
	JWTSecret        string `mapstructure:"jwt_secret"`
	TokenExpireHours int    `mapstructure:"token_expire_hours"`
}

type MessageConfig struct {
	BatchSize   int `mapstructure:"batch_size"`
	MaxLength   int `mapstructure:"max_length"`
	OfflineDays int `mapstructure:"offline_days"`
}

type ConnectionConfig struct {
	HeartbeatInterval int `mapstructure:"heartbeat_interval"`
	HeartbeatTimeout  int `mapstructure:"heartbeat_timeout"`
	ReadTimeout       int `mapstructure:"read_timeout"`
	WriteTimeout      int `mapstructure:"write_timeout"`
	MaxMessageSize    int `mapstructure:"max_message_size"`
}

type RateLimitConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	RequestsPerSecond  int  `mapstructure:"requests_per_second"`
	Burst              int  `mapstructure:"burst"`
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.ws_port", 8081)
	viper.SetDefault("server.tcp_port", 8082)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

