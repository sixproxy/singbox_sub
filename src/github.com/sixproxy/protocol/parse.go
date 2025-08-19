package protocol

import (
	"errors"
	"strings"
)

type Parser interface {
	Proto() string                       // 返回协议标识符，例如 "ss"、"trojan"
	Parse(rawURL string) (string, error) // 把原始URL解析
}

var parsers = make(map[string]Parser)

func Parse(rawURL string) (string, error) {
	proto := ProtoOf(rawURL)
	p, ok := parsers[proto]
	if !ok {
		return "", errors.New("unsupported protocol: " + proto)
	}
	return p.Parse(rawURL)
}

func ProtoOf(s string) string {
	if i := strings.Index(s, "://"); i > 0 {
		return s[:i]
	}
	return ""
}
