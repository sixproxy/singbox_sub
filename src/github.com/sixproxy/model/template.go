package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
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

const (
	Include string = "include"
	Exclude string = "exclude"
)

type DelegateParseNodesFunc func(nodes []string) []string

func (cfg *Config) nodesWithHttpGet(delegateParse DelegateParseNodesFunc) []string {
	// 检查订阅地址是否有效
	if len(cfg.Subs) == 0 {
		logger.Error("没有配置订阅地址")
		return []string{}
	}

	url := cfg.Subs[0].URL
	if url == "" {
		logger.Error("订阅地址为空，请在 config.yaml 中设置有效的订阅 URL")
		return []string{}
	}

	logger.Info("正在获取订阅内容: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("无法获取订阅内容: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态
	if resp.StatusCode != 200 {
		logger.Error("订阅服务器返回错误状态: %d %s", resp.StatusCode, resp.Status)
		return []string{}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取订阅内容失败: %v", err)
		return []string{}
	}

	if len(body) == 0 {
		logger.Error("订阅内容为空")
		return []string{}
	}

	// 尝试解码 base64
	bodyStr := string(body)
	decoded, err := base64.StdEncoding.DecodeString(bodyStr)
	if err != nil {
		logger.Warn("订阅内容不是 base64 编码，尝试直接解析: %v", err)
		// 如果不是 base64，直接使用原内容
		decoded = body
	}

	nodes := strings.Split(string(decoded), "\n")

	// 过滤空行
	var validNodes []string
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node != "" {
			validNodes = append(validNodes, node)
		}
	}

	if len(validNodes) == 0 {
		logger.Error("订阅内容中没有找到有效的节点")
		return []string{}
	}

	logger.Info("从订阅中获取到 %d 个节点", len(validNodes))
	configNodes := delegateParse(validNodes)

	return configNodes
}

func (cfg *Config) RenderTemplate(delegateParse DelegateParseNodesFunc) error {

	// 解析模版并渲染
	proxyNode := cfg.nodesWithHttpGet(delegateParse)

	for i := range cfg.Outbounds {
		// 获取第i行 出站模版
		outbound := cfg.Outbounds[i]

		// 判断出站模版类型
		switch o := outbound.(type) {
		case *URLTestOutbound:
			outbounds := o.Outbounds // 获取 出站 表达式
			filters := o.Filters     // 获取 出站 过滤器
			if len(outbounds) == 1 && outbounds[0] == constant.ALL_NODES {

				tmpNodes := FilterNodes(proxyNode, filters)
				o.Outbounds = GetTags(tmpNodes)
			}
			// 删除filter
			o.Filters = nil
		case *SelectorOutbound:
			outbounds := o.Outbounds // 获取 出站 表达式
			filters := o.Filters     // 获取 出站 过滤器
			if len(outbounds) == 1 && outbounds[0] == constant.ALL_NODES {

				tmpNodes := FilterNodes(proxyNode, filters)
				o.Outbounds = GetTags(tmpNodes)
			}
			// 删除filter
			o.Filters = nil
		default:
			logger.Debug("未处理的出站类型: %+v", o)
		}
	}
	cfg.Subs = nil

	// 合并所有节点到Outbounds
	for _, nodeJson := range proxyNode {
		outbound := NewOutbound(getNodeType(nodeJson))

		err := json.Unmarshal([]byte(nodeJson), &outbound)
		if err != nil {
			return err
		}
		if unvalidNode(outbound.GetTag()) {
			continue
		}
		cfg.Outbounds = append(cfg.Outbounds, outbound)
	}
	return nil
}

func unvalidNode(tag string) bool {
	switch {
	case strings.Contains(tag, "官网"):
		return true
	case strings.Contains(tag, "流量"):
		return true
	default:
		return false
	}
}

func getNodeType(node string) string {
	switch {
	case strings.Contains(node, constant.OUTBOUND_SS):
		return constant.OUTBOUND_SS
	case strings.Contains(node, constant.OUTBOUND_SSR):
		return constant.OUTBOUND_SSR
	case strings.Contains(node, constant.OUTBOUND_HY2):
		return constant.OUTBOUND_HY2
	case strings.Contains(node, constant.OUTBOUND_TROJAN):
		return constant.OUTBOUND_TROJAN
	case strings.Contains(node, constant.OUTBOUND_ANYTLS):
		return constant.OUTBOUND_ANYTLS
	case strings.Contains(node, constant.OUTBOUND_SELECTOR):
		return constant.OUTBOUND_SELECTOR
	case strings.Contains(node, constant.OUTBOUND_URLTEST):
		return constant.OUTBOUND_URLTEST
	default:
		return ""
	}
}

