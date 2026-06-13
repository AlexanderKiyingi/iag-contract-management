package models

import "time"

// ---------------- Requisitions ----------------

// RequisitionStages is the pre-contract approval chain.
var RequisitionStages = []string{"Project Manager", "Department Head", "Finance", "Management"}

type GovRequisition struct {
	ID                string              `json:"id"`
	No                string              `json:"no"`
	Title             string              `json:"title"`
	Department        string              `json:"department,omitempty"`
	Requester         string              `json:"requester,omitempty"`
	Type              string              `json:"type,omitempty"`
	ProcurementMethod string              `json:"procurementMethod,omitempty"`
	Supplier          string              `json:"supplier,omitempty"`
	Estimate          int64               `json:"estimate"`
	BudgetCode        string              `json:"budgetCode,omitempty"`
	Urgency           string              `json:"urgency,omitempty"`
	Status            string              `json:"status"`
	Stage             int                 `json:"stage"`
	Approvals         []VariationApproval `json:"approvals"`
	Justification     string              `json:"justification,omitempty"`
	Docs              []GovDoc            `json:"docs"`
	LinkedContract    *string             `json:"linkedContract,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}

// NewRequisition opens a requisition with the raiser recorded as the first
// (Project Manager) approval.
func NewRequisition(id, no, title string, est int64, raisedBy, date string) GovRequisition {
	app := make([]VariationApproval, len(RequisitionStages))
	for i, s := range RequisitionStages {
		app[i] = VariationApproval{Step: s}
	}
	app[0] = VariationApproval{Step: RequisitionStages[0], By: raisedBy, Date: date}
	return GovRequisition{
		ID: id, No: no, Title: title, Estimate: est,
		Status: "Pending Approval", Stage: 1, Approvals: app, Docs: []GovDoc{},
	}
}

// Advance records the next approval; returns true once fully approved.
func (rq *GovRequisition) Advance(by, date string) (approved bool, err error) {
	if rq.Status != "Pending Approval" {
		return false, ErrWorkflowComplete
	}
	if len(rq.Approvals) < len(RequisitionStages) {
		rq.Approvals = make([]VariationApproval, len(RequisitionStages))
		for i, s := range RequisitionStages {
			rq.Approvals[i].Step = s
		}
	}
	if rq.Stage >= len(RequisitionStages) {
		rq.Status = "Approved"
		return true, nil
	}
	rq.Approvals[rq.Stage] = VariationApproval{Step: RequisitionStages[rq.Stage], By: by, Date: date}
	rq.Stage++
	if rq.Stage >= len(RequisitionStages) {
		rq.Status = "Approved"
		return true, nil
	}
	return false, nil
}

func (rq *GovRequisition) Reject() { rq.Status = "Rejected" }

type RequisitionInput struct {
	No                string   `json:"no"`
	Title             string   `json:"title"`
	Department        string   `json:"department"`
	Requester         string   `json:"requester"`
	Type              string   `json:"type"`
	ProcurementMethod string   `json:"procurementMethod"`
	Supplier          string   `json:"supplier"`
	Estimate          int64    `json:"estimate"`
	BudgetCode        string   `json:"budgetCode"`
	Urgency           string   `json:"urgency"`
	Justification     string   `json:"justification"`
	Docs              []GovDoc `json:"docs"`
}

// ---------------- Obligations ----------------

type GovObligation struct {
	ID         string    `json:"id"`
	ContractID string    `json:"contractId"`
	Type       string    `json:"type"`
	Owner      string    `json:"owner,omitempty"`
	DueDate    string    `json:"dueDate,omitempty"`
	Frequency  string    `json:"frequency,omitempty"`
	Evidence   string    `json:"evidence,omitempty"`
	Status     string    `json:"status"`
	Escalation string    `json:"escalation,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ObligationInput struct {
	Type       string `json:"type"`
	Owner      string `json:"owner"`
	DueDate    string `json:"dueDate"`
	Frequency  string `json:"frequency"`
	Evidence   string `json:"evidence"`
	Status     string `json:"status"`
	Escalation string `json:"escalation"`
}

