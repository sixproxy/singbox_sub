package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/util"
	"strings"
)

type DelegateParseNodesFunc func(nodes []string) []string

type SubService struct {
	Cfg *model.Config
}

func (sub *SubService) nodesWithHttpGet(delegateParse DelegateParseNodesFunc) []string {
	// 检查订阅地址是否有效
	if len(sub.Cfg.Subs) == 0 {
		logger.Error("没有配置订阅地址")
		return []string{}
	}

	url := sub.Cfg.Subs[0].URL
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

func (sub *SubService) RenderTemplate(delegateParse DelegateParseNodesFunc) error {

	// 解析模版并渲染
	proxyNode := sub.nodesWithHttpGet(delegateParse)

	for i := range sub.Cfg.Outbounds {
		// 获取第i行 出站模版
		outbound := sub.Cfg.Outbounds[i]

		// 判断出站模版类型
		switch o := outbound.(type) {
		case *model.URLTestOutbound:
			outbounds := o.Outbounds // 获取 出站 表达式
			filters := o.Filters     // 获取 出站 过滤器
			if len(outbounds) == 1 && outbounds[0] == constant.ALL_NODES {

				tmpNodes := FilterNodes(proxyNode, filters)
				o.Outbounds = GetTags(tmpNodes)
				if len(o.Outbounds) == 0 {
					o.Outbounds = []string{constant.OUTBOUND_BLOCK}
				}
			}
			// 删除filter
			o.Filters = nil
		case *model.SelectorOutbound:
			outbounds := o.Outbounds // 获取 出站 表达式
			filters := o.Filters     // 获取 出站 过滤器
			if len(outbounds) == 1 && outbounds[0] == constant.ALL_NODES {

				tmpNodes := FilterNodes(proxyNode, filters)
				o.Outbounds = GetTags(tmpNodes)
				if len(o.Outbounds) == 0 {
					o.Outbounds = []string{constant.OUTBOUND_BLOCK}
				}
			}
			// 删除filter
			o.Filters = nil
		default:
			logger.Debug("未处理的出站类型: %+v", o)
		}
	}
	sub.Cfg.Subs = nil

	// 合并所有节点到Outbounds
	for _, nodeJson := range proxyNode {
		outbound := model.NewOutbound(util.GetNodeType(nodeJson))

		err := json.Unmarshal([]byte(nodeJson), &outbound)
		if err != nil {
			return err
		}
		if util.InvalidNode(outbound.GetTag()) {
			continue
		}
		sub.Cfg.Outbounds = append(sub.Cfg.Outbounds, outbound)
	}
	return nil
}

func (sub *SubService) LinuxConfig(path string) error {
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

	err = encoder.Encode(sub.Cfg)
	if err != nil {
		return fmt.Errorf("写入JSON失败: %v", err)
	}

	logger.Info("配置文件已写入: %s", path)
	return nil
}

func (sub *SubService) MacConfig(path string) error {
	// 如果没有指定路径，使用默认路径
	if path == "" {
		path = "mac_config.json"
	}
	sub.Cfg.Inbounds = make([]model.Inbound, 0)
	sub.Cfg.Inbounds = append(sub.Cfg.Inbounds, model.Inbound{
		Tag:                    "tun-in",
		Type:                   "tun",
		Address:                []string{"10.8.8.8/30"},
		MTU:                    9000,
		AutoRoute:              true,
		Stack:                  "system",
		RouteExcludeAddressSet: []string{"geosite-private", "geosite-ctm_cn", "geoip-cn"},
	})

	sub.Cfg.DNS.Servers = []model.DNSServer{
		model.DNSServer{
			Tag:             constant.DNS_LOCAL,
			Address:         "114.114.114.114",
			AddressStrategy: "ipv4_only",
			Strategy:        "ipv4_only",
			Detour:          "DirectConn",
		},
		model.DNSServer{
			Tag:     constant.DNS_PROXY,
			Address: "https://8.8.8.8/dns-query",
			Detour:  "DNS",
		},
	}

	sub.Cfg.DNS.Rules = []model.DNSRule{
		{Outbound: "any", Server: "dns_local", DisableCache: true},
		{ClashMode: "Direct", Server: constant.DNS_LOCAL},
		{ClashMode: "Global", Server: constant.DNS_PROXY},
		{RuleSet: "geosite-cn", Server: constant.DNS_LOCAL},
		{RuleSet: "geosite-geolocation-!cn", Server: constant.DNS_PROXY},
	}

	// 设置这些字段但不会输出到JSON（已通过json标签控制）
	sub.Cfg.Route.DefaultMark = 0
	sub.Cfg.Route.DefaultDomainResolver = nil

	sub.Cfg.Experimental.CacheFile.Path = ""
	sub.Cfg.Experimental.ClashAPI.ExternalController = "127.0.0.1:9095"

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

	err = encoder.Encode(sub.Cfg)
	if err != nil {
		return fmt.Errorf("写入JSON失败: %v", err)
	}

	logger.Info("配置文件已写入: %s", path)
	return nil
}

func GetTags(nodes []string) []string {
	var tags []string
	for _, nodeJSON := range nodes {
		var node model.Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			// 如果解析失败，跳过这个节点
			continue
		}
		tags = append(tags, node.Tag)
	}
	return tags
}

func FilterNodes(nodes []string, rules []model.Filter) []string {
	var result []string

	for _, nodeJSON := range nodes {
		// 解析JSON获取tag值
		var node model.Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			// 如果解析失败，跳过这个节点
			continue
		}

		shouldInclude := true

		// 应用所有过滤规则
		for _, rule := range rules {
			matched := matchPatterns(node.Tag, rule.Patterns)

			switch rule.Action {
			case constant.INCLUDE:
				// include规则：如果没有匹配到任何pattern，则排除
				if !matched {
					shouldInclude = false
				}
			case constant.EXCLUDE:
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
			if util.MatchPattern(tag, p) {
				return true
			}
		}
	}
	return false
}
