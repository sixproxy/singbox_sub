package util

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDownloadConfig 测试下载配置
func TestDownloadConfig(t *testing.T) {
	config := DownloadConfig{
		URL:          "http://example.com/test.zip",
		DestDir:      "/tmp",
		Filename:     "test.zip",
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		ShowProgress: true,
	}

	if config.URL != "http://example.com/test.zip" {
		t.Errorf("Expected URL to be 'http://example.com/test.zip', got %s", config.URL)
	}

	if config.DestDir != "/tmp" {
		t.Errorf("Expected DestDir to be '/tmp', got %s", config.DestDir)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}
}

// TestDownloadFile 测试文件下载
func TestDownloadFile(t *testing.T) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "test file content")
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "download_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 使用downloadFileOnce函数来绕过验证
	config := DownloadConfig{
		URL:          server.URL + "/test.txt",
		DestDir:      tempDir,
		Filename:     "test.txt",
		Timeout:      10 * time.Second,
		MaxRetries:   1,
		ShowProgress: false,
	}

	result, err := downloadFileOnce(config)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// 验证结果
	if result.FilePath != filepath.Join(tempDir, "test.txt") {
		t.Errorf("Expected file path %s, got %s", filepath.Join(tempDir, "test.txt"), result.FilePath)
	}

	if result.Size != 17 { // "test file content" length
		t.Errorf("Expected file size 17, got %d", result.Size)
	}

	// 验证文件内容
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "test file content" {
		t.Errorf("Expected content 'test file content', got '%s'", string(content))
	}
}

// TestValidateDownloadedFile 测试文件验证
func TestValidateDownloadedFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "validate_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	content := make([]byte, 2*1024*1024) // 2MB
	for i := range content {
		content[i] = byte(i % 256)
	}

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 测试验证
	if err := ValidateDownloadedFile(testFile); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// 测试小文件验证失败
	smallFile := filepath.Join(tempDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create small test file: %v", err)
	}

	if err := ValidateDownloadedFile(smallFile); err == nil {
		t.Error("Expected validation to fail for small file")
	}
}

// TestCalculateFileChecksum 测试校验和计算
func TestCalculateFileChecksum(t *testing.T) {
	// 创建临时文件
	tempDir, err := os.MkdirTemp("", "checksum_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "test content for checksum"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 计算校验和
	checksum, err := CalculateFileChecksum(testFile)
	if err != nil {
		t.Errorf("Failed to calculate checksum: %v", err)
	}

	if len(checksum) != 64 { // SHA256 hex string length
		t.Errorf("Expected checksum length 64, got %d", len(checksum))
	}

	// 验证校验和是否一致
	checksum2, err := CalculateFileChecksum(testFile)
	if err != nil {
		t.Errorf("Failed to calculate checksum again: %v", err)
	}

	if checksum != checksum2 {
		t.Errorf("Checksums don't match: %s != %s", checksum, checksum2)
	}
}

// TestCopyFile 测试文件复制
func TestCopyFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "copy_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建源文件
	srcFile := filepath.Join(tempDir, "source.txt")
	content := "test content for copy"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 复制文件
	dstFile := filepath.Join(tempDir, "destination.txt")
	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Errorf("Failed to copy file: %v", err)
	}

	// 验证目标文件内容
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(dstContent))
	}
}

// TestProgressReader 测试进度读取器
func TestProgressReader(t *testing.T) {
	content := "test content for progress reader"
	reader := &ProgressReader{
		Reader: io.NopCloser(strings.NewReader(content)),
		Total:  int64(len(content)),
	}

	// 读取所有内容
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read from progress reader: %v", err)
	}

	if string(result) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(result))
	}

	if reader.read != int64(len(content)) {
		t.Errorf("Expected read bytes %d, got %d", len(content), reader.read)
	}
}