package singboxs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/util"
	"singbox_sub/src/github.com/sixproxy/util/files"
)

// startSingBox 启动sing-box服务（带失败检测和回滚）
func StartSingBox() error {
	logger.Info("正在启动sing-box服务...")

	scriptPath := "bash/start_singbox.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("启动脚本不存在: %s", scriptPath)
	}

	shell := util.GetAvailableShell()
	if shell == "" {
		return fmt.Errorf("未找到可用的shell执行器")
	}

	// 备份当前配置
	configBackupPath := "/etc/sing-box/config.json.backup"
	configPath := "/etc/sing-box/config.json"
	if _, err := os.Stat(configPath); err == nil {
		if err := files.CopyFile(configPath, configBackupPath); err != nil {
			logger.Warn("备份配置文件失败: %v", err)
		} else {
			logger.Debug("已备份配置文件到: %s", configBackupPath)
		}
	}

	logger.Debug("使用shell: %s", shell)
	cmd := exec.Command(shell, scriptPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("启动sing-box服务失败: %v", err)
	}

	logger.Info("sing-box服务启动命令已执行")
	if len(output) > 0 {
		logger.Debug("脚本输出: %s", string(output))
	}
	return nil
}

func IsSingBoxRunning() bool {
	// 使用pgrep命令检查sing-box进程
	cmd := exec.Command("pgrep", "sing-box")
	err := cmd.Run()
	// 如果pgrep找到进程，返回码为0；找不到进程返回码为1
	return err == nil
}

func StopSingBox() {
	logger.Info("正在停止sing-box服务...")

	scriptPath := "bash/stop_singbox.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		logger.Warn("停止脚本不存在: %s，跳过停止步骤", scriptPath)
		return
	}

	shell := util.GetAvailableShell()
	if shell == "" {
		logger.Error("未找到可用的shell执行器，跳过停止步骤")
		return
	}

	logger.Debug("使用shell: %s", shell)
	cmd := exec.Command(shell, scriptPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Warn("停止sing-box服务失败: %v", err)
		logger.Debug("脚本输出: %s", string(output))
	} else {
		logger.Info("sing-box服务停止命令已执行")
		if len(output) > 0 {
			logger.Debug("脚本输出: %s", string(output))
		}

		// 验证服务是否真的停止了
		if IsSingBoxRunning() {
			logger.Warn("sing-box进程可能仍在运行，建议手动检查")
		} else {
			logger.Info("确认sing-box服务已完全停止")
		}
	}
}

func DeployLinuxConfig() {
	logger.Info("正在部署Linux配置文件...")

	sourceFile := "linux_config.json"
	targetDir := "/etc/sing-box"
	targetFile := filepath.Join(targetDir, "config.json")

	// 检查源文件是否存在
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		logger.Error("源配置文件不存在: %s", sourceFile)
		return
	}

	// 创建目标目录（如果不存在）
	logger.Debug("创建配置目录: %s", targetDir)
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		logger.Error("创建配置目录失败: %v", err)
		return
	}

	// 拷贝配置文件
	logger.Debug("拷贝配置文件: %s -> %s", sourceFile, targetFile)
	err = files.CopyFile(sourceFile, targetFile)
	if err != nil {
		logger.Error("拷贝配置文件失败: %v", err)
		return
	}

	// 设置文件权限
	err = os.Chmod(targetFile, 0644)
	if err != nil {
		logger.Warn("设置配置文件权限失败: %v", err)
	}

	logger.Info("配置文件已成功部署到: %s", targetFile)
}
