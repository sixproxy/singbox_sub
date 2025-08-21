package util

import (
	"regexp"
	"singbox_sub/src/github.com/sixproxy/constant"
	"strings"
)

// matchPattern 检查tag是否匹配单个pattern
// 支持普通字符串匹配和正则表达式匹配
func MatchPattern(tag, pattern string) bool {
	// 检查是否为正则表达式（简单判断是否包含正则特殊字符）
	if IsRegexPattern(pattern) {
		// 尝试正则匹配
		matched, err := regexp.MatchString(pattern, tag)
		if err != nil {
			// 如果正则表达式无效，回退到字符串包含匹配
			return strings.Contains(tag, pattern)
		}
		return matched
	} else {
		// 普通字符串包含匹配
		return strings.Contains(tag, pattern)
	}
}

// IsRegexPattern 简单判断是否可能是正则表达式
// 这里使用启发式方法检测常见的正则表达式特征
func IsRegexPattern(pattern string) bool {
	// 包含正则表达式特殊字符的话，认为是正则
	regexChars := []string{"^", "$", "*", "+", "?", ".", "[", "]", "(", ")", "{", "}", "\\"}
	for _, char := range regexChars {
		if strings.Contains(pattern, char) {
			return true
		}
	}
	return false
}

func GetNodeType(node string) string {
	switch {
	case strings.Contains(node, constant.OUTBOUND_SS):
		return constant.OUTBOUND_SS
	case strings.Contains(node, constant.OUTBOUND_SSR):
		return constant.OUTBOUND_SSR
	case strings.Contains(node, constant.OUTBOUND_HY2):
		return constant.OUTBOUND_HY2
	case strings.Contains(node, constant.OUTBOUND_TROJAN):
		return constant.OUTBOUND_TROJAN
	case strings.Contains(node, constant.OUTBOUND_ANYTLS):
		return constant.OUTBOUND_ANYTLS
	case strings.Contains(node, constant.OUTBOUND_SELECTOR):
		return constant.OUTBOUND_SELECTOR
	case strings.Contains(node, constant.OUTBOUND_URLTEST):
		return constant.OUTBOUND_URLTEST
	default:
		return ""
	}
}

func InvalidNode(tag string) bool {
	switch {
	case strings.Contains(tag, "官网"):
		return true
	case strings.Contains(tag, "流量"):
		return true
	default:
		return false
	}
}
