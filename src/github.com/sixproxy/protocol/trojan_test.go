package protocol

import (
	"encoding/json"
	"testing"
)

func TestTrojanParser_Proto(t *testing.T) {
	parser := &trojanParser{}
	if parser.Proto() != "trojan" {
		t.Errorf("Proto() = %v, 期望 trojan", parser.Proto())
	}
}

func TestTrojanParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		trojanURL   string
		expectError bool
		checkFields map[string]interface{} // 检查JSON输出中的特定字段
	}{
		{
			name:        "标准trojan URL",
			trojanURL:   "trojan://password123@example.com:443#测试节点",
			expectError: false,
			checkFields: map[string]interface{}{
				"type":        "trojan",
				"tag":         "测试节点",
				"server":      "example.com",
				"server_port": float64(443), // JSON数字会被解析为float64
				"password":    "password123",
			},
		},
		{
			name:        "带SNI的trojan URL",
			trojanURL:   "trojan://secret@server.com:8443?sni=custom.com#SNI节点",
			expectError: false,
			checkFields: map[string]interface{}{
				"type":        "trojan",
				"tag":         "SNI节点",
				"server":      "server.com",
				"server_port": float64(8443),
				"password":    "secret",
			},
		},
		{
			name:        "allowInsecure参数",
			trojanURL:   "trojan://testpass@insecure.com:443?allowInsecure=true#不安全连接",
			expectError: false,
			checkFields: map[string]interface{}{
				"type":        "trojan",
				"tag":         "不安全连接",
				"server":      "insecure.com",
				"server_port": float64(443),
				"password":    "testpass",
			},
		},
		{
			name:        "默认端口443",
			trojanURL:   "trojan://defaultport@default.com#默认端口",
			expectError: false,
			checkFields: map[string]interface{}{
				"type":        "trojan",
				"tag":         "默认端口",
				"server":      "default.com",
				"server_port": float64(443),
				"password":    "defaultport",
			},
		},
		{
			name:        "无效URL - 缺少密码",
			trojanURL:   "trojan://@example.com:443#无密码",
			expectError: true,
		},
		{
			name:        "无效URL - 缺少服务器",
			trojanURL:   "trojan://password@:443#无服务器",
			expectError: true,
		},
		{
			name:        "无效URL - 错误格式",
			trojanURL:   "trojan://invalid-format",
			expectError: true,
		},
	}

	parser := &trojanParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.trojanURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但解析成功")
				}
				return
			}

			if err != nil {
				t.Errorf("意外的错误: %v", err)
				return
			}

			if result == "" {
				t.Error("解析结果为空")
				return
			}

			// 验证返回的JSON是否有效
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(result), &parsed)
			if err != nil {
				t.Errorf("返回的JSON无效: %v", err)
				return
			}

			// 检查特定字段
			for field, expectedValue := range tt.checkFields {
				actualValue, exists := parsed[field]
				if !exists {
					t.Errorf("缺少字段: %s", field)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("字段 %s = %v, 期望 %v", field, actualValue, expectedValue)
				}
			}

			// 验证TLS配置存在且正确
			tls, exists := parsed["tls"]
			if !exists {
				t.Error("缺少TLS配置")
			} else {
				tlsMap, ok := tls.(map[string]interface{})
				if !ok {
					t.Error("TLS配置格式错误")
				} else {
					if enabled, exists := tlsMap["enabled"]; !exists || enabled != true {
						t.Error("TLS应该被启用")
					}
				}
			}
		})
	}
}

func TestTrojanParser_Registration(t *testing.T) {
	// 验证解析器是否正确注册
	result, err := Parse("trojan://testpass@example.com:443#注册测试")
	if err != nil {
		t.Errorf("通过注册解析器解析失败: %v", err)
	}
	if result == "" {
		t.Error("通过注册解析器解析结果为空")
	}

	// 验证协议识别
	proto := ProtoOf("trojan://testpass@example.com:443#注册测试")
	if proto != "trojan" {
		t.Errorf("ProtoOf() = %v, 期望 trojan", proto)
	}
}

func TestTrojanParser_EdgeCases(t *testing.T) {
	parser := &trojanParser{}

	tests := []struct {
		name      string
		trojanURL string
		shouldPass bool
	}{
		{
			name:       "IPv6地址",
			trojanURL:  "trojan://password@[2001:db8::1]:443#IPv6节点",
			shouldPass: true,
		},
		{
			name:       "复杂密码字符",
			trojanURL:  "trojan://P%40ssw0rd%21@example.com:443#复杂密码",
			shouldPass: true,
		},
		{
			name:       "URL编码的标签",
			trojanURL:  "trojan://password@example.com:443#%E4%B8%AD%E6%96%87%E8%8A%82%E7%82%B9",
			shouldPass: true,
		},
		{
			name:       "多个查询参数",
			trojanURL:  "trojan://password@example.com:443?sni=test.com&allowInsecure=1&other=value#多参数",
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.trojanURL)
			
			if tt.shouldPass {
				if err != nil {
					t.Errorf("解析失败: %v", err)
				}
				if result == "" {
					t.Error("解析结果为空")
				}
				
				// 验证JSON有效性
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(result), &parsed); err != nil {
					t.Errorf("生成的JSON无效: %v", err)
				}
			} else {
				if err == nil {
					t.Error("期望解析失败，但成功了")
				}
			}
		})
	}
}