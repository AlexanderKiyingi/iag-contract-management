package models

import (
	"errors"
	"time"
)

// Governance financial-execution workflows: milestone payments (4-stage) and
// contract variations (4-stage approval). Server-side stage sequencing is the
// integrity the client-only prototype could not enforce.

// ErrWorkflowComplete is returned when an advance is attempted past the end.
var ErrWorkflowComplete = errors.New("workflow already complete")

// ErrSelfSuccession enforces four-eyes: the actor who completed the previous
// stage of a workflow may not also complete the next one. This is the
// segregation-of-duties control the single `<module>.update` permission cannot
// express on its own.
var ErrSelfSuccession = errors.New("segregation of duties: a different approver is required to advance this stage")

// The system-actor exemption now lives in the shared engine, which applies it
// to every chain rather than to these three by hand.

// PaymentStages is the ordered payment workflow.
var PaymentStages = []string{"PM Approval", "Finance Review", "Payment Authorization", "Paid"}

// VariationStages is the ordered variation approval chain.
var VariationStages = []string{"Project Manager", "Department Head", "Procurement", "Management"}

const (
	// PaymentAuthorizedIdx is the stage index whose completion authorizes
	// disbursement (the point finance should book the AP).
	paymentAuthorizeIdx = 2
)

type PaymentStep struct {
	Step string `json:"step"`
	By   string `json:"by,omitempty"`
	Date string `json:"date,omitempty"`
}

type GovPayment struct {
	ID          string        `json:"id"`
	MilestoneID string        `json:"milestoneId"`
	ContractID  string        `json:"contractId"`
	Amount      int64         `json:"amount"`
	Retention   int           `json:"retention"`
	Payable     int64         `json:"payable"`
	Stage       int           `json:"stage"`
	Status      string        `json:"status"`
	History     []PaymentStep `json:"history"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// NewPayment builds a payment at stage 0 with the payable computed and an empty
// per-stage history.
func NewPayment(id, milestoneID, contractID string, amount int64, retention int) GovPayment {
	hist := make([]PaymentStep, len(PaymentStages))
	for i, s := range PaymentStages {
		hist[i] = PaymentStep{Step: s}
	}
	return GovPayment{
		ID: id, MilestoneID: milestoneID, ContractID: contractID,
		Amount: amount, Retention: retention,
		Payable: amount * int64(100-retention) / 100,
		Stage:   0, Status: PaymentStages[0], History: hist,
	}
}

// Advance completes the current pending stage, returns the index just completed
// and whether that completion authorizes disbursement / marks paid.
func (p *GovPayment) Advance(by, date string) (completed int, authorized, paid bool, err error) {
	if p.Stage >= len(PaymentStages) {
		return -1, false, false, ErrWorkflowComplete
	}
	if len(p.History) < len(PaymentStages) {
		p.History = make([]PaymentStep, len(PaymentStages))
		for i, s := range PaymentStages {
			p.History[i].Step = s
		}
	}
	// Sequencing and segregation of duties are the shared engine's, not this
	// file's — see governance_chains.go.
	completed, _, err = govAdvance("gov.payment", "", p.Stage, paymentSteps(p.History), by)
	if err != nil {
		return -1, false, false, err
	}
	p.History[completed] = PaymentStep{Step: PaymentStages[completed], By: by, Date: date}
	p.Stage++
	if p.Stage >= len(PaymentStages) {
		p.Status = "Paid"
		paid = true
	} else {
		p.Status = PaymentStages[p.Stage]
	}
	authorized = completed == paymentAuthorizeIdx
	return completed, authorized, paid, nil
}

type VariationApproval struct {
	Step string `json:"step"`
	By   string `json:"by,omitempty"`
	Date string `json:"date,omitempty"`
	// Note carries the decline reason on the step that rejected. A refusal with
	// no stated reason leaves the raiser nothing to act on, and the platform
	// requires one at every other approval gate.
	Note string `json:"note,omitempty"`
}

type GovVariation struct {
	ID            string              `json:"id"`
	ContractID    string              `json:"contractId"`
	Number        string              `json:"number"`
	Title         string              `json:"title"`
	Amount        int64               `json:"amount"`
	ExtensionDays int                 `json:"extensionDays"`
	Description   string              `json:"description,omitempty"`
	Reason        string              `json:"reason,omitempty"`
	Impact        string              `json:"impact,omitempty"`
	Status        string              `json:"status"`
	Stage         int                 `json:"stage"`
	Approvals     []VariationApproval `json:"approvals"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}

