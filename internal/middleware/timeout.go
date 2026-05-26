package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// GinTimeout installs a per-request deadline on the request context. Handlers
// that honour ctx.Done() will return early; SQL queries through pgx will
// cancel their underlying conn. Health probes bypass it so a stuck downstream
// can't make the pod look unready while it's just slow.
//
// Note: this only cancels the *context*. The HTTP response is still produced
// by the handler; if the handler ignores ctx and writes a slow response, the
// http.Server WriteTimeout will catch it instead.
func GinTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isHealthPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// isHealthPath mirrors the auth middleware's public-path list — used here
// independently because middleware ordering means we may run before/after
// auth depending on configuration.
func isHealthPath(path string) bool {
	for len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	switch path {
	case "/ready", "/health", "/health/live", "/health/ready",
		"/v1/health", "/v1/health/live", "/v1/health/ready":
		return true
	}
	return false
}

