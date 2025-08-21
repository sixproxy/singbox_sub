package service

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
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/util"
	"singbox_sub/src/github.com/sixproxy/util/comp"
	"singbox_sub/src/github.com/sixproxy/util/files"
	http2 "singbox_sub/src/github.com/sixproxy/util/https"
	"singbox_sub/src/github.com/sixproxy/util/shells"
	"strings"
	"time"
)

// sing-box外部程序管理器
type SingBoxService struct {
	installDir   string
	binaryPath   string
	configPath   string
	serviceName  string
	homebrewName string
	MirrorURL    string
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

// NewSingboxService 创建新的sing-box管理器
func NewSingboxService(mirrorURL string) *SingBoxService {
	m := &SingBoxService{
		homebrewName: SINGBOX_HOMEBREW_NAME,
		serviceName:  SINGBOX_SERVICE_NAME,
		MirrorURL:    mirrorURL,
	}

	// 首先尝试检测用户实际安装的sing-box路径
	m.detectSingboxPath()

	return m
}

// detectSingboxPath 检测sing-box实际安装路径
func (m *SingBoxService) detectSingboxPath() {
	// 1. 优先检查PATH中的sing-box
	if pathBinary, err := exec.LookPath("sing-box"); err == nil {
		m.binaryPath = pathBinary
		m.installDir = filepath.Dir(pathBinary)
		m.setConfigPath()
		logger.Debug("在PATH中找到sing-box: %s", pathBinary)
		return
	}

	// 2. 检查macOS Homebrew路径
	if runtime.GOOS == "darwin" {
		homebrewPaths := []string{
			"/opt/homebrew/bin/sing-box", // Apple Silicon
			"/usr/local/bin/sing-box",    // Intel
		}

		for _, path := range homebrewPaths {
			if _, err := os.Stat(path); err == nil {
				m.binaryPath = path
				m.installDir = filepath.Dir(path)
				m.setConfigPath()
				logger.Debug("找到Homebrew安装的sing-box: %s", path)
				return
			}
		}
	}

	// 3. 检查其他常见安装位置
	commonPaths := m.getCommonInstallPaths()
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			m.binaryPath = path
			m.installDir = filepath.Dir(path)
			m.setConfigPath()
			logger.Debug("找到sing-box: %s", path)
			return
		}
	}

	// 4. 如果都没找到，使用默认路径
	m.setDefaultPaths()
	logger.Debug("未找到现有安装，使用默认路径: %s", m.binaryPath)
}

// getCommonInstallPaths 获取常见的安装路径
func (m *SingBoxService) getCommonInstallPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/local/bin/sing-box",
			"/usr/bin/sing-box",
			"/opt/sing-box/sing-box",
			filepath.Join(os.Getenv("HOME"), ".local/bin/sing-box"),
		}
	case "darwin":
		paths = []string{
			"/usr/local/bin/sing-box",
			"/opt/homebrew/bin/sing-box",
			"/opt/local/bin/sing-box",
			filepath.Join(os.Getenv("HOME"), ".local/bin/sing-box"),
		}
	case "windows":
		programFiles := os.Getenv("PROGRAMFILES")
		if programFiles == "" {
			programFiles = "C:\\Program Files"
		}
		programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
		if programFilesX86 == "" {
			programFilesX86 = "C:\\Program Files (x86)"
		}

		paths = []string{
			filepath.Join(programFiles, "sing-box", "sing-box.exe"),
			filepath.Join(programFilesX86, "sing-box", "sing-box.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "sing-box", "sing-box.exe"),
		}
	}

	return paths
}

// setDefaultPaths 设置默认路径
func (m *SingBoxService) setDefaultPaths() {
	switch runtime.GOOS {
	case "linux":
		m.installDir = "/usr/local/bin"
		m.binaryPath = filepath.Join(m.installDir, "sing-box")
	case "darwin":
		// macOS优先使用Homebrew路径
		if util.IsHomebrewInstalled() {
			m.binaryPath = "/opt/homebrew/bin/sing-box" // Apple Silicon
			if _, err := os.Stat("/usr/local/bin/brew"); err == nil {
				m.binaryPath = "/usr/local/bin/sing-box" // Intel
			}
		} else {
			m.installDir = "/usr/local/bin"
			m.binaryPath = filepath.Join(m.installDir, "sing-box")
		}
	case "windows":
		m.installDir = "C:\\Program Files\\sing-box"
		m.binaryPath = filepath.Join(m.installDir, "sing-box.exe")
	default:
		m.installDir = "/usr/local/bin"
		m.binaryPath = filepath.Join(m.installDir, "sing-box")
	}

	m.setConfigPath()
}

