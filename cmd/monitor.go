package cmd

import (
	"dnsfailover/internal/logger"
	"dnsfailover/internal/monitor"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	daemonMode bool
	scheduler  *monitor.Scheduler
	
	monitorCmd = &cobra.Command{
		Use:   "monitor",
		Short: "监控管理命令",
		Long:  "启动或停止域名监控服务",
	}

	monitorStartCmd = &cobra.Command{
		Use:   "start",
		Short: "启动监控服务",
		Long:  "启动域名监控服务，定期检测域名可用性",
		Run: func(cmd *cobra.Command, args []string) {
			// 初始化系统
			if err := InitSystem(); err != nil {
				fmt.Fprintf(os.Stderr, "系统初始化失败: %v\n", err)
				os.Exit(1)
			}

			// 创建调度器
			scheduler = monitor.NewScheduler(GetConfig(), GetCFClient())

			// 启动监控
			if err := scheduler.Start(); err != nil {
				logger.Errorf("启动监控失败: %v", err)
				os.Exit(1)
			}

			if daemonMode {
				logger.Info("以后台模式运行")
				// TODO: 实现真正的daemon模式（需要使用systemd或其他服务管理工具）
			}

			// 等待中断信号
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			
			logger.Info("监控服务运行中，按 Ctrl+C 停止...")
			<-sigChan
			
			logger.Info("\n收到停止信号，正在关闭...")
			scheduler.Stop()
		},
	}

	monitorStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "停止监控服务",
		Long:  "停止正在运行的域名监控服务",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("停止监控服务...")
			// 此命令需要结合进程管理工具使用
			fmt.Println("请使用 Ctrl+C 或 kill 命令停止监控进程")
		},
	}
)

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	
	monitorStartCmd.Flags().BoolVarP(&daemonMode, "daemon", "d", false, "以后台模式运行")
}
