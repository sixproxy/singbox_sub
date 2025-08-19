package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"singbox_sub/src/github.com/sixproxy/constant"
	"singbox_sub/src/github.com/sixproxy/util"
	"strings"
	"strconv"
)

type AnytlsConfig struct {
	Outbound
	Server                     string    `json:"server"`
	ServerPort                 int       `json:"server_port"`
	Password                   string    `json:"password"`
	IdleSessionCheckInterval   string    `json:"idle_session_check_interval,omitempty"`
	IdleSessionTimeout         string    `json:"idle_session_timeout,omitempty"`
	MinIdleSession             int       `json:"min_idle_session,omitempty"`
	TLS                        TLSConfig `json:"tls"`
	Detour                     *string   `json:"detour,omitempty"`
}

func (c *AnytlsConfig) GetTag() string {
	return c.Tag
}

func (c *AnytlsConfig) GetType() string {
	return c.Type
}

func (c *AnytlsConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server is required")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

var _ OutboundConfig = (*AnytlsConfig)(nil)

func (c *AnytlsConfig) SetData(data string) error {
	c.Type = constant.OUTBOUND_ANYTLS
	
	// Set default values
	c.IdleSessionCheckInterval = "30s"
	c.IdleSessionTimeout = "30s"
	c.MinIdleSession = 0
	c.TLS.Enabled = true
	c.TLS.Insecure = false
	// AnyTLS is a standalone proxy protocol, no detour needed by default
	c.Detour = nil

	// Parse tag from URL fragment or remarks
	tag := util.ParseTag(data)
	c.Tag = tag

	u, err := url.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	// Parse server and port
	if u.Host == "" {
		return fmt.Errorf("host is missing")
	}
	
	// Handle IPv6 addresses and regular hostnames
	if strings.HasPrefix(u.Host, "[") && strings.Contains(u.Host, "]:") {
		// IPv6 with port: [::1]:8080
		parts := strings.Split(u.Host, "]:")
		if len(parts) != 2 {
			return fmt.Errorf("invalid IPv6 address format")
		}
		c.Server = parts[0] + "]"  // Keep the brackets for IPv6
		port := util.String2Int(parts[1])
		if port == -999999 {
			return fmt.Errorf("invalid port: %s", parts[1])
		}
		c.ServerPort = port
	} else if strings.HasPrefix(u.Host, "[") && strings.HasSuffix(u.Host, "]") {
		// IPv6 without port: [::1]
		c.Server = u.Host
		c.ServerPort = 443 // Default HTTPS port for TLS
	} else {
		// Regular hostname or IPv4
		hostAndPort := strings.Split(u.Host, ":")
		c.Server = hostAndPort[0]
		if len(hostAndPort) > 1 {
			port := util.String2Int(hostAndPort[1])
			if port == -999999 {
				return fmt.Errorf("invalid port: %s", hostAndPort[1])
			}
			c.ServerPort = port
		} else {
			c.ServerPort = 443 // Default HTTPS port for TLS
		}
	}

	// Parse password from URL user info
	if u.User != nil {
		// Try to get password first, then fallback to username if no password
		if password, hasPassword := u.User.Password(); hasPassword && password != "" {
			c.Password = password
		} else {
			c.Password = u.User.Username()
		}
	}

	// Parse query parameters
	query := u.Query()

	// TLS configuration
	if sni := query.Get("sni"); sni != "" {
		c.TLS.ServerName = sni
	} else if serverName := query.Get("server_name"); serverName != "" {
		c.TLS.ServerName = serverName
	} else {
		c.TLS.ServerName = c.Server
	}

	if alpn := query.Get("alpn"); alpn != "" {
		c.TLS.Alpn = strings.Split(alpn, ",")
	}

	if insecure := query.Get("insecure"); insecure == "1" || insecure == "true" {
		c.TLS.Insecure = true
	}

	// Session management parameters
	if checkInterval := query.Get("check_interval"); checkInterval != "" {
		c.IdleSessionCheckInterval = checkInterval
	}

	if timeout := query.Get("idle_timeout"); timeout != "" {
		c.IdleSessionTimeout = timeout
	}

	if minIdle := query.Get("min_idle"); minIdle != "" {
		if val, err := strconv.Atoi(minIdle); err == nil {
			c.MinIdleSession = val
		}
	}

	// Basic validation before returning
	if c.Server == "" {
		return fmt.Errorf("server is missing")
	}
	if c.Password == "" {
		return fmt.Errorf("password is missing")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}

	return nil
}

func (c *AnytlsConfig) String() string {
	// Remove optional fields if they have default values
	copy := *c
	if copy.IdleSessionCheckInterval == "30s" {
		copy.IdleSessionCheckInterval = ""
	}
	if copy.IdleSessionTimeout == "30s" {
		copy.IdleSessionTimeout = ""
	}
	if copy.MinIdleSession == 0 {
		copy.MinIdleSession = 0
	}

	b, err := json.Marshal(copy)
	if err != nil {
		return ""
	}
	return string(b)
}