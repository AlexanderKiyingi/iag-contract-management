package controllers

import (
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// ---------------- Zone-works milestone profiles ----------------

// GetMilestoneProfile returns the governance detail for one zone-works
// milestone, or a default empty profile (200) when none has been saved yet —
// mirroring how GetCloseout returns a default checklist for a fresh contract.
func (g *GovernanceController) GetMilestoneProfile(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.read") {
		return
	}
	milestoneID := pathSegmentAfter(r, "milestone-profiles")
	p, err := g.gov.GetMilestoneProfile(r.Context(), milestoneID)
	if err != nil {
		views.JSON(w, http.StatusOK, models.MilestoneProfile{
			MilestoneID:  milestoneID,
			ContractNo:   "",
			Value:        0,
			Checklist:    []models.MilestoneCheckItem{},
			Scope:        []models.MilestoneScopeTask{},
			Deliverables: []models.MilestoneDeliverable{},
			Comments:     []models.MilestoneComment{},
		})
		return
	}
	views.JSON(w, http.StatusOK, p)
}

// PutMilestoneProfile upserts a milestone's governance detail. The milestone id
// always comes from the path.
func (g *GovernanceController) PutMilestoneProfile(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.update") {
		return
	}
	milestoneID := pathSegmentAfter(r, "milestone-profiles")
	if strings.TrimSpace(milestoneID) == "" {
		views.Error(w, http.StatusBadRequest, "milestone id is required")
		return
	}
	var in models.MilestoneProfileInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	p := models.MilestoneProfile{
		MilestoneID:  milestoneID,
		ContractNo:   in.ContractNo,
		Value:        in.Value,
		Checklist:    in.Checklist,
		Scope:        in.Scope,
		Deliverables: in.Deliverables,
		Comments:     in.Comments,
	}
	if p.Checklist == nil {
		p.Checklist = []models.MilestoneCheckItem{}
	}
	if p.Scope == nil {
		p.Scope = []models.MilestoneScopeTask{}
	}
	if p.Deliverables == nil {
		p.Deliverables = []models.MilestoneDeliverable{}
	}
	if p.Comments == nil {
		p.Comments = []models.MilestoneComment{}
	}
	out, err := g.gov.UpsertMilestoneProfile(r.Context(), p)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

// ---------------- Zone-works execution trackers ----------------

// GetExecutionTracker returns a contract's execution tracker, or a default
// empty tracker (200) when none has been saved yet.
func (g *GovernanceController) GetExecutionTracker(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.read") {
		return
	}
	contractNo := pathSegmentAfter(r, "execution")
	t, err := g.gov.GetExecutionTracker(r.Context(), contractNo)
	if err != nil {
		views.JSON(w, http.StatusOK, models.ExecutionTracker{
			ContractNo:    contractNo,
			Stage:         0,
			Steps:         []string{},
			ExecutionDate: nil,
		})
		return
	}
	views.JSON(w, http.StatusOK, t)
}

// PutExecutionTracker upserts a contract's execution tracker. The contract
// number always comes from the path.
func (g *GovernanceController) PutExecutionTracker(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.update") {
		return
	}
	contractNo := pathSegmentAfter(r, "execution")
	if strings.TrimSpace(contractNo) == "" {
		views.Error(w, http.StatusBadRequest, "contract number is required")
		return
	}
	var in models.ExecutionTrackerInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	t := models.ExecutionTracker{
		ContractNo:    contractNo,
		Stage:         in.Stage,
		Steps:         in.Steps,
		ExecutionDate: in.ExecutionDate,
	}
	if t.Steps == nil {
		t.Steps = []string{}
	}
	out, err := g.gov.UpsertExecutionTracker(r.Context(), t)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}
