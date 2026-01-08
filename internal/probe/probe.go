package probe

import "time"

// ProbeType 检测类型
type ProbeType string

const (
	TypePing ProbeType = "PING"
	TypeTCP  ProbeType = "TCP"
	TypeHTTP ProbeType = "HTTP"
)

// Result 通用检测结果
type Result struct {
	Type    ProbeType     // 检测类型
	Target  string        // 检测目标
	Success bool          // 是否成功
	Latency time.Duration // 延迟
	Error   error         // 错误信息
}

// Checker 检测器接口
type Checker interface {
	// Check 执行检测
	Check(target string, timeout time.Duration) *Result
	// Type 返回检测类型
	Type() ProbeType
}
