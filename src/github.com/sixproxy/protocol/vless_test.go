package protocol

import (
	"encoding/json"
	"singbox_sub/src/github.com/sixproxy/model"
	"testing"
)

func TestVlessParser_Proto(t *testing.T) {
	parser := &VlessParser{}
	if parser.Proto() != "vless" {
		t.Errorf("expected 'vless', got %s", parser.Proto())
	}
}

func TestVlessParser_Parse_BasicTCP(t *testing.T) {
	parser := &VlessParser{}
	
	// 基本TCP连接
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=tcp&security=tls&sni=example.com#TCP-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证基本配置
	if config.Server != "example.com" {
		t.Errorf("expected server 'example.com', got %s", config.Server)
	}
	if config.ServerPort != 443 {
		t.Errorf("expected port 443, got %d", config.ServerPort)
	}
	if config.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", config.UUID)
	}
	if config.Tag != "TCP-Test" {
		t.Errorf("expected tag 'TCP-Test', got %s", config.Tag)
	}
	
	// 验证TLS配置
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
	}
	if !config.TLS.Enabled {
		t.Error("TLS应该被启用")
	}
	if config.TLS.ServerName != "example.com" {
		t.Errorf("expected SNI 'example.com', got %s", config.TLS.ServerName)
	}
}

func TestVlessParser_Parse_WebSocket(t *testing.T) {
	parser := &VlessParser{}
	
	// WebSocket传输
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=ws&path=/ws&host=cdn.example.com&security=tls&sni=example.com#WebSocket-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证传输配置
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

func TestVlessParser_Parse_Reality(t *testing.T) {
	parser := &VlessParser{}
	
	// Reality配置
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=tcp&security=reality&pbk=publickey123&sid=shortid123&sni=www.microsoft.com&fp=chrome#Reality-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证Reality配置
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
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
	if config.TLS.ServerName != "www.microsoft.com" {
		t.Errorf("expected SNI 'www.microsoft.com', got %s", config.TLS.ServerName)
	}
	
	// 验证uTLS配置
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

func TestVlessParser_Parse_gRPC(t *testing.T) {
	parser := &VlessParser{}
	
	// gRPC传输
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=grpc&serviceName=GunService&security=tls#gRPC-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证gRPC传输配置
	if config.Transport == nil {
		t.Fatal("Transport配置不能为空")
	}
	if config.Transport.Type != "grpc" {
		t.Errorf("expected transport type 'grpc', got %s", config.Transport.Type)
	}
	if config.Transport.ServiceName != "GunService" {
		t.Errorf("expected service name 'GunService', got %s", config.Transport.ServiceName)
	}
}

func TestVlessParser_Parse_WithFlow(t *testing.T) {
	parser := &VlessParser{}
	
	// 带流控的XTLS
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=tcp&security=tls&flow=xtls-rprx-vision#Flow-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证流控配置
	if config.Flow != "xtls-rprx-vision" {
		t.Errorf("expected flow 'xtls-rprx-vision', got %s", config.Flow)
	}
}

func TestVlessParser_Parse_InvalidURL(t *testing.T) {
	parser := &VlessParser{}
	
	testCases := []string{
		"invalid-url",
		"vless://",
		"vless://@example.com:443",  // 缺少UUID
		"vless://uuid@:443",         // 缺少服务器
		"vless://uuid@example.com",  // 缺少端口
		"vless://uuid@example.com:invalid-port", // 无效端口
		"vless://uuid@example.com:443?security=reality", // Reality缺少必要参数
	}
	
	for _, url := range testCases {
		_, err := parser.Parse(url)
		if err == nil {
			t.Errorf("expected error for invalid URL: %s", url)
		}
	}
}

func TestVlessParser_Parse_DefaultValues(t *testing.T) {
	parser := &VlessParser{}
	
	// 最小配置，使用默认值
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证默认值
	if config.Tag != "VLESS-example.com-443" {
		t.Errorf("expected default tag 'VLESS-example.com-443', got %s", config.Tag)
	}
	
	// 默认应该是无加密TCP
	if config.Transport != nil {
		t.Error("默认配置不应该有transport配置")
	}
	if config.TLS != nil {
		t.Error("默认配置不应该有TLS配置")
	}
}

func TestVlessParser_Parse_ALPN(t *testing.T) {
	parser := &VlessParser{}
	
	// 带ALPN配置
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?type=tcp&security=tls&alpn=h2,http/1.1#ALPN-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证ALPN配置
	if config.TLS == nil {
		t.Fatal("TLS配置不能为空")
	}
	if len(config.TLS.ALPN) != 2 {
		t.Errorf("expected 2 ALPN protocols, got %d", len(config.TLS.ALPN))
	}
	if config.TLS.ALPN[0] != "h2" {
		t.Errorf("expected first ALPN 'h2', got %s", config.TLS.ALPN[0])
	}
	if config.TLS.ALPN[1] != "http/1.1" {
		t.Errorf("expected second ALPN 'http/1.1', got %s", config.TLS.ALPN[1])
	}
}

func TestVlessParser_Parse_PacketEncoding(t *testing.T) {
	parser := &VlessParser{}
	
	// 带包编码配置
	vlessURL := "vless://550e8400-e29b-41d4-a716-446655440000@example.com:443?packetEncoding=xudp#PacketEncoding-Test"
	
	result, err := parser.Parse(vlessURL)
	if err != nil {
		t.Fatalf("解析VLESS URL失败: %v", err)
	}
	
	var config model.VlessConfig
	err = json.Unmarshal([]byte(result), &config)
	if err != nil {
		t.Fatalf("反序列化配置失败: %v", err)
	}
	
	// 验证包编码配置
	if config.PacketEncoding != "xudp" {
		t.Errorf("expected packet encoding 'xudp', got %s", config.PacketEncoding)
	}
}