package config

import (
	"dnsfailover/internal/storage"
	"fmt"
)

// LoadFromDB 从 SQLite 数据库加载配置
func LoadFromDB(dbPath string) (*Config, error) {
	// 初始化存储
	store, err := storage.Init(dbPath)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	// 检查是否有配置
	hasConfig, err := store.HasConfig()
	if err != nil {
		return nil, fmt.Errorf("检查配置失败: %w", err)
	}

	// 如果没有配置，创建默认配置
	if !hasConfig {
		defaultCfg := storage.GetDefaultConfig()
		if err := store.SaveConfig(defaultCfg); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %w", err)
		}
	}

	// 加载配置
	storedCfg, err := store.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 转换为 Config 结构
	cfg := &Config{}
	cfg.DBPath = dbPath
	ApplyStorageConfig(cfg, storedCfg)

	return cfg, nil
}

// ApplyStorageConfig 将存储配置应用到主配置
func ApplyStorageConfig(cfg *Config, storedCfg *storage.FullConfig) {
	// Ping 配置
	cfg.Ping = ProbeConfig{
		Enabled:          storedCfg.Ping.Enabled,
		Frequency:        storedCfg.Ping.Frequency,
		FailCount:        storedCfg.Ping.FailCount,
		Timeout:          storedCfg.Ping.Timeout,
		Retry:            storedCfg.Ping.Retry,
		RemoteUpdateFreq: storedCfg.Ping.RemoteUpdateFreq,
		Domains:          storedCfg.Ping.Domains,
	}

	// TCP 配置
	cfg.Tcp = ProbeConfig{
		Enabled:          storedCfg.Tcp.Enabled,
		Frequency:        storedCfg.Tcp.Frequency,
		FailCount:        storedCfg.Tcp.FailCount,
		Timeout:          storedCfg.Tcp.Timeout,
		Retry:            storedCfg.Tcp.Retry,
		RemoteUpdateFreq: storedCfg.Tcp.RemoteUpdateFreq,
		Domains:          storedCfg.Tcp.Domains,
	}

	// HTTP 配置
	cfg.Http = ProbeConfig{
		Enabled:          storedCfg.Http.Enabled,
		Frequency:        storedCfg.Http.Frequency,
		FailCount:        storedCfg.Http.FailCount,
		Timeout:          storedCfg.Http.Timeout,
		Retry:            storedCfg.Http.Retry,
		RemoteUpdateFreq: storedCfg.Http.RemoteUpdateFreq,
		Domains:          storedCfg.Http.Domains,
	}

	// Webhook 配置
	cfg.Webhook = WebhookConfig{
		URL:           storedCfg.Webhook.URL,
		Method:        storedCfg.Webhook.Method,
		Headers:       storedCfg.Webhook.Headers,
		Timeout:       storedCfg.Webhook.Timeout,
		Retry:         storedCfg.Webhook.Retry,
		SilencePeriod: storedCfg.Webhook.SilencePeriod,
	}
}

// SaveToDB 保存配置到 SQLite 数据库
func SaveToDB(cfg *Config) error {
	store := storage.GetStorage()
	if store == nil {
		return fmt.Errorf("数据库未初始化")
	}

	storedCfg := &storage.FullConfig{
		Ping: storage.ProbeConfig{
			Enabled:          cfg.Ping.Enabled,
			Frequency:        cfg.Ping.Frequency,
			FailCount:        cfg.Ping.FailCount,
			Timeout:          cfg.Ping.Timeout,
			Retry:            cfg.Ping.Retry,
			RemoteUpdateFreq: cfg.Ping.RemoteUpdateFreq,
			Domains:          cfg.Ping.Domains,
		},
		Tcp: storage.ProbeConfig{
			Enabled:          cfg.Tcp.Enabled,
			Frequency:        cfg.Tcp.Frequency,
			FailCount:        cfg.Tcp.FailCount,
			Timeout:          cfg.Tcp.Timeout,
			Retry:            cfg.Tcp.Retry,
			RemoteUpdateFreq: cfg.Tcp.RemoteUpdateFreq,
			Domains:          cfg.Tcp.Domains,
		},
		Http: storage.ProbeConfig{
			Enabled:          cfg.Http.Enabled,
			Frequency:        cfg.Http.Frequency,
			FailCount:        cfg.Http.FailCount,
			Timeout:          cfg.Http.Timeout,
			Retry:            cfg.Http.Retry,
			RemoteUpdateFreq: cfg.Http.RemoteUpdateFreq,
			Domains:          cfg.Http.Domains,
		},
		Webhook: storage.WebhookConfig{
			URL:           cfg.Webhook.URL,
			Method:        cfg.Webhook.Method,
			Headers:       cfg.Webhook.Headers,
			Timeout:       cfg.Webhook.Timeout,
			Retry:         cfg.Webhook.Retry,
			SilencePeriod: cfg.Webhook.SilencePeriod,
		},
	}

	return store.SaveConfig(storedCfg)
}
