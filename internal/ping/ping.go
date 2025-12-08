package ping

import (
	"fmt"
	"log"
	"net"
	"time"

	goping "github.com/go-ping/ping"
)

// Result Ping检测结果
type Result struct {
	Success bool          // 是否成功
	Latency time.Duration // 平均延迟
	Error   error         // 错误信息
}

// Check 执行ping检测
func Check(target string, timeout time.Duration) *Result {
	result := &Result{}

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
		log.Printf("[DEBUG] DNS解析: %s -> %s", target, ipAddr)
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

	log.Printf("[DEBUG] 开始ICMP Ping: %s (目标IP: %s, 超时: %v)", target, ipAddr, timeout)

	// 执行ping
	err = pinger.Run()
	if err != nil {
		result.Error = fmt.Errorf("执行ping失败: %w", err)
		log.Printf("[ERROR] ICMP Ping执行失败: %s - %v", ipAddr, err)
		return result
	}

	// 获取统计信息
	stats := pinger.Statistics()
	log.Printf("[DEBUG] ICMP Ping统计: %s - 发送: %d, 接收: %d, 丢包率: %.1f%%, 平均延迟: %v",
		ipAddr, stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss, stats.AvgRtt)

	// 判断是否成功（至少有一个包收到响应）
	if stats.PacketsRecv > 0 {
		result.Success = true
		result.Latency = stats.AvgRtt
		log.Printf("[DEBUG] ✓ ICMP Ping成功: %s (延迟: %v)", ipAddr, result.Latency)
	} else {
		result.Success = false
		result.Error = fmt.Errorf("ICMP应答超时 - 目标主机可能禁用了ICMP响应 (发送: %d, 接收: %d, 丢包率: %.0f%%)",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		log.Printf("[WARN] ✗ ICMP Ping失败: %s - %v", ipAddr, result.Error)
	}

	return result
}

// CheckWithRetry 带重试的ping检测
func CheckWithRetry(target string, timeout time.Duration, retryCount int) *Result {
	var result *Result

	for i := 0; i < retryCount; i++ {
		result = Check(target, timeout)
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
