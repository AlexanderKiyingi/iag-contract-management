package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type spyNotifier struct{ calls chan struct{} }

func newSpy() *spyNotifier { return &spyNotifier{calls: make(chan struct{}, 4)} }

func (s *spyNotifier) Notify(context.Context) { s.calls <- struct{}{} }

// notified reports whether Notify ran. The middleware fires it in a goroutine
// so the HTTP response is never delayed, hence the wait.
func (s *spyNotifier) notified() bool {
	select {
	case <-s.calls:
		return true
	case <-time.After(time.Second):
		return false
	}
}

func run(t *testing.T, method, path string, status int, notifier WorkspaceNotifier) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinBroadcastWorkspace(nil, notifier))
	r.Handle(method, path, func(c *gin.Context) { c.Status(status) })

	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
}

// A write served by one instance has to refresh sockets held by every
// instance, so the announcement — not a local-only push — is what must happen.
func TestMutationAnnouncesToEveryInstance(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		spy := newSpy()
		run(t, method, "/v1/contracts", http.StatusOK, spy)
		if !spy.notified() {
			t.Fatalf("%s did not announce the workspace change", method)
		}
	}
}

func TestReadsDoNotAnnounce(t *testing.T) {
	spy := newSpy()
	run(t, http.MethodGet, "/v1/contracts", http.StatusOK, spy)
	if spy.notified() {
		t.Fatal("a read announced a change; every poll would refresh every session on every instance")
	}
}

// A rejected write changed nothing, so refreshing every session across the
// platform for it is pure noise.
func TestFailedMutationDoesNotAnnounce(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
		spy := newSpy()
		run(t, http.MethodPost, "/v1/contracts", status, spy)
		if spy.notified() {
			t.Fatalf("status %d announced a change that never happened", status)
		}
	}
}

func TestWebSocketRouteDoesNotAnnounce(t *testing.T) {
	spy := newSpy()
	run(t, http.MethodPost, "/v1/ws/workspace", http.StatusOK, spy)
	if spy.notified() {
		t.Fatal("the websocket route itself announced a change")
	}
}

// Without Redis the middleware must behave exactly as it did before: no
// notifier, no panic, and the local hub path taken instead.
func TestNoNotifierIsSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinBroadcastWorkspace(nil, nil))
	r.POST("/v1/contracts", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/contracts", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
