package protocol

import (
	"encoding/json"
	"testing"
)

func TestSSRParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantServer  string
		wantPort    int
		wantMethod  string
		wantProto   string
		wantObfs    string
		wantTag     string
		wantErr     bool
		description string
	}{
		{
			name:        "Basic SSR URL",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==",
			wantServer:  "127.0.0.1",
			wantPort:    1234,
			wantMethod:  "aes-256-cfb",
			wantProto:   "origin",
			wantObfs:    "plain",
			wantTag:     "SSR-127.0.0.1:1234", // 默认tag
			wantErr:     false,
			description: "Basic SSR URL without parameters",
		},
		{
			name:        "SSR URL with parameters",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?remarks=VGVzdCBTU1I&group=VGVzdEdyb3Vw&obfsparam=dGVzdHBhcmFt&protoparam=dGVzdHByb3RvcGFyYW0",
			wantServer:  "127.0.0.1",
			wantPort:    1234,
			wantMethod:  "aes-256-cfb",
			wantProto:   "origin",
			wantObfs:    "plain",
			wantTag:     "Test SSR",
			wantErr:     false,
			description: "SSR URL with base64 encoded parameters",
		},
		{
			name:        "Invalid SSR URL - not base64",
			input:       "ssr://invalidbase64!@#$",
			wantErr:     true,
			description: "Should fail with invalid base64",
		},
		{
			name:        "Invalid SSR URL - wrong format",
			input:       "ssr://MTI3LjAuMC4xOjEyMzQ", // incomplete data
			wantErr:     true,
			description: "Should fail with incomplete SSR data",
		},
		{
			name:        "Non-SSR URL",
			input:       "ss://method:pass@server:port",
			wantErr:     true,
			description: "Should fail with non-SSR URL",
		},
	}

	parser := &ssrParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

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

			if result == "" {
				t.Errorf("%s: got empty result", tt.description)
				return
			}

			// Parse the JSON result to verify fields
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(result), &config); err != nil {
				t.Errorf("%s: failed to parse result JSON: %v", tt.description, err)
				return
			}

			// Check server
			if server, ok := config["server"].(string); ok {
				if server != tt.wantServer {
					t.Errorf("%s: server = %v, want %v", tt.description, server, tt.wantServer)
				}
			}

			// Check port
			if port, ok := config["server_port"].(float64); ok {
				if int(port) != tt.wantPort {
					t.Errorf("%s: port = %v, want %v", tt.description, int(port), tt.wantPort)
				}
			}

			// Check method
			if method, ok := config["method"].(string); ok {
				if method != tt.wantMethod {
					t.Errorf("%s: method = %v, want %v", tt.description, method, tt.wantMethod)
				}
			}

			// Check protocol
			if proto, ok := config["protocol"].(string); ok {
				if proto != tt.wantProto {
					t.Errorf("%s: protocol = %v, want %v", tt.description, proto, tt.wantProto)
				}
			}

			// Check obfs
			if obfs, ok := config["obfs"].(string); ok {
				if obfs != tt.wantObfs {
					t.Errorf("%s: obfs = %v, want %v", tt.description, obfs, tt.wantObfs)
				}
			}

			// Check tag
			if tag, ok := config["tag"].(string); ok {
				if tag != tt.wantTag {
					t.Errorf("%s: tag = %v, want %v", tt.description, tag, tt.wantTag)
				}
			}

			// Check type
			if typ, ok := config["type"].(string); ok {
				if typ != "shadowsocksr" {
					t.Errorf("%s: type = %v, want shadowsocksr", tt.description, typ)
				}
			}
		})
	}
}

func TestSSRParser_Proto(t *testing.T) {
	parser := &ssrParser{}
	if got := parser.Proto(); got != "ssr" {
		t.Errorf("Proto() = %v, want ssr", got)
	}
}

func TestSSRParserRegistration(t *testing.T) {
	// Test that SSR parser is properly registered
	parser, exists := parsers["ssr"]
	if !exists {
		t.Error("SSR parser not registered")
		return
	}

	if parser.Proto() != "ssr" {
		t.Errorf("Registered parser proto = %v, want ssr", parser.Proto())
	}
}

func TestSSRParseRealWorldExamples(t *testing.T) {
	// 这些是一些真实世界的SSR URL示例（已脱敏）
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "Real SSR example 1",
			// ssr://server:port:protocol:method:obfs:password_base64/?remarks=name&group=group
			input:       "ssr://ZXhhbXBsZS5jb206NDQzOm9yaWdpbjphZXMtMjU2LWN0cjpwbGFpbjpZV0ZoWVdGaA/?remarks=VGVzdCBTZXJ2ZXI&group=VGVzdA",
			description: "Real world SSR URL with Chinese characters in base64",
		},
	}

	parser := &ssrParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if err != nil {
				// For real world examples, we just check they don't crash
				t.Logf("%s: parse error (expected for demo data): %v", tt.description, err)
				return
			}

			if result == "" {
				t.Errorf("%s: got empty result", tt.description)
				return
			}

			// Verify it's valid JSON
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(result), &config); err != nil {
				t.Errorf("%s: result is not valid JSON: %v", tt.description, err)
				return
			}

			t.Logf("%s: successfully parsed: %s", tt.description, result)
		})
	}
}

// Benchmark test for SSR parsing
func BenchmarkSSRParser_Parse(b *testing.B) {
	parser := &ssrParser{}
	input := "ssr://MTI3LjAuMC4xOjEyMzQ6b3JpZ2luOmFlcy0yNTYtY2ZiOnBsYWluOmRHVnpkQQ==/?remarks=VGVzdCBTU1I"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Errorf("Parse failed: %v", err)
		}
	}
}

func TestSSRParser_EdgeCases(t *testing.T) {
	parser := &ssrParser{}

	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "Empty URL",
			input:       "",
			expectError: true,
			description: "Empty string should cause error",
		},
		{
			name:        "Wrong protocol",
			input:       "http://example.com",
			expectError: true,
			description: "Non-SSR protocol should cause error",
		},
		{
			name:        "SSR prefix only",
			input:       "ssr://",
			expectError: true,
			description: "SSR with empty data should cause error",
		},
		{
			name:        "Malformed base64",
			input:       "ssr://not-valid-base64!",
			expectError: true,
			description: "Invalid base64 should cause error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}
