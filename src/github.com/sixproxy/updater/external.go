package updater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/util"
	"strings"
	"time"
)

// SingboxManager sing-box外部程序管理器
type SingboxManager struct {
	installDir   string
	binaryPath   string
	configPath   string
	serviceName  string
	homebrewName string
}

// SingboxRelease GitHub发布信息
type SingboxRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// SingboxVersion 版本信息
type SingboxVersion struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

const (
	SINGBOX_GITHUB_REPO   = "SagerNet/sing-box"
	SINGBOX_HOMEBREW_NAME = "sing-box"
	SINGBOX_SERVICE_NAME  = "sing-box"
	SINGBOX_CONFIG_FILE   = "config.json"
)

// NewSingboxManager 创建新的sing-box管理器
func NewSingboxManager() *SingboxManager {
	m := &SingboxManager{
		homebrewName: SINGBOX_HOMEBREW_NAME,
		serviceName:  SINGBOX_SERVICE_NAME,
	}

	// 根据平台设置路径
	switch runtime.GOOS {
	case "linux":
		m.installDir = "/usr/local/bin"
		m.binaryPath = filepath.Join(m.installDir, "sing-box")
		m.configPath = "/etc/sing-box/config.json"
	case "darwin":
		// macOS默认使用Homebrew管理
		m.binaryPath = "/opt/homebrew/bin/sing-box" // Apple Silicon
		if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
			m.binaryPath = "/usr/local/bin/sing-box" // Intel
		}
		m.configPath = "/usr/local/etc/sing-box/config.json"
	case "windows":
		m.installDir = "C:\\Program Files\\sing-box"
		m.binaryPath = filepath.Join(m.installDir, "sing-box.exe")
		m.configPath = filepath.Join(m.installDir, "config.json")
	default:
		m.installDir = "/usr/local/bin"
		m.binaryPath = filepath.Join(m.installDir, "sing-box")
		m.configPath = "/etc/sing-box/config.json"
	}

	return m
}

// GetInstalledVersion 获取已安装的sing-box版本
func (m *SingboxManager) GetInstalledVersion() (*SingboxVersion, error) {
	if !m.IsInstalled() {
		return nil, fmt.Errorf("sing-box未安装")
	}

	cmd := exec.Command(m.binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取版本信息失败: %v", err)
	}

	// 解析版本输出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				version := strings.TrimSpace(parts[len(parts)-1])
				return &SingboxVersion{
					Version: version,
					Date:    time.Now().Format("2006-01-02"),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("无法解析版本信息")
}

// GetLatestVersion 获取最新版本信息
func (m *SingboxManager) GetLatestVersion() (*SingboxRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", SINGBOX_GITHUB_REPO)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	logger.Debug("正在请求GitHub API: %s", url)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法连接到GitHub API，请检查网络连接: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		// 正常
	case 403:
		return nil, fmt.Errorf("GitHub API访问受限（状态码: %d），请稍后再试或检查网络代理设置", resp.StatusCode)
	case 404:
		return nil, fmt.Errorf("找不到sing-box项目，可能GitHub仓库地址已更改")
	default:
		return nil, fmt.Errorf("GitHub API返回错误状态: %d，请稍后再试", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取GitHub API响应失败: %v", err)
	}

	var release SingboxRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析GitHub API响应失败，数据格式可能已更改: %v", err)
	}

	// 验证响应数据的完整性
	if release.TagName == "" {
		return nil, fmt.Errorf("GitHub API响应数据不完整，缺少版本标签")
	}

	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("GitHub API响应中没有找到下载资源")
	}

	logger.Debug("成功获取版本信息: %s，包含 %d 个资源", release.TagName, len(release.Assets))
	return &release, nil
}

// IsInstalled 检查sing-box是否已安装
func (m *SingboxManager) IsInstalled() bool {
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		// macOS优先检查Homebrew
		return m.isHomebrewPackageInstalled()
	}

	// 检查二进制文件是否存在
	_, err := os.Stat(m.binaryPath)
	return err == nil
}

