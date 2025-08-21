package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/model"
	"strconv"
	"strings"
	"time"
)

func String2Int(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return -999999
	}
	return i
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

// GetSystemDNS 自动获取当前系统配置的DNS服务器
func GetSystemDNS() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getDNSMacOS()
	case "linux":
		return getDNSLinux()
	case "windows":
		return getDNSWindows()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// getDNSMacOS 获取macOS系统的DNS设置
func getDNSMacOS() ([]string, error) {
	// 方法1: 尝试读取系统配置
	if dns, err := getDNSFromScutil(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法2: 尝试读取resolv.conf
	if dns, err := getDNSFromResolvConf(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法3: 使用Go的默认DNS解析器配置
	return getDNSFromGoResolver()
}

// getDNSLinux 获取Linux系统的DNS设置
func getDNSLinux() ([]string, error) {
	// 方法1: 读取 /etc/resolv.conf
	if dns, err := getDNSFromResolvConf(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法2: 尝试systemd-resolve
	if dns, err := getDNSFromSystemdResolve(); err == nil && len(dns) > 0 {
		return dns, nil
	}

	// 方法3: 使用Go的默认DNS解析器配置
	return getDNSFromGoResolver()
}

// getDNSWindows 获取Windows系统的DNS设置
func getDNSWindows() ([]string, error) {
	// Windows上主要通过Go的默认解析器获取
	return getDNSFromGoResolver()
}

// getDNSFromScutil macOS特有: 通过scutil命令获取DNS
func getDNSFromScutil() ([]string, error) {
	// 这里我们通过读取系统网络配置来获取DNS
	// macOS的网络配置通常存储在系统配置数据库中
	// 由于无法直接执行scutil，我们回退到其他方法
	return nil, fmt.Errorf("scutil method not available")
}

// getDNSFromResolvConf 从 /etc/resolv.conf 读取DNS服务器
func getDNSFromResolvConf() ([]string, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dnsServers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略注释行
		if strings.HasPrefix(line, "#") {
			continue
		}

		// 查找 nameserver 行
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.TrimSpace(parts[1])
				// 验证是否为有效IP
				if net.ParseIP(ip) != nil {
					dnsServers = append(dnsServers, ip)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dnsServers, nil
}

// getDNSFromSystemdResolve Linux特有: 通过systemd-resolve获取DNS
func getDNSFromSystemdResolve() ([]string, error) {
	// 尝试读取systemd-resolve的状态文件
	possiblePaths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/run/systemd/resolve/stub-resolv.conf",
	}

	for _, path := range possiblePaths {
		if dns, err := readDNSFromFile(path); err == nil && len(dns) > 0 {
			return dns, nil
		}
	}

	return nil, fmt.Errorf("systemd-resolve DNS not found")
}

// readDNSFromFile 从指定文件读取DNS配置
func readDNSFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dnsServers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.TrimSpace(parts[1])
				if net.ParseIP(ip) != nil {
					dnsServers = append(dnsServers, ip)
				}
			}
		}
	}

	return dnsServers, scanner.Err()
}

// getDNSFromGoResolver 使用Go内置方法获取DNS服务器
func getDNSFromGoResolver() ([]string, error) {
	// 这个方法通过解析一个已知域名来获取DNS服务器信息
	// 虽然不是直接获取系统DNS配置，但可以作为后备方案

	// 尝试查询一些常见的DNS服务器来确定当前使用的DNS
	testDomains := []string{
		"google.com",
		"cloudflare.com",
		"baidu.com",
	}

	// 获取当前网络接口的DNS信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return getDefaultDNS(), nil // 返回默认DNS作为后备
	}

	var dnsServers []string
	seen := make(map[string]bool)

	// 尝试从网络接口获取DNS信息
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					// 基于网络段猜测可能的DNS服务器
					possibleDNS := guessDNSFromNetwork(ipnet.IP)
					for _, dns := range possibleDNS {
						if !seen[dns] {
							seen[dns] = true
							dnsServers = append(dnsServers, dns)
						}
					}
				}
			}
		}
	}

	// 如果没有找到任何DNS，返回一些通用的DNS服务器
	if len(dnsServers) == 0 {
		return getDefaultDNS(), nil
	}

	// 验证这些DNS服务器是否可用
	validDNS := validateDNSServers(dnsServers, testDomains)
	if len(validDNS) > 0 {
		return validDNS, nil
	}

	return getDefaultDNS(), nil
}

