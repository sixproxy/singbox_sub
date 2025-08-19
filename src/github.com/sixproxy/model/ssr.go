package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"singbox_sub/src/github.com/sixproxy/util"
	"strconv"
	"strings"
)

type SsrConfig struct {
	Outbound
	Server        string `json:"server"`
	ServerPort    int    `json:"server_port"`
	Method        string `json:"method"`
	Password      string `json:"password"`
	Protocol      string `json:"protocol,omitempty"`
	ProtocolParam string `json:"protocol_param,omitempty"`
	Obfs          string `json:"obfs,omitempty"`
	ObfsParam     string `json:"obfs_param,omitempty"`
	data          *string
}

func (c *SsrConfig) GetTag() string {
	return c.Tag
}

func (c *SsrConfig) GetType() string {
	return c.Type
}

func (c *SsrConfig) Validate() error {
	if c.Server == "" || c.ServerPort == 0 {
		return fmt.Errorf("shadowsocksr outbound must have server and port")
	}
	if c.Method == "" || c.Password == "" {
		return fmt.Errorf("shadowsocksr outbound must have method and password")
	}
	return nil
}

var _ OutboundConfig = (*SsrConfig)(nil)

func (c *SsrConfig) SetData(data string) error {
	if !strings.HasPrefix(data, "ssr://") {
		return fmt.Errorf("invalid ssr:// URL")
	}

	tmp := data[len("ssr://"):]
	c.data = &tmp
	c.Type = "shadowsocksr"

	// 解析SSR URL
	err := c.parseSSRURL()
	if err != nil {
		return fmt.Errorf("parse SSR URL failed: %v", err)
	}

	return nil
}

func (c *SsrConfig) String() string {
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

func (c *SsrConfig) parseSSRURL() error {
	// 按照Python代码的逻辑，但先正确分离主体和参数
	info := *c.data
	if info == "" || strings.TrimSpace(info) == "" {
		return fmt.Errorf("empty SSR data")
	}

	// 首先分离主体部分和参数部分
	dataParts := strings.SplitN(info, "/?", 2)
	mainData := dataParts[0]
	paramPart := ""
	if len(dataParts) == 2 {
		paramPart = dataParts[1]
	}

	// 对主体部分进行base64解码
	var proxyStr string
	if decoded, err := base64.URLEncoding.DecodeString(mainData); err == nil {
		proxyStr = string(decoded)
	} else if decoded, err := base64.StdEncoding.DecodeString(mainData); err == nil {
		proxyStr = string(decoded)
	} else {
		// 如果解码失败，使用原始数据
		proxyStr = mainData
	}

	// 按冒号分割
	parts := strings.Split(proxyStr, ":")

	// 处理特殊情况 (len(parts) == 5)
	i := 0
	if len(parts) == 5 {
		i = 1
		// 这种情况下需要特殊处理
		nextPart := strings.SplitN(proxyStr, "=", 2)
		if len(nextPart) == 2 {
			parts = append(parts, nextPart[1])
		}
		// 检查常见的obfs类型
		obfsTypes := []string{"plain", "http_simple", "http_post", "random_head", "tls1.2_ticket_auth"}
		for _, obfsType := range obfsTypes {
			if strings.Contains(parts[4], obfsType) {
				splitResult := strings.Split(parts[4], obfsType)
				if len(splitResult) > 1 {
					parts[5] = splitResult[len(splitResult)-1]
					parts[4] = obfsType
				}
				break
			}
		}
	}

	if len(parts) < 6 {
		return fmt.Errorf("invalid SSR format, expected at least 6 parts but got %d", len(parts))
	}

	// 基本信息
	c.Server = parts[0]

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid port: %v", err)
	}
	c.ServerPort = port

	c.Protocol = parts[2]
	c.Method = parts[3]
	c.Obfs = parts[4]

	// 处理密码和参数 - 重新组合密码部分和参数
	var fullPasswordPart string
	if paramPart != "" {
		fullPasswordPart = parts[5] + "/?" + paramPart
	} else {
		fullPasswordPart = parts[5]
	}

	passwordParams := strings.SplitN(fullPasswordPart, "/?", 2)

	if i == 0 {
		// 正常情况
		if passwordDecoded, err := decodeBase64WithPadding(passwordParams[0]); err == nil {
			c.Password = passwordDecoded
		} else {
			c.Password = passwordParams[0]
		}

		if len(passwordParams) > 1 {
			params := strings.Split(passwordParams[1], "&")
			c.parseSSRParams(params)
		}
	} else {
		// 特殊情况 (i == 1)
		remarksParts := strings.Split(passwordParams[0], "remarks")
		if len(remarksParts) > 0 {
			if passwordDecoded, err := decodeBase64WithPadding(remarksParts[0]); err == nil {
				c.Password = passwordDecoded
			} else {
				c.Password = remarksParts[0]
			}
		}

		if len(passwordParams) > 1 {
			remaining := strings.Split(passwordParams[len(passwordParams)-1], remarksParts[0])
			if len(remaining) > 1 {
				params := strings.Split(remaining[len(remaining)-1], "&")
				c.parseSSRParams(params)
			}
		}
	}

	// 如果没有tag，生成默认tag
	if c.Tag == "" {
		c.Tag = fmt.Sprintf("SSR-%s:%d", c.Server, c.ServerPort)
	}

	return nil
}

// decodeBase64WithPadding 处理缺少padding的base64字符串
func decodeBase64WithPadding(s string) (string, error) {
	// 添加padding
	missing := len(s) % 4
	if missing != 0 {
		s += string(make([]byte, 4-missing))
		for i := len(s) - (4 - missing); i < len(s); i++ {
			s = s[:i] + "=" + s[i+1:]
		}
	}

	// 尝试URL safe解码
	if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}

	// 尝试标准base64解码
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return string(decoded), nil
	}

	return "", fmt.Errorf("failed to decode base64")
}

func (c *SsrConfig) parseSSRParams(params []string) {
	var groupTag string

	for _, param := range params {
		keyValue := strings.SplitN(param, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		keyName := keyValue[0]

		if value, err := decodeBase64WithPadding(keyValue[1]); err == nil {
			switch keyName {
			case "obfsparam":
				c.ObfsParam = value
			case "protoparam":
				c.ProtocolParam = value
			case "remarks":
				c.Tag = util.RemoveEmoji(strings.TrimSpace(value))
			case "group":
				// 只有在没有remarks的情况下才使用group作为tag
				groupTag = util.RemoveEmoji(strings.TrimSpace(value))
			}
		}
	}

	// 如果没有从remarks获取到tag，使用group
	if c.Tag == "" && groupTag != "" {
		c.Tag = groupTag
	}
}
