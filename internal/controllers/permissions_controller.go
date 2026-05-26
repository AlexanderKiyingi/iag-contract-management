package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type PermissionsController struct {
	model *models.Store
}

func NewPermissionsController(model *models.Store) *PermissionsController {
	return &PermissionsController{model: model}
}

func (c *PermissionsController) Catalog(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, models.PermissionCatalogData())
}

func (c *PermissionsController) Builtin(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, models.BuiltinRolesPermissions())
}

func (c *PermissionsController) Me(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, c.model.PermissionContextCtx(r.Context()))
}

func (c *PermissionsController) UserPermissions(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.read") {
		return
	}
	perms, err := c.model.EffectivePermissionsForUser(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{
		"userId":      lastPathSegment(r),
		"permissions": perms,
	})
}

func (c *PermissionsController) Check(w http.ResponseWriter, r *http.Request) {
	var in models.PermissionCheckInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	views.JSON(w, http.StatusOK, models.PermissionCheckResult{
		Allowed: c.model.CheckPermissionsCtx(r.Context(), in.Keys),
	})
}
