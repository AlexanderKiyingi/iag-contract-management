package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// WorkspaceController exposes the workspace snapshot — GET filters by
// the caller's permissions / contractor scope, PUT replaces the entire
// workspace and is restricted to super_admin only.
type WorkspaceController struct {
	model *models.Store
}

// NewWorkspaceController constructs a WorkspaceController.
func NewWorkspaceController(model *models.Store) *WorkspaceController {
	return &WorkspaceController{model: model}
}

// Get returns the per-caller projection of the workspace snapshot.
func (c *WorkspaceController) Get(w http.ResponseWriter, r *http.Request) {
	sess := c.model.SessionFromRequest(r.Context())
	views.Workspace(w, c.model.GetWorkspaceForSession(sess))
}

// Put replaces the entire workspace. Destructive (overwrites every entity in
// one shot) → super_admin only, with an audit entry. Day-to-day edits should
// use the per-entity endpoints.
func (c *WorkspaceController) Put(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(r.Context(), c.model, w) {
		return
	}
	var ws models.Workspace
	if err := decodeJSON(r, &ws); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := c.model.ReplaceWorkspace(ws); err != nil {
		views.WriteError(w, err)
		return
	}
	sess := c.model.SessionFromRequest(r.Context())
	c.model.RecordBulkReplace("workspace", sess.Email)
	views.JSON(w, http.StatusOK, c.model.GetWorkspace())
}
