package models

import (
	"strings"
	"time"
)

// ---------------- Requisitions ----------------

// RequisitionStages is the pre-contract approval chain.
var RequisitionStages = []string{"Project Manager", "Department Head", "Finance", "Management"}

// GovRequisitionPatch corrects a requisition's own description. As with
// GovVariationPatch, status/stage/approvals and linkedContract are absent: the
// chain routes own them.
type GovRequisitionPatch struct {
	No                *string `json:"no,omitempty"`
	Title             *string `json:"title,omitempty"`
	Department        *string `json:"department,omitempty"`
	Requester         *string `json:"requester,omitempty"`
	Type              *string `json:"type,omitempty"`
	ProcurementMethod *string `json:"procurementMethod,omitempty"`
	Supplier          *string `json:"supplier,omitempty"`
	Estimate          *int64  `json:"estimate,omitempty"`
	BudgetCode        *string `json:"budgetCode,omitempty"`
	Urgency           *string `json:"urgency,omitempty"`
	Justification     *string `json:"justification,omitempty"`
	PMProjectID       *string `json:"pmProjectId,omitempty"`
}

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
	// PMProjectID is carried onto the contract this requisition converts into,
	// so the resulting contract inherits its parent Project Management project.
	PMProjectID string    `json:"pmProjectId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	// Sequencing and segregation of duties are the shared engine's — see
	// governance_chains.go.
	completed, _, err := govAdvance("gov.requisition", "", rq.Stage, variationSteps(rq.Approvals), by)
	if err != nil {
		return false, err
	}
	rq.Approvals[completed] = VariationApproval{Step: RequisitionStages[completed], By: by, Date: date}
	rq.Stage++
	if rq.Stage >= len(RequisitionStages) {
		rq.Status = "Approved"
		return true, nil
	}
	return false, nil
}

// Reject declines the requisition at the stage it has reached — see
// GovVariation.Reject.
func (rq *GovRequisition) Reject(by, date, reason string) {
	rq.Status = "Rejected"
	at := rq.Stage
	if at >= len(rq.Approvals) {
		at = len(rq.Approvals) - 1
	}
	if at < 0 {
		return
	}
	step := rq.Approvals[at].Step
	if step == "" && at < len(RequisitionStages) {
		step = RequisitionStages[at]
	}
	rq.Approvals[at] = VariationApproval{Step: step, By: by, Date: date, Note: reason}
}

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
	PMProjectID       string   `json:"pmProjectId"`
	Docs              []GovDoc `json:"docs"`
}

// ---------------- Obligations ----------------

// Obligation status is a canonical, closed set shared by the create form, the
// dashboard/risk KPI filters, and this backend. "Open"/"In Progress"/"Overdue"
// are outstanding; "Met"/"Waived" are settled. IsObligationOpen is the single
// source of truth for the "open obligations" count.
const (
	ObligationOpen       = "Open"
	ObligationInProgress = "In Progress"
	ObligationMet        = "Met"
	ObligationOverdue    = "Overdue"
	ObligationWaived     = "Waived"
)

var obligationStatuses = map[string]bool{
	ObligationOpen: true, ObligationInProgress: true, ObligationMet: true,
	ObligationOverdue: true, ObligationWaived: true,
}

// ValidObligationStatus reports whether s is a known obligation status.
func ValidObligationStatus(s string) bool { return obligationStatuses[s] }

// NormalizeObligationStatus maps free-form/legacy status strings (the prototype
// used "Compliant", "Due Soon", "Pending"…) onto the canonical set, defaulting
// to Open when unrecognized.
func NormalizeObligationStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "met", "compliant", "complete", "completed", "done", "closed":
		return ObligationMet
	case "waived", "n/a", "na":
		return ObligationWaived
	case "overdue", "breached", "late":
		return ObligationOverdue
	case "in progress", "in-progress", "due soon", "active":
		return ObligationInProgress
	case "open", "pending", "outstanding", "":
		return ObligationOpen
	default:
		return ObligationOpen
	}
}

// IsObligationOpen reports whether an obligation still needs attention (i.e. is
// neither Met nor Waived). This is the definition the "open obligations" badge
// and dashboard KPI must both use.
func IsObligationOpen(status string) bool {
	switch NormalizeObligationStatus(status) {
	case ObligationMet, ObligationWaived:
		return false
	default:
		return true
	}
}

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

// ObligationPatch is the PATCH body. Every field is a pointer so that an empty
// string means "clear this" and an absent key means "leave it alone" — the
// previous handler treated both as "leave it alone", which made an obligation's
// owner, evidence and escalation impossible to clear once set.
type ObligationPatch struct {
	ContractID *string `json:"contractId,omitempty"`
	Type       *string `json:"type,omitempty"`
	Owner      *string `json:"owner,omitempty"`
	DueDate    *string `json:"dueDate,omitempty"`
	Frequency  *string `json:"frequency,omitempty"`
	Evidence   *string `json:"evidence,omitempty"`
	Status     *string `json:"status,omitempty"`
	Escalation *string `json:"escalation,omitempty"`
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
		if !strings.EqualFold(r.Status, "Active") {
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

// ---------------- Zone-works milestone governance ----------------

// MilestoneProfile carries the per zone-works milestone governance detail —
// the checklist, scope tasks, deliverables, and comment thread — migrated out
// of the frontend blob. Keyed by the milestone id it annotates.
type MilestoneProfile struct {
	MilestoneID  string                 `json:"milestoneId"`
	ContractNo   string                 `json:"contractNo"`
	Value        int64                  `json:"value"`
	Checklist    []MilestoneCheckItem   `json:"checklist"`
	Scope        []MilestoneScopeTask   `json:"scope"`
	Deliverables []MilestoneDeliverable `json:"deliverables"`
	Comments     []MilestoneComment     `json:"comments"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

type MilestoneCheckItem struct {
	Item string `json:"item"`
	Done bool   `json:"done"`
	By   string `json:"by,omitempty"`
	Date string `json:"date,omitempty"`
}

type MilestoneScopeTask struct {
	Task   string `json:"task"`
	Desc   string `json:"desc,omitempty"`
	Target string `json:"target,omitempty"`
	Status string `json:"status"`
}

type MilestoneDeliverable struct {
	Name string `json:"name"`
	Done bool   `json:"done"`
}

type MilestoneComment struct {
	Author string `json:"author"`
	Role   string `json:"role,omitempty"`
	Date   string `json:"date,omitempty"`
	Text   string `json:"text"`
}

// MilestoneProfileInput is the write shape; the milestone id comes from the path.
type MilestoneProfileInput struct {
	ContractNo   string                 `json:"contractNo"`
	Value        int64                  `json:"value"`
	Checklist    []MilestoneCheckItem   `json:"checklist"`
	Scope        []MilestoneScopeTask   `json:"scope"`
	Deliverables []MilestoneDeliverable `json:"deliverables"`
	Comments     []MilestoneComment     `json:"comments"`
}

// ExecutionTracker records the execution stage a zone-works contract has
// reached, the steps completed so far, and an optional execution date. Keyed
// by contract number.
type ExecutionTracker struct {
	ContractNo    string    `json:"contractNo"`
	Stage         int       `json:"stage"`
	Steps         []string  `json:"steps"`
	ExecutionDate *string   `json:"executionDate"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ExecutionTrackerInput is the write shape; the contract number comes from the path.
type ExecutionTrackerInput struct {
	Stage         int      `json:"stage"`
	Steps         []string `json:"steps"`
	ExecutionDate *string  `json:"executionDate"`
}
