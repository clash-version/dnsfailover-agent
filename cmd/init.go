package cmd

import (
	"dnsfailover/internal/cloudflare"
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"fmt"
	"sync"
)

var (
	globalConfig   *config.Config
	globalCFClient *cloudflare.Client
	initOnce       sync.Once
	initError      error
)

func InitSystem() error {
	initOnce.Do(func() {
		// 从环境变量加载配置
		cfg, err := config.Load()
		if err != nil {
			initError = fmt.Errorf("配置加载失败: %w", err)
			return
		}
		globalConfig = cfg

		// 根据配置决定是否启用文件日志
		if cfg.Log.Enabled {
			err = logger.Init(cfg.Log.Level, cfg.Log.Path, cfg.Log.MaxDays)
			if err != nil {
				initError = fmt.Errorf("日志初始化失败: %w", err)
				return
			}
			logger.Info("日志系统初始化成功（文件+控制台）")
		} else {
			logger.InitConsoleOnly(cfg.Log.Level)
			logger.Info("日志系统初始化成功（仅控制台）")
		}

		cfClient, err := cloudflare.NewClient(cfg.DNS.APIToken)
		if err != nil {
			initError = fmt.Errorf("Cloudflare客户端失败: %w", err)
			return
		}
		globalCFClient = cfClient

		logger.Info("Cloudflare客户端初始化成功")
		logger.Infof("远程配置地址: %s", cfg.Ping.RemoteConfigURL)
		logger.Info("系统初始化完成")
	})
	return initError
}

func GetConfig() *config.Config {
	return globalConfig
}

func GetCFClient() *cloudflare.Client {
	return globalCFClient
}
