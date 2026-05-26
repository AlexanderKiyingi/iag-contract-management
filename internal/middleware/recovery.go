package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// GinRecovery recovers from handler panics, logs the cause + a Go stack
// trace, and returns a JSON 500 to the client. The stack trace is essential
// for post-mortems — without it, you only have a one-line message and no
// way to locate the offending function.
func GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic recovered",
			"panic", recovered,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"stack", string(debug.Stack()),
		)
		views.Error(c.Writer, http.StatusInternalServerError, "internal server error")
		c.Abort()
	})
}
