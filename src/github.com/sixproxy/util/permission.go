package util

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
)

// PermissionChecker 权限检查器
type PermissionChecker struct {
	targetDir string
}

// NewPermissionChecker 创建权限检查器
func NewPermissionChecker(targetDir string) *PermissionChecker {
	return &PermissionChecker{
		targetDir: targetDir,
	}
}

// CheckInstallPermissions 检查安装权限
func (pc *PermissionChecker) CheckInstallPermissions() error {
	// macOS使用Homebrew时通常不需要特殊权限检查
	if runtime.GOOS == "darwin" && IsHomebrewInstalled() {
		return nil
	}

	// 检查是否可以写入目标目录
	if err := pc.checkWritePermission(); err != nil {
		return err
	}

	// 在Linux和Windows上检查是否以管理员身份运行
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		if !IsRunningAsAdmin() {
			logger.Warn("建议使用管理员权限运行以确保安装成功")
		}
	}

	return nil
}

// checkWritePermission 检查写入权限
func (pc *PermissionChecker) checkWritePermission() error {
	// 尝试在目标目录创建临时文件
	tempFile := filepath.Join(pc.targetDir, ".permission_test")
	
	file, err := os.Create(tempFile)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("没有写入权限到 %s，请使用管理员权限运行", pc.targetDir)
		}
		
		// 目录可能不存在，尝试创建
		if err := os.MkdirAll(pc.targetDir, 0755); err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("没有权限创建目录 %s，请使用管理员权限运行", pc.targetDir)
			}
			return fmt.Errorf("无法创建目录: %v", err)
		}
		
		// 重试创建测试文件
		file, err = os.Create(tempFile)
		if err != nil {
			return fmt.Errorf("权限验证失败: %v", err)
		}
	}
	
	file.Close()
	os.Remove(tempFile)
	return nil
}

// IsRunningAsAdmin 检查是否以管理员身份运行
func IsRunningAsAdmin() bool {
	switch runtime.GOOS {
	case "linux":
		return isLinuxAdmin()
	case "windows":
		return isWindowsAdmin()
	case "darwin":
		return isDarwinAdmin()
	default:
		return true // 其他系统默认返回true
	}
}

// isLinuxAdmin 检查Linux管理员权限
func isLinuxAdmin() bool {
	// 检查是否为root用户
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	if currentUser.Uid == "0" {
		return true
	}
	
	// 检查sudo权限
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

// isWindowsAdmin 检查Windows管理员权限
func isWindowsAdmin() bool {
	// 在Windows上检查管理员权限
	cmd := exec.Command("net", "session")
	return cmd.Run() == nil
}

// isDarwinAdmin 检查macOS管理员权限
func isDarwinAdmin() bool {
	// 检查是否为root用户
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	if currentUser.Uid == "0" {
		return true
	}
	
	// 检查sudo权限
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

// IsHomebrewInstalled 检查Homebrew是否安装
func IsHomebrewInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// RequireAdminRights 要求管理员权限的通用函数
func RequireAdminRights(operation string) error {
	if !IsRunningAsAdmin() {
		switch runtime.GOOS {
		case "linux", "darwin":
			return fmt.Errorf("需要管理员权限进行%s操作，请使用sudo运行", operation)
		case "windows":
			return fmt.Errorf("需要管理员权限进行%s操作，请以管理员身份运行", operation)
		default:
			return fmt.Errorf("需要管理员权限进行%s操作", operation)
		}
	}
	return nil
}

// CheckDirectoryWritable 检查目录是否可写
func CheckDirectoryWritable(dir string) error {
	checker := NewPermissionChecker(dir)
	return checker.checkWritePermission()
}

// GetAdminInstallationPaths 获取需要管理员权限的安装路径
func GetAdminInstallationPaths() map[string][]string {
	paths := make(map[string][]string)
	
	switch runtime.GOOS {
	case "linux":
		paths["binary"] = []string{"/usr/local/bin", "/usr/bin"}
		paths["config"] = []string{"/etc", "/usr/local/etc"}
		paths["service"] = []string{"/etc/systemd/system", "/lib/systemd/system"}
		
	case "windows":
		paths["binary"] = []string{"C:\\Program Files", "C:\\Windows\\System32"}
		paths["config"] = []string{"C:\\ProgramData"}
		
	case "darwin":
		paths["binary"] = []string{"/usr/local/bin", "/opt/homebrew/bin"}
		paths["config"] = []string{"/usr/local/etc", "/opt/homebrew/etc"}
		paths["service"] = []string{"/Library/LaunchDaemons"}
	}
	
	return paths
}

// GetUserInstallationPaths 获取用户权限可安装的路径
func GetUserInstallationPaths() map[string][]string {
	paths := make(map[string][]string)
	homeDir, _ := os.UserHomeDir()
	
	switch runtime.GOOS {
	case "linux", "darwin":
		paths["binary"] = []string{
			filepath.Join(homeDir, "bin"),
			filepath.Join(homeDir, ".local", "bin"),
		}
		paths["config"] = []string{
			filepath.Join(homeDir, ".config"),
			filepath.Join(homeDir, ".local", "etc"),
		}
		
	case "windows":
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		paths["binary"] = []string{
			filepath.Join(localAppData, "Programs"),
			filepath.Join(homeDir, "bin"),
		}
		paths["config"] = []string{
			appData,
			localAppData,
		}
	}
	
	return paths
}

// SuggestInstallationStrategy 建议安装策略
func SuggestInstallationStrategy() string {
	if IsRunningAsAdmin() {
		return "system" // 系统级安装
	}
	
	if runtime.GOOS == "darwin" && IsHomebrewInstalled() {
		return "homebrew" // 使用Homebrew
	}
	
	return "user" // 用户级安装
}