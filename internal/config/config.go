package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 主配置结构
type Config struct {
	Ping    ProbeConfig
	Tcp     ProbeConfig
	Http    ProbeConfig
	Webhook WebhookConfig
	Log     LogConfig
	DBPath  string // SQLite 数据库路径
}

// WebhookConfig Webhook 回调配置
type WebhookConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Timeout       int               `json:"timeout"`
	Retry         int               `json:"retry"`
	SilencePeriod int               `json:"silence_period"` // 静默期（秒），发送告警后暂停检测的时间
}

// ProbeConfig 通用检测配置
type ProbeConfig struct {
	Enabled          bool     `json:"enabled"`
	Frequency        int      `json:"frequency"`
	FailCount        int      `json:"failcount"`
	Timeout          int      `json:"timeout"`
	Retry            int      `json:"retry"`
	RemoteUpdateFreq int      `json:"remote_update_freq"`
	Domains          []string `json:"domains"`
}

// LogConfig 日志配置
type LogConfig struct {
	Enabled bool
	Level   string
	Path    string
	MaxDays int
}

// Load 从 .env 文件加载基础配置（日志配置等）
func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{}

	// 数据库路径
	cfg.DBPath = getEnvString("DB_PATH", "./data/probe.db")

	// Webhook 配置（可从环境变量覆盖）
	cfg.Webhook.URL = os.Getenv("WEBHOOK_URL")
	cfg.Webhook.Method = getEnvString("WEBHOOK_METHOD", "POST")
	cfg.Webhook.Timeout = getEnvInt("WEBHOOK_TIMEOUT", 10)

	// 日志配置
	cfg.Log.Enabled = getEnvBool("LOG_ENABLED", true)
	cfg.Log.Level = getEnvString("LOG_LEVEL", "info")
	cfg.Log.Path = getEnvString("LOG_PATH", "./logs/probe.log")
	cfg.Log.MaxDays = getEnvInt("LOG_MAX_DAYS", 30)

	return cfg, nil
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