// guessDNSFromNetwork 根据网络地址猜测可能的DNS服务器
func guessDNSFromNetwork(ip net.IP) []string {
	var dns []string

	// 基于IP地址的前三个八位组构造可能的网关/DNS地址
	if ip.To4() != nil {
		ipv4 := ip.To4()

		// 常见的网关地址通常是 x.x.x.1
		gateway := fmt.Sprintf("%d.%d.%d.1", ipv4[0], ipv4[1], ipv4[2])
		dns = append(dns, gateway)

		// 一些运营商使用的DNS模式
		switch {
		case ipv4[0] == 192 && ipv4[1] == 168:
			// 私有网络，可能使用路由器作为DNS
			dns = append(dns, gateway)
		case ipv4[0] == 10:
			// 企业网络
			dns = append(dns, gateway)
		case ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31:
			// 私有网络
			dns = append(dns, gateway)
		default:
			// 公网IP，可能使用运营商DNS
			// 中国移动: 221.5.88.88, 221.5.203.98
			// 中国联通: 202.106.0.20, 202.106.196.115
			// 中国电信: 202.101.172.35, 202.101.172.47
			dns = append(dns, "202.101.172.35", "202.106.0.20", "221.5.88.88")
		}
	}

	return dns
}

// validateDNSServers 验证DNS服务器是否可用
func validateDNSServers(dnsServers []string, testDomains []string) []string {
	var validDNS []string

	for _, dnsServer := range dnsServers {
		if isDNSServerValid(dnsServer, testDomains) {
			validDNS = append(validDNS, dnsServer)
		}
	}

	return validDNS
}

// isDNSServerValid 检查DNS服务器是否可用
func isDNSServerValid(dnsServer string, testDomains []string) bool {
	// 注意: 这里暂时简化实现，实际应该使用指定的DNS服务器来查询
	// 当前只是验证能否解析域名，作为基础可用性检查
	for _, domain := range testDomains {
		_, err := net.LookupHost(domain)
		if err == nil {
			return true
		}
	}
	return false
}

// getDefaultDNS 返回默认的DNS服务器列表
func getDefaultDNS() []string {
	return []string{
		"202.101.172.35", // 中国电信
		"202.106.0.20",   // 中国联通
		"221.5.88.88",    // 中国移动
		"8.8.8.8",        // Google
		"1.1.1.1",        // Cloudflare
	}
}

// GetISPDNS 获取运营商DNS (主要方法)
func GetISPDNS() []string {
	dns, err := GetSystemDNS()
	if err != nil {
		logger.NetworkWarn("获取系统DNS失败: %v，使用默认DNS", err)
		return getDefaultDNS()
	}

	if len(dns) == 0 {
		return getDefaultDNS()
	}

	// 过滤掉本地DNS (127.x.x.x)
	var filteredDNS []string
	for _, d := range dns {
		if !strings.HasPrefix(d, "127.") && !strings.HasPrefix(d, "::1") {
			filteredDNS = append(filteredDNS, d)
		}
	}

	if len(filteredDNS) == 0 {
		return getDefaultDNS()
	}

	return filteredDNS
}

// LocationInfo IP地理位置信息
type LocationInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Region  string `json:"region"` // 省份
	City    string `json:"city"`   // 城市
	ISP     string `json:"isp"`    // 运营商
}

