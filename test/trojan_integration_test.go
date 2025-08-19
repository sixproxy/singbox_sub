package test

import (
	"encoding/json"
	"os"
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/protocol"
	"strings"
	"testing"
)

// TestTrojanIntegration 测试trojan协议的完整集成流程
func TestTrojanIntegration(t *testing.T) {
	// 创建测试配置模板
	config := createTestConfig()

	// 准备测试数据：包含trojan节点的订阅内容
	testNodes := []string{
		"trojan://password123@example.com:443#日本节点1",
		"trojan://secret456@server.example.com:8443?sni=custom.domain.com#美国节点1",
		"trojan://testpass@192.168.1.100:443?allowInsecure=1#测试节点",
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@1.2.3.4:8388#SS节点", // 混合其他协议
		"trojan://mypassword@trojan.server.com:443?peer=peer.domain.com#香港节点",
	}

	// 解析测试节点
	parsedNodes := parseTestNodes(testNodes)

	// 手动模拟RenderTemplate的逻辑
	err := simulateRenderTemplate(config, parsedNodes)
	if err != nil {
		t.Fatalf("配置生成失败: %v", err)
	}

	// 验证生成的配置
	validateGeneratedConfig(t, config)

	// 测试配置文件输出
	testConfigOutput(t, config)
}

// createTestConfig 创建测试用的配置模板
func createTestConfig() *model.Config {
	return &model.Config{
		Subs: []model.Sub{
			{
				URL:     "https://example.com/subscription",
				Enabled: true,
				Prefix:  "test",
			},
		},
		Log: model.LogConfig{
			Level:     "info",
			Timestamp: true,
		},
		DNS: model.DNSConfig{
			Servers: []model.DNSServer{
				{Tag: "local", Server: "8.8.8.8"},
			},
			Final: "local",
		},
		Outbounds: []model.OutboundConfig{
			// 选择器出站，包含所有节点
			&model.SelectorOutbound{
				Outbound: model.Outbound{
					Type: "selector",
					Tag:  "proxy",
				},
				Outbounds: []string{"{all}"},
				Filters: []model.Filter{
					{Action: "include", Patterns: []string{""}}, // 空字符串匹配所有
				},
			},
			// URL测试出站，只包含日本节点
			&model.URLTestOutbound{
				Outbound: model.Outbound{
					Type: "urltest",
					Tag:  "auto-jp",
				},
				Outbounds: []string{"{all}"},
				Filters: []model.Filter{
					{Action: "include", Patterns: []string{"日本"}},
				},
			},
			// 直连出站
			&model.DirectOutbound{
				Outbound: model.Outbound{
					Type: "direct",
					Tag:  "direct",
				},
			},
		},
	}
}

// parseTestNodes 解析测试节点
func parseTestNodes(nodes []string) []string {
	var parsed []string
	for _, node := range nodes {
		if strings.TrimSpace(node) == "" {
			continue
		}
		
		result, err := protocol.Parse(node)
		if err != nil {
			continue // 跳过无法解析的节点
		}
		parsed = append(parsed, result)
	}
	return parsed
}

// simulateRenderTemplate 模拟RenderTemplate的逻辑，但不进行HTTP请求
func simulateRenderTemplate(config *model.Config, parsedNodes []string) error {
	// 模拟template.go中RenderTemplate方法的逻辑
	for i := range config.Outbounds {
		outbound := config.Outbounds[i]

		switch o := outbound.(type) {
		case *model.URLTestOutbound:
			outbounds := o.Outbounds
			filters := o.Filters
			if len(outbounds) == 1 && outbounds[0] == "{all}" {
				tmpNodes := filterNodes(parsedNodes, filters)
				o.Outbounds = getTags(tmpNodes)
			}
			o.Filters = nil
		case *model.SelectorOutbound:
			outbounds := o.Outbounds
			filters := o.Filters
			if len(outbounds) == 1 && outbounds[0] == "{all}" {
				tmpNodes := filterNodes(parsedNodes, filters)
				o.Outbounds = getTags(tmpNodes)
			}
			o.Filters = nil
		}
	}

	// 清除订阅配置
	config.Subs = nil

	// 合并所有节点到Outbounds
	for _, nodeJSON := range parsedNodes {
		outbound := model.NewOutbound(getNodeType(nodeJSON))
		if outbound == nil {
			continue
		}

		err := json.Unmarshal([]byte(nodeJSON), &outbound)
		if err != nil {
			return err
		}
		if unvalidNode(outbound.GetTag()) {
			continue
		}
		config.Outbounds = append(config.Outbounds, outbound)
	}
	return nil
}

