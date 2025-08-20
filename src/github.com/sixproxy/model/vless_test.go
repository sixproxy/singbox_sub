package model

import (
	"encoding/json"
	"singbox_sub/src/github.com/sixproxy/constant"
	"testing"
)

func TestNewVlessOutbound(t *testing.T) {
	config := NewVlessOutbound("test-vless", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "")
	
	if config.Tag != "test-vless" {
		t.Errorf("expected tag 'test-vless', got %s", config.Tag)
	}
	if config.Type != constant.OUTBOUND_VLESS {
		t.Errorf("expected type '%s', got %s", constant.OUTBOUND_VLESS, config.Type)
	}
	if config.Server != "example.com" {
		t.Errorf("expected server 'example.com', got %s", config.Server)
	}
	if config.ServerPort != 443 {
		t.Errorf("expected port 443, got %d", config.ServerPort)
	}
	if config.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", config.UUID)
	}
	if config.Detour != nil {
		t.Error("Detour应该为nil")
	}
}

func TestVlessConfig_SetReality(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetReality("publickey123", "shortid123", "www.microsoft.com")
	
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
	}
	if !config.TLS.Enabled {
		t.Error("TLS应该被启用")
	}
	if config.TLS.ServerName != "www.microsoft.com" {
		t.Errorf("expected SNI 'www.microsoft.com', got %s", config.TLS.ServerName)
	}
	
	if config.TLS.Reality == nil {
		t.Fatal("Reality配置不能为空")
	}
	if !config.TLS.Reality.Enabled {
		t.Error("Reality应该被启用")
	}
	if config.TLS.Reality.PublicKey != "publickey123" {
		t.Errorf("expected public key 'publickey123', got %s", config.TLS.Reality.PublicKey)
	}
	if config.TLS.Reality.ShortID != "shortid123" {
		t.Errorf("expected short ID 'shortid123', got %s", config.TLS.Reality.ShortID)
	}
}

func TestVlessConfig_SetTransport(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetTransport("ws", "/ws", "cdn.example.com")
	
	if config.Transport == nil {
		t.Fatal("Transport配置不能为空")
	}
	if config.Transport.Type != "ws" {
		t.Errorf("expected transport type 'ws', got %s", config.Transport.Type)
	}
	if config.Transport.Path != "/ws" {
		t.Errorf("expected path '/ws', got %s", config.Transport.Path)
	}
	if config.Transport.Host != "cdn.example.com" {
		t.Errorf("expected host 'cdn.example.com', got %s", config.Transport.Host)
	}
}

func TestVlessConfig_SetTLS(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetTLS("example.com", false)
	
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
	}
	if !config.TLS.Enabled {
		t.Error("TLS应该被启用")
	}
	if config.TLS.ServerName != "example.com" {
		t.Errorf("expected SNI 'example.com', got %s", config.TLS.ServerName)
	}
	if config.TLS.Insecure {
		t.Error("Insecure应该为false")
	}
}

func TestVlessConfig_SetUTLS(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetUTLS("chrome")
	
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
	}
	if config.TLS.UTLS == nil {
		t.Fatal("uTLS配置不能为空")
	}
	if !config.TLS.UTLS.Enabled {
		t.Error("uTLS应该被启用")
	}
	if config.TLS.UTLS.Fingerprint != "chrome" {
		t.Errorf("expected fingerprint 'chrome', got %s", config.TLS.UTLS.Fingerprint)
	}
}

func TestVlessConfig_SetFlow(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetFlow("xtls-rprx-vision")
	
	if config.Flow != "xtls-rprx-vision" {
		t.Errorf("expected flow 'xtls-rprx-vision', got %s", config.Flow)
	}
}

func TestVlessConfig_SetPacketEncoding(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "uuid", "")
	config.SetPacketEncoding("xudp")
	
	if config.PacketEncoding != "xudp" {
		t.Errorf("expected packet encoding 'xudp', got %s", config.PacketEncoding)
	}
}

func TestVlessConfig_Validate(t *testing.T) {
	// 有效配置
	validConfig := NewVlessOutbound("test", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "")
	if err := validConfig.Validate(); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}
	
	// 无效配置测试
	testCases := []struct {
		name   string
		config *VlessConfig
	}{
		{
			name:   "空服务器",
			config: NewVlessOutbound("test", "", 443, "uuid", ""),
		},
		{
			name:   "无效端口-负数",
			config: NewVlessOutbound("test", "example.com", -1, "uuid", ""),
		},
		{
			name:   "无效端口-过大",
			config: NewVlessOutbound("test", "example.com", 65536, "uuid", ""),
		},
		{
			name:   "空UUID",
			config: NewVlessOutbound("test", "example.com", 443, "", ""),
		},
		{
			name:   "无效UUID格式",
			config: NewVlessOutbound("test", "example.com", 443, "invalid-uuid", ""),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); err == nil {
				t.Errorf("期望验证失败，但通过了: %s", tc.name)
			}
		})
	}
}

