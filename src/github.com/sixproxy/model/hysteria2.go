package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/util"
	"strings"
)

type Hysteria2Config struct {
	Outbound
	Server     string      `json:"server"`
	ServerPort int         `json:"server_port"`
	Password   string      `json:"password"`
	UpMbps     int         `json:"up_mbps"`
	DownMbps   int         `json:"down_mbps"`
	TLS        TLSConfig   `json:"tls"`
	Detour     *string     `json:"detour"`
	Obfs       *ObfsConfig `json:"obfs,omitempty"`
}

func (c *Hysteria2Config) GetTag() string {
	return c.Tag
}

func (c *Hysteria2Config) GetType() string {
	return c.Type
}

func (c *Hysteria2Config) Validate() error {
	return nil
}

var _ OutboundConfig = (*Hysteria2Config)(nil)

func (c *Hysteria2Config) SetData(data string) error {

	c.Type = constant.OUTBOUND_HY2
	c.UpMbps = 100
	c.DownMbps = 500
	c.TLS.Enabled = true
	c.TLS.Insecure = false
	selectStr := "Select"
	c.Detour = &selectStr
	c.Obfs = &ObfsConfig{}
	c.TLS.Alpn = []string{"h3"}
	tag := util.ParseTag(data)

	// 1.解析tag
	c.Tag = tag

	u, err := url.Parse(data)
	if err != nil {
		return err
	}

	// 2.解析server、server_port
	if u.Host == "" {
		return fmt.Errorf("host is missing")
	}
	hostAndPort := strings.Split(u.Host, ":")
	c.Server = hostAndPort[0]
	if len(hostAndPort) > 1 {
		port := util.String2Int(hostAndPort[1])
		c.ServerPort = port
	} else {
		c.ServerPort = 443
	}

	// 3.解析password
	if u.User != nil {
		c.Password = u.User.Username()
	}

	query := u.Query()

	c.TLS.ServerName = query.Get("sni")

	if alpn := query.Get("alpn"); len(alpn) > 0 {
		c.TLS.Alpn = []string{alpn}
	}

	if obfs := query.Get("obfs"); len(obfs) > 0 {
		c.Obfs.Type = obfs
		c.Obfs.Password = query.Get("obfs-password")
	}
	return nil
}

func (c *Hysteria2Config) String() string {

	if c.Obfs != nil && c.Obfs.Type == "" {
		c.Obfs = nil
	}

	copy := *c
	if copy.Obfs != nil && copy.Obfs.Type == "" {
		copy.Obfs = nil
	}
	b, err := json.Marshal(copy)
	if err != nil {
		return ""
	}
	return string(b)
}

type TLSConfig struct {
	Enabled    bool     `json:"enabled"`
	ServerName string   `json:"server_name,omitempty"`
	Insecure   bool     `json:"insecure"`
	Alpn       []string `json:"alpn"`
}

type ObfsConfig struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}
