package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// HealthController exposes /health (liveness) and /ready (readiness).
type HealthController struct {
	pg *persistence.Postgres
}

// NewHealthController constructs a HealthController.
func NewHealthController(pg *persistence.Postgres) *HealthController {
	return &HealthController{pg: pg}
}

// Check returns liveness.
func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) { views.Health(w) }

// Live is an alias for Check.
func (c *HealthController) Live(w http.ResponseWriter, r *http.Request) { views.Health(w) }

// Ready pings Postgres.
func (c *HealthController) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	ok := true
	if c.pg != nil {
		if err := c.pg.Ping(ctx); err != nil {
			checks["postgres"] = err.Error()
			ok = false
		} else {
			checks["postgres"] = "ok"
		}
	}
	status := http.StatusOK
	if !ok {
		status = http.StatusServiceUnavailable
	}
	views.JSON(w, status, map[string]any{
		"status": map[string]bool{"ready": ok},
		"checks": checks,
	})
}
