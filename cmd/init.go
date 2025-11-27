package cmd

import (
"dnsfailover/internal/cloudflare"
"dnsfailover/internal/config"
"dnsfailover/internal/logger"
"fmt"
"os"
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
cfg, err := config.Load(cfgFile)
if err != nil {
if os.IsNotExist(err) {
fmt.Println("配置文件不存在，正在生成默认配置...")
if err := config.GenerateDefault(cfgFile); err != nil {
initError = fmt.Errorf("生成配置失败: %w", err)
return
}
fmt.Printf("已生成: %s\n", cfgFile)
fmt.Println("请填入Cloudflare API Token后重新运行")
os.Exit(0)
}
initError = err
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

// 跳过验证，直接在实际使用时验证（避免权限不足的警告）
logger.Info("Cloudflare客户端初始化成功")

logger.Infof("监控域名: %d 个", len(cfg.Ping.Domains))
logger.Infof("Failover地址: %d 个", len(cfg.Ping.Failover))
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
