package util

import (
	"net/url"
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

func RemoveEmoji(s string) string {
	bs := []byte(s)
	out := bs[:0] // 复用原切片，省一次分配

	for i := 0; i < len(bs); {
		// 4 字节 UTF-8 的首字节一定是 0xF0~0xF4
		if bs[i]&0xF8 == 0xF0 { // 0xF0 = 11110000
			// 跳过这 4 字节（一个 emoji）
			i += 4
			continue
		}
		// 普通字符，拷贝过去
		out = append(out, bs[i])
		i++
	}
	return string(out)
}

func ParseTag(data string) string {

	// 处理 # 标签
	if hashIndex := strings.Index(data, "#"); hashIndex != -1 {
		if hashIndex+1 < len(data) {
			tag, err := url.QueryUnescape(data[hashIndex+1:])
			if err == nil && tag != "" {
				return strings.TrimSpace(RemoveEmoji(tag))
			}
		}
		data = data[:hashIndex]
	}

	// 处理 ?remarks= 标签
	if remarksIndex := strings.Index(data, "?remarks="); remarksIndex != -1 {
		if remarksIndex+9 < len(data) {
			tag, err := url.QueryUnescape(data[remarksIndex+9:])
			if err == nil && tag != "" {
				return RemoveEmoji(tag)
			}
		}
		data = data[:remarksIndex]
	}

	return ""
}
