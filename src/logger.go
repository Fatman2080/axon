package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var logPID = os.Getpid()

func logf(level string, module string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "[%5s %s] [%8d] [%15s] %s\n", level, ts, logPID, module, msg)
}

func logInfo(module string, format string, args ...interface{}) {
	logf("INFO", module, format, args...)
}

func logWarn(module string, format string, args ...interface{}) {
	logf("WARN", module, format, args...)
}

func logError(module string, format string, args ...interface{}) {
	logf("ERROR", module, format, args...)
}

func logFatal(module string, format string, args ...interface{}) {
	logf("FATAL", module, format, args...)
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
