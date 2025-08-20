package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

// 版本信息变量 - 可以通过构建时的ldflags注入
var (
	// 当前版本号 - 每次发布时需要手动更新，也可以通过构建时注入
	VERSION = "1.0.0"
	// 构建时间 - 通过构建时注入
	buildTime = "unknown"
)

const (
	// GitHub仓库信息
	GITHUB_REPO = "sixproxy/singbox_sub"
	// 程序名称
	APP_NAME = "singbox_sub"
)

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GitHubRelease GitHub Release信息结构体
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Size        int    `json:"size"`
	} `json:"assets"`
	PublishedAt string `json:"published_at"`
}

// GetVersionInfo 获取当前版本信息
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   VERSION,
		BuildTime: getBuildTime(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// PrintVersion 打印版本信息
func PrintVersion() {
	info := GetVersionInfo()
	fmt.Printf("singbox_sub version %s\n", info.Version)
	fmt.Printf("  Go version: %s\n", info.GoVersion)
	fmt.Printf("  OS/Arch: %s/%s\n", info.OS, info.Arch)
	fmt.Printf("  Build time: %s\n", info.BuildTime)
}

// CheckLatestVersion 检查最新版本
func CheckLatestVersion() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GITHUB_REPO)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取最新版本信息失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API返回错误状态: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %v", err)
	}
	
	return &release, nil
}

// CompareVersions 比较版本号，如果远程版本更新返回true
func CompareVersions(currentVersion, remoteVersion string) bool {
	// 简单的版本比较，实际项目中可能需要更复杂的语义化版本比较
	return currentVersion != remoteVersion
}

// IsUpdateAvailable 检查是否有可用更新
func IsUpdateAvailable() (bool, *GitHubRelease, error) {
	latest, err := CheckLatestVersion()
	if err != nil {
		return false, nil, err
	}
	
	// 移除版本号中的 'v' 前缀进行比较
	latestVersion := latest.TagName
	if len(latestVersion) > 0 && latestVersion[0] == 'v' {
		latestVersion = latestVersion[1:]
	}
	
	hasUpdate := CompareVersions(VERSION, latestVersion)
	return hasUpdate, latest, nil
}

// getBuildTime 获取构建时间
func getBuildTime() string {
	// 构建时间通过ldflags注入
	// go build -ldflags "-X singbox_sub/src/github.com/sixproxy/version.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	return buildTime
}

// GetPlatformAssetName 根据当前平台获取对应的资源文件名
func GetPlatformAssetName() string {
	osName := runtime.GOOS
	archName := runtime.GOARCH
	
	// 根据不同平台返回不同的压缩包名格式，匹配GitHub Actions生成的文件
	switch osName {
	case "windows":
		return fmt.Sprintf("sub-%s-%s.zip", osName, archName)
	case "darwin", "linux", "freebsd":
		return fmt.Sprintf("sub-%s-%s.tar.gz", osName, archName)
	default:
		return fmt.Sprintf("sub-%s-%s.tar.gz", osName, archName)
	}
}

// FindDownloadURL 从release中找到当前平台的下载链接
func FindDownloadURL(release *GitHubRelease) (string, error) {
	targetAsset := GetPlatformAssetName()
	
	for _, asset := range release.Assets {
		if asset.Name == targetAsset {
			return asset.DownloadURL, nil
		}
	}
	
	return "", fmt.Errorf("未找到适用于 %s/%s 平台的安装包", runtime.GOOS, runtime.GOARCH)
}