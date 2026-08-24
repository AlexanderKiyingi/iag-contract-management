package controllers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// actor resolves the acting principal: an explicit body override, else the
// session display name/email, else "system".
func (g *GovernanceController) actor(r *http.Request, override string) string {
	if s := strings.TrimSpace(override); s != "" {
		return s
	}
	sess := g.model.SessionFromRequest(r.Context())
	if sess.DisplayName != "" {
		return sess.DisplayName
	}
	if sess.Email != "" {
		return sess.Email
	}
	return "system"
}

// ----- Payments -----

// CreatePayment opens the payment workflow for a milestone (amount defaults to
// the milestone value, retention to the contract's).
func (g *GovernanceController) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "payments.create") {
		return
	}
	mID := pathSegmentAfter(r, "milestones")
	ms, err := g.gov.GetMilestone(r.Context(), mID)
	if g.handleErr(w, err) {
		return
	}
	if existing, err := g.gov.GetPaymentByMilestone(r.Context(), mID); err == nil && existing != nil {
		views.JSON(w, http.StatusOK, existing) // idempotent
		return
	}
	contract, err := g.gov.GetContract(r.Context(), ms.ContractID)
	if g.handleErr(w, err) {
		return
	}
	var in models.CreatePaymentInput
	_ = decodeJSON(r, &in)
	amount := ms.Value
	if in.Amount != nil {
		amount = *in.Amount
	}
	if amount < 0 {
		views.Error(w, http.StatusBadRequest, "amount must not be negative")
		return
	}
	retention := contract.Retention
	if in.Retention != nil {
		retention = *in.Retention
	}
	// Retention is a percentage; clamp so payable = amount*(100-retention)/100
	// can never be negative or exceed the gross amount.
	if retention < 0 {
		retention = 0
	}
	if retention > 100 {
		retention = 100
	}
	p := models.NewPayment(models.NewGovID("GPAY"), ms.ID, contract.ID, amount, retention)
	created, err := g.gov.CreatePayment(r.Context(), p)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

// ListPayments returns the payment queue across all milestones, filterable by
// ?contractId= and ?status= — the finance payment dashboard.
func (g *GovernanceController) ListPayments(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "payments.read") {
		return
	}
	list, err := g.gov.ListPayments(r.Context(),
		strings.TrimSpace(r.URL.Query().Get("contractId")),
		strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetPayment(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "payments.read") {
		return
	}
	p, err := g.gov.GetPayment(r.Context(), pathSegmentAfter(r, "payments"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, p)
}

// AdvancePayment completes the current pending stage. Authorizing the payment
// emits an event finance consumes to book the AP; marking paid flips the
// milestone to Paid.
func (g *GovernanceController) AdvancePayment(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "payments.update") {
		return
	}
	p, err := g.gov.GetPayment(r.Context(), pathSegmentAfter(r, "payments"))
	if g.handleErr(w, err) {
		return
	}
	var in models.WorkflowActionInput
	_ = decodeJSON(r, &in)

	_, authorized, paid, aerr := p.Advance(g.actor(r, in.By), nowStamp())
	if errors.Is(aerr, models.ErrSelfSuccession) {
		views.Error(w, http.StatusForbidden, aerr.Error())
		return
	}
	if aerr != nil {
		views.Error(w, http.StatusUnprocessableEntity, "payment already fully processed")
		return
	}
	// The stage advance and the events it triggers commit together. The
	// authorization stage is what instructs finance to book the payable, so a
	// request that advanced the stage but failed to record that instruction
	// would leave a payment authorized here and invisible there — with nothing
	// to reconcile from, because contracts believes it is done.
	var updated *models.GovPayment
	err = g.gov.InTx(r.Context(), func(ctx context.Context) error {
		u, uerr := g.gov.UpdatePaymentTx(ctx, *p)
		if uerr != nil {
			return uerr
		}
		updated = u
		if g.events == nil || !(authorized || paid) {
			return nil
		}
		var contractor, number, currency string
		if c, cerr := g.gov.GetContract(ctx, updated.ContractID); cerr == nil {
			contractor, number, currency = c.Contractor, c.Number, c.Currency
		}
		// The contract's own currency. Without it finance booked every contract
		// payable as UGX, so a payment on a USD contract became the same number
		// of shillings.
		if strings.TrimSpace(currency) == "" {
			currency = "UGX"
		}
		if authorized {
			if perr := g.events.PublishCommercialTx(ctx, "contracts.payment.authorized", map[string]any{
				"paymentId": updated.ID, "contractId": updated.ContractID, "contractNumber": number,
				"contractor": contractor, "milestoneId": updated.MilestoneID,
				"amount": updated.Amount, "payable": updated.Payable, "retention": updated.Retention,
				"currency": currency,
			}, updated.ID); perr != nil {
				return perr
			}
		}
		if paid {
			if perr := g.events.PublishCommercialTx(ctx, "contracts.payment.paid", map[string]any{
				"paymentId": updated.ID, "contractId": updated.ContractID, "contractNumber": number,
				"contractor": contractor, "milestoneId": updated.MilestoneID, "payable": updated.Payable,
				"currency": currency,
			}, updated.ID); perr != nil {
				return perr
			}
		}
		return nil
	})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	if paid {
		if ms, err := g.gov.GetMilestone(r.Context(), updated.MilestoneID); err == nil {
			ms.Status = models.MSPaid
			_, _ = g.gov.UpdateMilestone(r.Context(), *ms)
		}
	}
	if authorized {
		g.postContractSystem(updated.ContractID, "Payment authorized (by "+g.actor(r, in.By)+")")
	}
	if paid {
		g.postContractSystem(updated.ContractID, "Payment marked paid")
	}
	views.JSON(w, http.StatusOK, updated)
}

