package probe

import (
	"fmt"
	"net"
	"time"

	goping "github.com/go-ping/ping"
)

// PingChecker ICMP Ping检测器
type PingChecker struct{}

// NewPingChecker 创建Ping检测器
func NewPingChecker() *PingChecker {
	return &PingChecker{}
}

// Type 返回检测类型
func (c *PingChecker) Type() ProbeType {
	return TypePing
}

// Check 执行ICMP Ping检测
func (c *PingChecker) Check(target string, timeout time.Duration) *Result {
	result := &Result{
		Type:   TypePing,
		Target: target,
	}

	// 如果是域名，先使用系统 DNS 解析
	ipAddr := target
	if net.ParseIP(target) == nil {
		// 是域名，需要解析
		ips, err := net.LookupHost(target)
		if err != nil {
			result.Error = fmt.Errorf("DNS解析失败 (%s): %w", target, err)
			return result
		}
		if len(ips) == 0 {
			result.Error = fmt.Errorf("DNS解析未返回IP地址: %s", target)
			return result
		}
		ipAddr = ips[0] // 使用第一个IP
	}

	pinger, err := goping.NewPinger(ipAddr)
	if err != nil {
		result.Error = fmt.Errorf("创建pinger失败: %w", err)
		return result
	}

	// Linux系统使用特权模式（ICMP）
	pinger.SetPrivileged(true)

	// 设置ping参数
	pinger.Count = 4                         // 发送4个包
	pinger.Timeout = timeout                 // 超时时间
	pinger.Interval = time.Millisecond * 300 // 包间隔300ms

	// 执行ping
	err = pinger.Run()
	if err != nil {
		result.Error = fmt.Errorf("执行ping失败: %w", err)
		return result
	}

	// 获取统计信息
	stats := pinger.Statistics()

	// 判断是否成功（至少有一个包收到响应）
	if stats.PacketsRecv > 0 {
		result.Success = true
		result.Latency = stats.AvgRtt
	} else {
		result.Success = false
		result.Error = fmt.Errorf("ICMP应答超时 (发送: %d, 接收: %d, 丢包率: %.0f%%)",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
	}

	return result
}

// CheckWithRetry 带重试的Ping检测
func (c *PingChecker) CheckWithRetry(target string, timeout time.Duration, retryCount int) *Result {
	var result *Result

	for i := 0; i < retryCount; i++ {
		result = c.Check(target, timeout)
		if result.Success {
			return result
		}

		// 如果不是最后一次重试，等待一小段时间
		if i < retryCount-1 {
			time.Sleep(time.Second)
		}
	}

	return result
}
