package ping

import (
	"fmt"
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

	pinger, err := goping.NewPinger(target)
	if err != nil {
		result.Error = fmt.Errorf("创建pinger失败: %w", err)
		return result
	}

	// Windows系统优先尝试特权模式，失败则降级为UDP模式
	// 首先尝试特权模式（ICMP，需要管理员权限）
	pinger.SetPrivileged(true)
	
	// 设置ping参数
	pinger.Count = 3                    // 发送3个包
	pinger.Timeout = timeout            // 超时时间
	pinger.Interval = time.Millisecond * 500 // 包间隔

	// 执行ping
	err = pinger.Run()
	if err != nil {
		// 如果特权模式失败，尝试非特权模式（UDP）
		pinger.SetPrivileged(false)
		err = pinger.Run()
		if err != nil {
			result.Error = fmt.Errorf("执行ping失败: %w", err)
			return result
		}
	}

	// 获取统计信息
	stats := pinger.Statistics()

	// 判断是否成功（至少有一个包收到响应）
	if stats.PacketsRecv > 0 {
		result.Success = true
		result.Latency = stats.AvgRtt
	} else {
		result.Success = false
		result.Error = fmt.Errorf("所有数据包丢失 (发送: %d, 接收: %d)", stats.PacketsSent, stats.PacketsRecv)
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
