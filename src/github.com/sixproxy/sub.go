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
		targetOS         = flag.String("os", "auto", "目标操作系统 (auto/darwin/linux/windows/all)")
		verbose          = flag.Bool("v", false, "详细输出 (启用DEBUG日志)")
		help             = flag.Bool("h", false, "显示帮助信息")
		versionFlag      = flag.Bool("version", false, "显示版本信息")
		update           = flag.Bool("update", false, "检查并更新到最新版本")
		skipSingboxCheck = flag.Bool("skip-singbox-check", false, "跳过sing-box版本检查")
	)
	flag.Parse()

	// 处理非标志参数命令 (如 "sub version", "sub update", "sub box")
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "version":
			version.PrintVersion()
			return
		case "update":
			handleUpdate()
			return
		case "box":
			handleBoxCommand(args[1:])
			return
		case "install-singbox":
			// 保持向后兼容
			handleSingboxInstall()
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

	// 0.3.检查sing-box状态 (如果没有跳过检查)
	if !*skipSingboxCheck {
		checkSingboxStatus()
	}

	// 0.5.Linux系统预处理 - 停止sing-box服务
	if runtime.GOOS == "linux" && (*targetOS == "auto" || *targetOS == "linux") {
		stopSingBoxService()
		// 等待1秒确保sing-box完全停止，避免网络检测时仍通过代理
		logger.Info("等待sing-box服务完全停止...")
		time.Sleep(1 * time.Second)
	}

	// 1.加载模版并合并YAML配置（包含GitHub镜像处理）
	logger.Info("🔄 加载配置文件...")
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
	logger.Info("  -os string              目标操作系统 (默认: auto)")
	logger.Info("                          可选值: auto, darwin, linux, windows, all")
	logger.Info("  -v                      详细输出模式 (启用DEBUG日志)")
	logger.Info("  -h                      显示此帮助信息")
	logger.Info("  -version                显示版本信息")
	logger.Info("  -update                 检查并更新到最新版本")
	logger.Info("")
	logger.Info("Linux自动化功能 (仅在Linux系统上生效):")
	logger.Info("示例:")
	logger.Info("  ./sub                            # 自动检测系统类型")
	logger.Info("  ./sub -os darwin                 # 强制生成macOS配置")
	logger.Info("  ./sub -os linux                  # 强制生成Linux配置")
	logger.Info("  ./sub -os all                    # 生成所有类型配置")
	logger.Info("  ./sub -v                         # 详细输出模式")
	logger.Info("  ./sub version                    # 查看版本信息")
	logger.Info("  ./sub update                     # 检查并更新程序")
	logger.Info("  ./sub box                        # 显示sing-box状态")
	logger.Info("  ./sub box install                # 安装/更新sing-box")
	logger.Info("  ./sub -version                   # 查看版本信息 (标志形式)")
	logger.Info("  ./sub -update                    # 检查并更新程序 (标志形式)")
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

// startSingBoxService 启动sing-box服务（带失败检测和回滚）
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

	// 备份当前配置
	configBackupPath := "/etc/sing-box/config.json.backup"
	configPath := "/etc/sing-box/config.json"
	if _, err := os.Stat(configPath); err == nil {
		if err := copyFile(configPath, configBackupPath); err != nil {
			logger.Warn("备份配置文件失败: %v", err)
		} else {
			logger.Debug("已备份配置文件到: %s", configBackupPath)
		}
	}

	logger.Debug("使用shell: %s", shell)
	cmd := exec.Command(shell, scriptPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error("启动sing-box服务失败: %v", err)
		logger.Debug("脚本输出: %s", string(output))

		// 尝试回滚配置并重新启动
		handleStartupFailure(configBackupPath, configPath)
		return
	}

	logger.Info("sing-box服务启动命令已执行")
	if len(output) > 0 {
		logger.Debug("脚本输出: %s", string(output))
	}

	// 等待并检查启动状态
	if !checkSingboxStartupStatus() {
		logger.Error("sing-box启动失败，正在回滚配置...")
		handleStartupFailure(configBackupPath, configPath)
	} else {
		logger.Info("✅ sing-box服务启动成功")
		// 清理备份文件
		if err := os.Remove(configBackupPath); err == nil {
			logger.Debug("已清理配置备份文件")
		}
	}
}

