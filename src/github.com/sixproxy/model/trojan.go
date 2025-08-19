package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"singbox_sub/src/github.com/sixproxy/util"
	"strconv"
	"strings"
)

type TrojanConfig struct {
	Outbound
	Server     string     `json:"server"`
	ServerPort int        `json:"server_port"`
	Password   string     `json:"password"`
	TLS        *TrojanTLS `json:"tls,omitempty"`
	data       *string
}

type TrojanTLS struct {
	Enabled    bool   `json:"enabled"`
	ServerName string `json:"server_name,omitempty"`
	Insecure   bool   `json:"insecure,omitempty"`
}

func (c *TrojanConfig) GetTag() string {
	return c.Tag
}

func (c *TrojanConfig) GetType() string {
	return c.Type
}

func (c *TrojanConfig) Validate() error {
	if c.Server == "" || c.ServerPort == 0 {
		return fmt.Errorf("trojan outbound must have server and port")
	}
	if c.Password == "" {
		return fmt.Errorf("trojan outbound must have password")
	}
	return nil
}

var _ OutboundConfig = (*TrojanConfig)(nil)

func (c *TrojanConfig) SetData(data string) error {
	// trojan URL format: trojan://password@server:port?params#tag
	tmp := data[len("trojan://"):]
	c.data = &tmp
	c.Type = "trojan"

	// 解析tag
	tag := util.ParseTag(*c.data)
	c.Tag = tag

	// 解析trojan参数
	err := c.parseTrojanURL()
	if err != nil {
		return err
	}
	return nil
}

func (c *TrojanConfig) String() string {
	copy := *c
	if copy.data != nil {
		copy.data = nil
	}
	b, err := json.Marshal(copy)
	if err != nil {
		return ""
	}
	return string(b)
}

func (c *TrojanConfig) parseTrojanURL() error {
	// 分离查询参数和片段
	urlStr := *c.data
	
	// 移除片段部分（#tag）
	if fragIndex := strings.Index(urlStr, "#"); fragIndex != -1 {
		urlStr = urlStr[:fragIndex]
	}

	// 解析URL
	parsedURL, err := url.Parse("trojan://" + urlStr)
	if err != nil {
		return fmt.Errorf("invalid trojan URL: %v", err)
	}

	// 解析password（用户信息部分）
	if parsedURL.User == nil {
		return fmt.Errorf("missing password in trojan URL")
	}
	c.Password = parsedURL.User.Username()
	if c.Password == "" {
		return fmt.Errorf("empty password in trojan URL")
	}

	// 解析server和port
	c.Server = parsedURL.Hostname()
	if c.Server == "" {
		return fmt.Errorf("missing server in trojan URL")
	}

	portStr := parsedURL.Port()
	if portStr == "" {
		c.ServerPort = 443 // trojan默认端口
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port in trojan URL: %s", portStr)
		}
		c.ServerPort = port
	}

	// 解析查询参数
	query := parsedURL.Query()
	
	// 设置TLS配置（trojan默认启用TLS）
	c.TLS = &TrojanTLS{
		Enabled: true,
	}

	// 检查sni参数
	if sni := query.Get("sni"); sni != "" {
		c.TLS.ServerName = sni
	} else if serverName := query.Get("peer"); serverName != "" {
		c.TLS.ServerName = serverName
	} else {
		// 如果没有指定SNI，使用服务器地址
		c.TLS.ServerName = c.Server
	}

	// 检查allowInsecure参数
	if allowInsecure := query.Get("allowInsecure"); allowInsecure == "1" || strings.ToLower(allowInsecure) == "true" {
		c.TLS.Insecure = true
	}

	return nil
}