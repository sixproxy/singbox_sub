package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"singbox_sub/src/github.com/sixproxy/util"
	"strconv"
	"strings"
)

type SsConfig struct {
	Outbound
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Method     string `json:"method"`
	Password   string `json:"password"`
	data       *string
}

func (c *SsConfig) GetTag() string {
	return c.Tag
}

func (c *SsConfig) GetType() string {
	return c.Type
}

func (c *SsConfig) Validate() error {
	if c.Server == "" || c.ServerPort == 0 {
		return fmt.Errorf("shadowsocks outbound must have server and port")
	}
	return nil
}

var _ OutboundConfig = (*SsConfig)(nil)

func (c *SsConfig) SetData(data string) error {
	tmp := data[len("ss://"):]
	c.data = &tmp
	c.Type = "shadowsocks"
	// 解析tag
	tag := util.ParseTag(*c.data)
	c.Tag = tag

	// 解析出ss认证参数
	err := c.parseBasicAuth()
	if err != nil {
		return err
	}
	return nil
}

func (c *SsConfig) String() string {
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

func (c *SsConfig) parseBasicAuth() error {
	// 清理查询参数
	if qIndex := strings.Index(*c.data, "?"); qIndex != -1 {
		tmp := (*c.data)[:qIndex]
		c.data = &tmp
	}

	// 检查是否包含@符号（表示明文格式）
	if strings.Contains(*c.data, "@") {
		return c.parseExplicitFormat()
	} else {
		return c.parseBase64Format()
	}
}

// 解析明文格式: method:password@server:port
func (c *SsConfig) parseExplicitFormat() error {
	// 正则匹配: (method:password)@(server):(port)
	pattern := regexp.MustCompile(`^(.*?)@([^:]+):(\d+)`)
	matches := pattern.FindStringSubmatch(*c.data)
	if len(matches) != 4 {
		return fmt.Errorf("无效的明文格式: %s", *c.data)
	}

	authPart := matches[1]                       // method:password
	server := matches[2]                         // server
	portStr := strings.Split(matches[3], "&")[0] // port，去掉可能的参数

	// 解析服务器和端口
	c.Server = server
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("无效的端口号: %s", portStr)
	}
	c.ServerPort = port

	// 解析认证信息
	return c.parseAuthInfo(authPart)
}

// 解析base64格式
func (c *SsConfig) parseBase64Format() error {
	// base64解码
	decoded, err := base64.StdEncoding.DecodeString(*c.data)
	if err != nil {
		return fmt.Errorf("base64解码失败: %v", err)
	}

	decodedStr := string(decoded)

	// 正则匹配: method:password@server:port
	pattern := regexp.MustCompile(`^([^:]+):([^@]+)@([^:]+):(\d+)`)
	matches := pattern.FindStringSubmatch(decodedStr)
	if len(matches) != 5 {
		return fmt.Errorf("无效的base64解码格式")
	}

	c.Method = matches[1]
	c.Password = matches[2]
	c.Server = matches[3]

	port, err := strconv.Atoi(matches[4])
	if err != nil {
		return fmt.Errorf("无效的端口号: %s", matches[4])
	}
	c.ServerPort = port

	return nil
}

// 解析认证信息（method:password，可能是base64编码的）
func (c *SsConfig) parseAuthInfo(authPart string) error {
	// 先尝试base64解码
	if decoded, err := base64.StdEncoding.DecodeString(authPart); err == nil {
		decodedStr := string(decoded)
		// 检查解码后的字符串是否包含冒号
		if strings.Contains(decodedStr, ":") {
			authPart = decodedStr
		}
		// 如果解码后没有冒号，则使用原始字符串
	}

	// 分离method和password
	parts := strings.SplitN(authPart, ":", 2) // 使用SplitN确保只分割成两部分
	if len(parts) != 2 {
		return fmt.Errorf("无效的认证格式: %s", authPart)
	}

	c.Method = parts[0]
	c.Password = parts[1]

	return nil
}
