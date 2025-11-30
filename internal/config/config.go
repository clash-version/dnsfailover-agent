package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 主配置结构
type Config struct {
	DNS  DNSConfig  `json:"dns"`
	Ping PingConfig `json:"ping"`
	Log  LogConfig  `json:"log"`
}

// DNSConfig DNS配置（仅支持API Token）
type DNSConfig struct {
	APIToken string `json:"api_token"` // Cloudflare API Token
}

// PingConfig Ping检测配置
type PingConfig struct {
	Frequency        int              `json:"frequency"`          // 检测频率(秒)
	FailCount        int              `json:"failcount"`          // 失败阈值
	Timeout          int              `json:"timeout"`            // 超时时间(秒)
	Retry            int              `json:"retry"`              // DNS更新重试次数
	DNSServer        string           `json:"dns_server"`         // 自定义DNS服务器(可选，如: 1.1.1.1:53)
	RemoteConfigURL  string           `json:"remote_config_url"`  // 远程配置URL（可选）
	RemoteUpdateFreq int              `json:"remote_update_freq"` // 远程配置更新频率(秒)，0表示不启用
	Domains          []string         `json:"domains"`            // 监控域名列表（本地配置）
	Failover         []FailoverTarget `json:"failover"`           // 备用地址列表（本地配置）
}

// FailoverTarget 备用地址
type FailoverTarget struct {
	Address string `json:"address"` // 备用地址
	Weight  int    `json:"weight"`  // 权重
}

// LogConfig 日志配置
type LogConfig struct {
	Enabled bool   `json:"enabled"`  // 是否启用文件日志
	Level   string `json:"level"`    // 日志级别
	Path    string `json:"path"`     // 日志文件路径
	MaxDays int    `json:"max_days"` // 日志保留天数
}

// Load 加载配置文件
func Load(configFile string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 设置默认值
	cfg.SetDefaults()

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证DNS配置（仅支持API Token）
	if c.DNS.APIToken == "" {
		return fmt.Errorf("dns.api_token 不能为空")
	}

	// 验证Ping配置
	if c.Ping.Frequency < 10 {
		return fmt.Errorf("ping.frequency 不能小于10秒")
	}
	if c.Ping.FailCount < 1 {
		return fmt.Errorf("ping.failcount 必须大于0")
	}

	return nil
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	// 设置Ping默认值
	if c.Ping.Timeout == 0 {
		c.Ping.Timeout = 5
	}
	if c.Ping.Retry == 0 {
		c.Ping.Retry = 3
	}
	if c.Ping.RemoteUpdateFreq == 0 && c.Ping.RemoteConfigURL != "" {
		c.Ping.RemoteUpdateFreq = 300 // 默认5分钟更新一次远程配置
	}

	// 设置日志默认值
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Path == "" {
		c.Log.Path = "./logs/failover.log"
	}
	if c.Log.MaxDays == 0 {
		c.Log.MaxDays = 30
	}
}

// GenerateDefault 生成默认配置文件
func GenerateDefault(configFile string) error {
	defaultConfig := Config{
		DNS: DNSConfig{
			APIToken: "your-cloudflare-api-token",
		},
		Ping: PingConfig{
			Frequency:        30,
			FailCount:        5,
			Timeout:          5,
			Retry:            3,
			RemoteConfigURL:  "", // 可选：远程配置地址，如 "https://download.clash.guide/dns/config.json"
			RemoteUpdateFreq: 0,  // 远程配置更新频率(秒)，0表示不启用，建议300秒(5分钟)
			Domains: []string{
				"example1.com",
				"example2.com",
			},
			Failover: []FailoverTarget{
				{Address: "backup1.example.com", Weight: 100},
				{Address: "backup2.example.com", Weight: 50},
			},
		},
		Log: LogConfig{
			Enabled: true,
			Level:   "info",
			Path:    "./logs/failover.log",
			MaxDays: 30,
		},
	}

	data, err := json.MarshalIndent(defaultConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("生成默认配置失败: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
