package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/protocol"
)

func main() {
	// 0.解析命令行参数
	var (
		targetOS = flag.String("os", "auto", "目标操作系统 (auto/darwin/linux/windows/all)")
		verbose  = flag.Bool("v", false, "详细输出 (启用DEBUG日志)")
		help     = flag.Bool("h", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	// 设置日志级别
	if *verbose {
		logger.SetLevel(logger.DEBUG)
		logger.Debug("已启用详细输出模式")
	}

	// 0.5.Linux系统预处理 - 停止sing-box服务
	if runtime.GOOS == "linux" && (*targetOS == "auto" || *targetOS == "linux") {
		stopSingBoxService()
	}

	// 1.加载模版并合并YAML配置
	cfg, err := model.LoadConfigWithYAML(
		"config/template-v1.12.json",
		"config/config.yaml",
	)
	if err != nil {
		logger.Fatal("加载配置文件失败: %v", err)
	}

	// 1.5.打印控制面板地址
	printControlPanelURL(cfg)

	// 2.渲染模版
	err = cfg.RenderTemplate(delegateParse)
	if err != nil {
		logger.Error("渲染模板失败: %v", err)
	}

	// 3.根据系统类型输出相应配置
	generateSystemConfig(cfg, *targetOS)

}

func delegateParse(nodes []string) []string {
	c := make(chan string, 50)
	for _, node := range nodes {
		node := node
		go func(n string) {
			res, err := protocol.Parse(n)
			if err != nil {
				logger.ParseWarn("节点解析失败: %v", err)
				c <- "" // 返回空字符串而不是错误信息
			} else {
				c <- res
			}
		}(node)
	}

	configNodes := make([]string, 0)
	for i := 0; i < len(nodes); i++ {
		result := <-c
		if result != "" { // 过滤掉空结果（解析失败的节点）
			configNodes = append(configNodes, result)
		}
	}
	logger.ParseInfo("成功解析 %d/%d 个节点", len(configNodes), len(nodes))
	return configNodes
}

// printUsage 显示使用帮助
func printUsage() {
	logger.Info("=== sing-box配置生成器 ===")
	logger.Info("用法: %s [选项]", "singbox_sub")
	logger.Info("")
	logger.Info("选项:")
	logger.Info("  -os string    目标操作系统 (默认: auto)")
	logger.Info("                可选值: auto, darwin, linux, windows, all")
	logger.Info("  -v            详细输出模式 (启用DEBUG日志)")
	logger.Info("  -h            显示此帮助信息")
	logger.Info("")
	logger.Info("Linux自动化功能 (仅在Linux系统上生效):")
	logger.Info("  • 程序启动时自动停止sing-box服务")
	logger.Info("  • 配置生成后自动部署到/etc/sing-box/config.json")
	logger.Info("  • 部署完成后自动启动sing-box服务")
	logger.Info("  • 需要bash/stop_singbox.sh和bash/start_singbox.sh脚本")
	logger.Info("")
	logger.Info("示例:")
	logger.Info("  ./singbox_sub                    # 自动检测系统类型")
	logger.Info("  ./singbox_sub -os darwin         # 强制生成macOS配置")
	logger.Info("  ./singbox_sub -os linux          # 强制生成Linux配置")
	logger.Info("  ./singbox_sub -os all            # 生成所有类型配置")
	logger.Info("  ./singbox_sub -v                 # 详细输出模式")
	logger.Info("")
	logger.Info("Linux生产环境:")
	logger.Info("  ./singbox_sub                    # 完整自动化部署")
	logger.Info("  ./singbox_sub -v                 # 详细查看部署过程")
}

// generateSystemConfig 根据系统类型生成相应的配置文件
func generateSystemConfig(cfg *model.Config, targetOS string) {
	currentOS := runtime.GOOS
	logger.Info("当前操作系统: %s", currentOS)
	
	// 确定要生成的目标系统
	var effectiveOS string
	if targetOS == "auto" {
		effectiveOS = currentOS
		logger.Info("使用自动检测的系统类型: %s", effectiveOS)
	} else {
		effectiveOS = targetOS
		logger.Info("使用指定的目标系统: %s", effectiveOS)
	}
	
	switch effectiveOS {
	case "darwin":
		// macOS系统
		logger.Info("开始生成macOS配置文件...")
		err := cfg.MacConfig("")
		if err != nil {
			logger.Error("生成macOS配置文件失败: %v", err)
		} else {
			logger.Info("macOS配置文件生成成功")
		}
		
	case "linux":
		// Linux系统
		logger.Info("开始生成Linux配置文件...")
		err := cfg.LinuxConfig("")
		if err != nil {
			logger.Error("生成Linux配置文件失败: %v", err)
		} else {
			logger.Info("Linux配置文件生成成功")
			
			// 如果是在Linux系统上运行，执行额外的部署步骤
			if currentOS == "linux" {
				deployLinuxConfig()
				startSingBoxService()
			}
		}
		
	case "windows":
		// Windows系统 - 目前使用Linux配置作为通用配置
		logger.Info("检测到Windows系统，使用通用配置...")
		err := cfg.LinuxConfig("")
		if err != nil {
			logger.Error("生成Windows配置文件失败: %v", err)
		} else {
			logger.Info("Windows配置文件生成成功")
		}
		
	case "all":
		// 生成所有类型的配置文件
		logger.Info("生成所有类型的配置文件...")
		
		// 生成Linux配置
		logger.Info("生成Linux配置文件...")
		err := cfg.LinuxConfig("")
		if err != nil {
			logger.Error("生成Linux配置文件失败: %v", err)
		} else {
			logger.Info("Linux配置文件生成成功")
		}
		
		// 生成macOS配置
		logger.Info("生成macOS配置文件...")
		err = cfg.MacConfig("")
		if err != nil {
			logger.Error("生成macOS配置文件失败: %v", err)
		} else {
			logger.Info("macOS配置文件生成成功")
		}
		
		logger.Info("所有配置文件生成完成，请根据你的系统选择合适的配置")
		
	default:
		// 未知系统
		if targetOS == "auto" {
			logger.Warn("未知操作系统: %s，生成所有类型的配置文件", effectiveOS)
		} else {
			logger.Error("不支持的目标系统: %s", effectiveOS)
			logger.Info("支持的系统类型: auto, darwin, linux, windows, all")
			return
		}
		
		// 生成Linux配置
		err := cfg.LinuxConfig("")
		if err != nil {
			logger.Error("生成Linux配置文件失败: %v", err)
		}
		
		// 生成macOS配置
		err = cfg.MacConfig("")
		if err != nil {
			logger.Error("生成macOS配置文件失败: %v", err)
		}
		
		logger.Info("所有配置文件生成完成，请根据你的系统选择合适的配置")
	}
}

// getAvailableShell 获取可用的shell
func getAvailableShell() string {
	shells := []string{"bash", "sh", "/bin/bash", "/bin/sh", "/system/bin/sh"}
	for _, shell := range shells {
		if _, err := exec.LookPath(shell); err == nil {
			return shell
		}
	}
	return ""
}

// stopSingBoxService 停止sing-box服务
func stopSingBoxService() {
	logger.Info("正在停止sing-box服务...")
	
	scriptPath := "bash/stop_singbox.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		logger.Warn("停止脚本不存在: %s，跳过停止步骤", scriptPath)
		return
	}
	
	shell := getAvailableShell()
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
		logger.Info("sing-box服务已停止")
		if len(output) > 0 {
			logger.Debug("脚本输出: %s", string(output))
		}
	}
}