// setConfigPath 根据二进制路径设置配置路径
func (m *SingBoxService) setConfigPath() {
	switch runtime.GOOS {
	case "linux":
		m.configPath = "/etc/sing-box/config.json"
	case "darwin":
		if strings.Contains(m.binaryPath, "homebrew") {
			if strings.Contains(m.binaryPath, "/opt/homebrew/") {
				m.configPath = "/opt/homebrew/etc/sing-box/config.json"
			} else {
				m.configPath = "/usr/local/etc/sing-box/config.json"
			}
		} else {
			m.configPath = "/usr/local/etc/sing-box/config.json"
		}
	case "windows":
		m.configPath = filepath.Join(filepath.Dir(m.binaryPath), "config.json")
	default:
		m.configPath = "/etc/sing-box/config.json"
	}
}

// GetInstalledVersion 获取已安装的sing-box版本
func (m *SingBoxService) GetInstalledVersion() (*SingboxVersion, error) {
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
func (m *SingBoxService) GetLatestVersion() (*SingboxRelease, error) {

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", SINGBOX_GITHUB_REPO)

	logger.Debug("正在请求GitHub API: %s", url)

	// 临时禁用镜像功能，直接使用原始API
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
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

	// 转换下载URL为镜像URL
	for i := range release.Assets {
		if m.MirrorURL != "" {
			release.Assets[i].BrowserDownloadURL = m.convertToMirrorURL(release.Assets[i].BrowserDownloadURL, m.MirrorURL)
		}
	}

	logger.Debug("成功获取版本信息: %s，包含 %d 个资源", release.TagName, len(release.Assets))
	return &release, nil
}

// fetchWithMirror 使用镜像获取数据
func (m *SingBoxService) fetchWithMirror(url string, githubConfig *model.GitHubConfig) (*http.Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var lastErr error

	// 尝试使用主要镜像
	if githubConfig.MirrorURL != "" {
		mirrorURL := m.convertToMirrorURL(url, githubConfig.MirrorURL)
		logger.Debug("尝试使用主镜像: %s", mirrorURL)

		resp, err := client.Get(mirrorURL)
		if err == nil && resp.StatusCode < 400 {
			logger.Debug("使用主镜像成功")
			return resp, nil
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			lastErr = err
			logger.Debug("主镜像访问失败: %v", err)
		}
	}

	// 尝试备用镜像
	if len(githubConfig.FallbackMirrors) > 0 {
		for _, mirror := range githubConfig.FallbackMirrors {
			mirrorURL := m.convertToMirrorURL(url, mirror)
			logger.Debug("尝试使用备用镜像: %s", mirrorURL)

			resp, err := client.Get(mirrorURL)
			if err == nil && resp.StatusCode < 400 {
				logger.Debug("使用备用镜像成功: %s", mirror)
				return resp, nil
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				lastErr = err
				logger.Debug("备用镜像 %s 访问失败: %v", mirror, err)
			}
		}
	}

	// 最后尝试原始URL
	logger.Debug("尝试使用原始GitHub API: %s", url)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("所有镜像都失败，原始URL也失败: %v (最后一个镜像错误: %v)", err, lastErr)
	}

	return resp, nil
}

// convertToMirrorURL 转换下载URL为镜像URL
func (m *SingBoxService) convertToMirrorURL(originalURL, mirrorBase string) string {
	if strings.HasSuffix(mirrorBase, "/") {
		return mirrorBase + originalURL
	} else {
		return mirrorBase + "/" + originalURL
	}
}

// IsInstalled 检查sing-box是否已安装
func (m *SingBoxService) IsInstalled() bool {
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		// macOS优先检查Homebrew
		return m.isHomebrewPackageInstalled()
	}

	// 检查二进制文件是否存在
	_, err := os.Stat(m.binaryPath)
	return err == nil
}

// IsUpdateAvailable 检查是否有可用更新
func (m *SingBoxService) IsUpdateAvailable() (bool, *SingboxRelease, error) {
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
func (m *SingBoxService) InstallOrUpdate() error {
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		return m.installViaHomebrew()
	}

	return m.installFromGitHub()
}

