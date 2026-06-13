package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// ---------------- Requisitions ----------------

func (g *GovernanceController) ListRequisitions(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.read") {
		return
	}
	list, err := g.gov.ListRequisitions(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetRequisition(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.read") {
		return
	}
	rq, err := g.gov.GetRequisition(r.Context(), pathSegmentAfter(r, "requisitions"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, rq)
}

func (g *GovernanceController) CreateRequisition(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.create") {
		return
	}
	var in models.RequisitionInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.No) == "" || strings.TrimSpace(in.Title) == "" {
		views.Error(w, http.StatusBadRequest, "no and title are required")
		return
	}
	rq := models.NewRequisition(models.NewGovID("GREQ"), strings.TrimSpace(in.No), strings.TrimSpace(in.Title), in.Estimate, g.actor(r, ""), nowStamp())
	rq.Department, rq.Requester, rq.Type = in.Department, in.Requester, in.Type
	rq.ProcurementMethod, rq.Supplier, rq.BudgetCode = in.ProcurementMethod, in.Supplier, in.BudgetCode
	rq.Urgency, rq.Justification = in.Urgency, in.Justification
	if in.Docs != nil {
		rq.Docs = in.Docs
	}
	if rq.Urgency == "" {
		rq.Urgency = "Medium"
	}
	created, err := g.gov.CreateRequisition(r.Context(), rq)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) AdvanceRequisition(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.update") {
		return
	}
	rq, err := g.gov.GetRequisition(r.Context(), pathSegmentAfter(r, "requisitions"))
	if g.handleErr(w, err) {
		return
	}
	var in models.WorkflowActionInput
	_ = decodeJSON(r, &in)
	approved, aerr := rq.Advance(g.actor(r, in.By), nowStamp())
	if aerr != nil {
		views.Error(w, http.StatusUnprocessableEntity, "requisition is not pending approval")
		return
	}
	updated, err := g.gov.UpdateRequisition(r.Context(), *rq)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	if approved && g.events != nil {
		// Procurement consumes this to begin sourcing / raise a PO.
		g.events.PublishCommercial(r.Context(), "contracts.requisition.approved", map[string]any{
			"requisitionId": updated.ID, "no": updated.No, "title": updated.Title,
			"estimate": updated.Estimate, "supplier": updated.Supplier,
			"procurementMethod": updated.ProcurementMethod, "budgetCode": updated.BudgetCode,
			"department": updated.Department,
		}, updated.No)
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) RejectRequisition(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.update") {
		return
	}
	rq, err := g.gov.GetRequisition(r.Context(), pathSegmentAfter(r, "requisitions"))
	if g.handleErr(w, err) {
		return
	}
	rq.Reject()
	updated, err := g.gov.UpdateRequisition(r.Context(), *rq)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

// ConvertRequisition turns an approved requisition into a draft governance
// contract and links them.
func (g *GovernanceController) ConvertRequisition(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "requisitions.update") {
		return
	}
	rq, err := g.gov.GetRequisition(r.Context(), pathSegmentAfter(r, "requisitions"))
	if g.handleErr(w, err) {
		return
	}
	if rq.Status != "Approved" {
		views.Error(w, http.StatusUnprocessableEntity, "only an approved requisition can be converted")
		return
	}
	if rq.LinkedContract != nil && *rq.LinkedContract != "" {
		views.Error(w, http.StatusConflict, "requisition already converted")
		return
	}
	contract := models.GovContract{
		ID: models.NewGovID("GCT"), Number: rq.No, Name: rq.Title, Contractor: rq.Supplier,
		Type: rq.Type, Department: rq.Department, Value: rq.Estimate, Status: models.GovDraft,
		Documents: rq.Docs, Activity: []models.GovActivity{{Date: nowStamp(), Actor: g.actor(r, ""), Action: "Created from requisition " + rq.No}},
	}
	created, err := g.gov.CreateContract(r.Context(), contract)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	rq.LinkedContract = &created.ID
	rq.Status = "Converted"
	if _, err := g.gov.UpdateRequisition(r.Context(), *rq); err != nil {
		views.WriteError(w, err)
		return
	}
	if g.events != nil {
		g.events.PublishCommercial(r.Context(), "contracts.requisition.converted", map[string]any{
			"requisitionId": rq.ID, "no": rq.No, "contractId": created.ID, "contractNumber": created.Number,
		}, rq.No)
	}
	views.JSON(w, http.StatusCreated, created)
}

// ---------------- Obligations ----------------

func (g *GovernanceController) ListObligations(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "obligations.read") {
		return
	}
	list, err := g.gov.ListObligations(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) CreateObligation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "obligations.create") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	var in models.ObligationInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Type) == "" {
		views.Error(w, http.StatusBadRequest, "type is required")
		return
	}
	o := models.GovObligation{
		ID: models.NewGovID("GOB"), ContractID: c.ID, Type: in.Type, Owner: in.Owner, DueDate: in.DueDate,
		Frequency: defStr(in.Frequency, "Once"), Evidence: in.Evidence, Status: defStr(in.Status, "Open"), Escalation: in.Escalation,
	}
	created, err := g.gov.CreateObligation(r.Context(), o)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) PatchObligation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "obligations.update") {
		return
	}
	o, err := g.gov.GetObligation(r.Context(), lastPathSegment(r))
	if g.handleErr(w, err) {
		return
	}
	var in models.ObligationInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if in.Type != "" {
		o.Type = in.Type
	}
	if in.Owner != "" {
		o.Owner = in.Owner
	}
	if in.DueDate != "" {
		o.DueDate = in.DueDate
	}
	if in.Frequency != "" {
		o.Frequency = in.Frequency
	}
	if in.Evidence != "" {
		o.Evidence = in.Evidence
	}
	if in.Status != "" {
		o.Status = in.Status
	}
	if in.Escalation != "" {
		o.Escalation = in.Escalation
	}
	updated, err := g.gov.UpdateObligation(r.Context(), *o)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteObligation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "obligations.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteObligation(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ---------------- Approval rules ----------------

