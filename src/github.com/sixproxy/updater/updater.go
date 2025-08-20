package updater

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger" // Assuming this is the correct import path
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

	// 3. 解压压缩包并获取二进制文件路径
	newBinaryPath, err := u.extractBinary(archivePath)
	if err != nil {
		return fmt.Errorf("解压更新文件失败: %v", err)
	}

	// 4. 验证下载的文件
	if err := u.validateBinary(newBinaryPath); err != nil {
		return fmt.Errorf("验证下载文件失败: %v", err)
	}

	// 5. 备份当前程序
	backupPath, err := u.backupCurrentBinary()
	if err != nil {
		return fmt.Errorf("备份当前程序失败: %v", err)
	}

	// 6. 替换程序
	if err := u.replaceCurrentBinary(newBinaryPath); err != nil {
		// 如果替换失败，尝试恢复备份
		logger.Error("替换程序失败，尝试恢复备份: %v", err)
		if restoreErr := u.restoreBackup(backupPath); restoreErr != nil {
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

// extractBinary 解压压缩包并返回二进制文件路径
func (u *Updater) extractBinary(archivePath string) (string, error) {
	logger.Info("正在解压更新文件...")

	// 创建解压目录
	extractDir := filepath.Join(u.tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %v", err)
	}

	var binaryPath string
	var err error

	if strings.HasSuffix(archivePath, ".zip") {
		binaryPath, err = u.extractZip(archivePath, extractDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") {
		binaryPath, err = u.extractTarGz(archivePath, extractDir)
	} else {
		return "", fmt.Errorf("不支持的压缩格式")
	}

	if err != nil {
		return "", err
	}

	logger.Info("解压完成，二进制文件路径: %s", binaryPath)
	return binaryPath, nil
}

// extractZip 解压ZIP文件
func (u *Updater) extractZip(zipPath, destDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var binaryPath string

	for _, file := range reader.File {
		// 跳过目录
		if file.FileInfo().IsDir() {
			continue
		}

		// 检查是否是二进制文件（支持嵌套路径）
		fileName := filepath.Base(file.Name)
		if fileName == "sub.exe" || fileName == "sub" {
			// 打开压缩文件中的文件
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			// 创建目标文件
			binaryPath = filepath.Join(destDir, fileName)
			outFile, err := os.OpenFile(binaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			// 复制文件内容
			_, err = io.Copy(outFile, rc)
			if err != nil {
				return "", err
			}

			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("在ZIP文件中找不到二进制文件")
	}

	return binaryPath, nil
}

// extractTarGz 解压tar.gz文件
func (u *Updater) extractTarGz(tarPath, destDir string) (string, error) {
	file, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var binaryPath string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// 跳过目录
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// 检查是否是二进制文件（支持嵌套路径，并检查 sub 和 sub.exe）
		fileName := filepath.Base(header.Name)
		if fileName == "sub" || fileName == "sub.exe" {
			// 创建目标文件
			binaryPath = filepath.Join(destDir, fileName)
			outFile, err := os.OpenFile(binaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			// 复制文件内容
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return "", err
			}

			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("在tar.gz文件中找不到二进制文件")
	}

	return binaryPath, nil
}

// validateBinary 验证下载的二进制文件
func (u *Updater) validateBinary(binaryPath string) error {
	logger.Info("验证下载的文件...")

	// 检查文件是否存在
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("下载的文件不存在: %v", err)
	}

	// 设置执行权限 (Unix系统)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
	}

	logger.Info("文件验证通过")
	return nil
}

// backupCurrentBinary 备份当前程序
func (u *Updater) backupCurrentBinary() (string, error) {
	logger.Info("备份当前程序...")

	backupPath := u.currentExePath + ".backup"

	// 如果备份文件已存在，先删除
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return "", fmt.Errorf("删除旧备份文件失败: %v", err)
		}
	}

	// 复制当前程序到备份位置
	if err := u.copyFile(u.currentExePath, backupPath); err != nil {
		return "", fmt.Errorf("备份文件失败: %v", err)
	}

	logger.Info("程序已备份到: %s", backupPath)
	return backupPath, nil
}

// replaceCurrentBinary 替换当前程序
func (u *Updater) replaceCurrentBinary(newBinaryPath string) error {
	logger.Info("正在替换程序...")

	if runtime.GOOS == "windows" {
		return u.replaceOnWindows(newBinaryPath)
	} else {
		return u.replaceOnUnix(newBinaryPath)
	}
}

// replaceOnWindows Windows平台的替换策略
func (u *Updater) replaceOnWindows(newBinaryPath string) error {
	// Windows下将当前exe重命名为.old
	tempOldPath := u.currentExePath + ".old"
	if err := os.Rename(u.currentExePath, tempOldPath); err != nil {
		return fmt.Errorf("重命名当前程序失败: %v", err)
	}

	// 复制新程序到原位置
	if err := u.copyFile(newBinaryPath, u.currentExePath); err != nil {
		// 恢复原文件
		os.Rename(tempOldPath, u.currentExePath)
		return fmt.Errorf("复制新程序失败: %v", err)
	}

	// 删除临时文件（可能会失败，移到启动时清理）
	logger.Info("旧程序文件 %s 将在下次启动时自动清理", tempOldPath)

	logger.Info("程序替换完成")
	return nil
}

// replaceOnUnix Unix平台的替换策略
func (u *Updater) replaceOnUnix(newBinaryPath string) error {
	// 尝试直接替换
	err := u.copyFile(newBinaryPath, u.currentExePath)
	if err == nil {
		// 设置执行权限
		if err := os.Chmod(u.currentExePath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
		logger.Info("程序替换完成")
		return nil
	} else if strings.Contains(err.Error(), "text file busy") || errors.Is(err, os.ErrPermission) {
		logger.Warn("程序正在运行，使用重命名策略进行更新...")
		return u.replaceWithRename(newBinaryPath)
	}
	return fmt.Errorf("复制新程序失败: %v", err)
}

// replaceWithRename 使用重命名策略替换程序
func (u *Updater) replaceWithRename(newBinaryPath string) error {
	// 1. 将当前程序重命名为.old
	oldPath := u.currentExePath + ".old"
	if err := os.Rename(u.currentExePath, oldPath); err != nil {
		return fmt.Errorf("重命名当前程序失败: %v", err)
	}

	// 2. 复制新程序到原位置
	if err := u.copyFile(newBinaryPath, u.currentExePath); err != nil {
		// 恢复原文件
		os.Rename(oldPath, u.currentExePath)
		return fmt.Errorf("复制新程序失败: %v", err)
	}

	// 3. 设置执行权限
	if err := os.Chmod(u.currentExePath, 0755); err != nil {
		logger.Warn("设置执行权限失败: %v", err)
	}

	// 4. 旧文件将在下次启动时清理
	logger.Info("旧程序文件 %s 将在下次启动时自动清理", oldPath)
	logger.Info("程序替换完成（使用重命名策略）")
	logger.Info("建议重启程序以确保更新完全生效")

	return nil
}

// restoreBackup 恢复备份
func (u *Updater) restoreBackup(backupPath string) error {
	logger.Info("正在恢复备份...")

	if err := u.copyFile(backupPath, u.currentExePath); err != nil {
		return fmt.Errorf("恢复备份失败: %v", err)
	}

	// Unix系统设置执行权限
	if runtime.GOOS != "windows" {
		if err := os.Chmod(u.currentExePath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
	}

	logger.Info("备份已恢复")
	return nil
}

// copyFile 复制文件
func (u *Updater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// Cleanup 清理临时文件
func (u *Updater) Cleanup() {
	if u.tempDir != "" {
		os.RemoveAll(u.tempDir)
	}
}
