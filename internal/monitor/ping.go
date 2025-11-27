package monitor

import (
	"dnsfailover/internal/ping"
	"time"
)

// Ping 执行ping检测
func Ping(target string, timeout time.Duration) *ping.Result {
	return ping.Check(target, timeout)
}

// PingWithRetry 带重试的ping检测
func PingWithRetry(target string, timeout time.Duration, retryCount int) *ping.Result {
	return ping.CheckWithRetry(target, timeout, retryCount)
}

// CheckDomain 检测域名可用性（简化版本，用于快速检测）
func CheckDomain(domain string) bool {
	result := ping.Check(domain, 5*time.Second)
	return result.Success
}
