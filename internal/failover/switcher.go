package failover

import (
	"dnsfailover/internal/cloudflare"
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"fmt"
)

// Switcher 域名切换器
type Switcher struct {
	cfClient *cloudflare.Client
	selector *Selector
	cfg      *config.Config
}

// NewSwitcher 创建域名切换器
func NewSwitcher(cfClient *cloudflare.Client, cfg *config.Config) *Switcher {
	return &Switcher{
		cfClient: cfClient,
		selector: NewSelector(cfg),
		cfg:      cfg,
	}
}

// SwitchDomain 切换域名到新地址
func (s *Switcher) SwitchDomain(domain, targetAddress string) error {
	logger.Infof("开始切换域名: %s -> %s", domain, targetAddress)

	// 获取当前DNS记录
	currentAddress, err := s.cfClient.GetCurrentTarget(domain)
	if err != nil {
		errMsg := fmt.Sprintf("获取当前DNS记录失败: %v", err)
		logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	// 检查是否需要切换
	if currentAddress == targetAddress {
		msg := "目标地址与当前地址相同，无需切换"
		logger.Info(msg)
		return fmt.Errorf(msg)
	}

	// 执行DNS更新（带重试）
	var lastErr error
	for i := 0; i < s.cfg.Ping.Retry; i++ {
		if i > 0 {
			logger.Warnf("重试切换 (%d/%d)", i+1, s.cfg.Ping.Retry)
		}

		err = s.cfClient.UpdateDNSRecord(domain, targetAddress)
		if err == nil {
			// 更新成功
			logger.Infof("✓ DNS记录更新成功: %s -> %s", domain, targetAddress)
			logger.Infof("切换记录: %s: %s -> %s (成功)", domain, currentAddress, targetAddress)
			return nil
		}

		lastErr = err
		logger.Errorf("DNS记录更新失败: %v", err)
	}

	// 所有重试都失败
	errMsg := fmt.Sprintf("DNS更新失败（已重试%d次）: %v", s.cfg.Ping.Retry, lastErr)
	logger.Error(errMsg)
	logger.Errorf("切换记录: %s: %s -> %s (失败: %v)", domain, currentAddress, targetAddress, lastErr)
	return fmt.Errorf(errMsg)
}

// AutoSwitch 自动选择failover并切换
func (s *Switcher) AutoSwitch(domain string) error {
	logger.Infof("开始自动切换域名: %s", domain)

	// 获取当前地址
	currentAddress, err := s.cfClient.GetCurrentTarget(domain)
	if err != nil {
		logger.Warnf("获取当前地址失败，继续选择failover: %v", err)
		currentAddress = ""
	}

	// 选择最佳failover地址（排除当前地址）
	var targetAddress string
	if currentAddress != "" {
		targetAddress, err = s.selector.SelectFailoverExcluding(currentAddress)
	} else {
		targetAddress, err = s.selector.SelectBestFailover()
	}

	if err != nil {
		errMsg := fmt.Sprintf("选择failover地址失败: %v", err)
		logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	logger.Infof("选择的failover地址: %s", targetAddress)

	// 执行切换
	return s.SwitchDomain(domain, targetAddress)
}

// GetCurrentAddress 获取域名当前指向的地址
func (s *Switcher) GetCurrentAddress(domain string) (string, error) {
	return s.cfClient.GetCurrentTarget(domain)
}
