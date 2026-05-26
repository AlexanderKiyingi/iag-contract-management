package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type WorkspaceResourcesController struct {
	model *models.Store
}

func NewWorkspaceResourcesController(model *models.Store) *WorkspaceResourcesController {
	return &WorkspaceResourcesController{model: model}
}

func (c *WorkspaceResourcesController) ListZones(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "zones.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListZones())
}

func (c *WorkspaceResourcesController) GetZone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "zones.read") {
		return
	}
	z, err := c.model.GetZone(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, z)
}

func (c *WorkspaceResourcesController) ListEngineers(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListEngineers())
}

func (c *WorkspaceResourcesController) GetEngineer(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.read") {
		return
	}
	eng, err := c.model.GetEngineer(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, eng)
}

func (c *WorkspaceResourcesController) CreateEngineer(w http.ResponseWriter, r *http.Request) {
	if !requireMutate(r.Context(), c.model, w) {
		return
	}
	var in models.EngineerInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	eng, err := c.model.CreateEngineer(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, eng)
}

func (c *WorkspaceResourcesController) PatchEngineer(w http.ResponseWriter, r *http.Request) {
	if !requireMutate(r.Context(), c.model, w) {
		return
	}
	var in models.EngineerInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	eng, err := c.model.PatchEngineer(lastPathSegment(r), in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, eng)
}

func (c *WorkspaceResourcesController) DeleteEngineer(w http.ResponseWriter, r *http.Request) {
	if !requireMutate(r.Context(), c.model, w) {
		return
	}
	if err := c.model.DeleteEngineer(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *WorkspaceResourcesController) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListUsers())
}

func (c *WorkspaceResourcesController) GetUser(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.read") {
		return
	}
	u, err := c.model.GetUser(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, u)
}

func (c *WorkspaceResourcesController) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.create") {
		return
	}
	var in models.UserInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u, err := c.model.CreateUser(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, u)
}

func (c *WorkspaceResourcesController) PatchUser(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.update") {
		return
	}
	var patch models.UserPatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u, err := c.model.PatchUser(lastPathSegment(r), patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, u)
}

func (c *WorkspaceResourcesController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "users.delete") {
		return
	}
	if err := c.model.DeactivateUser(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}
