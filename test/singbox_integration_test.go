package test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/service"
	"testing"
	"time"
)

// TestSingboxManagerIntegration 集成测试sing-box管理器
func TestSingboxManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	manager := service.NewSingboxService()

	// 测试获取最新版本（需要网络连接）
	t.Run("GetLatestVersion", func(t *testing.T) {
		latest, err := manager.GetLatestVersion()
		if err != nil {
			t.Fatalf("获取最新版本失败: %v", err)
		}

		if latest.TagName == "" {
			t.Error("版本标签不应为空")
		}

		if len(latest.Assets) == 0 {
			t.Error("应该有可用的资产")
		}

		t.Logf("最新版本: %s", latest.TagName)
		t.Logf("发布时间: %s", latest.PublishedAt)
		t.Logf("资产数量: %d", len(latest.Assets))
	})

	// 测试更新检查
	t.Run("IsUpdateAvailable", func(t *testing.T) {
		hasUpdate, latest, err := manager.IsUpdateAvailable()
		if err != nil {
			t.Fatalf("检查更新失败: %v", err)
		}

		t.Logf("是否有更新: %v", hasUpdate)
		if latest != nil {
			t.Logf("最新版本: %s", latest.TagName)
		}
	})

	// 测试下载URL获取
	t.Run("GetDownloadURL", func(t *testing.T) {
		latest, err := manager.GetLatestVersion()
		if err != nil {
			t.Fatalf("获取最新版本失败: %v", err)
		}

		// 使用反射访问私有方法（或者将方法设为公共的）
		// 这里我们跳过这个测试，因为方法是私有的
		t.Logf("当前平台: %s/%s", runtime.GOOS, runtime.GOARCH)

		// 检查是否有适合当前平台的资产
		found := false
		for _, asset := range latest.Assets {
			if containsPlatform(asset.Name, runtime.GOOS, runtime.GOARCH) {
				found = true
				t.Logf("找到适合的资产: %s", asset.Name)
				break
			}
		}

		if !found {
			t.Errorf("未找到适合 %s/%s 平台的资产", runtime.GOOS, runtime.GOARCH)
		}
	})
}