// checkSingboxStartupStatus 检查sing-box启动状态
func checkSingboxStartupStatus() bool {
	logger.Info("检查sing-box启动状态...")

	// 等待几秒钟让服务完全启动
	maxWait := 10 * time.Second
	checkInterval := 1 * time.Second
	waited := time.Duration(0)

	for waited < maxWait {
		time.Sleep(checkInterval)
		waited += checkInterval

		// 检查进程是否存在
		if isSingBoxRunning() {
			logger.Debug("sing-box进程运行中...")

			// 尝试获取版本信息来验证服务状态
			manager := updater.NewSingboxManager()
			if manager.IsInstalled() {
				if version, err := manager.GetInstalledVersion(); err == nil {
					logger.Debug("sing-box版本验证成功: %s", version.Version)

					// 额外等待2秒确保服务完全稳定
					time.Sleep(2 * time.Second)

					// 最后检查进程是否仍在运行
					if isSingBoxRunning() {
						return true
					} else {
						logger.Warn("sing-box进程意外停止")
						return false
					}
				} else {
					logger.Debug("版本验证失败，可能尚未完全启动: %v", err)
				}
			}
		} else {
			logger.Debug("sing-box进程未运行...")
		}
	}

	logger.Error("等待 %.0f 秒后，sing-box仍未成功启动", maxWait.Seconds())
	return false
}

// handleStartupFailure 处理启动失败，回滚配置并重启
func handleStartupFailure(backupPath, configPath string) {
	logger.Error("🚨 sing-box启动失败，开始故障处理...")

	// 1. 显示失败原因（尝试获取服务日志）
	showSingboxFailureReason()

	// 2. 停止可能存在的异常进程
	stopSingBoxService()
	time.Sleep(2 * time.Second)

	// 3. 检查是否有备份配置可以回滚
	if _, err := os.Stat(backupPath); err == nil {
		logger.Info("🔄 回滚到之前的配置...")

		if err := copyFile(backupPath, configPath); err != nil {
			logger.Error("回滚配置失败: %v", err)
			return
		}

		logger.Info("配置已回滚，尝试重新启动sing-box...")

		// 4. 尝试使用回滚的配置重新启动
		shell := getAvailableShell()
		if shell != "" {
			scriptPath := "bash/start_singbox.sh"
			cmd := exec.Command(shell, scriptPath)
			output, err := cmd.CombinedOutput()

			if err != nil {
				logger.Error("使用回滚配置启动失败: %v", err)
				logger.Debug("输出: %s", string(output))
			} else {
				logger.Info("正在验证回滚配置启动状态...")
				time.Sleep(3 * time.Second)

				if isSingBoxRunning() {
					logger.Info("✅ 使用回滚配置成功启动sing-box")
					// 清理失败的配置文件（重命名为.failed）
					failedConfigPath := configPath + ".failed"
					if newConfigExists(configPath, backupPath) {
						// 只有当新配置与备份配置不同时才保存失败配置
						copyFile(configPath, failedConfigPath)
						logger.Info("失败的配置已保存为: %s", failedConfigPath)
					}
				} else {
					logger.Error("❌ 即使使用回滚配置也无法启动sing-box")
				}
			}
		}
	} else {
		logger.Warn("⚠️  没有找到配置备份，无法自动回滚")
		logger.Info("请手动检查配置文件: %s", configPath)
	}
}

// showSingboxFailureReason 显示sing-box启动失败的具体原因
func showSingboxFailureReason() {
	logger.Info("🔍 分析启动失败原因...")

	// 1. 检查配置文件语法
	configPath := "/etc/sing-box/config.json"
	if _, err := os.Stat(configPath); err == nil {
		// 尝试使用sing-box检查配置
		manager := updater.NewSingboxManager()
		if manager.IsInstalled() {
			cmd := exec.Command(manager.GetBinaryPath(), "check", "-c", configPath)
			output, err := cmd.CombinedOutput()

			if err != nil {
				logger.Error("❌ 配置文件检查失败:")
				logger.Error(string(output))
			} else {
				logger.Info("✅ 配置文件语法正确")
			}
		}
	}

	// 2. 检查系统资源
	logger.Debug("检查系统资源...")

	// 3. 尝试获取系统日志中的错误信息
	if runtime.GOOS == "linux" {
		// 尝试从systemd日志获取错误
		cmd := exec.Command("journalctl", "-u", "sing-box", "--no-pager", "-n", "10")
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			logger.Info("📋 最近的系统日志:")
			logger.Info(string(output))
		}
	}
}

