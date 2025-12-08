package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// RemoteConfig 远程配置结构（从 remote-config.json 加载）
type RemoteConfig struct {
	Ping RemotePingConfig  `json:"ping"`
	Tcp  RemoteProbeConfig `json:"tcp"`
	Http RemoteProbeConfig `json:"http"`
}

// RemotePingConfig 远程Ping配置（包含远程更新频率）
type RemotePingConfig struct {
	Frequency        int              `json:"frequency"`
	FailCount        int              `json:"failcount"`
	Timeout          int              `json:"timeout"`
	Retry            int              `json:"retry"`
	RemoteUpdateFreq int              `json:"remote_update_freq"`
	Domains          []string         `json:"domains"`
	Failover         []FailoverTarget `json:"failover"`
}

// RemoteProbeConfig 通用远程检测配置（TCP/HTTP共用）
type RemoteProbeConfig struct {
	Frequency int              `json:"frequency"`
	FailCount int              `json:"failcount"`
	Timeout   int              `json:"timeout"`
	Retry     int              `json:"retry"`
	Domains   []string         `json:"domains"`
	Failover  []FailoverTarget `json:"failover"`
}

// FetchRemoteConfig 从远程URL拉取配置（支持HTTP和S3）
func FetchRemoteConfig(url string) (*RemoteConfig, error) {
	var body []byte
	var err error
	if strings.HasPrefix(url, "s3://") {
		body, err = fetchFromS3(url)
	} else {
		body, err = fetchFromHTTP(url)
	}
	if err != nil {
		return nil, err
	}

	// 调试：打印拉取到的内容
	fmt.Printf("DEBUG: 远程配置内容:\n%s\n", string(body))

	var remoteCfg RemoteConfig
	if err := json.Unmarshal(body, &remoteCfg); err != nil {
		return nil, fmt.Errorf("解析远程配置JSON失败: %w", err)
	}
	if err := remoteCfg.Validate(); err != nil {
		return nil, err
	}
	return &remoteCfg, nil
}

// Validate 验证远程配置
func (r *RemoteConfig) Validate() error {
	// Ping 配置验证（必须）
	if r.Ping.Frequency < 10 {
		return fmt.Errorf("远程配置 ping.frequency 不能小于10秒")
	}
	if r.Ping.FailCount < 1 {
		return fmt.Errorf("远程配置 ping.failcount 必须大于0")
	}
	if len(r.Ping.Domains) == 0 {
		return fmt.Errorf("远程配置 ping.domains 不能为空")
	}
	if len(r.Ping.Failover) == 0 {
		return fmt.Errorf("远程配置 ping.failover 不能为空")
	}
	// Tcp 配置验证（可选，有配置时才验证）
	if len(r.Tcp.Domains) > 0 {
		if r.Tcp.Frequency < 10 {
			return fmt.Errorf("远程配置 tcp.frequency 不能小于10秒")
		}
		if r.Tcp.FailCount < 1 {
			return fmt.Errorf("远程配置 tcp.failcount 必须大于0")
		}
	}
	// Http 配置验证（可选，有配置时才验证）
	if len(r.Http.Domains) > 0 {
		if r.Http.Frequency < 10 {
			return fmt.Errorf("远程配置 http.frequency 不能小于10秒")
		}
		if r.Http.FailCount < 1 {
			return fmt.Errorf("远程配置 http.failcount 必须大于0")
		}
	}
	return nil
}

// SetDefaults 设置远程配置默认值
func (r *RemoteConfig) SetDefaults() {
	// Ping 默认值
	if r.Ping.Timeout == 0 {
		r.Ping.Timeout = 5
	}
	if r.Ping.Retry == 0 {
		r.Ping.Retry = 3
	}
	if r.Ping.RemoteUpdateFreq == 0 {
		r.Ping.RemoteUpdateFreq = 60
	}
	// Tcp/Http 默认值
	setProbeDefaults(&r.Tcp)
	setProbeDefaults(&r.Http)
}

// setProbeDefaults 设置通用检测配置默认值
func setProbeDefaults(cfg *RemoteProbeConfig) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5
	}
	if cfg.Retry == 0 {
		cfg.Retry = 3
	}
}

// ApplyRemoteConfig 将远程配置应用到主配置
func (c *Config) ApplyRemoteConfig(remoteCfg *RemoteConfig) {
	// 应用 Ping 配置
	c.Ping.Frequency = remoteCfg.Ping.Frequency
	c.Ping.FailCount = remoteCfg.Ping.FailCount
	c.Ping.Timeout = remoteCfg.Ping.Timeout
	c.Ping.Retry = remoteCfg.Ping.Retry
	c.Ping.RemoteUpdateFreq = remoteCfg.Ping.RemoteUpdateFreq
	c.Ping.Domains = remoteCfg.Ping.Domains
	c.Ping.Failover = remoteCfg.Ping.Failover
	// 应用 Tcp 配置
	applyProbeConfig(&c.Tcp, &remoteCfg.Tcp)
	// 应用 Http 配置
	applyProbeConfig(&c.Http, &remoteCfg.Http)
}

// applyProbeConfig 应用通用检测配置
func applyProbeConfig(dst *ProbeConfig, src *RemoteProbeConfig) {
	dst.Frequency = src.Frequency
	dst.FailCount = src.FailCount
	dst.Timeout = src.Timeout
	dst.Retry = src.Retry
	dst.Domains = src.Domains
	dst.Failover = src.Failover
}

// fetchFromS3 从S3读取配置
func fetchFromS3(s3URL string) ([]byte, error) {
	s3URL = strings.TrimPrefix(s3URL, "s3://")
	parts := strings.SplitN(s3URL, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的S3 URL格式")
	}
	bucket := parts[0]
	key := parts[1]
	ctx := context.Background()

	// 从环境变量获取区域，默认 us-east-1
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	fmt.Printf("DEBUG: AWS_REGION=%s, bucket=%s, key=%s\n", region, bucket, key)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("加载AWS配置失败: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("从S3获取配置失败: %w", err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("读取S3配置内容失败: %w", err)
	}
	return body, nil
} // fetchFromHTTP 从HTTP/HTTPS读取配置
func fetchFromHTTP(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求远程配置失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("远程配置HTTP状态错误: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取远程配置失败: %w", err)
	}
	return body, nil
}