// deployLinuxConfig 部署Linux配置文件
func deployLinuxConfig() {
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
	err = copyFile(sourceFile, targetFile)
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

// startSingBoxService 启动sing-box服务
func startSingBoxService() {
	logger.Info("正在启动sing-box服务...")
	
	scriptPath := "bash/start_singbox.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		logger.Warn("启动脚本不存在: %s，跳过启动步骤", scriptPath)
		return
	}
	
	shell := getAvailableShell()
	if shell == "" {
		logger.Error("未找到可用的shell执行器，跳过启动步骤")
		return
	}
	
	logger.Debug("使用shell: %s", shell)
	cmd := exec.Command(shell, scriptPath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		logger.Error("启动sing-box服务失败: %v", err)
		logger.Debug("脚本输出: %s", string(output))
	} else {
		logger.Info("sing-box服务已启动")
		if len(output) > 0 {
			logger.Debug("脚本输出: %s", string(output))
		}
	}
}

// copyFile 拷贝文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("文件拷贝失败: %v", err)
	}
	
	return nil
}

// printControlPanelURL 打印控制面板地址
func printControlPanelURL(cfg *model.Config) {
	if cfg.Experimental.ClashAPI.ExternalController != "" {
		controlURL := fmt.Sprintf("http://%s/ui/#/proxies", cfg.Experimental.ClashAPI.ExternalController)
		logger.Success("控制面板地址：%s", controlURL)
	}
}
