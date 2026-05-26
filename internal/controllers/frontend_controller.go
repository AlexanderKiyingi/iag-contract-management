package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// FrontendController exposes the frontend store snapshot — GET filters by
// the caller's permissions, PUT replaces the entire snapshot and is
// restricted to super_admin only (it could otherwise overwrite the audit log).
type FrontendController struct {
	model *models.Store
}

// NewFrontendController constructs a FrontendController.
func NewFrontendController(model *models.Store) *FrontendController {
	return &FrontendController{model: model}
}

// Get returns the per-caller projection of the frontend snapshot.
func (c *FrontendController) Get(w http.ResponseWriter, r *http.Request) {
	sess := c.model.SessionFromRequest(r.Context())
	views.Frontend(w, c.model.GetFrontendForSession(sess))
}

// Put replaces the entire frontend store. Destructive (can erase the audit
// log) → super_admin only, with an audit entry.
func (c *FrontendController) Put(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(r.Context(), c.model, w) {
		return
	}
	var fe models.FrontendStore
	if err := decodeJSON(r, &fe); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := c.model.ReplaceFrontend(fe); err != nil {
		views.WriteError(w, err)
		return
	}
	sess := c.model.SessionFromRequest(r.Context())
	c.model.RecordBulkReplace("frontend", sess.Email)
	views.JSON(w, http.StatusOK, c.model.GetFrontend())
}
