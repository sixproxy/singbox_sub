package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
)

// ExtractConfig 解压配置
type ExtractConfig struct {
	ArchivePath   string
	DestDir       string
	TargetFiles   []string // 要提取的目标文件名（如"sing-box", "sing-box.exe"）
	CreateDestDir bool     // 是否自动创建目标目录
}

// ExtractResult 解压结果
type ExtractResult struct {
	ExtractedFiles map[string]string // 文件名 -> 完整路径的映射
	TargetFile     string            // 主要目标文件的路径（如果只有一个目标文件）
}

// ExtractArchive 解压压缩文件
func ExtractArchive(config ExtractConfig) (*ExtractResult, error) {
	logger.Info("解压文件: %s", config.ArchivePath)

	if config.CreateDestDir {
		if err := os.MkdirAll(config.DestDir, 0755); err != nil {
			return nil, fmt.Errorf("创建目标目录失败: %v", err)
		}
	}

	var result *ExtractResult
	var err error

	if strings.HasSuffix(config.ArchivePath, ".zip") {
		result, err = extractZip(config)
	} else if strings.HasSuffix(config.ArchivePath, ".tar.gz") || strings.HasSuffix(config.ArchivePath, ".tgz") {
		result, err = extractTarGz(config)
	} else {
		return nil, fmt.Errorf("不支持的压缩格式: %s", config.ArchivePath)
	}

	if err != nil {
		return nil, err
	}

	if len(result.ExtractedFiles) == 0 {
		return nil, fmt.Errorf("未找到目标文件: %v", config.TargetFiles)
	}

	logger.Info("解压完成，提取了 %d 个文件", len(result.ExtractedFiles))
	return result, nil
}

// extractZip 解压ZIP文件
func extractZip(config ExtractConfig) (*ExtractResult, error) {
	reader, err := zip.OpenReader(config.ArchivePath)
	if err != nil {
		return nil, fmt.Errorf("打开ZIP文件失败: %v", err)
	}
	defer reader.Close()

	result := &ExtractResult{
		ExtractedFiles: make(map[string]string),
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileName := filepath.Base(file.Name)
		
		// 检查是否为目标文件
		if !isTargetFile(fileName, config.TargetFiles) {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("打开ZIP文件内容失败: %v", err)
		}
		defer rc.Close()

		destPath := filepath.Join(config.DestDir, fileName)
		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return nil, fmt.Errorf("创建目标文件失败: %v", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return nil, fmt.Errorf("复制文件内容失败: %v", err)
		}

		result.ExtractedFiles[fileName] = destPath
		if result.TargetFile == "" {
			result.TargetFile = destPath
		}

		logger.Debug("提取文件: %s -> %s", file.Name, destPath)
	}

	return result, nil
}

// extractTarGz 解压tar.gz文件
func extractTarGz(config ExtractConfig) (*ExtractResult, error) {
	file, err := os.Open(config.ArchivePath)
	if err != nil {
		return nil, fmt.Errorf("打开tar.gz文件失败: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	result := &ExtractResult{
		ExtractedFiles: make(map[string]string),
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取tar文件条目失败: %v", err)
		}

		if header.Typeflag == tar.TypeDir {
			continue
		}

		fileName := filepath.Base(header.Name)
		
		// 检查是否为目标文件
		if !isTargetFile(fileName, config.TargetFiles) {
			continue
		}

		destPath := filepath.Join(config.DestDir, fileName)
		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return nil, fmt.Errorf("创建目标文件失败: %v", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, tarReader)
		if err != nil {
			return nil, fmt.Errorf("复制文件内容失败: %v", err)
		}

		result.ExtractedFiles[fileName] = destPath
		if result.TargetFile == "" {
			result.TargetFile = destPath
		}

		logger.Debug("提取文件: %s -> %s", header.Name, destPath)
	}

	return result, nil
}

// isTargetFile 检查文件名是否为目标文件
func isTargetFile(fileName string, targetFiles []string) bool {
	if len(targetFiles) == 0 {
		// 如果没有指定目标文件，提取所有文件
		return true
	}

	for _, target := range targetFiles {
		if fileName == target {
			return true
		}
	}
	return false
}

// ExtractSingboxBinary 专门用于提取sing-box二进制文件的便捷函数
func ExtractSingboxBinary(archivePath, destDir string) (string, error) {
	config := ExtractConfig{
		ArchivePath:   archivePath,
		DestDir:       destDir,
		TargetFiles:   []string{"sing-box", "sing-box.exe"},
		CreateDestDir: true,
	}

	result, err := ExtractArchive(config)
	if err != nil {
		return "", err
	}

	if result.TargetFile == "" {
		return "", fmt.Errorf("在压缩文件中找不到sing-box二进制文件")
	}

	return result.TargetFile, nil
}

// ListArchiveContents 列出压缩文件内容（用于调试）
func ListArchiveContents(archivePath string) ([]string, error) {
	var contents []string

	if strings.HasSuffix(archivePath, ".zip") {
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		for _, file := range reader.File {
			contents = append(contents, file.Name)
		}
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		file, err := os.Open(archivePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			contents = append(contents, header.Name)
		}
	} else {
		return nil, fmt.Errorf("不支持的压缩格式: %s", archivePath)
	}

	return contents, nil
}