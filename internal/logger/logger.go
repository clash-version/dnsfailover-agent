package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// MemoryHook 内存日志钩子
type MemoryHook struct {
	buffer *LogBuffer
}

// Levels 返回支持的日志级别
func (hook *MemoryHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 当日志触发时调用
func (hook *MemoryHook) Fire(entry *logrus.Entry) error {
	if hook.buffer != nil {
		message, _ := entry.String()
		hook.buffer.AddLog(entry.Level.String(), message)
	}
	return nil
}

// Init 初始化日志系统（仅控制台输出+内存缓冲）
func Init(level, logPath string, maxDays int) error {
	Log = logrus.New()

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Log.SetLevel(logLevel)

	// 设置日志格式
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	// 仅输出到控制台
	Log.SetOutput(os.Stdout)

	// 初始化内存缓冲区（保留最近1000条日志）
	InitBuffer(1000)

	// 添加内存钩子
	Log.AddHook(&MemoryHook{buffer: GetBuffer()})

	return nil
}

// InitConsoleOnly 初始化日志系统（仅控制台输出）
func InitConsoleOnly(level string) {
	Log = logrus.New()

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Log.SetLevel(logLevel)

	// 设置日志格式
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	// 仅输出到控制台
	Log.SetOutput(os.Stdout)
}

// Debug 调试日志
func Debug(args ...interface{}) {
	if Log != nil {
		Log.Debug(args...)
	}
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	if Log != nil {
		Log.Debugf(format, args...)
	}
}

// Info 信息日志
func Info(args ...interface{}) {
	if Log != nil {
		Log.Info(args...)
	}
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	if Log != nil {
		Log.Infof(format, args...)
	}
}

// Warn 警告日志
func Warn(args ...interface{}) {
	if Log != nil {
		Log.Warn(args...)
	}
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	if Log != nil {
		Log.Warnf(format, args...)
	}
}

// Error 错误日志
func Error(args ...interface{}) {
	if Log != nil {
		Log.Error(args...)
	}
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	if Log != nil {
		Log.Errorf(format, args...)
	}
}

// Fatal 致命错误日志
func Fatal(args ...interface{}) {
	if Log != nil {
		Log.Fatal(args...)
	}
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	if Log != nil {
		Log.Fatalf(format, args...)
	}
}