// newConfigExists 检查新配置是否与备份配置不同
func newConfigExists(configPath, backupPath string) bool {
	configData, err1 := os.ReadFile(configPath)
	backupData, err2 := os.ReadFile(backupPath)

	if err1 != nil || err2 != nil {
		return true // 如果无法读取，假设它们不同
	}

	return string(configData) != string(backupData)
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
	cmd := exec.Command("pgrep", "sing-box")
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

// handleBoxCommand 处理box子命令
func handleBoxCommand(args []string) {
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}

	manager := updater.NewSingboxManager()

	switch action {
	case "install", "i":
		if err := manager.CheckAndInstall(); err != nil {
			logger.Error("sing-box安装失败: %v", err)
			os.Exit(1)
		}
	case "update", "u":
		if err := manager.CheckAndInstall(); err != nil {
			logger.Error("sing-box更新失败: %v", err)
			os.Exit(1)
		}
	case "status", "s":
		showSingboxStatus(manager)
	case "version", "v":
		showSingboxVersion(manager)
	case "help", "h":
		printBoxUsage()
	default:
		logger.Error("未知的box命令: %s", action)
		printBoxUsage()
		os.Exit(1)
	}
}

// handleSingboxInstall 处理sing-box安装命令 (向后兼容)
func handleSingboxInstall() {
	manager := updater.NewSingboxManager()
	if err := manager.CheckAndInstall(); err != nil {
		logger.Error("sing-box安装/更新失败: %v", err)
		os.Exit(1)
	}
}

// showSingboxStatus 显示sing-box状态
func showSingboxStatus(manager *updater.SingboxManager) {
	logger.Info("🔍 sing-box状态检查")

	if manager.IsInstalled() {
		if version, err := manager.GetInstalledVersion(); err == nil {
			logger.Info("✅ 已安装版本: %s", version.Version)
		} else {
			logger.Warn("⚠️ 已安装但无法获取版本: %v", err)
		}

		if hasUpdate, latest, err := manager.IsUpdateAvailable(); err == nil {
			if hasUpdate {
				logger.Info("🆕 有新版本可用: %s", latest.TagName)
				logger.Info("💡 运行 './sub box install' 更新")
			} else {
				logger.Info("✅ 已是最新版本")
			}
		} else {
			logger.Warn("⚠️ 检查更新失败: %v", err)
		}
	} else {
		logger.Info("❌ sing-box未安装")
		logger.Info("💡 运行 './sub box install' 安装")
	}
}

// showSingboxVersion 显示sing-box版本信息
func showSingboxVersion(manager *updater.SingboxManager) {
	if !manager.IsInstalled() {
		logger.Error("❌ sing-box未安装")
		os.Exit(1)
	}

	version, err := manager.GetInstalledVersion()
	if err != nil {
		logger.Error("获取版本失败: %v", err)
		os.Exit(1)
	}

	logger.Info("sing-box version %s", version.Version)
	logger.Info("Binary path: %s", manager.GetBinaryPath())
	logger.Info("Config path: %s", manager.GetConfigPath())
}

// printBoxUsage 显示box命令帮助
func printBoxUsage() {
	logger.Info("=== sing-box管理命令 ===")
	logger.Info("用法: sub box <命令>")
	logger.Info("")
	logger.Info("可用命令:")
	logger.Info("  install, i     安装或更新sing-box")
	logger.Info("  update, u      更新sing-box (同install)")
	logger.Info("  status, s      显示sing-box状态")
	logger.Info("  version, v     显示sing-box版本信息")
	logger.Info("  help, h        显示此帮助信息")
	logger.Info("")
	logger.Info("示例:")
	logger.Info("  ./sub box                    # 显示状态")
	logger.Info("  ./sub box install            # 安装sing-box")
	logger.Info("  ./sub box status             # 检查状态")
	logger.Info("  ./sub box version            # 显示版本")
}

// checkSingboxStatus 检查sing-box状态
func checkSingboxStatus() {
	manager := updater.NewSingboxManager()

	if manager.IsInstalled() {
		version, err := manager.GetInstalledVersion()
		if err != nil {
			logger.Warn("无法获取sing-box版本信息: %v", err)
		} else {
			logger.Info("检测到sing-box版本: %s", version.Version)
		}

		// 检查是否有更新
		hasUpdate, latest, err := manager.IsUpdateAvailable()
		if err != nil {
			logger.Warn("检查sing-box更新失败: %v", err)
		} else if hasUpdate {
			logger.Info("发现sing-box新版本: %s", latest.TagName)
			logger.Info("提示: 使用 './sub box install' 更新到最新版本")
		}
	} else {
		logger.Warn("未检测到sing-box，建议先安装sing-box")
		logger.Info("提示: 使用 './sub box installx' 安装最新版本")
	}
}
