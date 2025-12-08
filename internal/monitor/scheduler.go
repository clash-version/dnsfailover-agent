package monitor

import (
	"dnsfailover/internal/cloudflare"
	"dnsfailover/internal/config"
	"dnsfailover/internal/failover"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/ping"
	"fmt"
	"sync"
	"time"
)

// Scheduler 监控调度器
type Scheduler struct {
	cfg            *config.Config
	cfClient       *cloudflare.Client
	switcher       *failover.Switcher
	stateManager   *StateManager
	ticker         *time.Ticker
	configTicker   *time.Ticker // 远程配置更新定时器
	stopChan       chan bool
	configStopChan chan bool // 配置更新停止信号
	isRunning      bool
	mu             sync.Mutex
	configMu       sync.RWMutex // 配置读写锁
}

// NewScheduler 创建监控调度器
func NewScheduler(cfg *config.Config, cfClient *cloudflare.Client) *Scheduler {
	switcher := failover.NewSwitcher(cfClient, cfg)

	return &Scheduler{
		cfg:            cfg,
		cfClient:       cfClient,
		switcher:       switcher,
		stateManager:   NewStateManager(),
		stopChan:       make(chan bool),
		configStopChan: make(chan bool),
		isRunning:      false,
	}
}

// Start 启动监控
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("监控服务已经在运行中")
	}

	logger.Info("启动DNS Failover监控服务")

	// 立即拉取一次远程配置
	if err := s.updateRemoteConfig(); err != nil {
		return fmt.Errorf("拉取远程配置失败: %w", err)
	}

	// 初始化所有域名的内存状态
	s.configMu.RLock()
	domains := s.cfg.Ping.Domains
	s.configMu.RUnlock()

	for _, domain := range domains {
		s.stateManager.InitDomain(domain)
	}

	// 创建定时器
	s.ticker = time.NewTicker(time.Duration(s.cfg.Ping.Frequency) * time.Second)
	s.isRunning = true

	// 启动监控循环
	go s.monitorLoop()

	// 启动远程配置更新循环
	s.configTicker = time.NewTicker(time.Duration(s.cfg.Ping.RemoteUpdateFreq) * time.Second)
	go s.configUpdateLoop()

	logger.Infof("监控服务启动成功 (检测频率: %ds, 失败阈值: %d次, 域名数: %d)",
		s.cfg.Ping.Frequency, s.cfg.Ping.FailCount, len(domains))
	return nil
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

	if s.configTicker != nil {
		s.configTicker.Stop()
		s.configStopChan <- true
	}

	s.isRunning = false
	logger.Info("监控服务已停止")
	return nil
}

// IsRunning 检查是否正在运行
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

// monitorLoop 监控循环
func (s *Scheduler) monitorLoop() {
	// 启动后立即执行一次检测
	s.checkAllDomains()

	// 定时执行检测
	for {
		select {
		case <-s.ticker.C:
			s.checkAllDomains()
		case <-s.stopChan:
			return
		}
	}
}

// checkAllDomains 检查所有域名
func (s *Scheduler) checkAllDomains() {
	// 获取当前域名列表（加读锁）
	s.configMu.RLock()
	domains := make([]string, len(s.cfg.Ping.Domains))
	copy(domains, s.cfg.Ping.Domains)
	s.configMu.RUnlock()

	if len(domains) == 0 {
		return
	}

	// 并发检测所有域名
	var wg sync.WaitGroup
	for _, domain := range domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			s.checkDomain(d)
		}(domain)
	}

	wg.Wait()
}

// checkDomain 检查单个域名
func (s *Scheduler) checkDomain(domain string) {
	// 检查是否在冷却期（仍然执行检测，但不触发故障转移）
	inCooldown := s.stateManager.IsInCooldown(domain)

	// Ping检测
	timeout := time.Duration(s.cfg.Ping.Timeout) * time.Second
	result := ping.Check(domain, timeout)

	if result.Success {
		logger.Infof("[PING] ✓ %s (延迟: %v)", domain, result.Latency)
		s.stateManager.ResetFailCount(domain)
	} else {
		newFailCount := s.stateManager.IncrementFailCount(domain)
		logger.Warnf("[PING] ✗ %s 失败 (%d/%d) - %v", domain, newFailCount, s.cfg.Ping.FailCount, result.Error)

		// 达到失败阈值且不在冷却期时触发故障转移
		if newFailCount >= s.cfg.Ping.FailCount && !inCooldown {
			logger.Errorf("[PING] ⚠ %s 触发故障转移", domain)
			s.handleDomainFailure(domain)
		}
	}
}

// handleDomainFailure 处理域名故障
func (s *Scheduler) handleDomainFailure(domain string) {
	err := s.switcher.AutoSwitch(domain)
	if err != nil {
		logger.Errorf("故障转移失败 [%s]: %v", domain, err)
		s.stateManager.ResetFailCount(domain)
	} else {
		logger.Infof("✓ 故障转移成功: %s", domain)
		s.stateManager.MarkSwitched(domain)
	}
}

// RunOnce 执行一次检测（用于测试）
func (s *Scheduler) RunOnce() {
	s.checkAllDomains()
}

// configUpdateLoop 配置更新循环
func (s *Scheduler) configUpdateLoop() {
	for {
		select {
		case <-s.configTicker.C:
			if err := s.updateRemoteConfig(); err != nil {
				logger.Errorf("更新远程配置失败: %v", err)
			}
		case <-s.configStopChan:
			return
		}
	}
}

// updateRemoteConfig 更新远程配置
func (s *Scheduler) updateRemoteConfig() error {
	remoteCfg, err := config.FetchRemoteConfig(s.cfg.Ping.RemoteConfigURL)
	if err != nil {
		return fmt.Errorf("拉取远程配置失败: %w", err)
	}

	remoteCfg.SetDefaults()

	// 获取当前配置（加读锁）
	s.configMu.RLock()
	oldDomains := make([]string, len(s.cfg.Ping.Domains))
	copy(oldDomains, s.cfg.Ping.Domains)
	s.configMu.RUnlock()

	// 应用新配置（加写锁）
	s.configMu.Lock()
	s.cfg.ApplyRemoteConfig(remoteCfg)
	newDomains := s.cfg.Ping.Domains
	s.configMu.Unlock()

	// 处理域名变更
	s.handleDomainChanges(oldDomains, newDomains)

	return nil
}

// handleDomainChanges 处理域名变更（新增/删除）
func (s *Scheduler) handleDomainChanges(oldDomains, newDomains []string) {
	// 转换为map方便查找
	oldMap := make(map[string]bool)
	for _, domain := range oldDomains {
		oldMap[domain] = true
	}

	newMap := make(map[string]bool)
	for _, domain := range newDomains {
		newMap[domain] = true
	}

	// 查找新增的域名
	for _, domain := range newDomains {
		if !oldMap[domain] {
			logger.Infof("➕ 新增监控域名: %s", domain)
			s.stateManager.InitDomain(domain)
		}
	}

	// 查找删除的域名
	for _, domain := range oldDomains {
		if !newMap[domain] {
			logger.Infof("➖ 移除监控域名: %s", domain)
			s.stateManager.RemoveDomain(domain)
		}
	}
}
