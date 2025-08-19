package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String 返回日志级别的字符串表示
func (level LogLevel) String() string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器结构
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// 全局日志实例
var globalLogger *Logger
var colorEnabled = true // 默认启用颜色

func init() {
	// 默认初始化为INFO级别，输出到标准输出
	globalLogger = &Logger{
		level:  INFO,
		logger: log.New(os.Stdout, "", 0), // 不使用默认前缀，我们自定义格式
	}
	
	// 检查环境变量，某些CI环境可能不支持颜色
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		colorEnabled = false
	}
}

// SetColorEnabled 设置是否启用颜色输出
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// IsColorEnabled 检查是否启用了颜色
func IsColorEnabled() bool {
	return colorEnabled
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	globalLogger.level = level
}

// GetLevel 获取当前日志级别
func GetLevel() LogLevel {
	return globalLogger.level
}

// SetOutput 设置日志输出目标
func SetOutput(output *os.File) {
	globalLogger.logger.SetOutput(output)
}

// getLevelColor 获取日志级别对应的颜色
func getLevelColor(level LogLevel) (string, string) {
	if !colorEnabled {
		return "", ""
	}
	
	switch level {
	case DEBUG:
		return ColorDim + ColorCyan, ColorReset  // 暗青色 - 调试信息不太重要
	case INFO:
		return ColorBlue, ColorReset              // 蓝色 - 普通信息
	case WARN:
		return ColorBold + ColorYellow, ColorReset // 粗体黄色 - 警告
	case ERROR:
		return ColorBold + ColorRed, ColorReset    // 粗体红色 - 错误
	case FATAL:
		return ColorBold + ColorBrightRed, ColorReset // 粗体亮红色 - 致命错误
	default:
		return "", ""
	}
}

// applyColor 应用颜色到文本（如果启用颜色）
func applyColor(color, text, reset string) string {
	if !colorEnabled {
		return text
	}
	return color + text + reset
}

// formatMessage 格式化日志消息
func (l *Logger) formatMessage(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3) // 跳过3层调用栈
	caller := ""
	if ok {
		// 只保留文件名，不要完整路径
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]
		caller = fmt.Sprintf("%s:%d", filename, line)
	}
	
	message := fmt.Sprintf(format, args...)
	
	// 应用颜色
	coloredTimestamp := applyColor(ColorDim, timestamp, ColorReset)
	levelColor, resetColor := getLevelColor(level)
	coloredLevel := levelColor + level.String() + resetColor
	coloredCaller := applyColor(ColorDim, caller, ColorReset)
	
	// 格式: [时间] [彩色级别] [调用位置] 消息
	return fmt.Sprintf("[%s] [%s] [%s] %s", 
		coloredTimestamp,
		coloredLevel,
		coloredCaller,
		message)
}

// shouldLog 检查是否应该记录此级别的日志
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// log 通用日志记录方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	
	message := l.formatMessage(level, format, args...)
	l.logger.Println(message)
	
	// FATAL级别日志后退出程序
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 记录DEBUG级别日志
func Debug(format string, args ...interface{}) {
	globalLogger.log(DEBUG, format, args...)
}

// Info 记录INFO级别日志
func Info(format string, args ...interface{}) {
	globalLogger.log(INFO, format, args...)
}

// Warn 记录WARN级别日志
func Warn(format string, args ...interface{}) {
	globalLogger.log(WARN, format, args...)
}

// Error 记录ERROR级别日志
func Error(format string, args ...interface{}) {
	globalLogger.log(ERROR, format, args...)
}

// Fatal 记录FATAL级别日志并退出程序
func Fatal(format string, args ...interface{}) {
	globalLogger.log(FATAL, format, args...)
}

// DebugEnabled 检查是否启用DEBUG日志
func DebugEnabled() bool {
	return globalLogger.shouldLog(DEBUG)
}

// InfoEnabled 检查是否启用INFO日志
func InfoEnabled() bool {
	return globalLogger.shouldLog(INFO)
}

// 便利方法：不带格式化的日志记录
func Debugln(args ...interface{}) {
	if globalLogger.shouldLog(DEBUG) {
		Debug("%s", fmt.Sprint(args...))
	}
}

func Infoln(args ...interface{}) {
	if globalLogger.shouldLog(INFO) {
		Info("%s", fmt.Sprint(args...))
	}
}

func Warnln(args ...interface{}) {
	if globalLogger.shouldLog(WARN) {
		Warn("%s", fmt.Sprint(args...))
	}
}

func Errorln(args ...interface{}) {
	if globalLogger.shouldLog(ERROR) {
		Error("%s", fmt.Sprint(args...))
	}
}

func Fatalln(args ...interface{}) {
	Fatal("%s", fmt.Sprint(args...))
}

// 配置相关的便利方法
func ConfigInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorGreen, "[CONFIG]", ColorReset)
	Info("%s %s", tag, message)
}

func ConfigWarn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorYellow, "[CONFIG]", ColorReset)
	Warn("%s %s", tag, message)
}

func ConfigError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorRed, "[CONFIG]", ColorReset)
	Error("%s %s", tag, message)
}

// 网络相关的便利方法
func NetworkInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorCyan, "[NETWORK]", ColorReset)
	Info("%s %s", tag, message)
}

func NetworkWarn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorYellow, "[NETWORK]", ColorReset)
	Warn("%s %s", tag, message)
}

func NetworkError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorRed, "[NETWORK]", ColorReset)
	Error("%s %s", tag, message)
}

// 解析相关的便利方法
func ParseInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorPurple, "[PARSE]", ColorReset)
	Info("%s %s", tag, message)
}

func ParseWarn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorYellow, "[PARSE]", ColorReset)
	Warn("%s %s", tag, message)
}

func ParseError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	tag := applyColor(ColorBold+ColorRed, "[PARSE]", ColorReset)
	Error("%s %s", tag, message)
}

// ANSI 颜色代码
const (
	ColorReset    = "\033[0m"
	ColorRed      = "\033[31m"
	ColorGreen    = "\033[32m"
	ColorYellow   = "\033[33m"
	ColorBlue     = "\033[34m"
	ColorPurple   = "\033[35m"
	ColorCyan     = "\033[36m"
	ColorWhite    = "\033[37m"
	ColorBold     = "\033[1m"
	ColorDim      = "\033[2m"
	
	// 亮色版本
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightPurple  = "\033[95m"
	ColorBrightCyan    = "\033[96m"
)

// Success 打印绿色高亮的成功信息
func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// 使用绿色高亮和加粗显示
	coloredMessage := fmt.Sprintf("[%s] %s%s%s%s", 
		timestamp,
		ColorBold,
		ColorGreen,
		message,
		ColorReset)
	
	globalLogger.logger.Println(coloredMessage)
}