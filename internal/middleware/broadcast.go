package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/realtime"
)

// WorkspaceNotifier tells every instance that the workspace changed. Satisfied
// by realtime.NudgeBridge; nil when no Redis is configured, in which case the
// refresh reaches only the sockets held by this instance.
type WorkspaceNotifier interface {
	Notify(ctx context.Context)
}

// GinBroadcastWorkspace pushes the updated workspace to live WebSocket clients
// after any successful mutating request. A single post-handler hook covers
// every entity endpoint (instead of editing each controller); reads and the WS
// route itself are skipped. The fan-out runs asynchronously so it never delays
// the HTTP response.
//
// With a notifier, the refresh is announced to every instance rather than
// applied here, because a write served by one instance has to reach sockets
// held by all of them. Without one, this keeps its original single-instance
// behaviour.
func GinBroadcastWorkspace(hub *realtime.Hub, notifier WorkspaceNotifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// A notifier alone is enough: it reaches every instance, including this
		// one, through the subscription.
		if hub == nil && notifier == nil {
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
		// Detached from the request context on purpose: the response has already
		// been written, and a cancelled request context would cancel the publish.
		if notifier != nil {
			go notifier.Notify(context.WithoutCancel(c.Request.Context()))
			return
		}
		go hub.Broadcast()
	}
}
