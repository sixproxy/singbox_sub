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

func init() {
	// 默认初始化为INFO级别，输出到标准输出
	globalLogger = &Logger{
		level:  INFO,
		logger: log.New(os.Stdout, "", 0), // 不使用默认前缀，我们自定义格式
	}
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
	
	// 格式: [时间] [级别] [调用位置] 消息
	return fmt.Sprintf("[%s] [%s] [%s] %s", 
		timestamp, 
		level.String(), 
		caller, 
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
	Info("[CONFIG] "+format, args...)
}

func ConfigWarn(format string, args ...interface{}) {
	Warn("[CONFIG] "+format, args...)
}

func ConfigError(format string, args ...interface{}) {
	Error("[CONFIG] "+format, args...)
}

// 网络相关的便利方法
func NetworkInfo(format string, args ...interface{}) {
	Info("[NETWORK] "+format, args...)
}

func NetworkWarn(format string, args ...interface{}) {
	Warn("[NETWORK] "+format, args...)
}

func NetworkError(format string, args ...interface{}) {
	Error("[NETWORK] "+format, args...)
}

// 解析相关的便利方法
func ParseInfo(format string, args ...interface{}) {
	Info("[PARSE] "+format, args...)
}

func ParseWarn(format string, args ...interface{}) {
	Warn("[PARSE] "+format, args...)
}

func ParseError(format string, args ...interface{}) {
	Error("[PARSE] "+format, args...)
}