func (g *GovernanceController) ListApprovalRules(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "approvals.read") {
		return
	}
	list, err := g.gov.ListApprovalRules(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) ResolveApprovalRoute(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "approvals.read") {
		return
	}
	value, _ := strconv.ParseInt(r.URL.Query().Get("value"), 10, 64)
	rules, err := g.gov.ListApprovalRules(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	rule := models.ResolveApprovalRoute(rules, value)
	if rule == nil {
		views.JSON(w, http.StatusOK, map[string]any{"value": value, "rule": nil, "route": []string{}})
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"value": value, "rule": rule, "route": rule.Route})
}

func (g *GovernanceController) UpsertApprovalRule(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "approvals.update") {
		return
	}
	var in models.ApprovalRuleInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	id := in.ID
	if id == "" {
		id = models.NewGovID("GAR")
	}
	rule := models.GovApprovalRule{
		ID: id, Name: in.Name, Applies: in.Applies, Threshold: in.Threshold,
		MinValue: in.MinValue, MaxValue: in.MaxValue, Route: in.Route, Status: defStr(in.Status, "Active"),
	}
	if rule.Route == nil {
		rule.Route = []string{}
	}
	out, err := g.gov.UpsertApprovalRule(r.Context(), rule)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

func (g *GovernanceController) DeleteApprovalRule(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "approvals.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteApprovalRule(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ---------------- Templates ----------------

func (g *GovernanceController) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "templates.read") {
		return
	}
	list, err := g.gov.ListTemplates(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) UpsertTemplate(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "templates.update") {
		return
	}
	var in models.TemplateInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	id := in.ID
	if id == "" {
		id = models.NewGovID("GTPL")
	}
	t := models.GovTemplate{ID: id, Name: in.Name, Type: in.Type, Owner: in.Owner, Version: defStr(in.Version, "v1.0"), Status: defStr(in.Status, "Draft"), Clauses: in.Clauses}
	if t.Clauses == nil {
		t.Clauses = []string{}
	}
	out, err := g.gov.UpsertTemplate(r.Context(), t)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

func (g *GovernanceController) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "templates.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteTemplate(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ---------------- Clauses ----------------

func (g *GovernanceController) ListClauses(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "clauses.read") {
		return
	}
	list, err := g.gov.ListClauses(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) UpsertClause(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "clauses.update") {
		return
	}
	var in models.ClauseInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		views.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	id := in.ID
	if id == "" {
		id = models.NewGovID("GCL")
	}
	c := models.GovClause{ID: id, Title: in.Title, Risk: defStr(in.Risk, "Medium"), Approved: in.Approved, Owner: in.Owner, Text: in.Text}
	out, err := g.gov.UpsertClause(r.Context(), c)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

func (g *GovernanceController) DeleteClause(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "clauses.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteClause(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ---------------- Budgets ----------------

func (g *GovernanceController) ListBudgets(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "budgets.read") {
		return
	}
	list, err := g.gov.ListBudgets(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "budgets.update") {
		return
	}
	var in models.BudgetInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Code) == "" {
		views.Error(w, http.StatusBadRequest, "code is required")
		return
	}
	out, err := g.gov.UpsertBudget(r.Context(), models.GovBudget{Code: in.Code, Name: in.Name, Owner: in.Owner, Approved: in.Approved, Committed: in.Committed, Paid: in.Paid})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

func (g *GovernanceController) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "budgets.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteBudget(r.Context(), lastPathSegment(r))) {
		return
	}
	views.NoContent(w)
}

// ---------------- Closeout ----------------

func (g *GovernanceController) GetCloseout(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "closeout.read") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	co, err := g.gov.GetCloseout(r.Context(), c.ID)
	if err != nil {
		// Default empty checklist when none exists yet.
		views.JSON(w, http.StatusOK, models.GovCloseout{ContractID: c.ID, Status: "Open"})
		return
	}
	views.JSON(w, http.StatusOK, co)
}

func (g *GovernanceController) UpsertCloseout(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "closeout.update") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	cur, err := g.gov.GetCloseout(r.Context(), c.ID)
	if err != nil {
		cur = &models.GovCloseout{ContractID: c.ID, Status: "Open"}
	}
	var in models.CloseoutInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	applyCloseout(cur, in)
	out, err := g.gov.UpsertCloseout(r.Context(), *cur)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, out)
}

// ---------------- helpers ----------------

func defStr(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func applyCloseout(c *models.GovCloseout, in models.CloseoutInput) {
	if in.FinalAccount != nil {
		c.FinalAccount = *in.FinalAccount
	}
	if in.RetentionDecision != nil {
		c.RetentionDecision = *in.RetentionDecision
	}
	if in.DefectsLiability != nil {
		c.DefectsLiability = *in.DefectsLiability
	}
	if in.DocumentsComplete != nil {
		c.DocumentsComplete = *in.DocumentsComplete
	}
	if in.UnresolvedVariations != nil {
		c.UnresolvedVariations = *in.UnresolvedVariations
	}
	if in.FinalReport != nil {
		c.FinalReport = *in.FinalReport
	}
	if in.Status != nil {
		c.Status = *in.Status
	}
}
