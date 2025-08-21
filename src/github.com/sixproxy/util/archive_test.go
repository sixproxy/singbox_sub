package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// TestExtractConfig 测试解压配置
func TestExtractConfig(t *testing.T) {
	config := ExtractConfig{
		ArchivePath:   "/tmp/test.tar.gz",
		DestDir:       "/tmp/extract",
		TargetFiles:   []string{"sing-box", "sing-box.exe"},
		CreateDestDir: true,
	}

	if config.ArchivePath != "/tmp/test.tar.gz" {
		t.Errorf("Expected ArchivePath to be '/tmp/test.tar.gz', got %s", config.ArchivePath)
	}

	if len(config.TargetFiles) != 2 {
		t.Errorf("Expected 2 target files, got %d", len(config.TargetFiles))
	}

	if !config.CreateDestDir {
		t.Error("Expected CreateDestDir to be true")
	}
}

// TestExtractTarGz 测试tar.gz文件解压
func TestExtractTarGz(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "archive_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试tar.gz文件
	tarGzFile := filepath.Join(tempDir, "test.tar.gz")
	if err := createTestTarGz(tarGzFile); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// 测试解压
	extractDir := filepath.Join(tempDir, "extract")
	config := ExtractConfig{
		ArchivePath:   tarGzFile,
		DestDir:       extractDir,
		TargetFiles:   []string{"sing-box"},
		CreateDestDir: true,
	}

	result, err := ExtractArchive(config)
	if err != nil {
		t.Fatalf("Failed to extract archive: %v", err)
	}

	// 验证结果
	if len(result.ExtractedFiles) != 1 {
		t.Errorf("Expected 1 extracted file, got %d", len(result.ExtractedFiles))
	}

	if result.TargetFile == "" {
		t.Error("Expected target file to be set")
	}

	// 验证文件存在
	if _, err := os.Stat(result.TargetFile); os.IsNotExist(err) {
		t.Error("Extracted file does not exist")
	}
}

// TestExtractZip 测试ZIP文件解压
func TestExtractZip(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "archive_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试ZIP文件
	zipFile := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipFile); err != nil {
		t.Fatalf("Failed to create test zip: %v", err)
	}

	// 测试解压
	extractDir := filepath.Join(tempDir, "extract")
	config := ExtractConfig{
		ArchivePath:   zipFile,
		DestDir:       extractDir,
		TargetFiles:   []string{"sing-box.exe"},
		CreateDestDir: true,
	}

	result, err := ExtractArchive(config)
	if err != nil {
		t.Fatalf("Failed to extract archive: %v", err)
	}

	// 验证结果
	if len(result.ExtractedFiles) != 1 {
		t.Errorf("Expected 1 extracted file, got %d", len(result.ExtractedFiles))
	}

	if result.TargetFile == "" {
		t.Error("Expected target file to be set")
	}

	// 验证文件存在
	if _, err := os.Stat(result.TargetFile); os.IsNotExist(err) {
		t.Error("Extracted file does not exist")
	}
}

// TestExtractSingboxBinary 测试sing-box二进制文件提取
func TestExtractSingboxBinary(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "archive_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试tar.gz文件
	tarGzFile := filepath.Join(tempDir, "test.tar.gz")
	if err := createTestTarGz(tarGzFile); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// 测试提取
	extractDir := filepath.Join(tempDir, "extract")
	binaryPath, err := ExtractSingboxBinary(tarGzFile, extractDir)
	if err != nil {
		t.Fatalf("Failed to extract singbox binary: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Extracted binary does not exist")
	}

	// 验证文件名
	if filepath.Base(binaryPath) != "sing-box" {
		t.Errorf("Expected binary name 'sing-box', got %s", filepath.Base(binaryPath))
	}
}

// TestListArchiveContents 测试压缩文件内容列表
func TestListArchiveContents(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "archive_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试tar.gz文件
	tarGzFile := filepath.Join(tempDir, "test.tar.gz")
	if err := createTestTarGz(tarGzFile); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// 测试列表内容
	contents, err := ListArchiveContents(tarGzFile)
	if err != nil {
		t.Fatalf("Failed to list archive contents: %v", err)
	}

	// 验证内容
	if len(contents) == 0 {
		t.Error("Expected archive to contain files")
	}

	found := false
	for _, content := range contents {
		if filepath.Base(content) == "sing-box" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find 'sing-box' in archive contents")
	}
}

// createTestTarGz 创建测试用的tar.gz文件
func createTestTarGz(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 添加sing-box文件
	header := &tar.Header{
		Name: "sing-box",
		Mode: 0755,
		Size: int64(len("fake sing-box binary")),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tarWriter.Write([]byte("fake sing-box binary")); err != nil {
		return err
	}

	return nil
}

// createTestZip 创建测试用的ZIP文件
func createTestZip(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// 添加sing-box.exe文件
	fileWriter, err := zipWriter.Create("sing-box.exe")
	if err != nil {
		return err
	}

	if _, err := fileWriter.Write([]byte("fake sing-box binary")); err != nil {
		return err
	}

	return nil
}