// 辅助函数，复制自template.go的逻辑
func getTags(nodes []string) []string {
	var tags []string
	for _, nodeJSON := range nodes {
		var node model.Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			continue
		}
		tags = append(tags, node.Tag)
	}
	return tags
}

func filterNodes(nodes []string, rules []model.Filter) []string {
	var result []string
	for _, nodeJSON := range nodes {
		var node model.Outbound
		if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
			continue
		}

		shouldInclude := true
		for _, rule := range rules {
			matched := matchPatterns(node.Tag, rule.Patterns)

			switch rule.Action {
			case "include":
				if !matched {
					shouldInclude = false
				}
			case "exclude":
				if matched {
					shouldInclude = false
				}
			}

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

func matchPatterns(tag string, patterns []string) bool {
	// 如果没有模式或模式为空，匹配所有
	if len(patterns) == 0 {
		return true
	}
	
	for _, pattern := range patterns {
		if pattern == "" {
			return true // 空模式匹配所有
		}
		ps := strings.Split(pattern, "|")
		for _, p := range ps {
			if strings.Contains(tag, p) {
				return true
			}
		}
	}
	return false
}

func getNodeType(node string) string {
	switch {
	case strings.Contains(node, "shadowsocks"):
		return "shadowsocks"
	case strings.Contains(node, "shadowsocksr"):
		return "shadowsocksr"
	case strings.Contains(node, "hysteria2"):
		return "hysteria2"
	case strings.Contains(node, "trojan"):
		return "trojan"
	case strings.Contains(node, "selector"):
		return "selector"
	case strings.Contains(node, "urltest"):
		return "urltest"
	default:
		return ""
	}
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

// validateGeneratedConfig 验证生成的配置
func validateGeneratedConfig(t *testing.T, config *model.Config) {
	// 验证outbounds数量（原有3个 + 解析出的节点）
	expectedMinOutbounds := 3 // 至少包含原有的3个出站
	if len(config.Outbounds) < expectedMinOutbounds {
		t.Errorf("Outbounds数量不足，期望至少%d个，实际%d个", expectedMinOutbounds, len(config.Outbounds))
	}

	// 统计不同类型的节点
	var trojanCount, selectorCount, urltestCount, directCount int
	var trojanNodes []model.OutboundConfig

	for _, outbound := range config.Outbounds {
		switch outbound.GetType() {
		case "trojan":
			trojanCount++
			trojanNodes = append(trojanNodes, outbound)
		case "selector":
			selectorCount++
		case "urltest":
			urltestCount++
		case "direct":
			directCount++
		}
	}

	// 验证各类型节点数量
	if trojanCount == 0 {
		t.Error("没有找到trojan节点")
	}
	if selectorCount != 1 {
		t.Errorf("selector节点数量错误，期望1个，实际%d个", selectorCount)
	}
	if urltestCount != 1 {
		t.Errorf("urltest节点数量错误，期望1个，实际%d个", urltestCount)
	}
	if directCount != 1 {
		t.Errorf("direct节点数量错误，期望1个，实际%d个", directCount)
	}

	// 验证trojan节点的具体配置
	for _, node := range trojanNodes {
		trojanConfig, ok := node.(*model.TrojanConfig)
		if !ok {
			t.Error("trojan节点类型断言失败")
			continue
		}

		// 验证必要字段
		if trojanConfig.Server == "" {
			t.Error("trojan节点缺少server字段")
		}
		if trojanConfig.ServerPort == 0 {
			t.Error("trojan节点缺少server_port字段")
		}
		if trojanConfig.Password == "" {
			t.Error("trojan节点缺少password字段")
		}
		if trojanConfig.TLS == nil {
			t.Error("trojan节点缺少TLS配置")
		} else if !trojanConfig.TLS.Enabled {
			t.Error("trojan节点TLS应该被启用")
		}

		// 验证节点有效性
		if err := trojanConfig.Validate(); err != nil {
			t.Errorf("trojan节点验证失败: %v", err)
		}
	}

	// 验证过滤器已被清理
	for _, outbound := range config.Outbounds {
		switch o := outbound.(type) {
		case *model.SelectorOutbound:
			if o.Filters != nil {
				t.Error("SelectorOutbound的Filters应该被清理")
			}
			if len(o.Outbounds) == 0 {
				t.Errorf("SelectorOutbound应该包含解析后的节点，当前outbounds数量: %d", len(o.Outbounds))
			}
			// 调试输出
			t.Logf("SelectorOutbound outbounds: %v", o.Outbounds)
		case *model.URLTestOutbound:
			if o.Filters != nil {
				t.Error("URLTestOutbound的Filters应该被清理")
			}
		}
	}
}

// testConfigOutput 测试配置文件输出
func testConfigOutput(t *testing.T, config *model.Config) {
	// 测试JSON序列化
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}

	// 验证生成的JSON是否有效
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("生成的JSON无效: %v", err)
	}

	// 验证关键字段存在
	expectedFields := []string{"log", "dns", "outbounds"}
	for _, field := range expectedFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("生成的配置缺少字段: %s", field)
		}
	}

	// 测试文件输出
	testFile := "test_trojan_config.json"
	err = config.SingboxConfig(testFile)
	if err != nil {
		t.Fatalf("配置文件输出失败: %v", err)
	}

	// 验证文件是否存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("配置文件未正确创建")
	} else {
		// 清理测试文件
		os.Remove(testFile)
	}
}

