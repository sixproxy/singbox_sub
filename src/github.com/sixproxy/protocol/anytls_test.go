package protocol

import (
	"encoding/json"
	"testing"
)

func TestAnytlsParser_Proto(t *testing.T) {
	parser := &AnytlsParser{}
	if parser.Proto() != "anytls" {
		t.Errorf("Proto() = %v, 期望 anytls", parser.Proto())
	}
}

func TestAnytlsParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(t *testing.T, result string)
	}{
		{
			name:        "有效的anytls URL",
			input:       "anytls://password123@example.com:443#测试节点",
			expectError: false,
			validate: func(t *testing.T, result string) {
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				if err != nil {
					t.Errorf("解析结果JSON失败: %v", err)
					return
				}

				// 验证关键字段
				expectedFields := map[string]interface{}{
					"tag":         "测试节点",
					"type":        "anytls",
					"server":      "example.com",
					"server_port": float64(443),
					"password":    "password123",
				}

				for key, expected := range expectedFields {
					if actual, exists := parsed[key]; !exists {
						t.Errorf("缺少字段: %s", key)
					} else if actual != expected {
						t.Errorf("字段 %s = %v, 期望 %v", key, actual, expected)
					}
				}

				// 验证TLS配置存在
				if _, exists := parsed["tls"]; !exists {
					t.Error("缺少tls字段")
				}
			},
		},
		{
			name:        "带完整参数的anytls URL",
			input:       "anytls://user:pass@server.com:8080?sni=custom.com&insecure=1&alpn=h2&check_interval=60s&idle_timeout=120s&min_idle=5#完整节点",
			expectError: false,
			validate: func(t *testing.T, result string) {
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				if err != nil {
					t.Errorf("解析结果JSON失败: %v", err)
					return
				}

				// 验证扩展字段
				expectedFields := map[string]interface{}{
					"tag":                         "完整节点",
					"type":                        "anytls",
					"server":                      "server.com",
					"server_port":                 float64(8080),
					"password":                    "pass",
					"idle_session_check_interval": "60s",
					"idle_session_timeout":        "120s",
					"min_idle_session":            float64(5),
				}

				for key, expected := range expectedFields {
					if actual, exists := parsed[key]; !exists {
						t.Errorf("缺少字段: %s", key)
					} else if actual != expected {
						t.Errorf("字段 %s = %v, 期望 %v", key, actual, expected)
					}
				}

				// 验证TLS配置
				if tlsConfig, exists := parsed["tls"].(map[string]interface{}); exists {
					if tlsConfig["server_name"] != "custom.com" {
						t.Errorf("TLS server_name = %v, 期望 custom.com", tlsConfig["server_name"])
					}
					if tlsConfig["insecure"] != true {
						t.Errorf("TLS insecure = %v, 期望 true", tlsConfig["insecure"])
					}
					if alpn, exists := tlsConfig["alpn"].([]interface{}); exists {
						if len(alpn) != 1 || alpn[0] != "h2" {
							t.Errorf("TLS alpn = %v, 期望 [h2]", alpn)
						}
					} else {
						t.Error("TLS配置中缺少alpn字段")
					}
				} else {
					t.Error("TLS配置解析失败")
				}
			},
		},
		{
			name:        "默认端口的anytls URL",
			input:       "anytls://defaultpass@default.example.com#默认端口",
			expectError: false,
			validate: func(t *testing.T, result string) {
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				if err != nil {
					t.Errorf("解析结果JSON失败: %v", err)
					return
				}

				if parsed["server_port"] != float64(443) {
					t.Errorf("server_port = %v, 期望 443", parsed["server_port"])
				}
			},
		},
		{
			name:        "多个ALPN协议",
			input:       "anytls://pass@example.com:443?alpn=h2,http/1.1#ALPN测试",
			expectError: false,
			validate: func(t *testing.T, result string) {
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				if err != nil {
					t.Errorf("解析结果JSON失败: %v", err)
					return
				}

				if tlsConfig, exists := parsed["tls"].(map[string]interface{}); exists {
					if alpn, exists := tlsConfig["alpn"].([]interface{}); exists {
						expectedALPN := []string{"h2", "http/1.1"}
						if len(alpn) != len(expectedALPN) {
							t.Errorf("ALPN长度 = %d, 期望 %d", len(alpn), len(expectedALPN))
							return
						}
						for i, expected := range expectedALPN {
							if alpn[i] != expected {
								t.Errorf("ALPN[%d] = %v, 期望 %v", i, alpn[i], expected)
							}
						}
					} else {
						t.Error("TLS配置中缺少alpn字段")
					}
				} else {
					t.Error("TLS配置解析失败")
				}
			},
		},
		{
			name:        "无效URL - 缺少密码",
			input:       "anytls://@example.com:443#无密码",
			expectError: true,
		},
		{
			name:        "无效URL - 缺少服务器",
			input:       "anytls://password@:443#无服务器",
			expectError: true,
		},
		{
			name:        "无效URL - 格式错误",
			input:       "invalid-url",
			expectError: true,
		},
		{
			name:        "无效URL - 无效端口",
			input:       "anytls://password@example.com:invalid#无效端口",
			expectError: true,
		},
		{
			name:        "无效URL - 端口超出范围",
			input:       "anytls://password@example.com:99999#端口超范围",
			expectError: true,
		},
	}

	parser := &AnytlsParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("期望错误，但解析成功")
				}
				return
			}

			if err != nil {
				t.Errorf("意外的解析错误: %v", err)
				return
			}

			if result == "" {
				t.Error("解析结果为空字符串")
				return
			}

			// 验证结果是有效的JSON
			var testJSON map[string]interface{}
			if err := json.Unmarshal([]byte(result), &testJSON); err != nil {
				t.Errorf("解析结果不是有效的JSON: %v", err)
				return
			}

			// 运行自定义验证
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestAnytlsParser_ParseSession(t *testing.T) {
	// 测试解析会话是否正确创建和使用
	session := &AnytlsParseSession{}
	
	result, err := session.parse("anytls://testpass@test.com:443#会话测试")
	if err != nil {
		t.Errorf("会话解析失败: %v", err)
	}

	if result == "" {
		t.Error("会话解析结果为空")
	}

	// 验证配置对象是否正确设置
	if session.Config.Tag != "会话测试" {
		t.Errorf("会话配置Tag = %v, 期望 会话测试", session.Config.Tag)
	}
	if session.Config.Server != "test.com" {
		t.Errorf("会话配置Server = %v, 期望 test.com", session.Config.Server)
	}
}

