package model

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	Subs         []Sub            `json:"subs,omitempty"`
	Log          LogConfig        `json:"log"`
	Experimental Experimental     `json:"experimental"`
	DNS          DNSConfig        `json:"dns"`
	Inbounds     []Inbound        `json:"inbounds"`
	Outbounds    []OutboundConfig `json:"outbounds"`
	Route        Route            `json:"route"`
}

type Sub struct {
	URL      string `json:"url"`
	Enabled  bool   `json:"enabled"`
	Prefix   string `json:"prefix"`
	Insecure bool   `json:"insecure"`
	Detour   string `json:"detour"`
}

// --- log -------------------------------------------------
type LogConfig struct {
	Disabled  bool   `json:"disabled"`
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

// --- experimental ----------------------------------------
type Experimental struct {
	ClashAPI  ClashAPI  `json:"clash_api"`
	CacheFile CacheFile `json:"cache_file"`
}

type ClashAPI struct {
	ExternalController       string `json:"external_controller"`
	ExternalUI               string `json:"external_ui"`
	Secret                   string `json:"secret"`
	ExternalUIDownloadDetour string `json:"external_ui_download_detour"`
	DefaultMode              string `json:"default_mode"`
	ExternalUIDownloadURL    string `json:"external_ui_download_url"`
}

type CacheFile struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
}

// --- dns -------------------------------------------------
type DNSConfig struct {
	Servers          []DNSServer `json:"servers"`
	Rules            []DNSRule   `json:"rules"`
	Final            string      `json:"final"`
	Strategy         string      `json:"strategy"`
	ClientSubnet     string      `json:"client_subnet"`
	DisableCache     bool        `json:"disable_cache"`
	DisableExpire    bool        `json:"disable_expire"`
	IndependentCache bool        `json:"independent_cache"`
}

type DNSServer struct {
	Tag             string  `json:"tag"`
	Type            *string `json:"type,omitempty"`
	Server          string  `json:"server,omitempty"`
	Detour          string  `json:"detour,omitempty"`
	Address         string  `json:"address,omitempty"`
	AddressStrategy string  `json:"address_strategy,omitempty"`
	Strategy        string  `json:"strategy,omitempty"`
}

type DNSRule struct {
	ClashMode string `json:"clash_mode,omitempty"`
	Server    string `json:"server"`
	RuleSet   string `json:"rule_set,omitempty"`
	// 兼容v1.12 之前版本
	Outbound     string `json:"outbound,omitempty"`
	DisableCache bool   `json:"disable_cache,omitempty"`
}

// --- inbounds / outbounds -------------------------------
type Inbound struct {
	Type                   string   `json:"type"`
	Tag                    string   `json:"tag"`
	Listen                 string   `json:"listen,omitempty"`
	ListenPort             int      `json:"listen_port,omitempty"`
	UDPTimeout             string   `json:"udp_timeout,omitempty"`
	Address                []string `json:"address,omitempty"`
	MTU                    int      `json:"mtu,omitempty"`
	AutoRoute              bool     `json:"auto_route,omitempty"`
	Stack                  string   `json:"stack,omitempty"`
	RouteExcludeAddressSet []string `json:"route_exclude_address_set,omitempty"`
}

type Filter struct {
	Action   string   `json:"action"`
	Patterns []string `json:"keywords"`
}

// --- route -----------------------------------------------
type Route struct {
	Final                 string                 `json:"final"`
	AutoDetectInterface   bool                   `json:"auto_detect_interface"`
	DefaultDomainResolver *DefaultDomainResolver `json:"default_domain_resolver,omitempty"`
	DefaultMark           int                    `json:"default_mark,omitempty"`
	Rules                 json.RawMessage        `json:"rules"`
	RuleSet               json.RawMessage        `json:"rule_set"`
}

type DefaultDomainResolver struct {
	Server       string `json:"server"`
	RewriteTll   int    `json:"rewrite_tll"`
	ClientSubnet string `json:"client_subnet"`
}

func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config

	// 序列化Outbounds
	var rawOutbounds []json.RawMessage
	for _, outbound := range c.Outbounds {
		data, err := json.Marshal(outbound)
		if err != nil {
			return nil, err
		}
		rawOutbounds = append(rawOutbounds, data)
	}

	return json.Marshal(&struct {
		*Alias
		Outbounds []json.RawMessage `json:"outbounds"`
	}{
		Alias:     (*Alias)(c),
		Outbounds: rawOutbounds,
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		*Alias
		Outbounds []json.RawMessage `json:"outbounds"`
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 反序列化Outbounds
	c.Outbounds = make([]OutboundConfig, 0, len(aux.Outbounds))

	for _, rawOutbound := range aux.Outbounds {
		// 先解析获取type字段
		var base Outbound
		if err := json.Unmarshal(rawOutbound, &base); err != nil {
			return err
		}

		// 根据type创建具体类型
		outbound := NewOutbound(base.Type)
		if outbound == nil {
			return fmt.Errorf("unknown outbound type: %s", base.Type)
		}

		// 完整解析
		if err := json.Unmarshal(rawOutbound, outbound); err != nil {
			return err
		}

		c.Outbounds = append(c.Outbounds, outbound)
	}

	return nil
}
