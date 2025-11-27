package cloudflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// Client Cloudflare客户端封装
type Client struct {
	api *cloudflare.API
	ctx context.Context
}

// NewClient 创建Cloudflare客户端（仅支持API Token）
func NewClient(apiToken string) (*Client, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("创建Cloudflare API客户端失败: %w", err)
	}

	return &Client{
		api: api,
		ctx: context.Background(),
	}, nil
}

// VerifyCredentials 验证API凭证（通过验证Token来验证）
func (c *Client) VerifyCredentials() error {
	// 使用 VerifyAPIToken 来验证token
	result, err := c.api.VerifyAPIToken(c.ctx)
	if err != nil {
		return fmt.Errorf("验证Cloudflare凭证失败: %w", err)
	}
	
	if result.Status != "active" {
		return fmt.Errorf("API Token状态异常: %s", result.Status)
	}
	
	return nil
}

// GetZoneID 获取域名的Zone ID
func (c *Client) GetZoneID(domain string) (string, error) {
	// 从完整域名中提取根域名
	// 例如: cn1.speedtest-node.com -> speedtest-node.com
	zoneName := extractRootDomain(domain)

	zoneID, err := c.api.ZoneIDByName(zoneName)
	if err != nil {
		return "", fmt.Errorf("获取Zone ID失败 (域名: %s): %w", zoneName, err)
	}

	return zoneID, nil
}

// extractRootDomain 从完整域名中提取根域名
// 简单实现：假设根域名是最后两个部分
func extractRootDomain(domain string) string {
	// 移除末尾的点
	if len(domain) > 0 && domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}

	// 简单处理：取最后两个部分
	// 例如: cn1.speedtest-node.com -> speedtest-node.com
	parts := []rune(domain)
	dotCount := 0
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '.' {
			dotCount++
			if dotCount == 2 {
				return string(parts[i+1:])
			}
		}
	}
	return domain
}
