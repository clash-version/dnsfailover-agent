package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "dnsfailover",
		Short: "DNS Failover Cloudflare域名故障转移系统",
		Long: `DNS Failover是一个自动监控域名可用性的工具，
当检测到域名不可达时，自动切换到备用地址，实现域名故障转移。`,
		Version: "1.0.0",
	}
)

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 全局flags - 支持本地文件路径或HTTP(S) URL
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.json", "配置文件路径或URL (支持 http:// 或 https://)")
}

// GetConfigFile 获取配置文件路径
func GetConfigFile() string {
	return cfgFile
}
