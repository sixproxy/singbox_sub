package model

import (
	"encoding/json"
	"testing"
)

func TestAnytlsConfig_SetData(t *testing.T) {
	tests := []struct {
		name                       string
		anytlsURL                  string
		expectTag                  string
		expectServer               string
		expectPort                 int
		expectPassword             string
		expectTLSEnabled           bool
		expectServerName           string
		expectInsecure             bool
		expectIdleCheckInterval    string
		expectIdleTimeout          string
		expectMinIdleSession       int
		expectError                bool
	}{
		{
			name:                    "基本anytls URL",
			anytlsURL:               "anytls://password123@example.com:443#测试节点",
			expectTag:               "测试节点",
			expectServer:            "example.com",
			expectPort:              443,
			expectPassword:          "password123",
			expectTLSEnabled:        true,
			expectServerName:        "example.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "带SNI参数的anytls URL",
			anytlsURL:               "anytls://mypassword@server.example.com:8443?sni=custom.domain.com#节点名称",
			expectTag:               "节点名称",
			expectServer:            "server.example.com",
			expectPort:              8443,
			expectPassword:          "mypassword",
			expectTLSEnabled:        true,
			expectServerName:        "custom.domain.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "带server_name参数的anytls URL",
			anytlsURL:               "anytls://testpass@192.168.1.100:8080?server_name=server.domain.com#IP节点",
			expectTag:               "IP节点",
			expectServer:            "192.168.1.100",
			expectPort:              8080,
			expectPassword:          "testpass",
			expectTLSEnabled:        true,
			expectServerName:        "server.domain.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "允许不安全连接的anytls URL",
			anytlsURL:               "anytls://secret@insecure.example.com:443?insecure=1#不安全节点",
			expectTag:               "不安全节点",
			expectServer:            "insecure.example.com",
			expectPort:              443,
			expectPassword:          "secret",
			expectTLSEnabled:        true,
			expectServerName:        "insecure.example.com",
			expectInsecure:          true,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "带ALPN参数的anytls URL",
			anytlsURL:               "anytls://alpnpass@alpn.example.com:443?alpn=h2,http/1.1#ALPN节点",
			expectTag:               "ALPN节点",
			expectServer:            "alpn.example.com",
			expectPort:              443,
			expectPassword:          "alpnpass",
			expectTLSEnabled:        true,
			expectServerName:        "alpn.example.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "带会话管理参数的anytls URL",
			anytlsURL:               "anytls://sessionpass@session.example.com:443?check_interval=60s&idle_timeout=120s&min_idle=5#会话节点",
			expectTag:               "会话节点",
			expectServer:            "session.example.com",
			expectPort:              443,
			expectPassword:          "sessionpass",
			expectTLSEnabled:        true,
			expectServerName:        "session.example.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "60s",
			expectIdleTimeout:       "120s",
			expectMinIdleSession:    5,
			expectError:             false,
		},
		{
			name:                    "不带端口的anytls URL（使用默认443端口）",
			anytlsURL:               "anytls://defaultport@default.example.com#默认端口",
			expectTag:               "默认端口",
			expectServer:            "default.example.com",
			expectPort:              443,
			expectPassword:          "defaultport",
			expectTLSEnabled:        true,
			expectServerName:        "default.example.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:                    "用户信息包含密码的URL",
			anytlsURL:               "anytls://user:pass123@userpass.example.com:8080#用户密码",
			expectTag:               "用户密码",
			expectServer:            "userpass.example.com",
			expectPort:              8080,
			expectPassword:          "pass123",
			expectTLSEnabled:        true,
			expectServerName:        "userpass.example.com",
			expectInsecure:          false,
			expectIdleCheckInterval: "30s",
			expectIdleTimeout:       "30s",
			expectMinIdleSession:    0,
			expectError:             false,
		},
		{
			name:        "缺少密码的无效URL",
			anytlsURL:   "anytls://@example.com:443#无密码",
			expectError: true,
		},
		{
			name:        "缺少服务器的无效URL",
			anytlsURL:   "anytls://password@:443#无服务器",
			expectError: true,
		},
		{
			name:        "无效端口的URL",
			anytlsURL:   "anytls://password@example.com:invalid#无效端口",
			expectError: true,
		},
		{
			name:        "端口超出范围的URL",
			anytlsURL:   "anytls://password@example.com:99999#端口超范围",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AnytlsConfig{}
			err := config.SetData(tt.anytlsURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("意外的错误: %v", err)
				return
			}

			// 验证基本字段
			if config.Tag != tt.expectTag {
				t.Errorf("Tag = %v, 期望 %v", config.Tag, tt.expectTag)
			}
			if config.Server != tt.expectServer {
				t.Errorf("Server = %v, 期望 %v", config.Server, tt.expectServer)
			}
			if config.ServerPort != tt.expectPort {
				t.Errorf("ServerPort = %v, 期望 %v", config.ServerPort, tt.expectPort)
			}
			if config.Password != tt.expectPassword {
				t.Errorf("Password = %v, 期望 %v", config.Password, tt.expectPassword)
			}
			if config.Type != "anytls" {
				t.Errorf("Type = %v, 期望 anytls", config.Type)
			}

			// 验证TLS配置
			if config.TLS.Enabled != tt.expectTLSEnabled {
				t.Errorf("TLS.Enabled = %v, 期望 %v", config.TLS.Enabled, tt.expectTLSEnabled)
			}
			if config.TLS.ServerName != tt.expectServerName {
				t.Errorf("TLS.ServerName = %v, 期望 %v", config.TLS.ServerName, tt.expectServerName)
			}
			if config.TLS.Insecure != tt.expectInsecure {
				t.Errorf("TLS.Insecure = %v, 期望 %v", config.TLS.Insecure, tt.expectInsecure)
			}

			// 验证会话管理参数
			if config.IdleSessionCheckInterval != tt.expectIdleCheckInterval {
				t.Errorf("IdleSessionCheckInterval = %v, 期望 %v", config.IdleSessionCheckInterval, tt.expectIdleCheckInterval)
			}
			if config.IdleSessionTimeout != tt.expectIdleTimeout {
				t.Errorf("IdleSessionTimeout = %v, 期望 %v", config.IdleSessionTimeout, tt.expectIdleTimeout)
			}
			if config.MinIdleSession != tt.expectMinIdleSession {
				t.Errorf("MinIdleSession = %v, 期望 %v", config.MinIdleSession, tt.expectMinIdleSession)
			}
		})
	}
}

func TestAnytlsConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected map[string]interface{}
	}{
		{
			name: "基本配置JSON序列化",
			url:  "anytls://testpassword@example.com:443?sni=test.com#测试节点",
			expected: map[string]interface{}{
				"tag":         "测试节点",
				"type":        "anytls",
				"server":      "example.com",
				"server_port": float64(443),
				"password":    "testpassword",
			},
		},
		{
			name: "包含会话管理参数的配置",
			url:  "anytls://sessionpass@session.com:8080?check_interval=60s&idle_timeout=120s&min_idle=3#会话测试",
			expected: map[string]interface{}{
				"tag":                         "会话测试",
				"type":                        "anytls",
				"server":                      "session.com",
				"server_port":                 float64(8080),
				"password":                    "sessionpass",
				"idle_session_check_interval": "60s",
				"idle_session_timeout":        "120s",
				"min_idle_session":            float64(3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AnytlsConfig{}
			err := config.SetData(tt.url)
			if err != nil {
				t.Fatalf("SetData失败: %v", err)
			}

			jsonStr := config.String()
			if jsonStr == "" {
				t.Error("String()返回空字符串")
			}

			// 验证生成的JSON是否有效
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(jsonStr), &parsed)
			if err != nil {
				t.Errorf("生成的JSON无效: %v", err)
			}

			// 验证关键字段是否存在且正确
			for key, expectedValue := range tt.expected {
				if actualValue, exists := parsed[key]; !exists {
					t.Errorf("缺少字段: %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("字段 %s = %v, 期望 %v", key, actualValue, expectedValue)
				}
			}

			// 验证TLS配置存在
			if _, exists := parsed["tls"]; !exists {
				t.Error("缺少tls字段")
			}
			
			// 验证detour不存在（AnyTLS是独立代理协议）
			if _, exists := parsed["detour"]; exists {
				t.Error("AnyTLS不应该有detour字段")
			}
		})
	}
}

func TestAnytlsConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      AnytlsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效配置",
			config: AnytlsConfig{
				Server:     "example.com",
				ServerPort: 443,
				Password:   "password123",
			},
			expectError: false,
		},
		{
			name: "缺少服务器",
			config: AnytlsConfig{
				ServerPort: 443,
				Password:   "password123",
			},
			expectError: true,
			errorMsg:    "server is required",
		},
		{
			name: "端口为0",
			config: AnytlsConfig{
				Server:   "example.com",
				Password: "password123",
			},
			expectError: true,
			errorMsg:    "invalid server port: 0",
		},
		{
			name: "端口超出范围",
			config: AnytlsConfig{
				Server:     "example.com",
				ServerPort: 70000,
				Password:   "password123",
			},
			expectError: true,
			errorMsg:    "invalid server port: 70000",
		},
		{
			name: "负端口",
			config: AnytlsConfig{
				Server:     "example.com",
				ServerPort: -1,
				Password:   "password123",
			},
			expectError: true,
			errorMsg:    "invalid server port: -1",
		},
		{
			name: "缺少密码",
			config: AnytlsConfig{
				Server:     "example.com",
				ServerPort: 443,
			},
			expectError: true,
			errorMsg:    "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("期望错误，但验证通过")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("错误消息 = %v, 期望 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("意外的验证错误: %v", err)
				}
			}
		})
	}
}

