package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// Level constants.
const (
	levelDebug = 0
	levelInfo  = 1
	levelWarn  = 2
	levelError = 3
	levelFatal = 4
)

var logState struct {
	mu       sync.Mutex
	file     *os.File
	dir      string
	curPath  string // path to active log file
	curSize  int64
	maxSize  int64
	maxFiles int
	level    int
	console  bool
	pid      int
}

func initLogger(logCfg struct {
	Dir      string `json:"dir"`
	Level    string `json:"level"`
	MaxSize  int    `json:"maxSize"`
	MaxFiles int    `json:"maxFiles"`
	Console  bool   `json:"console"`
}) {
	logState.pid = os.Getpid()
	logState.console = logCfg.Console

	maxSize := logCfg.MaxSize
	if maxSize <= 0 {
		maxSize = 100
	}
	logState.maxSize = int64(maxSize) * 1024 * 1024

	logState.maxFiles = logCfg.MaxFiles
	if logState.maxFiles <= 0 {
		logState.maxFiles = 10
	}

	switch strings.ToLower(strings.TrimSpace(logCfg.Level)) {
	case "debug":
		logState.level = levelDebug
	case "warn":
		logState.level = levelWarn
	case "error":
		logState.level = levelError
	default:
		logState.level = levelInfo
	}

	if logCfg.Dir != "" {
		logState.dir = logCfg.Dir
		if err := os.MkdirAll(logState.dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] failed to create log dir %s: %v\n", logState.dir, err)
			os.Exit(1)
		}
		logState.curPath = filepath.Join(logState.dir, "openfi.log")
		if info, err := os.Stat(logState.curPath); err == nil {
			logState.curSize = info.Size()
		}
		f, err := os.OpenFile(logState.curPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] failed to open log file %s: %v\n", logState.curPath, err)
			return
		}
		logState.file = f
	}
}

func rotateFile() {
	if logState.file != nil {
		logState.file.Close()
		logState.file = nil
	}

	// Delete oldest if it exists
	oldest := fmt.Sprintf("%s.%d.gz", logState.curPath, logState.maxFiles)
	os.Remove(oldest)

	// Shift .N.gz → .N+1.gz (from maxFiles-1 down to 1)
	for i := logState.maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d.gz", logState.curPath, i)
		dst := fmt.Sprintf("%s.%d.gz", logState.curPath, i+1)
		os.Rename(src, dst)
	}

	// Compress current log → .1.gz
	if err := compressFile(logState.curPath, logState.curPath+".1.gz"); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] failed to compress log file: %v\n", err)
	}
	os.Remove(logState.curPath)

	// Open fresh log file
	f, err := os.OpenFile(logState.curPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] failed to open new log file %s: %v\n", logState.curPath, err)
		return
	}
	logState.file = f
	logState.curSize = 0
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		return err
	}
	return gz.Close()
}

func logf(level int, levelStr string, module string, format string, args ...any) {
	if level < logState.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%5s %s] [%8d] [%15s] %s\n", levelStr, ts, logState.pid, module, msg)

	if logState.console {
		fmt.Fprint(os.Stderr, line)
	}

	if logState.dir != "" {
		logState.mu.Lock()
		if logState.file != nil {
			logState.file.WriteString(line)
			logState.curSize += int64(len(line))
			if logState.curSize >= logState.maxSize {
				rotateFile()
			}
		}
		logState.mu.Unlock()
	}
}

func logDebug(module string, format string, args ...any) {
	logf(levelDebug, "DEBUG", module, format, args...)
}

func logInfo(module string, format string, args ...any) {
	logf(levelInfo, "INFO", module, format, args...)
}

func logWarn(module string, format string, args ...any) {
	logf(levelWarn, "WARN", module, format, args...)
}

func logError(module string, format string, args ...any) {
	logf(levelError, "ERROR", module, format, args...)
}

func logFatal(module string, format string, args ...any) {
	logf(levelFatal, "FATAL", module, format, args...)
	os.Exit(1)
}

func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/admin/api/")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func securityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			return next(c)
		}
	}
}

// requestLoggerMiddleware returns an Echo middleware that logs API requests
// using the unified log format.
func requestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			path := c.Request().URL.Path
			if !isAPIPath(path) {
				return err
			}

			logInfo("http", "%s %s %d %s",
				c.Request().Method,
				c.Request().RequestURI,
				c.Response().Status,
				time.Since(start),
			)
			return err
		}
	}
}