// installViaHomebrew 通过Homebrew安装
func (m *SingBoxService) installViaHomebrew() error {
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
func (m *SingBoxService) installFromGitHub() error {
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
	downloadConfig := http2.DownloadConfig{
		URL:          downloadURL,
		DestDir:      tempDir,
		Timeout:      10 * time.Minute,
		MaxRetries:   3,
		ShowProgress: true,
	}

	result, err := http2.DownloadFile(downloadConfig)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}

	// 使用util包的解压功能
	tmpBinaryPath, err := comp.ExtractSingboxBinary(result.FilePath, filepath.Join(tempDir, "extracted"))
	if err != nil {
		return fmt.Errorf("解压失败: %v\n提示：下载的文件可能已损坏，请重试", err)
	}

	// 安装二进制文件
	if err := m.installBinary(tmpBinaryPath); err != nil {
		return fmt.Errorf("安装失败: %v\n提示：请检查是否有足够的权限", err)
	}

	logger.Info("✅ sing-box %s 安装成功！", release.TagName)
	return nil
}

// getDownloadURL 获取当前平台的下载链接
func (m *SingBoxService) getDownloadURL(release *SingboxRelease) (string, error) {
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
func (m *SingBoxService) installBinary(srcPath string) error {
	logger.Info("安装sing-box到: %s", m.binaryPath)

	// 确保安装目录存在
	err := files.ReplaceBinary(srcPath, m.binaryPath)
	if err != nil {
		logger.Error("安装失败: %v", err)
		return err
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
func (m *SingBoxService) isHomebrewPackageInstalled() bool {
	cmd := exec.Command("brew", "list", SINGBOX_HOMEBREW_NAME)
	return cmd.Run() == nil
}

// runHomebrewCommand 执行Homebrew命令
func (m *SingBoxService) runHomebrewCommand(action, formula string) error {
	cmd := exec.Command("brew", action, formula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CheckAndInstall 检查并安装/更新sing-box
func (m *SingBoxService) CheckAndInstall() error {
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
func (m *SingBoxService) checkPrivileges() error {
	// macOS使用Homebrew时不需要特殊权限检查
	if runtime.GOOS == "darwin" && util.IsHomebrewInstalled() {
		return nil
	}

	// 使用util包的权限检查
	checker := util.NewPermissionChecker(filepath.Dir(m.binaryPath))
	return checker.CheckInstallPermissions()
}

// GetBinaryPath 获取二进制文件路径
func (m *SingBoxService) GetBinaryPath() string {
	return m.binaryPath
}

// GetConfigPath 获取配置文件路径
func (m *SingBoxService) GetConfigPath() string {
	return m.configPath
}

func (m *SingBoxService) ShowSingboxStatus() {
	logger.Info("🔍 sing-box状态检查")

	if m.IsInstalled() {
		if version, err := m.GetInstalledVersion(); err == nil {
			logger.Info("✅ 已安装版本: %s", version.Version)
		} else {
			logger.Warn("⚠️ 已安装但无法获取版本: %v", err)
		}

		if hasUpdate, latest, err := m.IsUpdateAvailable(); err == nil {
			if hasUpdate {
				logger.Info("🆕 有新版本可用: %s", latest.TagName)
				logger.Info("💡 运行 './sub box install' 更新")
			} else {
				logger.Info("✅ 已是最新版本")
			}
		} else {
			logger.Warn("⚠️ 检查更新失败: %v", err)
		}
	} else {
		logger.Info("❌ sing-box未安装")
		logger.Info("💡 运行 './sub box install' 安装")
	}
}

func (m *SingBoxService) ShowSingboxVersion() {
	if !m.IsInstalled() {
		logger.Error("❌ sing-box未安装")
		os.Exit(1)
	}

	version, err := m.GetInstalledVersion()
	if err != nil {
		logger.Error("获取版本失败: %v", err)
		os.Exit(1)
	}

	logger.Info("sing-box version %s", version.Version)
	logger.Info("Binary path: %s", m.GetBinaryPath())
	logger.Info("Config path: %s", m.GetConfigPath())
}

func (m *SingBoxService) ShowSingboxFailureReason() {
	logger.Info("🔍 分析启动失败原因...")

	// 1. 检查配置文件语法
	configPath := "/etc/sing-box/config.json"
	if _, err := os.Stat(configPath); err == nil {
		// 尝试使用sing-box检查配置
		if m.IsInstalled() {
			cmd := exec.Command(m.GetBinaryPath(), "check", "-c", configPath)
			output, err := cmd.CombinedOutput()

			if err != nil {
				logger.Error("❌ 配置文件检查失败:")
				logger.Error(string(output))
			} else {
				logger.Info("✅ 配置文件语法正确")
			}
		}
	}

	// 2. 检查系统资源
	logger.Debug("检查系统资源...")

	// 3. 尝试获取系统日志中的错误信息
	if runtime.GOOS == "linux" {
		// 尝试从systemd日志获取错误
		cmd := exec.Command("journalctl", "-u", "sing-box", "--no-pager", "-n", "10")
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			logger.Info("📋 最近的系统日志:")
			logger.Info(string(output))
		}
	}
}

func (m *SingBoxService) HandleStartupFailure(backupPath, configPath string) {
	logger.Error("🚨 sing-box启动失败，开始故障处理...")

	// 1. 显示失败原因（尝试获取服务日志）
	m.ShowSingboxFailureReason()

	// 2. 停止可能存在的异常进程
	shells.StopSingBox()
	time.Sleep(2 * time.Second)

	// 3. 检查是否有备份配置可以回滚
	if _, err := os.Stat(backupPath); err == nil {
		logger.Info("🔄 回滚到之前的配置...")

		if err := files.CopyFile(backupPath, configPath); err != nil {
			logger.Error("回滚配置失败: %v", err)
			return
		}

		logger.Info("配置已回滚，尝试重新启动sing-box...")

		// 4. 尝试使用回滚的配置重新启动
		shell := util.GetAvailableShell()
		if shell != "" {
			scriptPath := "bash/start_singbox.sh"
			cmd := exec.Command(shell, scriptPath)
			output, err := cmd.CombinedOutput()

			if err != nil {
				logger.Error("使用回滚配置启动失败: %v", err)
				logger.Debug("输出: %s", string(output))
			} else {
				logger.Info("正在验证回滚配置启动状态...")
				time.Sleep(3 * time.Second)

				if shells.IsSingBoxRunning() {
					logger.Info("✅ 使用回滚配置成功启动sing-box")
					// 清理失败的配置文件（重命名为.failed）
					failedConfigPath := configPath + ".failed"
					if util.CheckNewConfigIsSameOldConfig(configPath, backupPath) {
						// 只有当新配置与备份配置不同时才保存失败配置
						files.CopyFile(configPath, failedConfigPath)
						logger.Info("失败的配置已保存为: %s", failedConfigPath)
					}
				} else {
					logger.Error("❌ 即使使用回滚配置也无法启动sing-box")
				}
			}
		}
	} else {
		logger.Warn("⚠️  没有找到配置备份，无法自动回滚")
		logger.Info("请手动检查配置文件: %s", configPath)
	}
}

// checkSingboxStartupStatus 检查sing-box启动状态
func (m *SingBoxService) CheckSingboxStartupStatus() bool {
	logger.Info("检查sing-box启动状态...")

	// 等待几秒钟让服务完全启动
	maxWait := 10 * time.Second
	checkInterval := 1 * time.Second
	waited := time.Duration(0)

	for waited < maxWait {
		time.Sleep(checkInterval)
		waited += checkInterval

		// 检查进程是否存在
		if shells.IsSingBoxRunning() {
			logger.Debug("sing-box进程运行中...")

			// 尝试获取版本信息来验证服务状态
			if m.IsInstalled() {
				if version, err := m.GetInstalledVersion(); err == nil {
					logger.Debug("sing-box版本验证成功: %s", version.Version)

					// 额外等待2秒确保服务完全稳定
					time.Sleep(2 * time.Second)

					// 最后检查进程是否仍在运行
					if shells.IsSingBoxRunning() {
						return true
					} else {
						logger.Warn("sing-box进程意外停止")
						return false
					}
				} else {
					logger.Debug("版本验证失败，可能尚未完全启动: %v", err)
				}
			}
		} else {
			logger.Debug("sing-box进程未运行...")
		}
	}

	logger.Error("等待 %.0f 秒后，sing-box仍未成功启动", maxWait.Seconds())
	return false
}
