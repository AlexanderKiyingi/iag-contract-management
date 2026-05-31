package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/alvor-technologies/iag-contract-management/internal/config"
)

func GinCORS(cfg config.Config) gin.HandlerFunc {
	origins := cfg.AllowedOrigins
	if len(origins) == 0 && !cfg.IsProduction() {
		origins = []string{"*"}
	}
	allowAll := len(origins) == 1 && origins[0] == "*"
	originSet := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		originSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			if _, ok := originSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func GinSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// JSON-only API: strictest CSP — nothing loads, nothing frames.
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		if cfg := c.GetHeader("X-Forwarded-Proto"); strings.EqualFold(cfg, "https") || c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), interest-cohort=()")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}