// ----- Variations -----

func (g *GovernanceController) ListVariations(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.read") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	list, err := g.gov.ListVariations(r.Context(), c.ID)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// ListAllVariations returns variations across all contracts, filterable by
// ?contractId= and ?status= — the variations approval queue.
func (g *GovernanceController) ListAllVariations(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.read") {
		return
	}
	list, err := g.gov.ListAllVariations(r.Context(),
		strings.TrimSpace(r.URL.Query().Get("contractId")),
		strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// GetVariation returns a single variation by id.
func (g *GovernanceController) GetVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.read") {
		return
	}
	v, err := g.gov.GetVariation(r.Context(), pathSegmentAfter(r, "variations"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, v)
}

func (g *GovernanceController) CreateVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.create") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	var in models.CreateVariationInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		views.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	v := models.NewVariation(models.NewGovID("GVAR"), c.ID, strings.TrimSpace(in.Number), strings.TrimSpace(in.Title),
		in.Amount, in.ExtensionDays, in.Description, in.Reason, in.Impact, g.actor(r, ""), nowStamp())
	created, err := g.gov.CreateVariation(r.Context(), v)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

// AdvanceVariation records the next approval. On full approval the contract
// value is adjusted and an event is emitted.
func (g *GovernanceController) AdvanceVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.update") {
		return
	}
	v, err := g.gov.GetVariation(r.Context(), pathSegmentAfter(r, "variations"))
	if g.handleErr(w, err) {
		return
	}
	var in models.WorkflowActionInput
	_ = decodeJSON(r, &in)
	approved, aerr := v.Advance(g.actor(r, in.By), nowStamp())
	if errors.Is(aerr, models.ErrSelfSuccession) {
		views.Error(w, http.StatusForbidden, aerr.Error())
		return
	}
	if aerr != nil {
		views.Error(w, http.StatusUnprocessableEntity, "variation is not pending")
		return
	}
	updated, err := g.gov.UpdateVariation(r.Context(), *v)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	if approved {
		if updated.Amount != 0 {
			_ = g.gov.AddContractValue(r.Context(), updated.ContractID, updated.Amount)
		}
		if g.events != nil {
			g.events.PublishCommercial(r.Context(), "contracts.variation.approved", map[string]any{
				"variationId": updated.ID, "contractId": updated.ContractID, "number": updated.Number,
				"amount": updated.Amount, "extensionDays": updated.ExtensionDays,
			}, updated.ID)
		}
		g.postContractSystem(updated.ContractID,
			"Variation "+strings.TrimSpace(updated.Number)+" approved")
	}
	g.notifyGovDecision(r.Context(), "Variation "+strings.TrimSpace(updated.Number),
		updated.ID, outcomeFor(approved), updated.Stage, "")
	views.JSON(w, http.StatusOK, updated)
}

// outcomeFor maps a stage advance onto the two approval templates: a variation
// that cleared its final stage is a decision, one that only moved forward is
// still pending the next desk.
func outcomeFor(complete bool) string {
	if complete {
		return "approved"
	}
	return "pending"
}

// notifyGovDecision reports a governance approval step to the notifications
// desk. Governance records carry an approver trail but usually no requester
// address — approvals are by named actor, not by account.
//
// preferred is the record's requester when it is an addressable one. That case
// deliberately sends NO audience: an audience resolves to whoever an admin has
// put on the approvals desk, which is right for a desk notification and wrong
// for one aimed at a named person. Only the desk fallback is routable.
//
// Best effort: the decision is committed and must not fail on a notification
// problem.
func (g *GovernanceController) notifyGovDecision(ctx context.Context, what, id, outcome string, stage int, preferred string) {
	if g.events == nil {
		return
	}
	// Requester is free text — a name on most records, an address on some.
	// Only route to it when it is actually addressable.
	recipient := events.DefaultNotifyRecipient()
	audience := "approvals.contracts"
	if strings.Contains(preferred, "@") {
		recipient = strings.TrimSpace(preferred)
		audience = ""
	}
	template := "approval.decision"
	title := what + " " + outcome
	body := what + " was " + outcome + "."
	if outcome == "pending" {
		template = "approval.pending"
		title = what + " awaiting approval"
		body = what + " advanced and is now awaiting stage " + strconv.Itoa(stage) + "."
	}
	g.events.PublishAlertTo(ctx, "", audience, recipient, template, map[string]string{
		"Title": title,
		"Body":  body,
	}, id)
}

func (g *GovernanceController) RejectVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "variations.update") {
		return
	}
	v, err := g.gov.GetVariation(r.Context(), pathSegmentAfter(r, "variations"))
	if g.handleErr(w, err) {
		return
	}
	v.Reject()
	updated, err := g.gov.UpdateVariation(r.Context(), *v)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	g.notifyGovDecision(r.Context(), "Variation "+strings.TrimSpace(updated.Number),
		updated.ID, "rejected", updated.Stage, "")
	views.JSON(w, http.StatusOK, updated)
}