func TestAnytlsConfig_Interface(t *testing.T) {
	config := &AnytlsConfig{
		Outbound: Outbound{
			Tag:  "test-tag",
			Type: "anytls",
		},
	}

	// 验证接口方法
	if config.GetTag() != "test-tag" {
		t.Errorf("GetTag() = %v, 期望 test-tag", config.GetTag())
	}
	if config.GetType() != "anytls" {
		t.Errorf("GetType() = %v, 期望 anytls", config.GetType())
	}

	// 验证实现了OutboundConfig接口
	var _ OutboundConfig = config
}

func TestAnytlsConfig_ALPNParsing(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedALPN []string
	}{
		{
			name:         "单个ALPN协议",
			url:          "anytls://pass@example.com:443?alpn=h2#test",
			expectedALPN: []string{"h2"},
		},
		{
			name:         "多个ALPN协议",
			url:          "anytls://pass@example.com:443?alpn=h2,http/1.1#test",
			expectedALPN: []string{"h2", "http/1.1"},
		},
		{
			name:         "没有ALPN参数",
			url:          "anytls://pass@example.com:443#test",
			expectedALPN: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AnytlsConfig{}
			err := config.SetData(tt.url)
			if err != nil {
				t.Fatalf("SetData失败: %v", err)
			}

			if len(config.TLS.Alpn) != len(tt.expectedALPN) {
				t.Errorf("ALPN长度 = %d, 期望 %d", len(config.TLS.Alpn), len(tt.expectedALPN))
				return
			}

			for i, expected := range tt.expectedALPN {
				if i >= len(config.TLS.Alpn) || config.TLS.Alpn[i] != expected {
					t.Errorf("ALPN[%d] = %v, 期望 %v", i, config.TLS.Alpn[i], expected)
				}
			}
		})
	}
}

func TestAnytlsConfig_DefaultValues(t *testing.T) {
	config := &AnytlsConfig{}
	err := config.SetData("anytls://testpass@example.com#test")
	if err != nil {
		t.Fatalf("SetData失败: %v", err)
	}

	// 验证默认值
	expectedDefaults := map[string]interface{}{
		"ServerPort":                443,
		"IdleSessionCheckInterval":  "30s",
		"IdleSessionTimeout":        "30s",
		"MinIdleSession":            0,
		"TLS.Enabled":               true,
		"TLS.Insecure":              false,
		"TLS.ServerName":            "example.com",
	}

	if config.ServerPort != expectedDefaults["ServerPort"].(int) {
		t.Errorf("ServerPort默认值 = %v, 期望 %v", config.ServerPort, expectedDefaults["ServerPort"])
	}
	if config.IdleSessionCheckInterval != expectedDefaults["IdleSessionCheckInterval"].(string) {
		t.Errorf("IdleSessionCheckInterval默认值 = %v, 期望 %v", config.IdleSessionCheckInterval, expectedDefaults["IdleSessionCheckInterval"])
	}
	if config.IdleSessionTimeout != expectedDefaults["IdleSessionTimeout"].(string) {
		t.Errorf("IdleSessionTimeout默认值 = %v, 期望 %v", config.IdleSessionTimeout, expectedDefaults["IdleSessionTimeout"])
	}
	if config.MinIdleSession != expectedDefaults["MinIdleSession"].(int) {
		t.Errorf("MinIdleSession默认值 = %v, 期望 %v", config.MinIdleSession, expectedDefaults["MinIdleSession"])
	}
	if config.TLS.Enabled != expectedDefaults["TLS.Enabled"].(bool) {
		t.Errorf("TLS.Enabled默认值 = %v, 期望 %v", config.TLS.Enabled, expectedDefaults["TLS.Enabled"])
	}
	if config.TLS.Insecure != expectedDefaults["TLS.Insecure"].(bool) {
		t.Errorf("TLS.Insecure默认值 = %v, 期望 %v", config.TLS.Insecure, expectedDefaults["TLS.Insecure"])
	}
	if config.TLS.ServerName != expectedDefaults["TLS.ServerName"].(string) {
		t.Errorf("TLS.ServerName默认值 = %v, 期望 %v", config.TLS.ServerName, expectedDefaults["TLS.ServerName"])
	}
}