package util

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
	"time"
)

// TemplateMirrorManager 模板镜像管理器
type TemplateMirrorManager struct {
	templatePath    string
	currentMirror   string
	availableMirrors []string
}

// NewTemplateMirrorManager 创建模板镜像管理器
func NewTemplateMirrorManager(templatePath string) *TemplateMirrorManager {
	return &TemplateMirrorManager{
		templatePath: templatePath,
	}
}

// UpdateTemplateMirrors 更新模板中的所有GitHub镜像地址
func (tmm *TemplateMirrorManager) UpdateTemplateMirrors(userMirror string) error {
	logger.Info("🔄 开始更新模板文件中的GitHub镜像地址...")
	
	// 1. 确保sing-box已停止（避免运行时修改配置文件）
	if err := tmm.ensureSingboxStopped(); err != nil {
		logger.Warn("停止sing-box时出现问题: %v", err)
	}
	
	// 2. 读取模板文件
	content, err := os.ReadFile(tmm.templatePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %v", err)
	}
	
	originalContent := string(content)
	
	// 2. 确定要使用的镜像地址
	targetMirror, err := tmm.selectBestMirror(userMirror)
	if err != nil {
		return fmt.Errorf("选择镜像地址失败: %v", err)
	}
	
	if targetMirror == "" {
		logger.Info("未配置镜像或镜像不可用，保持原有配置")
		return nil
	}
	
	// 3. 检查模板中是否包含占位符
	if !strings.Contains(originalContent, "{{mirror_url}}") {
		logger.Info("✅ 模板未使用{{mirror_url}}占位符，无需更新")
		return nil
	}
	
	// 4. 替换占位符
	newContent := tmm.replaceMirrorPlaceholder(originalContent, targetMirror)
	
	// 5. 备份原文件
	backupPath := tmm.templatePath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		logger.Warn("备份模板文件失败: %v", err)
	} else {
		logger.Debug("已备份原模板到: %s", backupPath)
	}
	
	// 6. 写入新内容
	if err := os.WriteFile(tmm.templatePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("写入模板文件失败: %v", err)
	}
	
	// 7. 验证更新结果
	tmm.currentMirror = targetMirror
	mirrorCount := strings.Count(originalContent, "{{mirror_url}}")
	
	logger.Info("✅ 成功更新模板镜像地址")
	logger.Info("   镜像地址: %s", targetMirror)
	logger.Info("   更新了 %d 个{{mirror_url}}占位符", mirrorCount)
	
	return nil
}

// selectBestMirror 选择最佳镜像地址
func (tmm *TemplateMirrorManager) selectBestMirror(userMirror string) (string, error) {
	// 如果用户没有配置镜像，直接返回空
	if userMirror == "" {
		logger.Info("用户未配置GitHub镜像，保持原始GitHub地址")
		return "", nil
	}
	
	logger.Info("🧪 测试用户配置的镜像: %s", userMirror)
	if tmm.testMirrorAvailability(userMirror) {
		logger.Info("✅ 用户镜像可用")
		return strings.TrimSuffix(userMirror, "/"), nil
	} else {
		return "", fmt.Errorf("用户配置的GitHub镜像 %s 不可用，请检查网络连接或更换镜像地址", userMirror)
	}
}

// testMirrorAvailability 测试镜像可用性
func (tmm *TemplateMirrorManager) testMirrorAvailability(mirrorURL string) bool {
	// 使用之前实现的testMirrorConnectivity函数
	return testMirrorConnectivity(mirrorURL)
}

// replaceMirrorPlaceholder 替换模板中的{{mirror_url}}占位符
func (tmm *TemplateMirrorManager) replaceMirrorPlaceholder(content, mirrorURL string) string {
	// 确保镜像URL末尾没有斜杠（模板中已经包含了斜杠）
	cleanMirrorURL := strings.TrimSuffix(mirrorURL, "/")
	
	// 简单的字符串替换
	return strings.ReplaceAll(content, "{{mirror_url}}", cleanMirrorURL)
}

// RestoreTemplate 从备份恢复模板
func (tmm *TemplateMirrorManager) RestoreTemplate() error {
	backupPath := tmm.templatePath + ".backup"
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}
	
	if err := os.WriteFile(tmm.templatePath, content, 0644); err != nil {
		return fmt.Errorf("恢复模板文件失败: %v", err)
	}
	
	logger.Info("✅ 已从备份恢复模板文件")
	return nil
}

