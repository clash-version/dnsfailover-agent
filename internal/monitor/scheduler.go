package monitor

import (
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/probe"
	"dnsfailover/internal/webhook"
	"fmt"
	"sync"
	"time"
)

// Scheduler 监控调度器
type Scheduler struct {
	cfg           *config.Config
	stateManager  *StateManager
	webhookClient *webhook.Client
	ticker        *time.Ticker
	stopChan      chan bool
	isRunning     bool
	mu            sync.Mutex
	configMu      sync.RWMutex // 配置读写锁

	// 检测器
	pingChecker *probe.PingChecker
	tcpChecker  *probe.TCPChecker
	httpChecker *probe.HTTPChecker
}

// NewScheduler 创建监控调度器
func NewScheduler(cfg *config.Config) *Scheduler {
	stateManager := NewStateManager()

	// 初始化静默期配置
	if cfg.Webhook.SilencePeriod > 0 {
		DefaultSilenceDuration = time.Duration(cfg.Webhook.SilencePeriod) * time.Second
	}

	s := &Scheduler{
		cfg:           cfg,
		stateManager:  stateManager,
		webhookClient: webhook.NewClient(&cfg.Webhook),
		stopChan:      make(chan bool),
		isRunning:     false,
		pingChecker:   probe.NewPingChecker(),
		tcpChecker:    probe.NewTCPChecker(),
		httpChecker:   probe.NewHTTPChecker(5 * time.Second),
	}

	return s
}

// Start 启动监控
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("监控服务已经在运行中")
	}

	logger.Info("启动探针监控服务")

	// 初始化所有检测目标的内存状态
	s.initAllTargets()

	// 获取主循环频率（取最小的频率）
	frequency := s.getMinFrequency()
	s.ticker = time.NewTicker(time.Duration(frequency) * time.Second)
	s.isRunning = true

	// 启动监控循环
	go s.monitorLoop()

	// 打印启动信息
	s.printStartupInfo()

	return nil
}

// getMinFrequency 获取最小检测频率
func (s *Scheduler) getMinFrequency() int {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	minFreq := 30 // 默认30秒

	if s.cfg.Ping.Enabled && s.cfg.Ping.Frequency > 0 && s.cfg.Ping.Frequency < minFreq {
		minFreq = s.cfg.Ping.Frequency
	}
	if s.cfg.Tcp.Enabled && s.cfg.Tcp.Frequency > 0 && s.cfg.Tcp.Frequency < minFreq {
		minFreq = s.cfg.Tcp.Frequency
	}
	if s.cfg.Http.Enabled && s.cfg.Http.Frequency > 0 && s.cfg.Http.Frequency < minFreq {
		minFreq = s.cfg.Http.Frequency
	}

	return minFreq
}

// printStartupInfo 打印启动信息
func (s *Scheduler) printStartupInfo() {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	pingStatus := "禁用"
	if s.cfg.Ping.Enabled {
		pingStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Ping.Domains))
	}

	tcpStatus := "禁用"
	if s.cfg.Tcp.Enabled {
		tcpStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Tcp.Domains))
	}

	httpStatus := "禁用"
	if s.cfg.Http.Enabled {
		httpStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Http.Domains))
	}

	webhookStatus := "未配置"
	if s.cfg.Webhook.URL != "" {
		webhookStatus = s.cfg.Webhook.URL
	}

	logger.Infof("监控服务启动成功 (Ping: %s, TCP: %s, HTTP: %s)", pingStatus, tcpStatus, httpStatus)
	logger.Infof("Webhook: %s", webhookStatus)
}

// initAllTargets 初始化所有检测目标
func (s *Scheduler) initAllTargets() {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	// 初始化Ping目标
	for _, target := range s.cfg.Ping.Domains {
		s.stateManager.InitDomain(target)
		logger.Infof("[PING] ➕ 新增监控目标: %s", target)
	}

	// 初始化TCP目标
	for _, target := range s.cfg.Tcp.Domains {
		s.stateManager.InitDomain(target)
		logger.Infof("[TCP ] ➕ 新增监控目标: %s", target)
	}

	// 初始化HTTP目标
	for _, target := range s.cfg.Http.Domains {
		s.stateManager.InitDomain(target)
		logger.Infof("[HTTP] ➕ 新增监控目标: %s", target)
	}
}

// Stop 停止监控
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("监控服务未在运行")
	}

	s.ticker.Stop()
	s.stopChan <- true

	s.isRunning = false
	logger.Info("监控服务已停止")
	return nil
}

// IsRunning 检查是否在运行
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

// monitorLoop 监控主循环
func (s *Scheduler) monitorLoop() {
	// 立即执行一次检测
	s.checkAllTargets()

	for {
		select {
		case <-s.ticker.C:
			s.checkAllTargets()
		case <-s.stopChan:
			return
		}
	}
}

