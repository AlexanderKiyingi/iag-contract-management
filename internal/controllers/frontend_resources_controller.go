package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type FrontendResourcesController struct {
	model  *models.Store
	events *events.Bus
}

func NewFrontendResourcesController(model *models.Store, bus *events.Bus) *FrontendResourcesController {
	return &FrontendResourcesController{model: model, events: bus}
}

func (c *FrontendResourcesController) ListMilestones(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "milestones.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListMilestones())
}

func (c *FrontendResourcesController) GetMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "milestones.read") {
		return
	}
	m, err := c.model.GetMilestone(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, m)
}

func (c *FrontendResourcesController) GetRole(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "roles.read") {
		return
	}
	role, err := c.model.GetRole(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, role)
}

func (c *FrontendResourcesController) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "milestones.create") {
		return
	}
	var in models.MilestoneInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, err := c.model.CreateMilestone(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, m)
}

func (c *FrontendResourcesController) PatchMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "milestones.update") {
		return
	}
	var patch models.MilestonePatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, err := c.model.PatchMilestone(lastPathSegment(r), patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, m)
}

func (c *FrontendResourcesController) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "milestones.delete") {
		return
	}
	if err := c.model.DeleteMilestone(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) ListMaterials(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "materials.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListMaterials())
}

func (c *FrontendResourcesController) CreateMaterial(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "materials.create") {
		return
	}
	var in models.MaterialInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, err := c.model.CreateMaterial(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, m)
}

func (c *FrontendResourcesController) PatchMaterial(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "materials.update") {
		return
	}
	var patch models.MaterialPatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, err := c.model.PatchMaterial(lastPathSegment(r), patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, m)
}

func (c *FrontendResourcesController) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "materials.delete") {
		return
	}
	if err := c.model.DeleteMaterial(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) ListProjects(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListProjects())
}

func (c *FrontendResourcesController) CreateProject(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.create") {
		return
	}
	var in models.ProjectInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	p, idx, err := c.model.CreateProject(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, map[string]any{"index": idx, "project": p})
}

func (c *FrontendResourcesController) PatchProject(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.update") {
		return
	}
	idx, ok := pathIntAfter(r, "projects")
	if !ok {
		views.Error(w, http.StatusBadRequest, "invalid project index")
		return
	}
	var patch models.ProjectPatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	p, err := c.model.PatchProject(idx, patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, p)
}

func (c *FrontendResourcesController) DeleteProject(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.delete") {
		return
	}
	idx, ok := pathIntAfter(r, "projects")
	if !ok {
		views.Error(w, http.StatusBadRequest, "invalid project index")
		return
	}
	if err := c.model.DeleteProject(idx); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) CreateTask(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.create") {
		return
	}
	idx, ok := pathIntAfter(r, "projects")
	if !ok {
		views.Error(w, http.StatusBadRequest, "invalid project index")
		return
	}
	var in models.TaskInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	t, err := c.model.CreateTask(idx, in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, t)
}

func (c *FrontendResourcesController) PatchTask(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.update") {
		return
	}
	idx, ok := pathIntAfter(r, "projects")
	if !ok {
		views.Error(w, http.StatusBadRequest, "invalid project index")
		return
	}
	var patch models.TaskPatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	t, err := c.model.PatchTask(idx, lastPathSegment(r), patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, t)
}

func (c *FrontendResourcesController) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "tasks.delete") {
		return
	}
	idx, ok := pathIntAfter(r, "projects")
	if !ok {
		views.Error(w, http.StatusBadRequest, "invalid project index")
		return
	}
	if err := c.model.DeleteTask(idx, lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) ListRoles(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "roles.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListRoles())
}

func (c *FrontendResourcesController) CreateRole(w http.ResponseWriter, r *http.Request) {
	if !requireManageRoles(r.Context(), c.model, w) {
		return
	}
	var in models.RoleInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	role, err := c.model.CreateRole(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, role)
}

func (c *FrontendResourcesController) PatchRole(w http.ResponseWriter, r *http.Request) {
	if !requireManageRoles(r.Context(), c.model, w) {
		return
	}
	var patch models.RolePatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	role, err := c.model.PatchRole(lastPathSegment(r), patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, role)
}

func (c *FrontendResourcesController) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if !requireManageRoles(r.Context(), c.model, w) {
		return
	}
	if err := c.model.DeleteRole(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) ListAudit(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "audit.read") {
		return
	}
	views.JSON(w, http.StatusOK, c.model.ListAudit())
}

func (c *FrontendResourcesController) GetAudit(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "audit.read") {
		return
	}
	a, err := c.model.GetAudit(lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, a)
}

func (c *FrontendResourcesController) AppendAudit(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "audit.create") {
		return
	}
	var in models.AuditInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	a, err := c.model.AppendAudit(in, c.model.GetSessionCtx(r.Context()).Email)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, a)
}

func (c *FrontendResourcesController) ListAssistance(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, c.model.ListAssistance())
}

func (c *FrontendResourcesController) PostAssistance(w http.ResponseWriter, r *http.Request) {
	if !requireMutate(r.Context(), c.model, w) {
		return
	}
	var in models.AssistanceInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	msg, err := c.model.PostAssistance(in.Text, c.model.GetSessionCtx(r.Context()).Email)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	events.PublishAssistanceRequested(r.Context(), c.events, msg)
	views.JSON(w, http.StatusCreated, msg)
}

func (c *FrontendResourcesController) PutProfilePhoto(w http.ResponseWriter, r *http.Request) {
	var in models.ProfilePhotoInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	email := in.Email
	if email == "" {
		email = c.model.GetSessionCtx(r.Context()).Email
	}
	if !canEditProfile(r.Context(), c.model, email) {
		views.WriteError(w, models.ErrForbidden)
		return
	}
	if err := c.model.SetProfilePhoto(email, in.DataURL); err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{"email": email, "dataUrl": c.model.GetProfilePhoto(email)})
}

func (c *FrontendResourcesController) DeleteProfilePhoto(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		email = c.model.GetSessionCtx(r.Context()).Email
	}
	if !canEditProfile(r.Context(), c.model, email) {
		views.WriteError(w, models.ErrForbidden)
		return
	}
	if err := c.model.SetProfilePhoto(email, ""); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

func (c *FrontendResourcesController) PutAiScan(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "insights.update") {
		return
	}
	var scan any
	if err := decodeJSON(r, &scan); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	c.model.SetAiScan(scan)
	views.NoContent(w)
}