// TestTrojanParserRegistration 测试trojan解析器注册
func TestTrojanParserRegistration(t *testing.T) {
	testURL := "trojan://testpass@example.com:443#测试"
	
	// 通过protocol.Parse测试解析器注册
	result, err := protocol.Parse(testURL)
	if err != nil {
		t.Fatalf("trojan解析器未正确注册: %v", err)
	}

	if result == "" {
		t.Error("trojan解析器返回空结果")
	}

	// 验证返回的JSON
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("trojan解析器返回无效JSON: %v", err)
	}

	if parsed["type"] != "trojan" {
		t.Errorf("解析器返回错误的类型: %v", parsed["type"])
	}
}

// TestTrojanWithOtherProtocols 测试trojan与其他协议的兼容性
func TestTrojanWithOtherProtocols(t *testing.T) {
	mixedNodes := []string{
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@1.2.3.4:8388#SS节点",
		"trojan://password@example.com:443#Trojan节点",
		"ssr://base64encoded#SSR节点",
		"hysteria2://password@server.com:8443#HY2节点",
	}

	var results []string
	for _, node := range mixedNodes {
		if strings.TrimSpace(node) == "" {
			continue
		}
		
		result, err := protocol.Parse(node)
		if err == nil {
			results = append(results, result)
		}
	}

	// 验证至少解析出trojan节点
	var hasTrojan bool
	for _, result := range results {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err == nil {
			if parsed["type"] == "trojan" {
				hasTrojan = true
				break
			}
		}
	}

	if !hasTrojan {
		t.Error("混合协议测试中未找到trojan节点")
	}
}