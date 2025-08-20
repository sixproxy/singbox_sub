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
	"singbox_sub/src/github.com/sixproxy/updater"
	"singbox_sub/src/github.com/sixproxy/version"
	"time"
)

func main() {
	// 0.解析命令行参数
	var (
		targetOS    = flag.String("os", "auto", "目标操作系统 (auto/darwin/linux/windows/all)")
		verbose     = flag.Bool("v", false, "详细输出 (启用DEBUG日志)")
		help        = flag.Bool("h", false, "显示帮助信息")
		versionFlag = flag.Bool("version", false, "显示版本信息")
		update      = flag.Bool("update", false, "检查并更新到最新版本")
	)
	flag.Parse()

	// 处理非标志参数命令 (如 "sub version", "sub update")
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "version":
			version.PrintVersion()
			return
		case "update":
			handleUpdate()
			return
		case "help":
			printUsage()
			return
		}
	}

	// 处理版本命令
	if *versionFlag {
		version.PrintVersion()
		return
	}

	// 处理更新命令
	if *update {
		handleUpdate()
		return
	}

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
		// 等待1秒确保sing-box完全停止，避免网络检测时仍通过代理
		logger.Info("等待sing-box服务完全停止...")
		time.Sleep(1 * time.Second)
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

// handleUpdate 处理更新命令
func handleUpdate() {
	updaterInstance, err := updater.NewUpdater()
	if err != nil {
		logger.Error("创建更新器失败: %v", err)
		return
	}
	defer updaterInstance.Cleanup()

	if err := updaterInstance.CheckUpdate(); err != nil {
		logger.Error("更新失败: %v", err)
	}
}

// printUsage 显示使用帮助
func printUsage() {
	logger.Info("=== sing-box配置生成器 ===")
	logger.Info("用法: %s [选项]", "sub")
	logger.Info("")
	logger.Info("选项:")
	logger.Info("  -os string    目标操作系统 (默认: auto)")
	logger.Info("                可选值: auto, darwin, linux, windows, all")
	logger.Info("  -v            详细输出模式 (启用DEBUG日志)")
	logger.Info("  -h            显示此帮助信息")
	logger.Info("  -version      显示版本信息")
	logger.Info("  -update       检查并更新到最新版本")
	logger.Info("")
	logger.Info("Linux自动化功能 (仅在Linux系统上生效):")
	logger.Info("  • 程序启动时自动停止sing-box服务")
	logger.Info("  • 等待1秒确保服务完全停止，避免网络检测错误")
	logger.Info("  • 配置生成后自动部署到/etc/sing-box/config.json")
	logger.Info("  • 部署完成后自动启动sing-box服务")
	logger.Info("  • 需要bash/stop_singbox.sh和bash/start_singbox.sh脚本")
	logger.Info("")
	logger.Info("示例:")
	logger.Info("  ./sub                            # 自动检测系统类型")
	logger.Info("  ./sub -os darwin                 # 强制生成macOS配置")
	logger.Info("  ./sub -os linux                  # 强制生成Linux配置")
	logger.Info("  ./sub -os all                    # 生成所有类型配置")
	logger.Info("  ./sub -v                         # 详细输出模式")
	logger.Info("  ./sub version                    # 查看版本信息")
	logger.Info("  ./sub update                     # 检查并更新程序")
	logger.Info("  ./sub -version                   # 查看版本信息 (标志形式)")
	logger.Info("  ./sub -update                    # 检查并更新程序 (标志形式)")
	logger.Info("")
	logger.Info("Linux生产环境:")
	logger.Info("  ./sub                            # 完整自动化部署")
	logger.Info("  ./sub -v                         # 详细查看部署过程")
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
		logger.Info("sing-box服务停止命令已执行")
		if len(output) > 0 {
			logger.Debug("脚本输出: %s", string(output))
		}
		
		// 验证服务是否真的停止了
		if isSingBoxRunning() {
			logger.Warn("sing-box进程可能仍在运行，建议手动检查")
		} else {
			logger.Info("确认sing-box服务已完全停止")
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

// isSingBoxRunning 检查sing-box进程是否仍在运行
func isSingBoxRunning() bool {
	// 使用pgrep命令检查sing-box进程
	cmd := exec.Command("pgrep", "-x", "sing-box")
	err := cmd.Run()
	// 如果pgrep找到进程，返回码为0；找不到进程返回码为1
	return err == nil
}

// printControlPanelURL 打印控制面板地址
func printControlPanelURL(cfg *model.Config) {
	if cfg.Experimental.ClashAPI.ExternalController != "" {
		controlURL := fmt.Sprintf("http://%s/ui/#/proxies", cfg.Experimental.ClashAPI.ExternalController)
		logger.Success("控制面板地址：%s", controlURL)
	}
}