// GovVariationPatch corrects a variation's own description. Status, stage and
// approvals are absent on purpose: those belong to /variations/:id/advance and
// /reject, which enforce sequencing and four-eyes. A patch that could set
// `status: "Approved"` would walk around both.
type GovVariationPatch struct {
	Number        *string `json:"number,omitempty"`
	Title         *string `json:"title,omitempty"`
	Amount        *int64  `json:"amount,omitempty"`
	ExtensionDays *int    `json:"extensionDays,omitempty"`
	Description   *string `json:"description,omitempty"`
	Reason        *string `json:"reason,omitempty"`
	Impact        *string `json:"impact,omitempty"`
}

// NewVariation builds a Pending variation, auto-recording the first stage
// (Project Manager) as approved by the raiser.
func NewVariation(id, contractID, number, title string, amount int64, days int, description, reason, impact, raisedBy, date string) GovVariation {
	app := make([]VariationApproval, len(VariationStages))
	for i, s := range VariationStages {
		app[i] = VariationApproval{Step: s}
	}
	app[0] = VariationApproval{Step: VariationStages[0], By: raisedBy, Date: date}
	return GovVariation{
		ID: id, ContractID: contractID, Number: number, Title: title, Amount: amount,
		ExtensionDays: days, Description: description, Reason: reason, Impact: impact,
		Status: "Pending", Stage: 1, Approvals: app,
	}
}

// Advance records the next approval. Returns true once fully approved.
func (v *GovVariation) Advance(by, date string) (approved bool, err error) {
	if v.Status != "Pending" {
		return false, ErrWorkflowComplete
	}
	if v.Stage >= len(VariationStages) {
		v.Status = "Approved"
		return true, nil
	}
	if len(v.Approvals) < len(VariationStages) {
		v.Approvals = make([]VariationApproval, len(VariationStages))
		for i, s := range VariationStages {
			v.Approvals[i].Step = s
		}
	}
	// Sequencing and segregation of duties are the shared engine's — see
	// governance_chains.go.
	completed, _, err := govAdvance("gov.variation", "", v.Stage, variationSteps(v.Approvals), by)
	if err != nil {
		return false, err
	}
	v.Approvals[completed] = VariationApproval{Step: VariationStages[completed], By: by, Date: date}
	v.Stage++
	if v.Stage >= len(VariationStages) {
		v.Status = "Approved"
		return true, nil
	}
	return false, nil
}

// Reject terminates the variation.
// Reject declines the variation at the stage it has reached, recording who
// declined it and why on that step.
func (v *GovVariation) Reject(by, date, reason string) {
	v.Status = "Rejected"
	at := v.Stage
	if at >= len(v.Approvals) {
		at = len(v.Approvals) - 1
	}
	if at < 0 {
		return
	}
	step := v.Approvals[at].Step
	if step == "" && at < len(VariationStages) {
		step = VariationStages[at]
	}
	v.Approvals[at] = VariationApproval{Step: step, By: by, Date: date, Note: reason}
}

// ----- inputs -----

type CreatePaymentInput struct {
	Amount    *int64 `json:"amount"`    // optional override; defaults to milestone value
	Retention *int   `json:"retention"` // optional override; defaults to contract retention
}

type CreateVariationInput struct {
	Number        string `json:"number"`
	Title         string `json:"title"`
	Amount        int64  `json:"amount"`
	ExtensionDays int    `json:"extensionDays"`
	Description   string `json:"description"`
	Reason        string `json:"reason"`
	Impact        string `json:"impact"`
}

type WorkflowActionInput struct {
	By string `json:"by"` // optional actor override; defaults to the session user
	// Reason is required on a reject and ignored on an advance.
	Reason string `json:"reason"`
}