// IsUpdateAvailable 检查是否有可用更新
func (m *SingboxManager) IsUpdateAvailable() (bool, *SingboxRelease, error) {
	latest, err := m.GetLatestVersion()
	if err != nil {
		return false, nil, err
	}

	if !m.IsInstalled() {
		return true, latest, nil
	}

	current, err := m.GetInstalledVersion()
	if err != nil {
		return true, latest, nil
	}

	// 比较版本
	currentVersion := strings.TrimPrefix(current.Version, "v")
	latestVersion := strings.TrimPrefix(latest.TagName, "v")

	return currentVersion != latestVersion, latest, nil
}

// InstallOrUpdate 安装或更新sing-box
func (m *SingboxManager) InstallOrUpdate() error {
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		return m.installViaHomebrew()
	}

	return m.installFromGitHub()
}

// installViaHomebrew 通过Homebrew安装
func (m *SingboxManager) installViaHomebrew() error {
	logger.Info("🍺 检测到Homebrew，使用Homebrew管理sing-box...")

	// 检查是否已安装
	if m.isHomebrewPackageInstalled() {
		logger.Info("正在更新sing-box...")
		if err := m.runHomebrewCommand("upgrade", SINGBOX_HOMEBREW_NAME); err != nil {
			// 如果升级失败，可能是因为没有新版本
			if strings.Contains(err.Error(), "already installed") ||
				strings.Contains(err.Error(), "already up-to-date") {
				logger.Info("✅ sing-box已是最新版本")
				return nil
			}
			return fmt.Errorf("Homebrew更新失败: %v", err)
		}
		logger.Info("✅ sing-box更新成功！")
		return nil
	} else {
		logger.Info("正在安装sing-box...")
		if err := m.runHomebrewCommand("install", SINGBOX_HOMEBREW_NAME); err != nil {
			return fmt.Errorf("Homebrew安装失败: %v\n提示：请检查Homebrew是否正常工作，或尝试手动运行 'brew install sing-box'", err)
		}
		logger.Info("✅ sing-box安装成功！")
		return nil
	}
}

// installFromGitHub 从GitHub下载安装
func (m *SingboxManager) installFromGitHub() error {
	logger.Info("从GitHub下载最新版本...")

	release, err := m.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("获取最新版本失败: %v", err)
	}

	logger.Info("最新版本: %s (发布时间: %s)", release.TagName, release.PublishedAt)

	// 获取下载URL
	downloadURL, err := m.getDownloadURL(release)
	if err != nil {
		return fmt.Errorf("获取下载链接失败: %v", err)
	}

	logger.Info("下载地址: %s", downloadURL)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "singbox_download_*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 确保临时目录被清理
	defer func() {
		logger.Debug("清理临时目录: %s", tempDir)
		if err := os.RemoveAll(tempDir); err != nil {
			logger.Warn("清理临时目录失败: %v", err)
		}
	}()

	// 使用util包的下载功能
	downloadConfig := util.DownloadConfig{
		URL:          downloadURL,
		DestDir:      tempDir,
		Timeout:      10 * time.Minute,
		MaxRetries:   3,
		ShowProgress: true,
	}

	result, err := util.DownloadFile(downloadConfig)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}

	// 使用util包的解压功能
	binaryPath, err := util.ExtractSingboxBinary(result.FilePath, filepath.Join(tempDir, "extracted"))
	if err != nil {
		return fmt.Errorf("解压失败: %v\n提示：下载的文件可能已损坏，请重试", err)
	}

	// 安装二进制文件
	if err := m.installBinary(binaryPath); err != nil {
		return fmt.Errorf("安装失败: %v\n提示：请检查是否有足够的权限", err)
	}

	logger.Info("✅ sing-box %s 安装成功！", release.TagName)
	return nil
}

// getDownloadURL 获取当前平台的下载链接
func (m *SingboxManager) getDownloadURL(release *SingboxRelease) (string, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// 平台映射
	var platformName string
	switch osName {
	case "darwin":
		platformName = "darwin"
	case "linux":
		platformName = "linux"
	case "windows":
		platformName = "windows"
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", osName)
	}

	// 架构映射
	var archSuffix string
	switch archName {
	case "amd64":
		archSuffix = "amd64"
	case "arm64":
		archSuffix = "arm64"
	case "386":
		archSuffix = "386"
	default:
		return "", fmt.Errorf("不支持的架构: %s", archName)
	}

	// 查找匹配的资产
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, platformName) && strings.Contains(name, archSuffix) {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("未找到适合 %s/%s 的安装包", osName, archName)
}

