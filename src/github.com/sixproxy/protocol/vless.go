package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/util"
	"strconv"
	"strings"
)

type VlessParser struct{}

func (p *VlessParser) Proto() string {
	return "vless"
}

func (p *VlessParser) Parse(rawURL string) (string, error) {
	// 解析VLESS URL格式：
	// vless://uuid@server:port?type=tcp&security=tls&sni=example.com&fp=chrome&pbk=publickey&sid=shortid#tag
	// vless://uuid@server:port?type=ws&path=/path&host=example.com&security=reality&pbk=publickey&sid=shortid#tag

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("解析VLESS URL失败: %v", err)
	}

	// 提取基本信息
	uuid := parsedURL.User.Username()
	if uuid == "" {
		return "", fmt.Errorf("VLESS UUID不能为空")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return "", fmt.Errorf("VLESS服务器地址不能为空")
	}

	port, err := strconv.Atoi(parsedURL.Port())
	if err != nil {
		return "", fmt.Errorf("VLESS端口格式错误: %v", err)
	}

	// 提取标签
	tag := parsedURL.Fragment
	if tag == "" {
		tag = fmt.Sprintf("VLESS-%s-%d", host, port)
	}
	tag = util.RemoveEmoji(tag)

	// 解析查询参数
	query := parsedURL.Query()

	// 创建VLESS配置
	vlessConfig := model.NewVlessOutbound(tag, host, port, uuid, "")

	// 解析传输层类型
	transportType := query.Get("type")
	if transportType == "" {
		transportType = "tcp" // 默认TCP
	}

	// 解析安全类型
	security := query.Get("security")

	// 配置传输层
	switch transportType {
	case "tcp":
		// TCP直连，无需额外配置

	case "ws", "websocket":
		// WebSocket传输
		path := query.Get("path")
		if path == "" {
			path = "/"
		}
		host := query.Get("host")
		vlessConfig.SetTransport("ws", path, host)

	case "grpc":
		// gRPC传输
		serviceName := query.Get("serviceName")
		if serviceName == "" {
			serviceName = query.Get("service") // 兼容不同参数名
		}
		vlessConfig.Transport = &model.Transport{
			Type:        "grpc",
			ServiceName: serviceName,
		}

	case "https", "h2":
		// HTTP/2传输
		path := query.Get("path")
		if path == "" {
			path = "/"
		}
		host := query.Get("host")
		vlessConfig.SetTransport("https", path, host)

	default:
		// 未知传输类型，使用TCP
		transportType = "tcp"
	}

	// 配置安全层
	switch security {
	case "tls":
		// 普通TLS
		sni := query.Get("sni")
		if sni == "" {
			sni = query.Get("peer") // 兼容不同参数名
		}
		if sni == "" {
			sni = host
		}

		insecure := query.Get("allowInsecure") == "1" || query.Get("skip-cert-verify") == "1"
		vlessConfig.SetTLS(sni, insecure)

		// 设置uTLS指纹
		if fp := query.Get("fp"); fp != "" {
			vlessConfig.SetUTLS(fp)
		}

		// 设置ALPN
		if alpn := query.Get("alpn"); alpn != "" {
			alpnList := strings.Split(alpn, ",")
			vlessConfig.TLS.ALPN = alpnList
		}

	case "reality":
		// Reality TLS
		publicKey := query.Get("pbk")
		shortID := query.Get("sid")
		sni := query.Get("sni")

		if publicKey == "" {
			return "", fmt.Errorf("Reality配置缺少public key")
		}
		if shortID == "" {
			return "", fmt.Errorf("Reality配置缺少short ID")
		}
		if sni == "" {
			sni = host
		}

		vlessConfig.SetReality(publicKey, shortID, sni)

		// 设置uTLS指纹（Reality通常需要）
		if fp := query.Get("fp"); fp != "" {
			vlessConfig.SetUTLS(fp)
		} else {
			// Reality默认使用chrome指纹
			vlessConfig.SetUTLS("chrome")
		}

	case "none", "":
		// 无加密

	default:
		return "", fmt.Errorf("不支持的安全类型: %s", security)
	}

	// 设置流控（仅在使用XTLS时）
	if flow := query.Get("flow"); flow != "" {
		vlessConfig.SetFlow(flow)
	}

	// 设置包编码
	if packetEncoding := query.Get("packetEncoding"); packetEncoding != "" {
		vlessConfig.SetPacketEncoding(packetEncoding)
	}

	// 验证配置
	if err := vlessConfig.Validate(); err != nil {
		return "", fmt.Errorf("VLESS配置验证失败: %v", err)
	}

	// 序列化为JSON
	configJSON, err := json.Marshal(vlessConfig)
	if err != nil {
		return "", fmt.Errorf("序列化VLESS配置失败: %v", err)
	}

	return string(configJSON), nil
}

// 注册解析器
func init() {
	parsers["vless"] = &VlessParser{}
}
