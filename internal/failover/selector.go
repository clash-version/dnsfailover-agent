package failover

import (
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/ping"
	"fmt"
	"sort"
	"time"
)

// Selector Failover选择器
type Selector struct {
	cfg *config.Config
}

// NewSelector 创建Failover选择器
func NewSelector(cfg *config.Config) *Selector {
	return &Selector{
		cfg: cfg,
	}
}

// SelectBestFailover 选择最佳的failover地址
// 按权重从高到低依次检测，返回第一个可用的地址
func (s *Selector) SelectBestFailover() (string, error) {
	targets := s.cfg.Ping.Failover
	if len(targets) == 0 {
		return "", fmt.Errorf("没有配置failover地址")
	}

	// 按权重降序排序
	sortedTargets := make([]config.FailoverTarget, len(targets))
	copy(sortedTargets, targets)
	sort.Slice(sortedTargets, func(i, j int) bool {
		return sortedTargets[i].Weight > sortedTargets[j].Weight
	})

	logger.Infof("开始选择failover地址，共 %d 个候选", len(sortedTargets))

	// 依次检测每个failover地址
	for _, target := range sortedTargets {
		logger.Infof("检测failover地址: %s (权重: %d)", target.Address, target.Weight)

		// Ping检测
		result := ping.Check(target.Address, 5*time.Second)

		if result.Success {
			logger.Infof("✓ Failover地址可用: %s (延迟: %v)", target.Address, result.Latency)
			return target.Address, nil
		} else {
			logger.Warnf("✗ Failover地址不可用: %s (错误: %v)", target.Address, result.Error)
		}
	}

	return "", fmt.Errorf("所有failover地址都不可用")
}

// SelectFailoverExcluding 选择failover地址（排除指定地址）
func (s *Selector) SelectFailoverExcluding(excludeAddress string) (string, error) {
	targets := s.cfg.Ping.Failover
	if len(targets) == 0 {
		return "", fmt.Errorf("没有配置failover地址")
	}

	// 按权重降序排序
	sortedTargets := make([]config.FailoverTarget, len(targets))
	copy(sortedTargets, targets)
	sort.Slice(sortedTargets, func(i, j int) bool {
		return sortedTargets[i].Weight > sortedTargets[j].Weight
	})

	logger.Infof("选择failover地址（排除: %s）", excludeAddress)

	// 依次检测每个failover地址（跳过excludeAddress）
	for _, target := range sortedTargets {
		if target.Address == excludeAddress {
			logger.Infof("跳过当前地址: %s", target.Address)
			continue
		}

		logger.Infof("检测failover地址: %s (权重: %d)", target.Address, target.Weight)

		// Ping检测
		result := ping.Check(target.Address, 5*time.Second)

		if result.Success {
			logger.Infof("✓ Failover地址可用: %s (延迟: %v)", target.Address, result.Latency)
			return target.Address, nil
		} else {
			logger.Warnf("✗ Failover地址不可用: %s (错误: %v)", target.Address, result.Error)
		}
	}

	return "", fmt.Errorf("所有failover地址都不可用")
}
