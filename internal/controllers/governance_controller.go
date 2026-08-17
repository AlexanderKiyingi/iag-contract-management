package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/chat"
	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/objstore"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// GovernanceController serves the contract-governance domain (governance
// contracts + rich milestones) that backs the Contract Governance UI.
type GovernanceController struct {
	model  *models.Store
	gov    *persistence.GovStore
	events *events.Bus
	docs   *objstore.Presigner // nil when object storage is unconfigured
	chat   *chat.Service       // nil when chat is unconfigured
}

func NewGovernanceController(model *models.Store, gov *persistence.GovStore, bus *events.Bus, docs *objstore.Presigner, chatSvc *chat.Service) *GovernanceController {
	return &GovernanceController{model: model, gov: gov, events: bus, docs: docs, chat: chatSvc}
}

func govActor(c models.GovContract) string {
	if c.PM != "" {
		return c.PM
	}
	return "system"
}

// ensureContractThread find-or-creates a contract's chat discussion thread and
// persists the conversation id back onto the contract. Runs in the background,
// best-effort, so a chat outage never blocks or fails contract creation.
// Participants seeded: the creating user and (if the contract is linked to a
// contractor with a platform login) that contractor, so they see the thread.
func (g *GovernanceController) ensureContractThread(actorUserID string, c models.GovContract) {
	if g.chat == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	seen := map[string]bool{}
	var parts []string
	add := func(uid string) {
		uid = strings.TrimSpace(uid)
		if uid != "" && !seen[uid] {
			seen[uid] = true
			parts = append(parts, uid)
		}
	}
	add(actorUserID)
	if c.ContractorID != "" {
		if k, err := g.gov.GetContractor(ctx, c.ContractorID); err == nil && k != nil {
			add(k.PlatformUserID)
		}
	}

	convID, err := g.chat.EnsureContractThread(ctx, c.ID, contractThreadTitle(c), parts)
	if err != nil {
		slog.Warn("contract chat thread create failed", "contract", c.ID, "err", err)
		return
	}
	if convID == "" {
		return
	}
	if err := g.gov.SetContractConversationID(ctx, c.ID, convID); err != nil {
		slog.Warn("persist contract conversation id failed", "contract", c.ID, "err", err)
	}
}

func contractThreadTitle(c models.GovContract) string {
	return strings.TrimSpace(strings.TrimSpace(c.Number) + " " + strings.TrimSpace(c.Name))
}

