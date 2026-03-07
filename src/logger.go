package main

import (
	"fmt"
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
	mu      sync.Mutex
	file    *os.File
	dir     string
	today   string // "2006-01-02"
	level   int
	console bool
	maxDays int
	pid     int
}

func initLogger(logCfg struct {
	Dir     string `json:"dir"`
	Level   string `json:"level"`
	MaxDays int    `json:"maxDays"`
	Console bool   `json:"console"`
}, configDir string) {
	logState.pid = os.Getpid()
	logState.console = logCfg.Console
	logState.maxDays = logCfg.MaxDays
	if logState.maxDays <= 0 {
		logState.maxDays = 30
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
		openLogFile()
		go cleanupOldLogs()
	}
}

func openLogFile() {
	today := time.Now().Format("2006-01-02")
	logState.today = today
	name := filepath.Join(logState.dir, fmt.Sprintf("openfi-%s.log", today))
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] failed to open log file %s: %v\n", name, err)
		return
	}
	if logState.file != nil {
		logState.file.Close()
	}
	logState.file = f
}

func cleanupOldLogs() {
	entries, err := os.ReadDir(logState.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -logState.maxDays)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "openfi-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logState.dir, name))
		}
	}
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
		today := time.Now().Format("2006-01-02")
		if today != logState.today {
			openLogFile()
		}
		if logState.file != nil {
			logState.file.WriteString(line)
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
