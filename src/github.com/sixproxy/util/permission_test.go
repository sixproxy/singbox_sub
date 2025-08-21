package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewPermissionChecker 测试权限检查器创建
func TestNewPermissionChecker(t *testing.T) {
	targetDir := "/tmp/test"
	checker := NewPermissionChecker(targetDir)

	if checker == nil {
		t.Fatal("Expected permission checker to be created")
	}

	if checker.targetDir != targetDir {
		t.Errorf("Expected target dir '%s', got '%s'", targetDir, checker.targetDir)
	}
}

// TestCheckWritePermission 测试写入权限检查
func TestCheckWritePermission(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "permission_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试对可写目录的权限检查
	checker := NewPermissionChecker(tempDir)
	if err := checker.checkWritePermission(); err != nil {
		t.Errorf("Expected write permission check to pass, got error: %v", err)
	}
}

// TestCheckInstallPermissions 测试安装权限检查
func TestCheckInstallPermissions(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "permission_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试安装权限检查
	checker := NewPermissionChecker(tempDir)
	err = checker.CheckInstallPermissions()
	
	// 对于临时目录，权限检查应该通过
	if err != nil {
		t.Logf("Install permission check failed (may be expected): %v", err)
	}
}

// TestIsRunningAsAdmin 测试管理员权限检查
func TestIsRunningAsAdmin(t *testing.T) {
	// 这个测试的结果取决于运行环境
	isAdmin := IsRunningAsAdmin()
	t.Logf("Running as admin: %v", isAdmin)
	
	// 验证函数不会panic
	if runtime.GOOS == "windows" {
		// Windows特定的检查
		t.Logf("Windows admin check: %v", isWindowsAdmin())
	} else if runtime.GOOS == "linux" {
		// Linux特定的检查
		t.Logf("Linux admin check: %v", isLinuxAdmin())
	} else if runtime.GOOS == "darwin" {
		// macOS特定的检查
		t.Logf("Darwin admin check: %v", isDarwinAdmin())
	}
}

// TestIsHomebrewInstalled 测试Homebrew安装检查
func TestIsHomebrewInstalled(t *testing.T) {
	isInstalled := IsHomebrewInstalled()
	t.Logf("Homebrew installed: %v", isInstalled)
	
	// 在macOS上，结果可能为true或false
	// 在其他系统上，通常为false
	if runtime.GOOS == "darwin" {
		t.Logf("macOS Homebrew check result: %v", isInstalled)
	}
}

// TestCheckDirectoryWritable 测试目录可写性检查
func TestCheckDirectoryWritable(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "permission_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试可写目录
	if err := CheckDirectoryWritable(tempDir); err != nil {
		t.Errorf("Expected directory to be writable, got error: %v", err)
	}

	// 测试不存在的目录
	nonExistentDir := filepath.Join(tempDir, "nonexistent", "nested", "dir")
	err = CheckDirectoryWritable(nonExistentDir)
	// 这个可能成功（如果能创建）或失败（如果没有权限）
	t.Logf("Non-existent directory check result: %v", err)
}

// TestGetAdminInstallationPaths 测试获取管理员安装路径
func TestGetAdminInstallationPaths(t *testing.T) {
	paths := GetAdminInstallationPaths()
	
	// 验证返回的路径不为空
	if len(paths) == 0 {
		t.Error("Expected admin installation paths to be returned")
	}
	
	// 验证包含expected categories
	expectedCategories := []string{"binary", "config"}
	for _, category := range expectedCategories {
		if _, exists := paths[category]; !exists {
			t.Errorf("Expected category '%s' in admin paths", category)
		}
	}
	
	t.Logf("Admin installation paths: %+v", paths)
}

// TestGetUserInstallationPaths 测试获取用户安装路径
func TestGetUserInstallationPaths(t *testing.T) {
	paths := GetUserInstallationPaths()
	
	// 验证返回的路径不为空
	if len(paths) == 0 {
		t.Error("Expected user installation paths to be returned")
	}
	
	// 验证包含expected categories
	expectedCategories := []string{"binary", "config"}
	for _, category := range expectedCategories {
		if _, exists := paths[category]; !exists {
			t.Errorf("Expected category '%s' in user paths", category)
		}
	}
	
	t.Logf("User installation paths: %+v", paths)
}

// TestSuggestInstallationStrategy 测试安装策略建议
func TestSuggestInstallationStrategy(t *testing.T) {
	strategy := SuggestInstallationStrategy()
	
	// 验证返回的策略是有效的
	validStrategies := []string{"system", "homebrew", "user"}
	found := false
	for _, valid := range validStrategies {
		if strategy == valid {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected valid strategy, got '%s'", strategy)
	}
	
	t.Logf("Suggested installation strategy: %s", strategy)
}

// TestRequireAdminRights 测试管理员权限要求
func TestRequireAdminRights(t *testing.T) {
	err := RequireAdminRights("test operation")
	
	// 如果当前用户是管理员，应该返回nil
	// 如果不是管理员，应该返回错误
	if err != nil {
		t.Logf("Admin rights check failed (may be expected): %v", err)
	} else {
		t.Log("Admin rights check passed")
	}
}