package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
)

type Files struct {
}

// extractBinary 解压压缩包并返回二进制文件路径
func (f *Files) ExtractBinary(archivePath string, tempDir string, targetFile string) (string, error) {
	logger.Info("正在解压更新文件...")

	// 创建解压目录
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %v", err)
	}

	config := ExtractConfig{
		ArchivePath:   archivePath,
		DestDir:       extractDir,
		TargetFiles:   []string{targetFile},
		CreateDestDir: true,
	}

	archive, err := ExtractArchive(config)
	if err != nil {
		return "解压失败", err
	}

	logger.Info("解压完成，二进制文件路径: %s", archive.TargetFile)
	return archive.TargetFile, nil
}

// backupCurrentBinary 备份当前程序
func (f *Files) BackupFile(src string, dst string) (string, error) {
	logger.Info("备份当前程序...")

	backupPath := dst

	// 如果备份文件已存在，先删除
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return "", fmt.Errorf("删除旧备份文件失败: %v", err)
		}
	}

	// 复制当前程序到备份位置
	if err := CopyFile(src, backupPath); err != nil {
		return "", fmt.Errorf("备份文件失败: %v", err)
	}

	logger.Info("程序已备份到: %s", backupPath)
	return backupPath, nil
}

// replaceCurrentBinary 替换当前程序
func (f *Files) ReplaceBinary(srcBinaryPath string, dstBinaryPath string) error {
	logger.Info("正在替换程序...")

	if runtime.GOOS == "windows" {
		return f.replaceOnWindows(srcBinaryPath, dstBinaryPath)
	} else {
		return f.replaceOnUnix(srcBinaryPath, dstBinaryPath)
	}
}

// replaceOnWindows Windows平台的替换策略
func (f *Files) replaceOnWindows(srcBinaryPath string, dstBinaryPath string) error {
	// Windows下将当前exe重命名为.old
	tempOldPath := dstBinaryPath + ".old"

	if err := os.Rename(dstBinaryPath, tempOldPath); err != nil {
		return fmt.Errorf("重命名当前程序失败: %v", err)
	}

	// 复制新程序到原位置
	if err := CopyFile(srcBinaryPath, dstBinaryPath); err != nil {
		// 恢复原文件
		os.Rename(tempOldPath, dstBinaryPath)
		return fmt.Errorf("复制新程序失败: %v", err)
	}

	// 删除临时文件（可能会失败，移到启动时清理）
	logger.Info("旧程序文件 %s 将在下次启动时自动清理", tempOldPath)

	logger.Info("程序替换完成")
	return nil
}

// replaceOnUnix Unix平台的替换策略
func (f *Files) replaceOnUnix(srcBinaryPath string, dstBinaryPath string) error {
	// 尝试直接替换
	err := CopyFile(srcBinaryPath, dstBinaryPath)
	if err == nil {
		// 设置执行权限
		if err := os.Chmod(dstBinaryPath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
		logger.Info("程序替换完成")
		return nil
	} else if strings.Contains(err.Error(), "text file busy") || errors.Is(err, os.ErrPermission) {
		logger.Warn("程序正在运行，使用重命名策略进行更新...")
		return f.replaceWithRename(srcBinaryPath, dstBinaryPath)
	}
	return fmt.Errorf("复制新程序失败: %v", err)
}

// replaceWithRename 使用重命名策略替换程序
func (f *Files) replaceWithRename(srcBinaryPath string, dstBinaryPath string) error {
	// 1. 将当前程序重命名为.old
	oldPath := dstBinaryPath + ".old"
	if err := os.Rename(dstBinaryPath, oldPath); err != nil {
		return fmt.Errorf("重命名当前程序失败: %v", err)
	}

	// 2. 复制新程序到原位置
	if err := CopyFile(srcBinaryPath, dstBinaryPath); err != nil {
		// 恢复原文件
		os.Rename(oldPath, dstBinaryPath)
		return fmt.Errorf("复制新程序失败: %v", err)
	}

	// 3. 设置执行权限
	if err := os.Chmod(dstBinaryPath, 0755); err != nil {
		logger.Warn("设置执行权限失败: %v", err)
	}

	// 4. 旧文件将在下次启动时清理
	logger.Info("旧程序文件 %s 将在下次启动时自动清理", oldPath)
	logger.Info("程序替换完成（使用重命名策略）")
	logger.Info("建议重启程序以确保更新完全生效")

	return nil
}

// restoreBackup 恢复备份
func (f *Files) RestoreBackup(srcPath string, dstPath string) error {
	logger.Info("正在恢复备份...")

	if err := CopyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("恢复备份失败: %v", err)
	}

	// Unix系统设置执行权限
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
	}

	logger.Info("备份已恢复")
	return nil
}

// copyFile 复制文件
func CopyFile(src, dst string) error {
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
func (f *Files) Cleanup(tempDir string) error {
	if tempDir != "" {
		err := os.RemoveAll(tempDir)
		if err != nil {
			return err
		}
	}
	return nil
}
