package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *logrus.Logger

// Init 初始化日志系统（带文件输出）
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

	// 创建日志目录
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 配置日志轮转
	fileWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,      // MB
		MaxBackups: maxDays,  // 保留备份数
		MaxAge:     maxDays,  // 天
		Compress:   true,     // 压缩
		LocalTime:  true,
	}

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)
	Log.SetOutput(multiWriter)

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