func TestVlessConfig_ValidateFlow(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "")
	
	// 有效的flow参数
	validFlows := []string{"", "xtls-rprx-vision", "xtls-rprx-vision-udp443"}
	for _, flow := range validFlows {
		config.SetFlow(flow)
		if err := config.Validate(); err != nil {
			t.Errorf("有效flow '%s' 验证失败: %v", flow, err)
		}
	}
	
	// 无效的flow参数
	config.SetFlow("invalid-flow")
	if err := config.Validate(); err == nil {
		t.Error("期望无效flow验证失败，但通过了")
	}
}

func TestVlessConfig_ValidateReality(t *testing.T) {
	config := NewVlessOutbound("test", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "")
	
	// Reality缺少public key
	config.TLS = &VlessTLSConfig{
		Enabled: true,
		Reality: &RealityConfig{
			Enabled:   true,
			PublicKey: "",
			ShortID:   "shortid",
		},
	}
	if err := config.Validate(); err == nil {
		t.Error("期望Reality缺少public key验证失败，但通过了")
	}
	
	// Reality缺少short ID
	config.TLS.Reality.PublicKey = "publickey"
	config.TLS.Reality.ShortID = ""
	if err := config.Validate(); err == nil {
		t.Error("期望Reality缺少short ID验证失败，但通过了")
	}
	
	// 完整的Reality配置
	config.TLS.Reality.ShortID = "shortid"
	if err := config.Validate(); err != nil {
		t.Errorf("完整Reality配置验证失败: %v", err)
	}
}

func TestVlessConfig_MarshalJSON(t *testing.T) {
	// 基本配置
	config := NewVlessOutbound("test-vless", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "")
	
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}
	
	// 验证JSON结构
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}
	
	// 验证基本字段
	if result["tag"] != "test-vless" {
		t.Errorf("expected tag 'test-vless', got %v", result["tag"])
	}
	if result["type"] != constant.OUTBOUND_VLESS {
		t.Errorf("expected type '%s', got %v", constant.OUTBOUND_VLESS, result["type"])
	}
	if result["server"] != "example.com" {
		t.Errorf("expected server 'example.com', got %v", result["server"])
	}
	if result["server_port"] != float64(443) {
		t.Errorf("expected port 443, got %v", result["server_port"])
	}
	if result["uuid"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %v", result["uuid"])
	}
	
	// 空字段应该不存在
	if _, exists := result["flow"]; exists {
		t.Error("空flow字段不应该存在于JSON中")
	}
	if _, exists := result["packet_encoding"]; exists {
		t.Error("空packet_encoding字段不应该存在于JSON中")
	}
	if _, exists := result["transport"]; exists {
		t.Error("空transport字段不应该存在于JSON中")
	}
	if _, exists := result["tls"]; exists {
		t.Error("空tls字段不应该存在于JSON中")
	}
}

func TestVlessConfig_MarshalJSON_WithAllFields(t *testing.T) {
	// 完整配置
	config := NewVlessOutbound("test-vless", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "xtls-rprx-vision")
	config.SetPacketEncoding("xudp")
	config.SetTransport("ws", "/ws", "cdn.example.com")
	config.SetReality("publickey123", "shortid123", "www.microsoft.com")
	config.SetUTLS("chrome")
	
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}
	
	// 验证JSON结构
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}
	
	// 验证所有字段都存在
	if result["flow"] != "xtls-rprx-vision" {
		t.Errorf("expected flow 'xtls-rprx-vision', got %v", result["flow"])
	}
	if result["packet_encoding"] != "xudp" {
		t.Errorf("expected packet_encoding 'xudp', got %v", result["packet_encoding"])
	}
	if result["transport"] == nil {
		t.Error("transport字段应该存在")
	}
	if result["tls"] == nil {
		t.Error("tls字段应该存在")
	}
}

func TestVlessConfig_ChainedConfiguration(t *testing.T) {
	// 测试链式配置调用
	config := NewVlessOutbound("test", "example.com", 443, "550e8400-e29b-41d4-a716-446655440000", "").
		SetFlow("xtls-rprx-vision").
		SetPacketEncoding("xudp").
		SetTransport("ws", "/ws", "cdn.example.com").
		SetTLS("example.com", false).
		SetUTLS("chrome")
	
	// 验证所有配置都正确设置
	if config.Flow != "xtls-rprx-vision" {
		t.Errorf("expected flow 'xtls-rprx-vision', got %s", config.Flow)
	}
	if config.PacketEncoding != "xudp" {
		t.Errorf("expected packet encoding 'xudp', got %s", config.PacketEncoding)
	}
	if config.Transport == nil || config.Transport.Type != "ws" {
		t.Error("Transport配置不正确")
	}
	if config.TLS == nil || !config.TLS.Enabled {
		t.Error("TLS配置不正确")
	}
	if config.TLS.UTLS == nil || !config.TLS.UTLS.Enabled {
		t.Error("uTLS配置不正确")
	}
}