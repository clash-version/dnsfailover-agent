package probe

import (
	"fmt"
	"net"
	"time"
)

// TCPChecker TCP端口检测器
type TCPChecker struct{}

// NewTCPChecker 创建TCP检测器
func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

// Type 返回检测类型
func (c *TCPChecker) Type() ProbeType {
	return TypeTCP
}

// Check 执行TCP端口检测
// target 格式: host:port (例如: example.com:443)
func (c *TCPChecker) Check(target string, timeout time.Duration) *Result {
	result := &Result{
		Type:   TypeTCP,
		Target: target,
	}

	// 验证目标格式
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		result.Error = fmt.Errorf("无效的目标格式 (应为 host:port): %w", err)
		return result
	}

	if host == "" || port == "" {
		result.Error = fmt.Errorf("无效的目标格式: host=%s, port=%s", host, port)
		return result
	}

	// 记录开始时间
	start := time.Now()

	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		result.Error = fmt.Errorf("TCP连接失败: %w", err)
		return result
	}
	defer conn.Close()

	// 计算延迟
	result.Latency = time.Since(start)
	result.Success = true

	return result
}

// CheckWithRetry 带重试的TCP检测
func (c *TCPChecker) CheckWithRetry(target string, timeout time.Duration, retryCount int) *Result {
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
