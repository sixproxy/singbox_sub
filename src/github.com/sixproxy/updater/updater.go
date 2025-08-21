package updater

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"singbox_sub/src/github.com/sixproxy/logger" // Assuming this is the correct import path
	"singbox_sub/src/github.com/sixproxy/util"
	"singbox_sub/src/github.com/sixproxy/version"
	"strings"
	"time"
)

// Updater 自动更新器
type Updater struct {
	currentExePath string
	tempDir        string
}

// NewUpdater 创建新的更新器
func NewUpdater() (*Updater, error) {
	// 获取当前执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("无法获取当前程序路径: %v", err)
	}

	// 清理上次的残留文件（.old 和 .backup）
	cleanupLeftovers(exePath)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "singbox_update_*")
	if err != nil {
		return nil, fmt.Errorf("无法创建临时目录: %v", err)
	}

	return &Updater{
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

	backupPath := exePath + ".backup"
	if err := os.Remove(backupPath); err == nil {
		logger.Info("已清理残留的备份文件: %s", backupPath)
	} else if !os.IsNotExist(err) {
		logger.Warn("清理残留备份文件失败: %v", err)
	}
}

// CheckUpdate 检查更新
func (u *Updater) CheckUpdate() error {
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
func (u *Updater) AutoUpdate() error {
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
func (u *Updater) performUpdate(release *version.GitHubRelease) error {
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

	// 2. 下载新版本（添加重试）
	archivePath, err := u.downloadUpdateWithRetry(downloadURL, 3)
	if err != nil {
		return fmt.Errorf("下载更新失败: %v", err)
	}

	fs := &util.Files{}

	// 3. 解压压缩包并获取二进制文件路径
	tmpBinaryPath, err := fs.ExtractBinary(archivePath, u.tempDir)
	if err != nil {
		return fmt.Errorf("解压更新文件失败: %v", err)
	}

	// 5. 备份当前程序
	backupPath, err := fs.BackupFile(u.currentExePath, u.currentExePath+".backup")
	if err != nil {
		return fmt.Errorf("备份当前程序失败: %v", err)
	}

	// 6. 替换程序
	if err := fs.ReplaceBinary(tmpBinaryPath, u.currentExePath); err != nil {
		// 如果替换失败，尝试恢复备份
		logger.Error("替换程序失败，尝试恢复备份: %v", err)
		if restoreErr := fs.RestoreBackup(backupPath, u.currentExePath); restoreErr != nil {
			return fmt.Errorf("替换失败且恢复备份失败: 替换错误=%v, 恢复错误=%v", err, restoreErr)
		}
		return fmt.Errorf("替换程序失败: %v", err)
	}

	// 7. 清理备份文件（但移到启动时清理，以处理锁定）
	logger.Info("备份文件 %s 将在下次启动时自动清理", backupPath)

	logger.Info("更新成功！程序已更新到版本 %s", release.TagName)
	logger.Info("请重启程序以使用新版本")

	return nil
}

// downloadUpdateWithRetry 下载更新压缩包（添加重试）
func (u *Updater) downloadUpdateWithRetry(downloadURL string, retries int) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		logger.Info("正在下载更新文件... (尝试 %d/%d)", attempt, retries)

		// 创建HTTP客户端
		client := &http.Client{
			Timeout: 5 * time.Minute, // 5分钟下载超时
		}

		resp, err := client.Get(downloadURL)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second) // 简单退避
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
			continue
		}

		// 根据URL确定文件扩展名（添加更多检查）
		var filename string
		if strings.Contains(downloadURL, ".zip") {
			filename = "update.zip"
		} else if strings.Contains(downloadURL, ".tar.gz") || strings.Contains(downloadURL, ".tgz") {
			filename = "update.tar.gz"
		} else {
			return "", fmt.Errorf("不支持的下载文件格式: %s", downloadURL)
		}

		// 创建临时文件
		tempFile := filepath.Join(u.tempDir, filename)

		file, err := os.Create(tempFile)
		if err != nil {
			return "", err
		}
		defer file.Close()

		// 显示下载进度
		size := resp.ContentLength
		if size > 0 {
			logger.Info("压缩包大小: %.2f MB", float64(size)/(1024*1024))
		}

		// 复制文件内容
		written, err := io.Copy(file, resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		logger.Info("下载完成，文件大小: %.2f MB", float64(written)/(1024*1024))

		return tempFile, nil
	}
	return "", lastErr
}

func (u *Updater) Cleanup() {
	fs := &util.Files{}
	fs.Cleanup(u.tempDir)
}
