package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"singbox_sub/src/github.com/sixproxy/logger"
	"singbox_sub/src/github.com/sixproxy/model"
	"singbox_sub/src/github.com/sixproxy/protocol"
	"singbox_sub/src/github.com/sixproxy/service"
	"singbox_sub/src/github.com/sixproxy/util/files"
	"singbox_sub/src/github.com/sixproxy/util/shells"
	"singbox_sub/src/github.com/sixproxy/version"
	"time"
)

func main() {

	// 配置处理逻辑
	logger.Info("🔄 加载配置文件...")
	userService := service.ConfigService{}
	userConfig, err := userService.LoadConfig("config/config.yaml")
	if err != nil {
		logger.Fatal("加载用户自定义配置文件失败: %v", err)
	}

	// 注入singbox处理逻辑
	boxService := &service.SingBoxService{MirrorURL: userConfig.GitHub.MirrorURL}

	// 解析命令行参数
	var (
		targetOS    = flag.String("os", "auto", "目标操作系统 (auto/darwin/linux/windows/all)")
		verbose     = flag.Bool("v", false, "详细输出 (启用DEBUG日志)")
		help        = flag.Bool("h", false, "显示帮助信息")
		versionFlag = flag.Bool("version", false, "显示版本信息")
		update      = flag.Bool("update", false, "检查并更新到最新版本")
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
			handleBoxCommand(args[1:], boxService)
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

	// 停止sing-box服务
	if runtime.GOOS == "linux" && (*targetOS == "auto" || *targetOS == "linux") {
		shells.StopSingBox()
		// 等待1秒确保sing-box完全停止，避免网络检测时仍通过代理
		logger.Info("等待sing-box服务完全停止...")
		time.Sleep(1 * time.Second)
	}

	// 加载模版并合并YAML配置（包含GitHub镜像处理）
	template, err := userService.LoadTemplate("config/template-v1.12.json")
	if err != nil {
		logger.Fatal("加载模版失败: %v", err)
	}
	// 打印控制面板地址
	printControlPanelURL(template)

	// 订阅处理逻辑
	var subService = &service.SubService{Cfg: template}
	err = subService.RenderTemplate(delegateParse)
	if err != nil {
		logger.Error("渲染模板失败: %v", err)
	}

	// 生成配置
	generateSystemConfig(subService, boxService, *targetOS, userService.UserConfig.GitHub.MirrorURL)

}

// 按协议解析节点
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
	updaterInstance, err := service.NewUpdaterService()
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
func generateSystemConfig(cfg *service.SubService, boxService *service.SingBoxService, targetOS, mirrorURL string) {
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
				shells.StartSingBox()
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

// printControlPanelURL 打印控制面板地址
func printControlPanelURL(cfg *model.Config) {
	if cfg.Experimental.ClashAPI.ExternalController != "" {
		controlURL := fmt.Sprintf("http://%s/ui/#/proxies", cfg.Experimental.ClashAPI.ExternalController)
		logger.Success("控制面板地址：%s", controlURL)
	}
}

// handleBoxCommand 处理box子命令
func handleBoxCommand(args []string, boxService *service.SingBoxService) {
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}

	switch action {
	case "install", "i":
		if err := boxService.CheckAndInstall(); err != nil {
			logger.Error("sing-box安装失败: %v", err)
			os.Exit(1)
		}
	case "update", "u":
		if err := boxService.CheckAndInstall(); err != nil {
			logger.Error("sing-box更新失败: %v", err)
			os.Exit(1)
		}
	case "status", "s":
		boxService.ShowSingboxStatus()
	case "version", "v":
		boxService.ShowSingboxVersion()
	case "help", "h":
		printBoxUsage()
	default:
		logger.Error("未知的box命令: %s", action)
		printBoxUsage()
		os.Exit(1)
	}
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
