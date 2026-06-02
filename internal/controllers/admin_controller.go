package controllers

import (
	"net/http"
	"strconv"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type AdminController struct {
	model *models.Store
	pg    *persistence.Postgres
}

func NewAdminController(model *models.Store, pg *persistence.Postgres) *AdminController {
	return &AdminController{model: model, pg: pg}
}

func (c *AdminController) ListAPIAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "audit.read") {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, total, err := c.pg.ListAPIAuditLogs(r.Context(), limit)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	})
}
