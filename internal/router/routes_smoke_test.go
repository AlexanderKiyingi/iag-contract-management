package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestGovernanceRoutePatternsRegister guards against Gin radix-tree conflicts
// (e.g. a static segment colliding with a wildcard) for the governance route
// group, including the monthly-report additions. Route registration panics on
// conflict, so a clean registration is the assertion.
func TestGovernanceRoutePatternsRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("governance route registration panicked: %v", rec)
		}
	}()

	r := gin.New()
	gov := r.Group("/v1/governance")
	h := func(c *gin.Context) {}

	// Existing governance patterns that share the /contracts/:id subtree.
	gov.GET("/contracts", h)
	gov.GET("/contracts/:id", h)
	gov.GET("/contracts/:id/milestones", h)
	gov.GET("/contracts/:id/variations", h)
	gov.GET("/contracts/:id/obligations", h)
	gov.GET("/contracts/:id/closeout", h)
	gov.GET("/milestones/:id", h)
	gov.GET("/variations/:id/advance", h)

	// Monthly-report additions — these must coexist with the above.
	gov.GET("/contractors", h)
	gov.POST("/contractors", h)
	gov.GET("/contractors/:id", h)
	gov.PATCH("/contractors/:id", h)
	gov.DELETE("/contractors/:id", h)
	gov.GET("/contracts/:id/reports", h)
	gov.PUT("/contracts/:id/reports", h)
	gov.GET("/reports", h)
	gov.DELETE("/reports/:id", h)
	gov.GET("/valuations", h)
	gov.POST("/valuations", h)
	gov.GET("/valuations/:id", h)
	gov.PATCH("/valuations/:id", h)
	gov.DELETE("/valuations/:id", h)
	gov.GET("/summary", h)

	if len(r.Routes()) == 0 {
		t.Fatal("no routes registered")
	}
	_ = http.StatusOK
}
