package controllers

import (
	"net/http"
	"strings"

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
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
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
	retention := contract.Retention
	if in.Retention != nil {
		retention = *in.Retention
	}
	p := models.NewPayment(models.NewGovID("GPAY"), ms.ID, contract.ID, amount, retention)
	created, err := g.gov.CreatePayment(r.Context(), p)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) GetPayment(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
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
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
		return
	}
	p, err := g.gov.GetPayment(r.Context(), pathSegmentAfter(r, "payments"))
	if g.handleErr(w, err) {
		return
	}
	var in models.WorkflowActionInput
	_ = decodeJSON(r, &in)

	_, authorized, paid, aerr := p.Advance(g.actor(r, in.By), nowStamp())
	if aerr != nil {
		views.Error(w, http.StatusUnprocessableEntity, "payment already fully processed")
		return
	}
	updated, err := g.gov.UpdatePayment(r.Context(), *p)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	if g.events != nil && (authorized || paid) {
		// Enrich with the contractor (vendor) and contract number so finance can
		// book the AP without a second lookup.
		var contractor, number string
		if c, cerr := g.gov.GetContract(r.Context(), updated.ContractID); cerr == nil {
			contractor, number = c.Contractor, c.Number
		}
		if authorized {
			g.events.PublishCommercial(r.Context(), "contracts.payment.authorized", map[string]any{
				"paymentId": updated.ID, "contractId": updated.ContractID, "contractNumber": number,
				"contractor": contractor, "milestoneId": updated.MilestoneID,
				"amount": updated.Amount, "payable": updated.Payable, "retention": updated.Retention,
			}, updated.ID)
		}
		if paid {
			g.events.PublishCommercial(r.Context(), "contracts.payment.paid", map[string]any{
				"paymentId": updated.ID, "contractId": updated.ContractID, "contractNumber": number,
				"contractor": contractor, "milestoneId": updated.MilestoneID, "payable": updated.Payable,
			}, updated.ID)
		}
	}
	if paid {
		if ms, err := g.gov.GetMilestone(r.Context(), updated.MilestoneID); err == nil {
			ms.Status = models.MSPaid
			_, _ = g.gov.UpdateMilestone(r.Context(), *ms)
		}
	}
	views.JSON(w, http.StatusOK, updated)
}

// ----- Variations -----

func (g *GovernanceController) ListVariations(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
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

func (g *GovernanceController) CreateVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
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
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
		return
	}
	v, err := g.gov.GetVariation(r.Context(), pathSegmentAfter(r, "variations"))
	if g.handleErr(w, err) {
		return
	}
	var in models.WorkflowActionInput
	_ = decodeJSON(r, &in)
	approved, aerr := v.Advance(g.actor(r, in.By), nowStamp())
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
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) RejectVariation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.create") {
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
	views.JSON(w, http.StatusOK, updated)
}