func TestAnytlsParser_RegistrationInParsers(t *testing.T) {
	// 测试解析器是否正确注册到parsers映射中
	if parser, exists := parsers["anytls"]; !exists {
		t.Error("anytls解析器未注册到parsers映射中")
	} else {
		if anytlsParser, ok := parser.(*AnytlsParser); !ok {
			t.Error("anytls解析器类型错误")
		} else if anytlsParser.Proto() != "anytls" {
			t.Errorf("注册的解析器协议 = %v, 期望 anytls", anytlsParser.Proto())
		}
	}
}

func TestAnytlsParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "空字符串",
			input:       "",
			expectError: true,
			description: "空输入应该返回错误",
		},
		{
			name:        "只有协议名",
			input:       "anytls://",
			expectError: true,
			description: "不完整的URL应该返回错误",
		},
		{
			name:        "特殊字符在密码中",
			input:       "anytls://p@ssw0rd!@example.com:443#特殊字符",
			expectError: false,
			description: "密码中的特殊字符应该正常处理",
		},
		{
			name:        "IPv6地址",
			input:       "anytls://password@[::1]:8080#IPv6",
			expectError: false,
			description: "IPv6地址应该正常处理",
		},
		{
			name:        "长标签名",
			input:       "anytls://password@example.com:443#这是一个非常长的节点标签名称用于测试解析器是否能正确处理长标签",
			expectError: false,
			description: "长标签名应该正常处理",
		},
	}

	parser := &AnytlsParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("%s: 期望错误但解析成功", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: 意外的解析错误: %v", tt.description, err)
			}
		})
	}
}