// GetOptimalClientSubnet 获取优化的client_subnet值
func GetOptimalClientSubnet() string {
	// 1. 尝试获取真实的公网IP和地理位置
	location, err := getRealLocation()
	if err != nil {
		logger.NetworkWarn("获取地理位置失败: %v，使用默认策略", err)
		return model.GetDefaultClientSubnet()
	}

	// 2. 根据城市和运营商获取精确的网段
	if subnet := model.GetCityISPSubnet(location.City, location.ISP); subnet != "" {
		logger.NetworkInfo("检测到位置: %s %s，使用client_subnet: %s", location.City, location.ISP, subnet)
		return subnet
	}

	// 3. 如果精确匹配失败，尝试模糊匹配
	if subnet := model.GetFallbackSubnet(location); subnet != "" {
		logger.NetworkInfo("使用备选匹配: %s，client_subnet: %s", location.City, subnet)
		return subnet
	}

	// 4. 默认策略
	logger.NetworkInfo("无法精确匹配，使用默认client_subnet")
	return model.GetDefaultClientSubnet()
}

// getRealLocation 获取真实的公网IP地理位置
func getRealLocation() (*LocationInfo, error) {
	// 使用多个IP查询服务，提高成功率
	services := []string{
		"https://ipapi.co/json/",
		"http://ip.sb/geoip/",
		"http://ip-api.com/json/?fields=status,country,regionName,city,isp,query",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		location, err := queryLocationService(client, service)

		if err == nil && location != nil {
			originalCity := location.City

			// 先处理空字段的情况
			if originalCity == "" {
				logger.NetworkWarn("API返回的城市字段为空，尝试从省份推导")
				// 尝试从省份/地区推导城市
				if location.Region != "" {
					originalCity = model.InferCityFromRegion(location.Region)
					location.City = originalCity
					logger.NetworkInfo("从省份 '%s' 推导城市: %s", location.Region, originalCity)
				}
			}

			// 进行城市名映射
			mappedCity := model.GetCityNameCH(originalCity)

			// 详细的调试信息
			logger.NetworkInfo("地理位置查询成功 - 服务: %s", service)
			logger.NetworkInfo("原始数据 - 城市:'%s', 省份:'%s', ISP:'%s', IP:'%s'", originalCity, location.Region, location.ISP, location.IP)
			logger.NetworkInfo("城市映射: '%s' -> '%s'", originalCity, mappedCity)

			// 验证映射结果
			if mappedCity != "" && mappedCity != originalCity {
				// 映射成功
				location.City = mappedCity
				logger.NetworkInfo("城市映射成功，使用: %s", mappedCity)
			} else if originalCity != "" {
				// 映射失败但原始城市不为空，保留原始名称
				location.City = originalCity
				logger.NetworkWarn("未找到城市映射，保留原始名称: %s", originalCity)
			} else {
				// 城市字段完全为空，使用ISP推导
				logger.NetworkWarn("城市信息完全缺失，尝试从ISP推导默认城市")
				location.City = model.GetDefaultCityByISP(location.ISP)
			}

			logger.NetworkInfo("最终结果 - 城市:'%s', ISP:'%s', 省份:'%s'", location.City, location.ISP, location.Region)

			// 确保关键字段不为空
			if location.City == "" {
				logger.NetworkWarn("城市字段仍为空，使用默认值")
				location.City = "北京" // 使用默认城市
			}
			if location.ISP == "" {
				logger.NetworkWarn("ISP字段为空，使用默认值")
				location.ISP = "电信" // 使用默认ISP
			}

			return location, nil
		}
		logger.NetworkWarn("地理位置查询服务 %s 失败: %v", service, err)
	}

	return nil, fmt.Errorf("所有地理位置查询服务都失败了")
}

