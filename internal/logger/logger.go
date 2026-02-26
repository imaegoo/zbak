package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger 日志记录器接口
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	SetOutput(w io.Writer)
}

// logger 日志记录器实现
type logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	writer      io.Writer
}

// NewLogger 创建新的日志记录器
// targetDir: 目标目录，用于创建日志文件
func NewLogger(targetDir string) (Logger, error) {
	// 创建日志文件名：backup-YYYY-MM-DD-HH-MM-SS.log
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	logFileName := fmt.Sprintf("backup-%s.log", timestamp)
	logFilePath := filepath.Join(targetDir, logFileName)

	// 创建目标目录（如果不存在）
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 打开日志文件
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %w", err)
	}

	// 同时输出到标准输出和日志文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	l := &logger{
		infoLogger:  log.New(multiWriter, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(multiWriter, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(multiWriter, "[ERROR] ", log.LstdFlags),
		writer:      multiWriter,
	}

	return l, nil
}

// Info 记录信息日志
func (l *logger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.infoLogger.Printf(msg, args...)
	} else {
		l.infoLogger.Println(msg)
	}
}

// Warn 记录警告日志
func (l *logger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.warnLogger.Printf(msg, args...)
	} else {
		l.warnLogger.Println(msg)
	}
}

// Error 记录错误日志
func (l *logger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.errorLogger.Printf(msg, args...)
	} else {
		l.errorLogger.Println(msg)
	}
}

// SetOutput 设置输出目标
func (l *logger) SetOutput(w io.Writer) {
	l.writer = w
	l.infoLogger.SetOutput(w)
	l.warnLogger.SetOutput(w)
	l.errorLogger.SetOutput(w)
}
