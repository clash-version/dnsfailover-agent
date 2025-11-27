package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RemoteConfig 远程配置结构（仅包含需要动态更新的部分）
type RemoteConfig struct {
	Domains  []string         `json:"domains"`  // 监控域名列表
	Failover []FailoverTarget `json:"failover"` // 备用地址列表
}

// FetchRemoteConfig 从远程URL拉取配置
func FetchRemoteConfig(url string) (*RemoteConfig, error) {
	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发起GET请求
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求远程配置失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("远程配置HTTP状态错误: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取远程配置失败: %w", err)
	}

	// 解析JSON
	var remoteCfg RemoteConfig
	if err := json.Unmarshal(body, &remoteCfg); err != nil {
		return nil, fmt.Errorf("解析远程配置JSON失败: %w", err)
	}

	// 验证远程配置
	if len(remoteCfg.Domains) == 0 {
		return nil, fmt.Errorf("远程配置中domains不能为空")
	}
	if len(remoteCfg.Failover) == 0 {
		return nil, fmt.Errorf("远程配置中failover不能为空")
	}

	return &remoteCfg, nil
}

// ApplyRemoteConfig 应用远程配置到当前配置
func (c *Config) ApplyRemoteConfig(remoteCfg *RemoteConfig) {
	c.Ping.Domains = remoteCfg.Domains
	c.Ping.Failover = remoteCfg.Failover
}

// IsRemoteConfigEnabled 检查是否启用了远程配置
func (c *Config) IsRemoteConfigEnabled() bool {
	return c.Ping.RemoteConfigURL != "" && c.Ping.RemoteUpdateFreq > 0
}
