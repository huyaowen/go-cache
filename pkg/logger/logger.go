package logger

import (
	"log"
	"os"
)

// Logger 提供分级别的日志记录器
type Logger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
}

// Default 默认日志器实例
var Default = New()

// New 创建新的 Logger 实例
func New() *Logger {
	return &Logger{
		debug: log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		info:  log.New(os.Stderr, "[INFO] ", log.LstdFlags|log.Lshortfile),
		warn:  log.New(os.Stderr, "[WARN] ", log.LstdFlags|log.Lshortfile),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
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
