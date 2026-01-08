package cmd

import (
	"dnsfailover/internal/api"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/monitor"
	"dnsfailover/internal/schedule"
	"dnsfailover/internal/storage"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	daemonMode      bool
	apiPort         int
	enableWeb       bool
	scheduler       *monitor.Scheduler
	scheduleManager *schedule.Manager
	apiServer       *api.Server

	monitorCmd = &cobra.Command{
		Use:   "monitor",
		Short: "监控管理命令",
		Long:  "启动或停止探针监控服务",
	}

	monitorStartCmd = &cobra.Command{
		Use:   "start",
		Short: "启动监控服务",
		Long:  "启动探针监控服务，定期检测目标可用性，触发阈值时回调 Webhook",
		Run: func(cmd *cobra.Command, args []string) {
			// 初始化系统
			if err := InitSystem(); err != nil {
				fmt.Fprintf(os.Stderr, "系统初始化失败: %v\n", err)
				os.Exit(1)
			}

			// 创建探针调度器
			scheduler = monitor.NewScheduler(GetConfig())

			// 启动监控
			if err := scheduler.Start(); err != nil {
				logger.Errorf("启动监控失败: %v", err)
				os.Exit(1)
			}

			// 创建并启动定时任务调度器
			scheduleManager = schedule.NewManager()
			scheduleManager.SetTaskUpdateCallback(func(task *schedule.Task) error {
				// 任务执行后更新数据库状态
				store := storage.GetStorage()
				if store == nil {
					return nil
				}
				lastRunAt := ""
				if task.LastRunAt != nil {
					lastRunAt = task.LastRunAt.Format("2006-01-02 15:04:05")
				}
				return store.UpdateScheduleTaskStatus(task.ID, lastRunAt, task.LastResult)
			})

			// 从数据库加载已有任务
			if err := loadScheduleTasksFromDB(); err != nil {
				logger.Warnf("加载定时任务失败: %v", err)
			}

			scheduleManager.Start()
			logger.Infof("定时任务调度器已启动，共 %d 个任务", scheduleManager.GetTaskCount())

			// 启动 Web 管理界面
			if enableWeb {
				apiServer = api.NewServer(GetConfig(), scheduler, scheduleManager, apiPort)
				if err := apiServer.Start(); err != nil {
					logger.Warnf("启动 Web 管理界面失败: %v", err)
				}
			}

			if daemonMode {
				logger.Info("以后台模式运行")
			}

			// 等待中断信号
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			logger.Info("监控服务运行中，按 Ctrl+C 停止...")
			<-sigChan

			logger.Info("\n收到停止信号，正在关闭...")
			if apiServer != nil {
				apiServer.Stop()
			}
			if scheduleManager != nil {
				scheduleManager.Stop()
			}
			scheduler.Stop()
		},
	}

	monitorStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "停止监控服务",
		Long:  "停止正在运行的探针监控服务",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("停止监控服务...")
			fmt.Println("请使用 Ctrl+C 或 kill 命令停止监控进程")
		},
	}
)

// loadScheduleTasksFromDB 从数据库加载定时任务
func loadScheduleTasksFromDB() error {
	store := storage.GetStorage()
	if store == nil {
		return fmt.Errorf("数据库未初始化")
	}

	tasks, err := store.GetAllScheduleTasks()
	if err != nil {
		return err
	}

	for _, st := range tasks {
		task := &schedule.Task{
			ID:          st.ID,
			Name:        st.Name,
			Enabled:     st.Enabled,
			Cron:        st.Cron,
			CheckType:   schedule.CheckType(st.CheckType),
			Target:      st.Target,
			Port:        st.Port,
			Timeout:     st.Timeout,
			WebhookURL:  st.WebhookURL,
			WebhookData: st.WebhookData,
			LastResult:  st.LastResult,
		}

		if st.CreatedAt != "" {
			task.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", st.CreatedAt)
		}
		if st.UpdatedAt != "" {
			task.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", st.UpdatedAt)
		}
		if st.LastRunAt != nil {
			t, _ := time.Parse("2006-01-02 15:04:05", *st.LastRunAt)
			task.LastRunAt = &t
		}

		if err := scheduleManager.AddTask(task); err != nil {
			logger.Warnf("加载任务失败 [%s]: %v", st.Name, err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)

	monitorStartCmd.Flags().BoolVarP(&daemonMode, "daemon", "d", false, "以后台模式运行")
	monitorStartCmd.Flags().BoolVarP(&enableWeb, "web", "w", true, "启用 Web 管理界面")
	monitorStartCmd.Flags().IntVarP(&apiPort, "port", "p", 8080, "Web 管理界面端口")
}