// checkAllTargets 检查所有目标
func (s *Scheduler) checkAllTargets() {
	var wg sync.WaitGroup

	// 获取配置（加读锁）
	s.configMu.RLock()
	pingEnabled := s.cfg.Ping.Enabled
	pingTargets := make([]string, len(s.cfg.Ping.Domains))
	copy(pingTargets, s.cfg.Ping.Domains)
	pingTimeout := time.Duration(s.cfg.Ping.Timeout) * time.Second
	pingFailCount := s.cfg.Ping.FailCount

	tcpEnabled := s.cfg.Tcp.Enabled
	tcpTargets := make([]string, len(s.cfg.Tcp.Domains))
	copy(tcpTargets, s.cfg.Tcp.Domains)
	tcpTimeout := time.Duration(s.cfg.Tcp.Timeout) * time.Second
	tcpFailCount := s.cfg.Tcp.FailCount

	httpEnabled := s.cfg.Http.Enabled
	httpTargets := make([]string, len(s.cfg.Http.Domains))
	copy(httpTargets, s.cfg.Http.Domains)
	httpTimeout := time.Duration(s.cfg.Http.Timeout) * time.Second
	httpFailCount := s.cfg.Http.FailCount
	s.configMu.RUnlock()

	// 并发检测Ping目标（如果启用）
	if pingEnabled {
		for _, target := range pingTargets {
			wg.Add(1)
			go func(t string) {
				defer wg.Done()
				s.checkTarget(t, probe.TypePing, pingTimeout, pingFailCount)
			}(target)
		}
	}

	// 并发检测TCP目标（如果启用）
	if tcpEnabled {
		for _, target := range tcpTargets {
			wg.Add(1)
			go func(t string) {
				defer wg.Done()
				s.checkTarget(t, probe.TypeTCP, tcpTimeout, tcpFailCount)
			}(target)
		}
	}

	// 并发检测HTTP目标（如果启用）
	if httpEnabled {
		for _, target := range httpTargets {
			wg.Add(1)
			go func(t string) {
				defer wg.Done()
				s.checkTarget(t, probe.TypeHTTP, httpTimeout, httpFailCount)
			}(target)
		}
	}

	wg.Wait()
}

// checkTarget 检查单个目标
func (s *Scheduler) checkTarget(target string, probeType probe.ProbeType, timeout time.Duration, failThreshold int) {
	// 格式化类型标签，保持对齐
	typeTag := fmt.Sprintf("%-4s", probeType)

	// 检查是否在静默期内
	if s.stateManager.IsSilenced(target) {
		remaining := s.stateManager.GetSilenceRemaining(target)
		logger.Debugf("[%s] ⏸ %s 处于静默期，剩余 %v", typeTag, target, remaining.Round(time.Second))
		return
	}

	// 执行检测
	var result *probe.Result
	switch probeType {
	case probe.TypePing:
		result = s.pingChecker.Check(target, timeout)
	case probe.TypeTCP:
		result = s.tcpChecker.Check(target, timeout)
	case probe.TypeHTTP:
		result = s.httpChecker.Check(target, timeout)
	default:
		logger.Errorf("[%s] 未知的检测类型", typeTag)
		return
	}

	// 获取当前状态
	state := s.stateManager.GetState(target)
	wasDown := state.IsDown

	if result.Success {
		// 检测成功
		logger.Infof("[%s] ✓ %s (延迟: %v)", typeTag, target, result.Latency)

		// 如果之前是故障状态，现在恢复了，发送恢复通知
		if wasDown {
			logger.Infof("[%s] ✓ %s 已恢复正常", typeTag, target)
			s.webhookClient.SendRecoveryAlert(string(probeType), target)
		}

		// 重置失败计数和静默期
		s.stateManager.ResetFailCount(target)
		s.stateManager.ClearSilence(target)
	} else {
		// 检测失败
		currentFailCount := s.stateManager.IncrementFailCount(target)
		errMsg := ""
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		logger.Warnf("[%s] ✗ %s 失败 (%d/%d) - %s", typeTag, target, currentFailCount, failThreshold, errMsg)

		// 达到阈值，触发告警
		if currentFailCount >= failThreshold {
			logger.Errorf("[%s] ⚠ %s 触发告警 (连续失败 %d 次)，进入静默期 %v", typeTag, target, currentFailCount, DefaultSilenceDuration)
			s.webhookClient.SendDownAlert(string(probeType), target, currentFailCount, failThreshold, errMsg)
			s.stateManager.MarkDown(target)
		}
	}
}

// GetConfig 获取当前配置
func (s *Scheduler) GetConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg
}
