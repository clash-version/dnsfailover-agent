package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "dnsfailover",
		Short: "DNS Failover Cloudflare域名故障转移系统",
		Long: `DNS Failover是一个自动监控域名可用性的工具，
当检测到域名不可达时，自动切换到备用地址，实现域名故障转移。

配置说明:
  - .env 文件: 存放敏感配置 (CF_API_TOKEN, AWS凭证, REMOTE_CONFIG_URL等)
  - 远程配置: 存放运行时配置 (ping参数, domains, failover地址)`,
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
