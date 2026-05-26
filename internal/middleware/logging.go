package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLogger emits one slog record per request, skipping health probes to
// keep the log stream readable when k8s probes hit /v1/health/ready every
// few seconds. The handler-applied status code is recorded so callers can
// filter on 4xx/5xx.
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		if isHealthPath(path) {
			return
		}
		dur := time.Since(start)
		level := slog.LevelInfo
		status := c.Writer.Status()
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		slog.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("duration", dur.Round(time.Millisecond)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
