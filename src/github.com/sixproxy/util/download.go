package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
	"time"
)

// DownloadConfig 下载配置
type DownloadConfig struct {
	URL         string
	DestDir     string
	Filename    string
	Timeout     time.Duration
	MaxRetries  int
	ShowProgress bool
}

// DownloadResult 下载结果
type DownloadResult struct {
	FilePath string
	Size     int64
	Checksum string
}

// ProgressReader 带进度显示的读取器
type ProgressReader struct {
	Reader      io.Reader
	Total       int64
	read        int64
	lastPercent int
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)

	if pr.Total > 0 {
		percent := int((pr.read * 100) / pr.Total)
		// 每10%显示一次进度，避免输出过多
		if percent >= pr.lastPercent+10 {
			logger.Info("下载进度: %d%% (%.1f MB / %.1f MB)",
				percent,
				float64(pr.read)/(1024*1024),
				float64(pr.Total)/(1024*1024))
			pr.lastPercent = percent
		}
	}

	return n, err
}

// DownloadFile 下载文件到指定目录
func DownloadFile(config DownloadConfig) (*DownloadResult, error) {
	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	var result *DownloadResult
	var err error

	// 重试机制
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		if attempt > 1 {
			logger.Info("第 %d 次重试下载...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second) // 递增延迟
		}

		result, err = downloadFileOnce(config)
		if err != nil {
			if attempt == config.MaxRetries {
				return nil, fmt.Errorf("下载失败（已重试 %d 次）: %v", config.MaxRetries, err)
			}
			logger.Warn("下载失败，准备重试: %v", err)
			continue
		}

		// 验证下载的文件
		if err := ValidateDownloadedFile(result.FilePath); err != nil {
			if attempt == config.MaxRetries {
				return nil, fmt.Errorf("文件验证失败（已重试 %d 次）: %v", config.MaxRetries, err)
			}
			logger.Warn("文件验证失败，准备重试: %v", err)
			continue
		}

		// 下载和验证都成功
		break
	}

	return result, nil
}

// downloadFileOnce 执行一次下载
func downloadFileOnce(config DownloadConfig) (*DownloadResult, error) {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	resp, err := client.Get(config.URL)
	if err != nil {
		return nil, fmt.Errorf("下载请求失败: %v，请检查网络连接", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("下载失败，HTTP状态码: %d，URL: %s", resp.StatusCode, config.URL)
	}

	// 确定文件名
	filename := config.Filename
	if filename == "" {
		filename = filepath.Base(config.URL)
		if filename == "" || filename == "." {
			if strings.Contains(config.URL, ".zip") {
				filename = "download.zip"
			} else {
				filename = "download.tar.gz"
			}
		}
	}

	filePath := filepath.Join(config.DestDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建下载文件失败: %v", err)
	}
	defer file.Close()

	// 显示下载进度
	size := resp.ContentLength
	var written int64

	if config.ShowProgress && size > 0 {
		logger.Info("开始下载: %.2f MB", float64(size)/(1024*1024))

		// 使用进度读取器
		progressReader := &ProgressReader{
			Reader: resp.Body,
			Total:  size,
		}

		written, err = io.Copy(file, progressReader)
		if err != nil {
			return nil, fmt.Errorf("下载过程中发生错误: %v", err)
		}

		logger.Info("✓ 下载完成: %.2f MB", float64(written)/(1024*1024))
	} else {
		// 没有内容长度信息或不显示进度，使用简单下载
		written, err = io.Copy(file, resp.Body)
		if err != nil {
			return nil, fmt.Errorf("下载过程中发生错误: %v", err)
		}
		if config.ShowProgress {
			logger.Info("✓ 下载完成: %.2f MB", float64(written)/(1024*1024))
		}
	}

	// 计算文件校验和
	file.Seek(0, 0)
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		logger.Warn("计算文件校验和失败: %v", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	return &DownloadResult{
		FilePath: filePath,
		Size:     written,
		Checksum: checksum,
	}, nil
}

// ValidateDownloadedFile 验证下载的文件
func ValidateDownloadedFile(filePath string) error {
	logger.Debug("验证下载文件: %s", filePath)

	// 检查文件是否存在
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("文件不存在: %v", err)
	}

	// 检查文件大小（至少应该有1MB）
	if stat.Size() < 1024*1024 {
		return fmt.Errorf("文件大小异常，可能下载不完整（大小: %d bytes）", stat.Size())
	}

	// 检查文件类型
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件进行验证: %v", err)
	}
	defer file.Close()

	// 读取文件头来验证格式
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证文件格式
	if strings.HasSuffix(filePath, ".zip") {
		// ZIP文件应该以"PK"开头
		if n < 2 || string(header[:2]) != "PK" {
			return fmt.Errorf("文件不是有效的ZIP格式")
		}
	} else if strings.HasSuffix(filePath, ".tar.gz") || strings.HasSuffix(filePath, ".tgz") {
		// GZIP文件应该以特定的魔数开头
		if n < 2 || header[0] != 0x1f || header[1] != 0x8b {
			return fmt.Errorf("文件不是有效的GZIP格式")
		}
	}

	logger.Debug("✓ 文件验证通过")
	return nil
}

// CalculateFileChecksum 计算文件SHA256校验和
func CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// CopyFile 复制文件
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

	_, err = io.Copy(dstFile, srcFile)
	return err
}