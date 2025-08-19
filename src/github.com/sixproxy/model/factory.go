package model

import "singbox_sub/src/github.com/sixproxy/constant"

func NewOutbound(outboundType string) OutboundConfig {
	switch outboundType {
	case constant.OUTBOUND_SS:
		return &SsConfig{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_SSR:
		return &SsrConfig{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_HY2:
		return &Hysteria2Config{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_TROJAN:
		return &TrojanConfig{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_ANYTLS:
		return &AnytlsConfig{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_SELECTOR:
		return &SelectorOutbound{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_URLTEST:
		return &URLTestOutbound{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_DIRECT:
		return &DirectOutbound{Outbound: Outbound{Type: outboundType}}
	case constant.OUTBOUND_SOCKS:
		return &SocksOutbound{Outbound: Outbound{Type: outboundType}}
	default:
		return nil
	}
}