func (cfg *Config) LinuxConfig(path string) error {
	// 如果没有指定路径，使用默认路径
	if path == "" {
		path = "linux_config.json"
	}

	// 创建或打开文件
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	// 创建encoder，输出到文件
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // 可选：不转义HTML字符

	err = encoder.Encode(cfg)
	if err != nil {
		return fmt.Errorf("写入JSON失败: %v", err)
	}

	logger.Info("配置文件已写入: %s", path)
	return nil
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

func (cfg *Config) MacConfig(path string) error {
	// 如果没有指定路径，使用默认路径
	if path == "" {
		path = "mac_config.json"
	}
	cfg.Inbounds = make([]Inbound, 0)
	cfg.Inbounds = append(cfg.Inbounds, Inbound{
		Tag:                    "tun-in",
		Type:                   "tun",
		Address:                []string{"10.8.8.8/30"},
		MTU:                    9000,
		AutoRoute:              true,
		Stack:                  "system",
		RouteExcludeAddressSet: []string{"geosite-private", "geosite-ctm_cn", "geoip-cn"},
	})

	cfg.DNS.Servers = []DNSServer{
		DNSServer{
			Tag:             constant.DNS_LOCAL,
			Address:         "114.114.114.114",
			AddressStrategy: "ipv4_only",
			Strategy:        "ipv4_only",
			Detour:          "DirectConn",
		},
		DNSServer{
			Tag:     constant.DNS_PROXY,
			Address: "https://8.8.8.8/dns-query",
			Detour:  "DNS",
		},
	}

	cfg.DNS.Rules = []DNSRule{
		{Outbound: "any", Server: "dns_local", DisableCache: true},
		{ClashMode: "Direct", Server: constant.DNS_LOCAL},
		{ClashMode: "Global", Server: constant.DNS_PROXY},
		{RuleSet: "geosite-cn", Server: constant.DNS_LOCAL},
		{RuleSet: "geosite-geolocation-!cn", Server: constant.DNS_PROXY},
	}

	// 设置这些字段但不会输出到JSON（已通过json标签控制）
	cfg.Route.DefaultMark = 0
	cfg.Route.DefaultDomainResolver = nil

	cfg.Experimental.CacheFile.Path = ""
	cfg.Experimental.ClashAPI.ExternalController = "127.0.0.1:9095"

	// 创建或打开文件
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	// 创建encoder，输出到文件
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // 可选：不转义HTML字符

	err = encoder.Encode(cfg)
	if err != nil {
		return fmt.Errorf("写入JSON失败: %v", err)
	}

	logger.Info("配置文件已写入: %s", path)
	return nil
}

func GetTags(nodes []string) []string {
	var tags []string
	for _, nodeJSON := range nodes {
		var node Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			// 如果解析失败，跳过这个节点
			continue
		}
		tags = append(tags, node.Tag)
	}
	return tags
}

func FilterNodes(nodes []string, rules []Filter) []string {
	var result []string

	for _, nodeJSON := range nodes {
		// 解析JSON获取tag值
		var node Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			// 如果解析失败，跳过这个节点
			continue
		}

		shouldInclude := true

		// 应用所有过滤规则
		for _, rule := range rules {
			matched := matchPatterns(node.Tag, rule.Patterns)

			switch rule.Action {
			case Include:
				// include规则：如果没有匹配到任何pattern，则排除
				if !matched {
					shouldInclude = false
				}
			case Exclude:
				// exclude规则：如果匹配到任何pattern，则排除
				if matched {
					shouldInclude = false
				}
			}

			// 如果已经决定排除，就不用继续检查其他规则了
			if !shouldInclude {
				break
			}
		}

		if shouldInclude {
			result = append(result, nodeJSON)
		}
	}

	return result
}

// matchPatterns 检查tag是否匹配任何一个pattern
func matchPatterns(tag string, patterns []string) bool {
	for _, pattern := range patterns {
		ps := strings.Split(pattern, "|")
		for _, p := range ps {
			if matchPattern(tag, p) {
				return true
			}
		}
	}
	return false
}

// matchPattern 检查tag是否匹配单个pattern
// 支持普通字符串匹配和正则表达式匹配
func matchPattern(tag, pattern string) bool {
	// 检查是否为正则表达式（简单判断是否包含正则特殊字符）
	if isRegexPattern(pattern) {
		// 尝试正则匹配
		matched, err := regexp.MatchString(pattern, tag)
		if err != nil {
			// 如果正则表达式无效，回退到字符串包含匹配
			return strings.Contains(tag, pattern)
		}
		return matched
	} else {
		// 普通字符串包含匹配
		return strings.Contains(tag, pattern)
	}
}

// isRegexPattern 简单判断是否可能是正则表达式
// 这里使用启发式方法检测常见的正则表达式特征
func isRegexPattern(pattern string) bool {
	// 包含正则表达式特殊字符的话，认为是正则
	regexChars := []string{"^", "$", "*", "+", "?", ".", "[", "]", "(", ")", "{", "}", "\\"}
	for _, char := range regexChars {
		if strings.Contains(pattern, char) {
			return true
		}
	}
	return false
}
