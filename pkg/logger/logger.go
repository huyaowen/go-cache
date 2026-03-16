package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	NONE // 禁用所有日志
)

// ParseLevel 解析日志级别字符串
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "NONE", "OFF":
		return NONE
	default:
		return INFO
	}
}

// Logger 提供分级别的日志记录器
type Logger struct {
	level Level
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
}

// Default 默认日志器实例
var Default = New()

// New 创建新的 Logger 实例
func New() *Logger {
	return NewWithLevel(INFO)
}

// NewWithLevel 创建指定级别的 Logger 实例
func NewWithLevel(level Level) *Logger {
	var debugOut, infoOut, warnOut, errorOut io.Writer

	// 根据级别决定输出
	if level <= DEBUG {
		debugOut = os.Stderr
	} else {
		debugOut = io.Discard
	}

	if level <= INFO {
		infoOut = os.Stderr
	} else {
		infoOut = io.Discard
	}

	if level <= WARN {
		warnOut = os.Stderr
	} else {
		warnOut = io.Discard
	}

	if level <= ERROR {
		errorOut = os.Stderr
	} else {
		errorOut = io.Discard
	}

	return &Logger{
		level: level,
		debug: log.New(debugOut, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		info:  log.New(infoOut, "[INFO] ", log.LstdFlags|log.Lshortfile),
		warn:  log.New(warnOut, "[WARN] ", log.LstdFlags|log.Lshortfile),
		error: log.New(errorOut, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.level = level

	// 重新配置输出
	var debugOut, infoOut, warnOut, errorOut io.Writer

	if level <= DEBUG {
		debugOut = os.Stderr
	} else {
		debugOut = io.Discard
	}

	if level <= INFO {
		infoOut = os.Stderr
	} else {
		infoOut = io.Discard
	}

	if level <= WARN {
		warnOut = os.Stderr
	} else {
		warnOut = io.Discard
	}

	if level <= ERROR {
		errorOut = os.Stderr
	} else {
		errorOut = io.Discard
	}

	l.debug.SetOutput(debugOut)
	l.info.SetOutput(infoOut)
	l.warn.SetOutput(warnOut)
	l.error.SetOutput(errorOut)
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() Level {
	return l.level
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

// Info 记录信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	l.warn.Printf(format, v...)
}

// Error 记录错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}

// 包级别的便捷函数，直接使用 logger.Debug() 等

// Debug 记录调试日志（使用默认日志器）
func Debug(format string, v ...interface{}) {
	Default.Debug(format, v...)
}

// Info 记录信息日志（使用默认日志器）
func Info(format string, v ...interface{}) {
	Default.Info(format, v...)
}

// Warn 记录警告日志（使用默认日志器）
func Warn(format string, v ...interface{}) {
	Default.Warn(format, v...)
}

// Error 记录错误日志（使用默认日志器）
func Error(format string, v ...interface{}) {
	Default.Error(format, v...)
}

// SetLevel 设置全局日志级别（通过环境变量 LOG_LEVEL 控制）
func SetLevel(level Level) {
	Default.SetLevel(level)
}

// GetLevel 获取全局日志级别
func GetLevel() Level {
	return Default.GetLevel()
}

// init 初始化时从环境变量读取日志级别
func init() {
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		Default.SetLevel(ParseLevel(levelStr))
	}
}
