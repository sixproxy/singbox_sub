package model

import (
	"encoding/json"
	"testing"
)

func TestSsrConfig_SetData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantServer  string
		wantPort    int
		wantMethod  string
		wantProto   string
		wantObfs    string
		wantPass    string
		wantTag     string
		wantErr     bool
		description string
	}{
		{
			name:        "Valid SSR URL basic",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==",
			wantServer:  "127.0.0.1",
			wantPort:    1234,
			wantMethod:  "aes-256-cfb",
			wantProto:   "origin",
			wantObfs:    "plain",
			wantPass:    "test", // base64 decode of "dGVzdA=="
			wantTag:     "SSR-127.0.0.1:1234",
			wantErr:     false,
			description: "Basic SSR configuration parsing",
		},
		{
			name:        "Valid SSR URL with remarks",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?remarks=VGVzdCBTZXJ2ZXI",
			wantServer:  "127.0.0.1",
			wantPort:    1234,
			wantMethod:  "aes-256-cfb",
			wantProto:   "origin",
			wantObfs:    "plain",
			wantPass:    "test",
			wantTag:     "Test Server", // base64 decode of "VGVzdCBTZXJ2ZXI"
			wantErr:     false,
			description: "SSR with remarks parameter",
		},
		{
			name:        "Invalid protocol",
			input:       "ss://invalid",
			wantErr:     true,
			description: "Non-SSR URL should fail",
		},
		{
			name:        "Invalid base64",
			input:       "ssr://invalid-base64!",
			wantErr:     true,
			description: "Invalid base64 should fail",
		},
		{
			name:        "Incomplete SSR data",
			input:       "ssr://MTI3LjAuMC4x", // only has server
			wantErr:     true,
			description: "Incomplete SSR data should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SsrConfig{}
			err := config.SetData(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
				return
			}

			// Verify parsed values
			if config.Server != tt.wantServer {
				t.Errorf("%s: Server = %v, want %v", tt.description, config.Server, tt.wantServer)
			}

			if config.ServerPort != tt.wantPort {
				t.Errorf("%s: ServerPort = %v, want %v", tt.description, config.ServerPort, tt.wantPort)
			}

			if config.Method != tt.wantMethod {
				t.Errorf("%s: Method = %v, want %v", tt.description, config.Method, tt.wantMethod)
			}

			if config.Protocol != tt.wantProto {
				t.Errorf("%s: Protocol = %v, want %v", tt.description, config.Protocol, tt.wantProto)
			}

			if config.Obfs != tt.wantObfs {
				t.Errorf("%s: Obfs = %v, want %v", tt.description, config.Obfs, tt.wantObfs)
			}

			if config.Password != tt.wantPass {
				t.Errorf("%s: Password = %v, want %v", tt.description, config.Password, tt.wantPass)
			}

			if config.Tag != tt.wantTag {
				t.Errorf("%s: Tag = %v, want %v", tt.description, config.Tag, tt.wantTag)
			}

			if config.Type != "shadowsocksr" {
				t.Errorf("%s: Type = %v, want shadowsocksr", tt.description, config.Type)
			}
		})
	}
}

func TestSsrConfig_String(t *testing.T) {
	config := &SsrConfig{
		Outbound: Outbound{
			Tag:  "test-ssr",
			Type: "shadowsocksr",
		},
		Server:        "127.0.0.1",
		ServerPort:    1234,
		Method:        "aes-256-cfb",
		Password:      "testpass",
		Protocol:      "origin",
		ProtocolParam: "test-proto-param",
		Obfs:          "plain",
		ObfsParam:     "test-obfs-param",
	}

	jsonStr := config.String()
	if jsonStr == "" {
		t.Error("String() returned empty string")
		return
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("String() returned invalid JSON: %v", err)
		return
	}

	// Check required fields
	if parsed["type"] != "shadowsocksr" {
		t.Errorf("JSON type = %v, want shadowsocksr", parsed["type"])
	}

	if parsed["tag"] != "test-ssr" {
		t.Errorf("JSON tag = %v, want test-ssr", parsed["tag"])
	}

	if parsed["server"] != "127.0.0.1" {
		t.Errorf("JSON server = %v, want 127.0.0.1", parsed["server"])
	}

	if parsed["server_port"] != float64(1234) {
		t.Errorf("JSON server_port = %v, want 1234", parsed["server_port"])
	}
}

func TestSsrConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *SsrConfig
		wantErr     bool
		description string
	}{
		{
			name: "Valid config",
			config: &SsrConfig{
				Server:     "127.0.0.1",
				ServerPort: 1234,
				Method:     "aes-256-cfb",
				Password:   "testpass",
			},
			wantErr:     false,
			description: "Complete valid configuration",
		},
		{
			name: "Missing server",
			config: &SsrConfig{
				ServerPort: 1234,
				Method:     "aes-256-cfb",
				Password:   "testpass",
			},
			wantErr:     true,
			description: "Missing server should fail validation",
		},
		{
			name: "Missing port",
			config: &SsrConfig{
				Server:   "127.0.0.1",
				Method:   "aes-256-cfb",
				Password: "testpass",
			},
			wantErr:     true,
			description: "Missing port should fail validation",
		},
		{
			name: "Missing method",
			config: &SsrConfig{
				Server:     "127.0.0.1",
				ServerPort: 1234,
				Password:   "testpass",
			},
			wantErr:     true,
			description: "Missing method should fail validation",
		},
		{
			name: "Missing password",
			config: &SsrConfig{
				Server:     "127.0.0.1",
				ServerPort: 1234,
				Method:     "aes-256-cfb",
			},
			wantErr:     true,
			description: "Missing password should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			} else if !tt.wantErr && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}

func TestSsrConfig_GettersAndSetters(t *testing.T) {
	config := &SsrConfig{
		Outbound: Outbound{
			Tag:  "test-tag",
			Type: "test-type",
		},
	}

	if got := config.GetTag(); got != "test-tag" {
		t.Errorf("GetTag() = %v, want test-tag", got)
	}

	if got := config.GetType(); got != "test-type" {
		t.Errorf("GetType() = %v, want test-type", got)
	}
}

func TestSsrConfig_ParseParams(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantObfsParam  string
		wantProtoParam string
		wantTag        string
		description    string
	}{
		{
			name:           "With obfsparam and protoparam",
			input:          "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?obfsparam=dGVzdG9iZnM&protoparam=dGVzdHByb3Rv&remarks=VGVzdCBTZXJ2ZXI",
			wantObfsParam:  "testobfs",    // base64 decode of "dGVzdG9iZnM"
			wantProtoParam: "testproto",   // base64 decode of "dGVzdHByb3Rv"
			wantTag:        "Test Server", // base64 decode of "VGVzdCBTZXJ2ZXI"
			description:    "Parameters should be base64 decoded correctly",
		},
		{
			name:        "With group as tag fallback",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?group=VGVzdEdyb3Vw",
			wantTag:     "TestGroup", // base64 decode of "VGVzdEdyb3Vw"
			description: "Group should be used as tag when remarks not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SsrConfig{}
			err := config.SetData(tt.input)

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
				return
			}

			if tt.wantObfsParam != "" && config.ObfsParam != tt.wantObfsParam {
				t.Errorf("%s: ObfsParam = %v, want %v", tt.description, config.ObfsParam, tt.wantObfsParam)
			}

			if tt.wantProtoParam != "" && config.ProtocolParam != tt.wantProtoParam {
				t.Errorf("%s: ProtocolParam = %v, want %v", tt.description, config.ProtocolParam, tt.wantProtoParam)
			}

			if tt.wantTag != "" && config.Tag != tt.wantTag {
				t.Errorf("%s: Tag = %v, want %v", tt.description, config.Tag, tt.wantTag)
			}
		})
	}
}

// Benchmark test for SSR config parsing
func BenchmarkSsrConfig_SetData(b *testing.B) {
	input := "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?remarks=VGVzdCBTZXJ2ZXI"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &SsrConfig{}
		err := config.SetData(input)
		if err != nil {
			b.Errorf("SetData failed: %v", err)
		}
	}
}