// GetCurrentMirror 获取当前使用的镜像
func (tmm *TemplateMirrorManager) GetCurrentMirror() string {
	return tmm.currentMirror
}

// ensureSingboxStopped 确保sing-box已停止
func (tmm *TemplateMirrorManager) ensureSingboxStopped() error {
	logger.Debug("检查sing-box运行状态...")
	
	// 检查sing-box是否在运行
	if !tmm.isSingboxRunning() {
		logger.Debug("sing-box未运行，可以安全修改模板")
		return nil
	}
	
	logger.Info("⏹️ 检测到sing-box正在运行，正在停止...")
	
	// 尝试优雅停止sing-box
	if err := tmm.stopSingbox(); err != nil {
		return fmt.Errorf("停止sing-box失败: %v", err)
	}
	
	// 等待进程完全停止
	maxWait := 10 * time.Second
	waited := time.Duration(0)
	checkInterval := 500 * time.Millisecond
	
	for waited < maxWait {
		time.Sleep(checkInterval)
		waited += checkInterval
		
		if !tmm.isSingboxRunning() {
			logger.Info("✅ sing-box已成功停止")
			return nil
		}
	}
	
	return fmt.Errorf("等待sing-box停止超时（%v），但将继续模板更新", maxWait)
}

// isSingboxRunning 检查sing-box是否运行
func (tmm *TemplateMirrorManager) isSingboxRunning() bool {
	// 使用pgrep检查进程
	cmd := exec.Command("pgrep", "-x", "sing-box")
	err := cmd.Run()
	return err == nil
}

// stopSingbox 停止sing-box服务
func (tmm *TemplateMirrorManager) stopSingbox() error {
	// 根据不同系统使用不同的停止方法
	switch runtime.GOOS {
	case "linux":
		return tmm.stopLinuxSingbox()
	case "darwin":
		return tmm.stopDarwinSingbox()
	case "windows":
		return tmm.stopWindowsSingbox()
	default:
		return tmm.stopGenericSingbox()
	}
}

// stopLinuxSingbox 在Linux上停止sing-box
func (tmm *TemplateMirrorManager) stopLinuxSingbox() error {
	// 优先尝试systemd服务
	cmd := exec.Command("systemctl", "is-active", "--quiet", "sing-box")
	if cmd.Run() == nil {
		logger.Debug("使用systemctl停止sing-box服务")
		return exec.Command("systemctl", "stop", "sing-box").Run()
	}
	
	// 如果不是systemd服务，尝试脚本
	scriptPath := "bash/stop_singbox.sh"
	if _, err := os.Stat(scriptPath); err == nil {
		logger.Debug("使用停止脚本停止sing-box")
		return exec.Command("bash", scriptPath).Run()
	}
	
	// 最后尝试直接杀进程
	return tmm.stopGenericSingbox()
}

// stopDarwinSingbox 在macOS上停止sing-box
func (tmm *TemplateMirrorManager) stopDarwinSingbox() error {
	// 检查是否有launchd服务
	cmd := exec.Command("launchctl", "list")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "sing-box") {
		logger.Debug("使用launchctl停止sing-box服务")
		return exec.Command("launchctl", "stop", "sing-box").Run()
	}
	
	// 否则直接终止进程
	return tmm.stopGenericSingbox()
}

// stopWindowsSingbox 在Windows上停止sing-box
func (tmm *TemplateMirrorManager) stopWindowsSingbox() error {
	// 尝试停止Windows服务
	cmd := exec.Command("sc", "query", "sing-box")
	if cmd.Run() == nil {
		logger.Debug("使用sc停止sing-box服务")
		return exec.Command("sc", "stop", "sing-box").Run()
	}
	
	// 否则使用taskkill
	return exec.Command("taskkill", "/F", "/IM", "sing-box.exe").Run()
}

// stopGenericSingbox 通用的停止方法（发送SIGTERM信号）
func (tmm *TemplateMirrorManager) stopGenericSingbox() error {
	logger.Debug("使用SIGTERM信号停止sing-box")
	return exec.Command("pkill", "-TERM", "sing-box").Run()
}