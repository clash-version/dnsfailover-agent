package cmd

import (
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	globalConfig *config.Config
	initOnce     sync.Once
	initError    error
)

func InitSystem() error {
	initOnce.Do(func() {
		// 从环境变量加载基础配置（日志配置等）
		baseCfg, err := config.Load()
		if err != nil {
			initError = fmt.Errorf("配置加载失败: %w", err)
			return
		}

		// 根据配置决定是否启用文件日志
		if baseCfg.Log.Enabled {
			err = logger.Init(baseCfg.Log.Level, baseCfg.Log.Path, baseCfg.Log.MaxDays)
			if err != nil {
				initError = fmt.Errorf("日志初始化失败: %w", err)
				return
			}
			logger.Info("日志系统初始化成功（文件+控制台）")
		} else {
			logger.InitConsoleOnly(baseCfg.Log.Level)
			logger.Info("日志系统初始化成功（仅控制台）")
		}

		// 确保数据库目录存在
		dbDir := filepath.Dir(baseCfg.DBPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			initError = fmt.Errorf("创建数据库目录失败: %w", err)
			return
		}

		// 初始化 SQLite 并加载配置
		logger.Infof("数据库路径: %s", baseCfg.DBPath)
		_, err = storage.Init(baseCfg.DBPath)
		if err != nil {
			initError = fmt.Errorf("数据库初始化失败: %w", err)
			return
		}

		// 从数据库加载配置
		cfg, err := config.LoadFromDB(baseCfg.DBPath)
		if err != nil {
			initError = fmt.Errorf("从数据库加载配置失败: %w", err)
			return
		}

		// 合并日志配置
		cfg.Log = baseCfg.Log
		cfg.DBPath = baseCfg.DBPath

		// 环境变量中的 Webhook 可覆盖数据库配置
		if baseCfg.Webhook.URL != "" {
			cfg.Webhook.URL = baseCfg.Webhook.URL
		}

		globalConfig = cfg

		if cfg.Webhook.URL != "" {
			logger.Infof("Webhook 地址: %s", cfg.Webhook.URL)
		}
		logger.Info("系统初始化完成")
	})
	return initError
}

func GetConfig() *config.Config {
	return globalConfig
}