// postContractSystem posts a system line into a contract's discussion thread
// (find-or-creating it by link). Async + best-effort so it never affects the
// request; no-ops when chat is unconfigured or the contract id is empty.
func (g *GovernanceController) postContractSystem(contractID, message string) {
	if g.chat == nil || strings.TrimSpace(contractID) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := g.chat.PostSystem(ctx, contractID, message); err != nil {
			slog.Warn("contract chat system post failed", "contract", contractID, "err", err)
		}
	}()
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
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	// The contract number (human code) is auto-generated when left blank, so
	// staff don't have to invent one; they may still supply their own.
	number := strings.TrimSpace(in.Number)
	if number == "" {
		gen, err := g.gov.NextContractNumber(r.Context())
		if err != nil {
			views.WriteError(w, err)
			return
		}
		number = gen
	}
	status := in.Status
	if status == "" {
		status = models.GovDraft
	}
	if !status.Valid() {
		views.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	execStatus := in.ExecutionStatus
	if execStatus == "" {
		execStatus = models.ExecNotStarted
	}
	if !execStatus.Valid() {
		views.Error(w, http.StatusBadRequest, "invalid executionStatus")
		return
	}
	c := models.GovContract{
		ID:                models.NewGovID("GCT"),
		Number:            number,
		Name:              strings.TrimSpace(in.Name),
		Contractor:        in.Contractor,
		ContractorID:      in.ContractorID,
		ContractorContact: in.ContractorContact,
		Type:              in.Type,
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
		Location:          in.Location,
		PM:                in.PM,
		Department:        in.Department,
		Value:             in.Value,
		Currency:          strings.ToUpper(strings.TrimSpace(in.Currency)),
		Retention:         in.Retention,
		Status:            status,
		ExecutionStatus:   execStatus,
		Progress:          clampProgress(in.Progress),
		Received:          in.Received,
		VariationTotal:    in.VariationTotal,
		PlannedCompletion: in.PlannedCompletion,
		PMProjectID:       strings.TrimSpace(in.PMProjectID),
		Documents:         in.Documents,
		Activity:          []models.GovActivity{{Date: nowStamp(), Actor: in.PM, Action: "Contract created in " + string(status) + " status"}},
	}
	created, err := g.gov.CreateContract(r.Context(), c)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	g.publishStatus(r, *created, "", created.Status)
	g.publishPMProjectLink(r, *created, events.TypeContractCreated)
	if g.chat != nil {
		go g.ensureContractThread(g.model.SessionFromRequest(r.Context()).UserID, *created)
	}
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
	if p.ExecutionStatus != nil && *p.ExecutionStatus != "" && !p.ExecutionStatus.Valid() {
		views.Error(w, http.StatusBadRequest, "invalid executionStatus")
		return
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
	// Keep the PM project link in sync (name/status/link changes); the consumer
	// upserts the contract onto its project.
	g.publishPMProjectLink(r, *updated, events.TypeContractUpdated)
	if statusChanged {
		g.postContractSystem(updated.ID,
			"Status changed: "+string(prevStatus)+" → "+string(updated.Status)+" (by "+g.actor(r, "")+")")
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteContract(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.delete") {
		return
	}
	id := pathSegmentAfter(r, "contracts")
	// Load first so we can tell the PM service to detach the contract from its
	// project (best-effort; a missing contract just yields no event).
	existing, _ := g.gov.GetContract(r.Context(), id)
	if g.handleErr(w, g.gov.DeleteContract(r.Context(), id)) {
		return
	}
	if existing != nil {
		g.publishPMProjectLink(r, *existing, events.TypeContractDeleted)
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

// ListAllMilestones returns every milestone across the portfolio (one
// round-trip), so the frontend portfolio page needn't fan out per contract.
func (g *GovernanceController) ListAllMilestones(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "milestones.read") {
		return
	}
	list, err := g.gov.ListAllMilestones(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// Counts serves the server-computed governance nav badge numbers.
func (g *GovernanceController) Counts(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
		return
	}
	counts, err := g.gov.Counts(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, counts)
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
	if !models.ValidMilestoneStatus(status) {
		views.Error(w, http.StatusBadRequest, "invalid milestone status")
		return
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
	if p.Status != nil && !models.ValidMilestoneStatus(*p.Status) {
		views.Error(w, http.StatusBadRequest, "invalid milestone status")
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

// publishPMProjectLink emits a project-management-consumable contract lifecycle
// event so the PM service can attach/detach this contract on its parent project.
// It reuses the legacy contracts.contract.* event types (the PM consumer keys on
// pmProjectId) and only fires when the contract is actually linked to a project.
func (g *GovernanceController) publishPMProjectLink(r *http.Request, c models.GovContract, eventType string) {
	if g.events == nil || strings.TrimSpace(c.PMProjectID) == "" {
		return
	}
	g.events.PublishCommercial(r.Context(), eventType, map[string]any{
		"no":          c.Number,
		"name":        c.Name,
		"status":      string(c.Status),
		"pmProjectId": c.PMProjectID,
	}, c.ID)
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
	if p.Currency != nil {
		c.Currency = strings.ToUpper(strings.TrimSpace(*p.Currency))
	}
	if p.Retention != nil {
		c.Retention = *p.Retention
	}
	if p.Status != nil {
		c.Status = *p.Status
	}
	if p.ContractorID != nil {
		c.ContractorID = *p.ContractorID
	}
	if p.ExecutionStatus != nil {
		c.ExecutionStatus = *p.ExecutionStatus
	}
	if p.Progress != nil {
		c.Progress = clampProgress(*p.Progress)
	}
	if p.Received != nil {
		c.Received = *p.Received
	}
	if p.VariationTotal != nil {
		c.VariationTotal = *p.VariationTotal
	}
	if p.PlannedCompletion != nil {
		c.PlannedCompletion = *p.PlannedCompletion
	}
	if p.PMProjectID != nil {
		c.PMProjectID = strings.TrimSpace(*p.PMProjectID)
	}
	if p.Documents != nil {
		c.Documents = *p.Documents
	}
}

// clampProgress bounds a progress percentage to [0, 100].
func clampProgress(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
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
