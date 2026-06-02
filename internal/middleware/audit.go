package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

func RequestAudit(pg *persistence.Postgres) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if isPublicPath(path) || strings.HasPrefix(path, "/health") {
			return
		}

		userName := "anonymous"
		if sess, ok := models.RequestSession(c.Request.Context()); ok {
			if sess.Email != "" {
				userName = sess.Email
			} else if sess.DisplayName != "" {
				userName = sess.DisplayName
			}
		}

		duration := int(time.Since(start).Milliseconds())
		_ = pg.LogAPIRequest(
			c.Request.Context(),
			c.Request.Method,
			path,
			c.Writer.Status(),
			userName,
			duration,
			c.ClientIP(),
		)
	}
}
