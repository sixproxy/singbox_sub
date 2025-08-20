package model

import (
	"encoding/json"
	"fmt"
	"singbox_sub/src/github.com/sixproxy/constant"
)

// VlessConfig VLESS出站配置
type VlessConfig struct {
	Outbound
	Server     string      `json:"server"`
	ServerPort int         `json:"server_port"`
	UUID       string      `json:"uuid"`
	Flow       string      `json:"flow,omitempty"`       // xtls流控：xtls-rprx-vision等
	PacketEncoding string  `json:"packet_encoding,omitempty"` // udp包编码方式
	Transport  *Transport  `json:"transport,omitempty"`  // 传输层配置
	TLS        *VlessTLSConfig  `json:"tls,omitempty"`        // TLS配置
	Detour     *string     `json:"detour,omitempty"`     // 出站代理标签
}

// Transport 传输层配置
type Transport struct {
	Type     string                 `json:"type"`               // 传输类型：tcp, ws, grpc, http等
	Path     string                 `json:"path,omitempty"`     // WebSocket/HTTP路径
	Host     string                 `json:"host,omitempty"`     // HTTP Host头
	Headers  map[string]interface{} `json:"headers,omitempty"`  // 自定义头
	Method   string                 `json:"method,omitempty"`   // HTTP方法
	ServiceName string              `json:"service_name,omitempty"` // gRPC服务名
}

// VlessTLSConfig VLESS专用TLS配置（支持Reality）
type VlessTLSConfig struct {
	Enabled            bool           `json:"enabled"`
	DisableSNI         bool           `json:"disable_sni,omitempty"`
	ServerName         string         `json:"server_name,omitempty"`
	Insecure           bool           `json:"insecure,omitempty"`
	ALPN               []string       `json:"alpn,omitempty"`
	MinVersion         string         `json:"min_version,omitempty"`
	MaxVersion         string         `json:"max_version,omitempty"`
	CipherSuites       []string       `json:"cipher_suites,omitempty"`
	Certificate        []string       `json:"certificate,omitempty"`
	CertificatePath    string         `json:"certificate_path,omitempty"`
	ECH                *ECHConfig     `json:"ech,omitempty"`
	UTLS               *UTLSConfig    `json:"utls,omitempty"`
	Reality            *RealityConfig `json:"reality,omitempty"` // Reality配置
}

// ECHConfig ECH配置
type ECHConfig struct {
	Enabled                bool   `json:"enabled"`
	PQSignatureSchemesEnabled bool `json:"pq_signature_schemes_enabled,omitempty"`
	DynamicRecordSizingDisabled bool `json:"dynamic_record_sizing_disabled,omitempty"`
	Config                 []string `json:"config,omitempty"`
}

// UTLSConfig uTLS配置  
type UTLSConfig struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint,omitempty"` // chrome, firefox, safari等
}

// RealityConfig Reality配置
type RealityConfig struct {
	Enabled    bool   `json:"enabled"`
	PublicKey  string `json:"public_key"`              // 公钥
	ShortID    string `json:"short_id"`               // 短ID
}

// NewVlessOutbound 创建VLESS出站配置
func NewVlessOutbound(tag, server string, port int, uuid, flow string) *VlessConfig {
	return &VlessConfig{
		Outbound: Outbound{
			Tag:  tag,
			Type: constant.OUTBOUND_VLESS,
		},
		Server:     server,
		ServerPort: port,
		UUID:       uuid,
		Flow:       flow,
		Detour:     nil, // VLESS通常不需要链式代理
	}
}

// SetReality 设置Reality配置
func (c *VlessConfig) SetReality(publicKey, shortID, serverName string) *VlessConfig {
	if c.TLS == nil {
		c.TLS = &VlessTLSConfig{Enabled: true}
	}
	
	c.TLS.Reality = &RealityConfig{
		Enabled:   true,
		PublicKey: publicKey,
		ShortID:   shortID,
	}
	
	c.TLS.ServerName = serverName
	return c
}

// SetTransport 设置传输层配置
func (c *VlessConfig) SetTransport(transportType, path, host string) *VlessConfig {
	c.Transport = &Transport{
		Type: transportType,
		Path: path,
		Host: host,
	}
	return c
}

// SetTLS 设置普通TLS配置
func (c *VlessConfig) SetTLS(serverName string, insecure bool) *VlessConfig {
	c.TLS = &VlessTLSConfig{
		Enabled:    true,
		ServerName: serverName,
		Insecure:   insecure,
	}
	return c
}

// SetUTLS 设置uTLS指纹
func (c *VlessConfig) SetUTLS(fingerprint string) *VlessConfig {
	if c.TLS == nil {
		c.TLS = &VlessTLSConfig{Enabled: true}
	}
	
	c.TLS.UTLS = &UTLSConfig{
		Enabled:     true,
		Fingerprint: fingerprint,
	}
	return c
}

// SetFlow 设置XTLS流控
func (c *VlessConfig) SetFlow(flow string) *VlessConfig {
	c.Flow = flow
	return c
}

// SetPacketEncoding 设置UDP包编码
func (c *VlessConfig) SetPacketEncoding(encoding string) *VlessConfig {
	c.PacketEncoding = encoding
	return c
}

// Validate 验证配置有效性
func (c *VlessConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("VLESS server不能为空")
	}
	
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("VLESS server_port必须在1-65535范围内")
	}
	
	if c.UUID == "" {
		return fmt.Errorf("VLESS UUID不能为空")
	}
	
	// 验证UUID格式（简单检查）
	if len(c.UUID) != 36 {
		return fmt.Errorf("VLESS UUID格式不正确")
	}
	
	// 验证flow参数
	if c.Flow != "" {
		validFlows := []string{"", "xtls-rprx-vision", "xtls-rprx-vision-udp443"}
		valid := false
		for _, f := range validFlows {
			if c.Flow == f {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("不支持的flow参数: %s", c.Flow)
		}
	}
	
	// 验证Reality配置
	if c.TLS != nil && c.TLS.Reality != nil && c.TLS.Reality.Enabled {
		if c.TLS.Reality.PublicKey == "" {
			return fmt.Errorf("Reality public_key不能为空")
		}
		if c.TLS.Reality.ShortID == "" {
			return fmt.Errorf("Reality short_id不能为空")
		}
	}
	
	return nil
}

// MarshalJSON 自定义JSON序列化
func (c *VlessConfig) MarshalJSON() ([]byte, error) {
	// 创建一个包含所有字段的map
	data := map[string]interface{}{
		"tag":         c.Tag,
		"type":        c.Type,
		"server":      c.Server,
		"server_port": c.ServerPort,
		"uuid":        c.UUID,
	}
	
	// 添加可选字段
	if c.Flow != "" {
		data["flow"] = c.Flow
	}
	
	if c.PacketEncoding != "" {
		data["packet_encoding"] = c.PacketEncoding
	}
	
	if c.Transport != nil {
		data["transport"] = c.Transport
	}
	
	if c.TLS != nil {
		data["tls"] = c.TLS
	}
	
	if c.Detour != nil {
		data["detour"] = c.Detour
	}
	
	return json.Marshal(data)
}