// installBinary 安装二进制文件
func (m *SingboxManager) installBinary(srcPath string) error {
	logger.Info("安装sing-box到: %s", m.binaryPath)

	// 确保安装目录存在
	if err := os.MkdirAll(filepath.Dir(m.binaryPath), 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %v", err)
	}

	// 备份现有文件
	if _, err := os.Stat(m.binaryPath); err == nil {
		backupPath := m.binaryPath + ".backup"
		if err := util.CopyFile(m.binaryPath, backupPath); err != nil {
			logger.Warn("备份现有文件失败: %v", err)
		} else {
			logger.Info("已备份现有文件到: %s", backupPath)
		}
	}

	// 复制新文件
	if err := util.CopyFile(srcPath, m.binaryPath); err != nil {
		return fmt.Errorf("复制文件失败: %v", err)
	}

	// 设置执行权限
	if runtime.GOOS != "windows" {
		if err := os.Chmod(m.binaryPath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
	}

	logger.Info("sing-box安装成功!")
	return nil
}

// isHomebrewPackageInstalled 检查Homebrew包是否安装
func (m *SingboxManager) isHomebrewPackageInstalled() bool {
	cmd := exec.Command("brew", "list", SINGBOX_HOMEBREW_NAME)
	return cmd.Run() == nil
}

// runHomebrewCommand 执行Homebrew命令
func (m *SingboxManager) runHomebrewCommand(action, formula string) error {
	cmd := exec.Command("brew", action, formula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CheckAndInstall 检查并安装/更新sing-box
func (m *SingboxManager) CheckAndInstall() error {
	logger.Info("🔍 检查sing-box状态...")

	// 检查权限
	if err := m.checkPrivileges(); err != nil {
		return fmt.Errorf("❌ 权限检查失败: %v", err)
	}

	isInstalled := m.IsInstalled()
	var currentVersion string

	if isInstalled {
		current, err := m.GetInstalledVersion()
		if err == nil {
			currentVersion = current.Version
			logger.Info("📦 当前版本: %s", currentVersion)
		} else {
			logger.Warn("⚠️  无法获取当前版本信息: %v", err)
		}
	} else {
		logger.Info("❌ sing-box未安装")
	}

	logger.Info("🌐 正在检查最新版本...")
	hasUpdate, latest, err := m.IsUpdateAvailable()
	if err != nil {
		return fmt.Errorf("❌ 检查更新失败: %v\n💡 提示：请检查网络连接或稍后再试", err)
	}

	if !hasUpdate && isInstalled {
		logger.Info("✅ sing-box已是最新版本 (%s)", currentVersion)
		return nil
	}

	// 显示版本对比信息
	if isInstalled {
		logger.Info("🆕 发现新版本!")
		logger.Info("   当前版本: %s", currentVersion)
		logger.Info("   最新版本: %s", latest.TagName)
		logger.Info("   发布时间: %s", latest.PublishedAt)
		fmt.Printf("\n是否更新到最新版本? [y/N]: ")
	} else {
		logger.Info("📋 最新版本信息:")
		logger.Info("   版本: %s", latest.TagName)
		logger.Info("   发布时间: %s", latest.PublishedAt)
		if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
			logger.Info("   安装方式: Homebrew")
		} else {
			logger.Info("   安装方式: GitHub直接下载")
		}
		fmt.Printf("\n是否安装sing-box? [y/N]: ")
	}

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice != "y" && choice != "yes" {
		logger.Info("🚫 用户取消操作")
		return nil
	}

	logger.Info("🚀 开始安装/更新过程...")
	return m.InstallOrUpdate()
}

// checkPrivileges 检查是否有足够的权限进行安装
func (m *SingboxManager) checkPrivileges() error {
	// macOS使用Homebrew时不需要特殊权限检查
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		return nil
	}

	// 使用util包的权限检查
	checker := util.NewPermissionChecker(filepath.Dir(m.binaryPath))
	return checker.CheckInstallPermissions()
}

// GetBinaryPath 获取二进制文件路径
func (m *SingboxManager) GetBinaryPath() string {
	return m.binaryPath
}

// GetConfigPath 获取配置文件路径
func (m *SingboxManager) GetConfigPath() string {
	return m.configPath
}