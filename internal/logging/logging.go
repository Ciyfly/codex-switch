package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level 表示日志级别
type Level int

const (
	// Debug 级别，输出最详细信息
	Debug Level = iota
	// Info 级别，输出关键运行信息
	Info
	// Warn 级别，提示潜在问题
	Warn
	// Error 级别，记录错误
	Error
)

var (
	once      sync.Once
	logger    *log.Logger
	level     = Info
	logWriter io.Writer
)

// Init 初始化日志系统
func Init(customPath string) error {
	var initErr error
	once.Do(func() {
		path := customPath
		if path == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				initErr = err
				return
			}
			dir := filepath.Join(home, ".codex-switch", "logs")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				initErr = err
				return
			}
			path = filepath.Join(dir, "ckm.log")
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				initErr = err
				return
			}
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			initErr = err
			return
		}

		logWriter = file
		logger = log.New(file, "", 0)

		if envLevel := os.Getenv("CKM_LOG_LEVEL"); envLevel != "" {
			if parsed, ok := parseLevel(envLevel); ok {
				level = parsed
			}
		}
	})
	return initErr
}

// Close 关闭日志文件
func Close() error {
	if closer, ok := logWriter.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Debugf 输出调试日志
func Debugf(format string, args ...any) {
	write(Debug, format, args...)
}

// Infof 输出普通信息
func Infof(format string, args ...any) {
	write(Info, format, args...)
}

// Warnf 输出警告
func Warnf(format string, args ...any) {
	write(Warn, format, args...)
}

// Errorf 输出错误
func Errorf(format string, args ...any) {
	write(Error, format, args...)
}

func write(l Level, format string, args ...any) {
	if logger == nil || l < level {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	prefix := levelPrefix(l)
	message := fmt.Sprintf(format, args...)
	logger.Printf("[%s] [%s] %s", ts, prefix, message)
}

func levelPrefix(l Level) string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	default:
		return "INFO"
	}
}

func parseLevel(v string) (Level, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return Debug, true
	case "info":
		return Info, true
	case "warn", "warning":
		return Warn, true
	case "error":
		return Error, true
	default:
		return Info, false
	}
}
