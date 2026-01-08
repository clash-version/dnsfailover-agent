package probe

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPChecker HTTP检测器
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker 创建HTTP检测器
func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	// 创建自定义Transport，跳过证书验证（用于自签名证书的内部服务）
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &HTTPChecker{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			// 不跟随重定向
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Type 返回检测类型
func (c *HTTPChecker) Type() ProbeType {
	return TypeHTTP
}

// Check 执行HTTP检测
// target 格式: URL (例如: http://example.com/health 或 https://example.com:8443/ping)
func (c *HTTPChecker) Check(target string, timeout time.Duration) *Result {
	result := &Result{
		Type:   TypeHTTP,
		Target: target,
	}

	// 验证URL格式
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		result.Error = fmt.Errorf("无效的URL格式 (应以 http:// 或 https:// 开头): %s", target)
		return result
	}

	// 更新超时时间
	c.client.Timeout = timeout

	// 记录开始时间
	start := time.Now()

	// 发送GET请求
	resp, err := c.client.Get(target)
	if err != nil {
		result.Error = fmt.Errorf("HTTP请求失败: %w", err)
		return result
	}
	defer resp.Body.Close()

	// 计算延迟
	result.Latency = time.Since(start)

	// 检查状态码 (2xx 和 3xx 都认为成功)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
	} else {
		result.Error = fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
	}

	return result
}

// CheckWithRetry 带重试的HTTP检测
func (c *HTTPChecker) CheckWithRetry(target string, timeout time.Duration, retryCount int) *Result {
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
