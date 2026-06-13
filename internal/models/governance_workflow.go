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
	completed = p.Stage
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
	v.Approvals[v.Stage] = VariationApproval{Step: VariationStages[v.Stage], By: by, Date: date}
	v.Stage++
	if v.Stage >= len(VariationStages) {
		v.Status = "Approved"
		return true, nil
	}
	return false, nil
}

// Reject terminates the variation.
func (v *GovVariation) Reject() { v.Status = "Rejected" }

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
}
