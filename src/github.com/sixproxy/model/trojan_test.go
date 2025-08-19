package model

import (
	"encoding/json"
	"testing"
)

func TestTrojanConfig_SetData(t *testing.T) {
	tests := []struct {
		name        string
		trojanURL   string
		expectTag   string
		expectServer string
		expectPort   int
		expectPassword string
		expectTLSEnabled bool
		expectServerName string
		expectInsecure   bool
		expectError      bool
	}{
		{
			name:             "基本trojan URL",
			trojanURL:        "trojan://password123@example.com:443#测试节点",
			expectTag:        "测试节点",
			expectServer:     "example.com",
			expectPort:       443,
			expectPassword:   "password123",
			expectTLSEnabled: true,
			expectServerName: "example.com",
			expectInsecure:   false,
			expectError:      false,
		},
		{
			name:             "带SNI参数的trojan URL",
			trojanURL:        "trojan://mypassword@server.example.com:8443?sni=custom.domain.com#节点名称",
			expectTag:        "节点名称",
			expectServer:     "server.example.com",
			expectPort:       8443,
			expectPassword:   "mypassword",
			expectTLSEnabled: true,
			expectServerName: "custom.domain.com",
			expectInsecure:   false,
			expectError:      false,
		},
		{
			name:             "允许不安全连接的trojan URL",
			trojanURL:        "trojan://secret@insecure.example.com:443?allowInsecure=1#不安全节点",
			expectTag:        "不安全节点",
			expectServer:     "insecure.example.com",
			expectPort:       443,
			expectPassword:   "secret",
			expectTLSEnabled: true,
			expectServerName: "insecure.example.com",
			expectInsecure:   true,
			expectError:      false,
		},
		{
			name:             "使用peer参数的trojan URL",
			trojanURL:        "trojan://testpass@192.168.1.100:8080?peer=peer.domain.com#IP节点",
			expectTag:        "IP节点",
			expectServer:     "192.168.1.100",
			expectPort:       8080,
			expectPassword:   "testpass",
			expectTLSEnabled: true,
			expectServerName: "peer.domain.com",
			expectInsecure:   false,
			expectError:      false,
		},
		{
			name:             "不带端口的trojan URL（使用默认443端口）",
			trojanURL:        "trojan://defaultport@default.example.com#默认端口",
			expectTag:        "默认端口",
			expectServer:     "default.example.com",
			expectPort:       443,
			expectPassword:   "defaultport",
			expectTLSEnabled: true,
			expectServerName: "default.example.com",
			expectInsecure:   false,
			expectError:      false,
		},
		{
			name:        "缺少密码的无效URL",
			trojanURL:   "trojan://@example.com:443#无密码",
			expectError: true,
		},
		{
			name:        "缺少服务器的无效URL",
			trojanURL:   "trojan://password@:443#无服务器",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TrojanConfig{}
			err := config.SetData(tt.trojanURL)

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
			if config.Type != "trojan" {
				t.Errorf("Type = %v, 期望 trojan", config.Type)
			}

			// 验证TLS配置
			if config.TLS == nil {
				t.Errorf("TLS配置不应该为nil")
				return
			}
			if config.TLS.Enabled != tt.expectTLSEnabled {
				t.Errorf("TLS.Enabled = %v, 期望 %v", config.TLS.Enabled, tt.expectTLSEnabled)
			}
			if config.TLS.ServerName != tt.expectServerName {
				t.Errorf("TLS.ServerName = %v, 期望 %v", config.TLS.ServerName, tt.expectServerName)
			}
			if config.TLS.Insecure != tt.expectInsecure {
				t.Errorf("TLS.Insecure = %v, 期望 %v", config.TLS.Insecure, tt.expectInsecure)
			}
		})
	}
}

func TestTrojanConfig_String(t *testing.T) {
	config := &TrojanConfig{}
	err := config.SetData("trojan://testpassword@example.com:443?sni=test.com#测试节点")
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

	// 验证关键字段是否存在
	expectedFields := []string{"tag", "type", "server", "server_port", "password", "tls"}
	for _, field := range expectedFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("缺少字段: %s", field)
		}
	}

	// 验证type字段值
	if parsed["type"] != "trojan" {
		t.Errorf("type字段值错误: %v, 期望 trojan", parsed["type"])
	}
}

func TestTrojanConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      TrojanConfig
		expectError bool
	}{
		{
			name: "有效配置",
			config: TrojanConfig{
				Server:     "example.com",
				ServerPort: 443,
				Password:   "password123",
			},
			expectError: false,
		},
		{
			name: "缺少服务器",
			config: TrojanConfig{
				ServerPort: 443,
				Password:   "password123",
			},
			expectError: true,
		},
		{
			name: "缺少端口",
			config: TrojanConfig{
				Server:   "example.com",
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "缺少密码",
			config: TrojanConfig{
				Server:     "example.com",
				ServerPort: 443,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("期望错误，但验证通过")
			}
			if !tt.expectError && err != nil {
				t.Errorf("意外的验证错误: %v", err)
			}
		})
	}
}

func TestTrojanConfig_Interface(t *testing.T) {
	config := &TrojanConfig{
		Outbound: Outbound{
			Tag:  "test-tag",
			Type: "trojan",
		},
	}

	// 验证接口方法
	if config.GetTag() != "test-tag" {
		t.Errorf("GetTag() = %v, 期望 test-tag", config.GetTag())
	}
	if config.GetType() != "trojan" {
		t.Errorf("GetType() = %v, 期望 trojan", config.GetType())
	}

	// 验证实现了OutboundConfig接口
	var _ OutboundConfig = config
}