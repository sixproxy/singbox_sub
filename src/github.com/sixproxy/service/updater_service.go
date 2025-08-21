package service

import (
	"bufio"
	"fmt"
	"os"
	"singbox_sub/src/github.com/sixproxy/logger" // Assuming this is the correct import path
	"singbox_sub/src/github.com/sixproxy/util/files"
	"singbox_sub/src/github.com/sixproxy/util/https"
	"singbox_sub/src/github.com/sixproxy/version"
	"strings"
	"time"
)

// UpdaterService 自动更新器
type UpdaterService struct {
	currentExePath string
	tempDir        string
}

// NewUpdaterService 创建新的更新器
func NewUpdaterService() (*UpdaterService, error) {
	// 获取当前执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("无法获取当前程序路径: %v", err)
	}

	// 清理上次的残留文件
	cleanupLeftovers(exePath)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "singbox_update_*")
	if err != nil {
		return nil, fmt.Errorf("无法创建临时目录: %v", err)
	}

	return &UpdaterService{
		currentExePath: exePath,
		tempDir:        tempDir,
	}, nil
}

// cleanupLeftovers 在启动时清理残留的 .old 和 .backup 文件
func cleanupLeftovers(exePath string) {
	oldPath := exePath + ".old"
	if err := os.Remove(oldPath); err == nil {
		logger.Info("已清理残留的旧程序文件: %s", oldPath)
	} else if !os.IsNotExist(err) {
		logger.Warn("清理残留旧文件失败: %v", err)
	}
}

// CheckUpdate 检查更新
func (u *UpdaterService) CheckUpdate() error {
	logger.Info("正在检查更新...")

	hasUpdate, release, err := version.IsUpdateAvailable()
	if err != nil {
		return fmt.Errorf("检查更新失败: %v", err)
	}

	if !hasUpdate {
		logger.Info("当前已是最新版本 v%s", version.VERSION)
		return nil
	}

	logger.Info("发现新版本: %s -> %s", version.VERSION, release.TagName)
	logger.Info("发布时间: %s", release.PublishedAt)

	// 询问用户是否要更新，使用 bufio 以处理输入
	fmt.Printf("发现新版本 %s，是否立即更新? (y/n): ", release.TagName)
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice != "y" && choice != "yes" {
		logger.Info("用户取消更新")
		return nil
	}

	return u.performUpdate(release)
}

// AutoUpdate 自动更新
func (u *UpdaterService) AutoUpdate() error {
	logger.Info("开始自动更新...")

	hasUpdate, release, err := version.IsUpdateAvailable()
	if err != nil {
		return fmt.Errorf("检查更新失败: %v", err)
	}

	if !hasUpdate {
		logger.Info("当前已是最新版本 v%s", version.VERSION)
		return nil
	}

	logger.Info("发现新版本: %s -> %s，开始自动更新...", version.VERSION, release.TagName)
	return u.performUpdate(release)
}

// performUpdate 执行更新
func (u *UpdaterService) performUpdate(release *version.GitHubRelease) error {
	// 清理临时目录
	defer func() {
		if err := os.RemoveAll(u.tempDir); err != nil {
			logger.Warn("清理临时目录失败: %v", err)
		}
	}()

	// 1. 获取下载链接
	downloadURL, err := version.FindDownloadURL(release)
	if err != nil {
		return fmt.Errorf("获取下载链接失败: %v", err)
	}

	logger.Info("下载地址: %s", downloadURL)

	// 2. 下载新版本
	result, err := https.DownloadFile(https.DownloadConfig{
		URL:        downloadURL,
		MaxRetries: 3,
		Timeout:    3 * time.Minute,
		DestDir:    u.tempDir,
	})
	if err != nil {
		return fmt.Errorf("下载更新失败: %v", err)
	}

	// 3. 解压压缩包并获取二进制文件路径
	tmpBinaryPath, err := files.ExtractBinary(result.FilePath, u.tempDir, "sub")
	if err != nil {
		return fmt.Errorf("解压更新文件失败: %v", err)
	}

	// 4. 替换程序
	if err := files.ReplaceBinary(tmpBinaryPath, u.currentExePath); err != nil {
		return fmt.Errorf("替换程序失败: %v", err)
	}

	logger.Info("更新成功！程序已更新到版本 %s", release.TagName)
	logger.Info("请重启程序以使用新版本")

	return nil
}

func (u *UpdaterService) Cleanup() {
	files.Cleanup(u.tempDir)
}
