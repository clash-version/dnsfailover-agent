package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 主配置结构（从 .env 文件加载）
type Config struct {
	DNS  DNSConfig
	Ping PingConfig
	Tcp  TcpConfig
	Http HttpConfig
	Log  LogConfig
}

// DNSConfig DNS配置
type DNSConfig struct {
	APIToken string
}

// PingConfig Ping检测配置
type PingConfig struct {
	Frequency        int
	FailCount        int
	Timeout          int
	Retry            int
	RemoteConfigURL  string // 仅从 .env 读取
	RemoteUpdateFreq int
	Domains          []string
	Failover         []FailoverTarget
}

// FailoverTarget 备用地址
type FailoverTarget struct {
	Address string `json:"address"`
	Weight  int    `json:"weight"`
}

// TcpConfig TCP检测配置
type TcpConfig = ProbeConfig

// HttpConfig HTTP检测配置
type HttpConfig = ProbeConfig

// ProbeConfig 通用检测配置（TCP/HTTP共用）
type ProbeConfig struct {
	Frequency int
	FailCount int
	Timeout   int
	Retry     int
	Domains   []string
	Failover  []FailoverTarget
}

// LogConfig 日志配置
type LogConfig struct {
	Enabled bool
	Level   string
	Path    string
	MaxDays int
}

// Load 从 .env 文件加载配置
func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{}
	cfg.DNS.APIToken = os.Getenv("CF_API_TOKEN")
	cfg.Ping.RemoteConfigURL = os.Getenv("REMOTE_CONFIG_URL")
	cfg.Log.Enabled = getEnvBool("LOG_ENABLED", true)
	cfg.Log.Level = getEnvString("LOG_LEVEL", "info")
	cfg.Log.Path = getEnvString("LOG_PATH", "./logs/failover.log")
	cfg.Log.MaxDays = getEnvInt("LOG_MAX_DAYS", 30)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	return cfg, nil
}

// Validate 验证本地配置
func (c *Config) Validate() error {
	if c.DNS.APIToken == "" {
		return fmt.Errorf("环境变量 CF_API_TOKEN 不能为空")
	}
	if c.Ping.RemoteConfigURL == "" {
		return fmt.Errorf("环境变量 REMOTE_CONFIG_URL 不能为空")
	}
	return nil
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}
