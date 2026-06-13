package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/realtime"
)

// GinBroadcastWorkspace pushes the updated workspace to live WebSocket clients
// after any successful mutating request. A single post-handler hook covers
// every entity endpoint (instead of editing each controller); reads and the WS
// route itself are skipped. The fan-out runs asynchronously so it never delays
// the HTTP response.
func GinBroadcastWorkspace(hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if hub == nil {
			return
		}
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return
		}
		if status := c.Writer.Status(); status < 200 || status >= 300 {
			return
		}
		if strings.Contains(c.Request.URL.Path, "/ws/") {
			return
		}
		go hub.Broadcast()
	}
}
