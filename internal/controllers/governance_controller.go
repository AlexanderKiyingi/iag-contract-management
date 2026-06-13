package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// GovernanceController serves the contract-governance domain (governance
// contracts + rich milestones) that backs the Contract Governance UI.
type GovernanceController struct {
	model  *models.Store
	gov    *persistence.GovStore
	events *events.Bus
}

func NewGovernanceController(model *models.Store, gov *persistence.GovStore, bus *events.Bus) *GovernanceController {
	return &GovernanceController{model: model, gov: gov, events: bus}
}

func govActor(c models.GovContract) string {
	if c.PM != "" {
		return c.PM
	}
	return "system"
}

// ----- Contracts -----

func (g *GovernanceController) ListContracts(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
		return
	}
	list, err := g.gov.ListContracts(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetContract(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

func (g *GovernanceController) CreateContract(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
		return
	}
	var in models.GovContractInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "number and name are required")
		return
	}
	status := in.Status
	if status == "" {
		status = models.GovDraft
	}
	if !status.Valid() {
		views.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	c := models.GovContract{
		ID:                models.NewGovID("GCT"),
		Number:            strings.TrimSpace(in.Number),
		Name:              strings.TrimSpace(in.Name),
		Contractor:        in.Contractor,
		ContractorContact: in.ContractorContact,
		Type:              in.Type,
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
		Location:          in.Location,
		PM:                in.PM,
		Department:        in.Department,
		Value:             in.Value,
		Retention:         in.Retention,
		Status:            status,
		Documents:         in.Documents,
		Activity:          []models.GovActivity{{Date: nowStamp(), Actor: in.PM, Action: "Contract created in " + string(status) + " status"}},
	}
	created, err := g.gov.CreateContract(r.Context(), c)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	g.publishStatus(r, *created, "", created.Status)
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) PatchContract(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.update") {
		return
	}
	existing, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	var p models.GovContractPatch
	if err := decodeJSON(r, &p); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	prevStatus := existing.Status
	statusChanged := false
	if p.Status != nil && *p.Status != existing.Status {
		if !p.Status.Valid() {
			views.Error(w, http.StatusBadRequest, "invalid status")
			return
		}
		if !existing.Status.CanTransitionTo(*p.Status) {
			views.Error(w, http.StatusUnprocessableEntity,
				"invalid transition: "+string(existing.Status)+" → "+string(*p.Status))
			return
		}
		statusChanged = true
	}

	applyContractPatch(existing, p)
	if statusChanged {
		existing.Activity = append(existing.Activity, models.GovActivity{
			Date: nowStamp(), Actor: govActor(*existing),
			Action: "Status changed: " + string(prevStatus) + " → " + string(existing.Status),
		})
	}
	updated, err := g.gov.UpdateContract(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	updated.Milestones = existing.Milestones
	if statusChanged {
		g.publishStatus(r, *updated, prevStatus, updated.Status)
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteContract(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteContract(r.Context(), pathSegmentAfter(r, "contracts"))) {
		return
	}
	views.NoContent(w)
}

// ----- Milestones -----

func (g *GovernanceController) ListMilestones(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.read") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": c.Milestones})
}

func (g *GovernanceController) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.create") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	var in models.GovMilestoneInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	status := in.Status
	if status == "" {
		status = models.MSPending
	}
	m := models.GovMilestone{
		ID: models.NewGovID("GMS"), ContractID: c.ID, Name: strings.TrimSpace(in.Name),
		Value: in.Value, TargetDate: in.TargetDate, Status: status,
		Scope: in.Scope, Deliverables: in.Deliverables, Checklist: in.Checklist,
		Docs: in.Docs, Comments: in.Comments,
	}
	created, err := g.gov.CreateMilestone(r.Context(), m)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) PatchMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.update") {
		return
	}
	existing, err := g.gov.GetMilestone(r.Context(), lastPathSegment(r))
	if g.handleErr(w, err) {
		return
	}
	var p models.GovMilestonePatch
	if err := decodeJSON(r, &p); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	applyMilestonePatch(existing, p)
	updated, err := g.gov.UpdateMilestone(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteMilestone(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ----- helpers -----

func (g *GovernanceController) handleErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, persistence.ErrGovNotFound) {
		views.Error(w, http.StatusNotFound, "not found")
		return true
	}
	views.WriteError(w, err)
	return true
}

func (g *GovernanceController) publishStatus(r *http.Request, c models.GovContract, prev, next models.GovStatus) {
	if g.events == nil {
		return
	}
	g.events.PublishCommercial(r.Context(), "contracts.governance.status_changed", map[string]any{
		"id":             c.ID,
		"number":         c.Number,
		"name":           c.Name,
		"previousStatus": string(prev),
		"status":         string(next),
		"value":          c.Value,
		"department":     c.Department,
	}, c.Number)
}

func nowStamp() string { return time.Now().UTC().Format("02 Jan 2006 15:04") }

func applyContractPatch(c *models.GovContract, p models.GovContractPatch) {
	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Contractor != nil {
		c.Contractor = *p.Contractor
	}
	if p.ContractorContact != nil {
		c.ContractorContact = *p.ContractorContact
	}
	if p.Type != nil {
		c.Type = *p.Type
	}
	if p.StartDate != nil {
		c.StartDate = *p.StartDate
	}
	if p.EndDate != nil {
		c.EndDate = *p.EndDate
	}
	if p.Location != nil {
		c.Location = *p.Location
	}
	if p.PM != nil {
		c.PM = *p.PM
	}
	if p.Department != nil {
		c.Department = *p.Department
	}
	if p.Value != nil {
		c.Value = *p.Value
	}
	if p.Retention != nil {
		c.Retention = *p.Retention
	}
	if p.Status != nil {
		c.Status = *p.Status
	}
	if p.Documents != nil {
		c.Documents = *p.Documents
	}
}

func applyMilestonePatch(m *models.GovMilestone, p models.GovMilestonePatch) {
	if p.Name != nil {
		m.Name = *p.Name
	}
	if p.Value != nil {
		m.Value = *p.Value
	}
	if p.TargetDate != nil {
		m.TargetDate = *p.TargetDate
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if p.Scope != nil {
		m.Scope = *p.Scope
	}
	if p.Deliverables != nil {
		m.Deliverables = *p.Deliverables
	}
	if p.Checklist != nil {
		m.Checklist = *p.Checklist
	}
	if p.Docs != nil {
		m.Docs = *p.Docs
	}
	if p.Comments != nil {
		m.Comments = *p.Comments
	}
	if p.Inspection != nil {
		m.Inspection = p.Inspection
	}
	if p.CompletionReport != nil {
		m.CompletionReport = p.CompletionReport
	}
}