// ---------------- Approval rules ----------------

type GovApprovalRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Applies   string    `json:"applies,omitempty"`
	Threshold string    `json:"threshold,omitempty"`
	MinValue  int64     `json:"minValue"`
	MaxValue  *int64    `json:"maxValue,omitempty"`
	Route     []string  `json:"route"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ApprovalRuleInput struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Applies   string   `json:"applies"`
	Threshold string   `json:"threshold"`
	MinValue  int64    `json:"minValue"`
	MaxValue  *int64   `json:"maxValue"`
	Route     []string `json:"route"`
	Status    string   `json:"status"`
}

// ResolveApprovalRoute returns the active rule whose value band contains value,
// preferring the narrowest band — the routing engine the prototype lacked.
func ResolveApprovalRoute(rules []GovApprovalRule, value int64) *GovApprovalRule {
	var best *GovApprovalRule
	for i := range rules {
		r := &rules[i]
		if r.Status != "Active" {
			continue
		}
		if value < r.MinValue {
			continue
		}
		if r.MaxValue != nil && value >= *r.MaxValue {
			continue
		}
		if best == nil || narrower(r, best) {
			best = r
		}
	}
	return best
}

func narrower(a, b *GovApprovalRule) bool {
	return bandWidth(a) < bandWidth(b)
}

func bandWidth(r *GovApprovalRule) int64 {
	if r.MaxValue == nil {
		return 1 << 62 // open-ended bands are the widest
	}
	return *r.MaxValue - r.MinValue
}

// ---------------- Templates & clauses ----------------

type GovTemplate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type,omitempty"`
	Owner     string    `json:"owner,omitempty"`
	Version   string    `json:"version,omitempty"`
	Status    string    `json:"status"`
	Clauses   []string  `json:"clauses"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TemplateInput struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Owner   string   `json:"owner"`
	Version string   `json:"version"`
	Status  string   `json:"status"`
	Clauses []string `json:"clauses"`
}

type GovClause struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Risk      string    `json:"risk"`
	Approved  bool      `json:"approved"`
	Owner     string    `json:"owner,omitempty"`
	Text      string    `json:"text,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ClauseInput struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Risk     string `json:"risk"`
	Approved bool   `json:"approved"`
	Owner    string `json:"owner"`
	Text     string `json:"text"`
}

// ---------------- Budgets ----------------

type GovBudget struct {
	Code      string    `json:"code"`
	Name      string    `json:"name,omitempty"`
	Owner     string    `json:"owner,omitempty"`
	Approved  int64     `json:"approved"`
	Committed int64     `json:"committed"`
	Paid      int64     `json:"paid"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type BudgetInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Owner     string `json:"owner"`
	Approved  int64  `json:"approved"`
	Committed int64  `json:"committed"`
	Paid      int64  `json:"paid"`
}

// ---------------- Closeout ----------------

type GovCloseout struct {
	ContractID           string    `json:"contractId"`
	FinalAccount         bool      `json:"finalAccount"`
	RetentionDecision    bool      `json:"retentionDecision"`
	DefectsLiability     bool      `json:"defectsLiability"`
	DocumentsComplete    bool      `json:"documentsComplete"`
	UnresolvedVariations int       `json:"unresolvedVariations"`
	FinalReport          bool      `json:"finalReport"`
	Status               string    `json:"status"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type CloseoutInput struct {
	FinalAccount         *bool   `json:"finalAccount"`
	RetentionDecision    *bool   `json:"retentionDecision"`
	DefectsLiability     *bool   `json:"defectsLiability"`
	DocumentsComplete    *bool   `json:"documentsComplete"`
	UnresolvedVariations *int    `json:"unresolvedVariations"`
	FinalReport          *bool   `json:"finalReport"`
	Status               *string `json:"status"`
}