// queryLocationService 查询单个地理位置服务
func queryLocationService(client *http.Client, serviceURL string) (*LocationInfo, error) {
	resp, err := client.Get(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否为空
	if len(body) == 0 {
		return nil, fmt.Errorf("响应为空")
	}

	// 检查是否有错误响应 (如rate limit)
	var errorCheck map[string]interface{}
	if err := json.Unmarshal(body, &errorCheck); err == nil {
		if errorVal, exists := errorCheck["error"]; exists {
			if errorBool, ok := errorVal.(bool); ok && errorBool {
				reason := "未知错误"
				if reasonVal, exists := errorCheck["reason"]; exists {
					reason = fmt.Sprintf("%v", reasonVal)
				}
				return nil, fmt.Errorf("服务返回错误: %s", reason)
			}
		}

		// 检查ip-api.com的状态字段
		if statusVal, exists := errorCheck["status"]; exists {
			if status, ok := statusVal.(string); ok && status != "success" {
				return nil, fmt.Errorf("API状态异常: %s", status)
			}
		}
	}

	// 解析不同服务的响应格式
	if strings.Contains(serviceURL, "ip-api.com") {
		return parseIPAPIResponse(body)
	} else if strings.Contains(serviceURL, "ipapi.co") {
		return parseIPAPICoResponse(body)
	} else if strings.Contains(serviceURL, "ip.sb") {
		return parseIPSBResponse(body)
	}

	return nil, fmt.Errorf("未知的服务格式")
}

// parseIPAPIResponse 解析ip-api.com的响应
func parseIPAPIResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
		Query      string `json:"query"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("API返回状态: %s", response.Status)
	}

	return &LocationInfo{
		IP:      response.Query,
		Country: response.Country,
		Region:  response.RegionName,
		City:    response.City,
		ISP:     model.NormalizeISPName(response.ISP),
	}, nil
}

// parseIPAPICoResponse 解析ipapi.co的响应

func parseIPAPICoResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		IP      string `json:"ip"`
		Country string `json:"country_name"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Org     string `json:"org"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &LocationInfo{
		IP:      response.IP,
		Country: response.Country,
		Region:  response.Region,
		City:    model.GetCityNameCH(response.City),
		ISP:     model.NormalizeISPName(response.Org),
	}, nil
}

// parseIPSBResponse 解析ip.sb的响应
func parseIPSBResponse(body []byte) (*LocationInfo, error) {
	var response struct {
		IP           string `json:"ip"`
		Country      string `json:"country"`
		Region       string `json:"region"`
		City         string `json:"city"`
		ISP          string `json:"isp"`
		Organization string `json:"organization"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	isp := response.ISP
	if isp == "" {
		isp = response.Organization
	}

	return &LocationInfo{
		IP:      response.IP,
		Country: response.Country,
		Region:  response.Region,
		City:    response.City,
		ISP:     model.NormalizeISPName(isp),
	}, nil
}

// GetISPName 获取当前运营商名称 (更新为使用新的地理位置API)
func GetISPName() string {
	// 尝试从真实位置获取运营商信息
	if location, err := getRealLocation(); err == nil {
		return fmt.Sprintf("%s (%s)", location.ISP, location.City)
	}

	// 备选方案：通过DNS推测
	currentDNS := GetISPDNS()
	if len(currentDNS) == 0 {
		return "未知运营商"
	}

	// 检查是否使用了知名公共DNS
	for _, dns := range currentDNS {
		switch dns {
		case "8.8.8.8", "8.8.4.4":
			return "Google DNS"
		case "1.1.1.1", "1.0.0.1":
			return "Cloudflare DNS"
		case "114.114.114.114", "114.114.115.115":
			return "114 DNS"
		case "223.5.5.5", "223.6.6.6":
			return "阿里 DNS"
		case "119.29.29.29", "182.254.116.116":
			return "腾讯 DNS"
		}
	}

	return "本地网络"
}

// isPrivateIP 检查是否为私有IP地址
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为私有IP段
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// GetInternalIP 获取本机的内网IP地址
func GetInternalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.NetworkWarn("获取网络接口失败: %v", err)
		return "127.0.0.1"
	}

	for _, iface := range interfaces {
		// 跳过禁用的接口和回环接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ip := ipnet.IP.String()
					// 优先返回私有IP地址
					if isPrivateIP(ip) {
						logger.NetworkInfo("检测到内网IP: %s (接口: %s)", ip, iface.Name)
						return ip
					}
				}
			}
		}
	}

	// 如果没有找到私有IP，尝试连接外部服务来获取本地IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logger.NetworkWarn("无法检测内网IP，使用默认地址: %v", err)
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.String()
	logger.NetworkInfo("通过连接测试检测到内网IP: %s", ip)
	return ip
}
