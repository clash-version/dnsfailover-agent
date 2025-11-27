package cloudflare

import (
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// DNSRecordType DNS记录类型
type DNSRecordType string

const (
	TypeA     DNSRecordType = "A"
	TypeCNAME DNSRecordType = "CNAME"
)

// DNSRecord DNS记录信息
type DNSRecord struct {
	ID      string
	Type    string
	Name    string
	Content string
	Proxied bool
	TTL     int
}

// GetDNSRecord 获取指定域名的DNS记录
func (c *Client) GetDNSRecord(domain string) (*DNSRecord, error) {
	zoneID, err := c.GetZoneID(domain)
	if err != nil {
		return nil, err
	}

	// 查询DNS记录
	records, _, err := c.api.ListDNSRecords(c.ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Name: domain,
	})
	if err != nil {
		return nil, fmt.Errorf("查询DNS记录失败: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("未找到域名的DNS记录: %s", domain)
	}

	// 返回第一条记录（通常只有一条A或CNAME记录）
	record := records[0]
	return &DNSRecord{
		ID:      record.ID,
		Type:    record.Type,
		Name:    record.Name,
		Content: record.Content,
		Proxied: *record.Proxied,
		TTL:     record.TTL,
	}, nil
}

// UpdateDNSRecord 更新DNS记录
func (c *Client) UpdateDNSRecord(domain, newTarget string) error {
	// 获取当前DNS记录
	currentRecord, err := c.GetDNSRecord(domain)
	if err != nil {
		return err
	}

	// 检查是否需要更新
	if currentRecord.Content == newTarget {
		return fmt.Errorf("目标地址与当前地址相同，无需更新: %s", newTarget)
	}

	zoneID, err := c.GetZoneID(domain)
	if err != nil {
		return err
	}

	// 判断新目标的类型（IP地址或域名）
	recordType := determineRecordType(newTarget)

	// 更新DNS记录
	params := cloudflare.UpdateDNSRecordParams{
		ID:      currentRecord.ID,
		Type:    string(recordType),
		Name:    domain,
		Content: newTarget,
		Proxied: cloudflare.BoolPtr(currentRecord.Proxied),
		TTL:     currentRecord.TTL,
	}

	_, err = c.api.UpdateDNSRecord(c.ctx, cloudflare.ZoneIdentifier(zoneID), params)
	if err != nil {
		return fmt.Errorf("更新DNS记录失败: %w", err)
	}

	return nil
}

// CreateDNSRecord 创建DNS记录
func (c *Client) CreateDNSRecord(domain, target string, proxied bool) error {
	zoneID, err := c.GetZoneID(domain)
	if err != nil {
		return err
	}

	recordType := determineRecordType(target)

	params := cloudflare.CreateDNSRecordParams{
		Type:    string(recordType),
		Name:    domain,
		Content: target,
		Proxied: cloudflare.BoolPtr(proxied),
		TTL:     300, // 5分钟
	}

	_, err = c.api.CreateDNSRecord(c.ctx, cloudflare.ZoneIdentifier(zoneID), params)
	if err != nil {
		return fmt.Errorf("创建DNS记录失败: %w", err)
	}

	return nil
}

// determineRecordType 判断记录类型（A记录或CNAME记录）
func determineRecordType(target string) DNSRecordType {
	// 简单判断：如果包含字母，则为CNAME，否则为A记录
	// 更准确的判断可以使用正则表达式判断是否为IP地址
	if strings.Contains(target, ".") {
		// 检查是否为IP地址格式
		parts := strings.Split(target, ".")
		if len(parts) == 4 {
			// 简单判断是否所有部分都是数字
			isIP := true
			for _, part := range parts {
				if len(part) == 0 || len(part) > 3 {
					isIP = false
					break
				}
				for _, ch := range part {
					if ch < '0' || ch > '9' {
						isIP = false
						break
					}
				}
				if !isIP {
					break
				}
			}
			if isIP {
				return TypeA
			}
		}
	}
	// 默认为CNAME
	return TypeCNAME
}

// GetCurrentTarget 获取当前DNS记录指向的地址
func (c *Client) GetCurrentTarget(domain string) (string, error) {
	record, err := c.GetDNSRecord(domain)
	if err != nil {
		return "", err
	}
	return record.Content, nil
}