// containsPlatform 检查资产名称是否包含指定平台和架构
func containsPlatform(assetName, os, arch string) bool {
	// 简单的字符串匹配检查
	osNames := map[string][]string{
		"linux":   {"linux"},
		"darwin":  {"darwin", "macos"},
		"windows": {"windows", "win"},
	}

	archNames := map[string][]string{
		"amd64": {"amd64", "x64", "x86_64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "x86"},
	}

	// 检查OS
	osFound := false
	if osList, exists := osNames[os]; exists {
		for _, osName := range osList {
			if contains(assetName, osName) {
				osFound = true
				break
			}
		}
	}

	// 检查架构
	archFound := false
	if archList, exists := archNames[arch]; exists {
		for _, archName := range archList {
			if contains(assetName, archName) {
				archFound = true
				break
			}
		}
	}

	return osFound && archFound
}

// contains 检查字符串是否包含子字符串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[0:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMainProgramSingboxIntegration 测试主程序的sing-box集成
func TestMainProgramSingboxIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	// 构建测试用的程序
	tempDir, err := os.MkdirTemp("", "singbox_main_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 获取项目根目录
	projectRoot, err := filepath.Abs("../")
	if err != nil {
		t.Fatalf("获取项目根目录失败: %v", err)
	}

	binaryPath := filepath.Join(tempDir, "sub_test")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// 构建程序
	buildCmd := exec.Command("go", "build", "-o", binaryPath,
		filepath.Join(projectRoot, "src/github.com/sixproxy/sub.go"))
	buildCmd.Dir = projectRoot

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("构建程序失败: %v\n输出: %s", err, string(output))
	}

	// 测试help命令
	t.Run("HelpCommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-h")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("执行help命令失败: %v", err)
		}

		outputStr := string(output)

		// 检查是否包含新添加的选项
		if !contains(outputStr, "install-singbox") {
			t.Error("帮助信息应包含 install-singbox 选项")
		}

		if !contains(outputStr, "skip-singbox-check") {
			t.Error("帮助信息应包含 skip-singbox-check 选项")
		}

		if !contains(outputStr, "sing-box管理功能") {
			t.Error("帮助信息应包含 sing-box管理功能 说明")
		}
	})

	// 测试版本命令
	t.Run("VersionCommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("执行version命令失败: %v", err)
		}

		outputStr := string(output)
		if !contains(outputStr, "singbox_sub version") {
			t.Error("版本信息格式不正确")
		}
	})

	// 测试跳过sing-box检查
	t.Run("SkipSingboxCheck", func(t *testing.T) {
		// 创建一个简单的配置目录结构
		configDir := filepath.Join(tempDir, "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("创建配置目录失败: %v", err)
		}

		// 创建最小配置文件
		configContent := `subs:
  - url: ""
    insecure: false
dns:
  auto_optimize: true`

		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("创建配置文件失败: %v", err)
		}

		// 创建模板文件（简化版）
		templateContent := `{
  "subs": [{"url": "", "prefix": "", "insecure": false}],
  "log": {"disabled": false, "level": "warn", "timestamp": true},
  "experimental": {
    "clash_api": {"external_controller": "", "external_ui": "", "secret": ""},
    "cache_file": {"enabled": true, "path": ""}
  },
  "dns": {
    "servers": [
      {"tag": "dns_local", "server": "114.114.114.114"},
      {"tag": "dns_proxy", "server": "8.8.8.8"}
    ],
    "rules": [],
    "final": "dns_proxy"
  },
  "inbounds": [],
  "outbounds": [],
  "route": {"final": "direct", "auto_detect_interface": true}
}`

		templatePath := filepath.Join(configDir, "template-v1.12.json")
		if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
			t.Fatalf("创建模板文件失败: %v", err)
		}

		// 测试跳过sing-box检查的命令
		cmd := exec.Command(binaryPath, "-skip-singbox-check")
		cmd.Dir = tempDir

		// 设置超时
		timer := time.AfterFunc(30*time.Second, func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
		defer timer.Stop()

		output, err := cmd.CombinedOutput()

		// 这个命令可能会失败（因为没有有效的订阅URL），但不应该因为sing-box检查而失败
		outputStr := string(output)
		t.Logf("输出: %s", outputStr)
		if err != nil {
			t.Logf("命令执行错误（预期的）: %v", err)
		}

		// 检查是否跳过了sing-box检查
		if contains(outputStr, "检测到sing-box") || contains(outputStr, "未检测到sing-box") {
			t.Error("应该跳过sing-box检查")
		}
	})
}

// TestSingboxManagerRealInstall 真实安装测试（可选，需要管理员权限）
func TestSingboxManagerRealInstall(t *testing.T) {
	if os.Getenv("SINGBOX_REAL_INSTALL_TEST") != "1" {
		t.Skip("跳过真实安装测试（设置 SINGBOX_REAL_INSTALL_TEST=1 启用）")
	}

	if runtime.GOOS == "windows" && os.Getenv("CI") == "" {
		t.Skip("Windows真实安装测试需要管理员权限，跳过")
	}

	manager := service.NewSingboxService()

	t.Run("RealInstall", func(t *testing.T) {
		// 备份现有安装（如果有）
		var backupPath string
		if manager.IsInstalled() {
			current, err := manager.GetInstalledVersion()
			if err == nil {
				t.Logf("备份当前版本: %s", current.Version)
				backupPath = manager.GetBinaryPath() + ".test_backup"
				if err := copyFile(manager.GetBinaryPath(), backupPath); err != nil {
					t.Logf("备份失败: %v", err)
				}
			}
		}

		// 执行安装
		if err := manager.InstallOrUpdate(); err != nil {
			t.Fatalf("安装失败: %v", err)
		}

		// 验证安装结果
		if !manager.IsInstalled() {
			t.Error("安装后应该能检测到sing-box")
		}

		version, err := manager.GetInstalledVersion()
		if err != nil {
			t.Errorf("获取安装后的版本失败: %v", err)
		} else {
			t.Logf("安装的版本: %s", version.Version)
		}

		// 恢复备份（如果有）
		if backupPath != "" {
			if err := copyFile(backupPath, manager.GetBinaryPath()); err != nil {
				t.Logf("恢复备份失败: %v", err)
			} else {
				os.Remove(backupPath)
				t.Log("已恢复原始版本")
			}
		}
	})
}

// copyFile 复制文件的辅助函数
func copyFile(src, dst string) error {
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
