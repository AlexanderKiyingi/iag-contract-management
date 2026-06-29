package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// Monthly-report endpoints (contractors, per-period progress reports, IPC
// valuations, and the executive-summary rollup) that back the Construction
// Department monthly-report workbook. They hang off GovernanceController so
// they share its GovStore + permission model.

// ----- Contractors -----

func (g *GovernanceController) ListContractors(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contractors.read") {
		return
	}
	list, err := g.gov.ListContractors(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetContractor(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contractors.read") {
		return
	}
	c, err := g.gov.GetContractor(r.Context(), pathSegmentAfter(r, "contractors"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

func (g *GovernanceController) CreateContractor(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contractors.create") {
		return
	}
	var in models.GovContractorInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	created, err := g.gov.CreateContractor(r.Context(), models.GovContractor{
		ID:             models.NewGovID("CON"),
		Name:           strings.TrimSpace(in.Name),
		Contact:        in.Contact,
		PlatformUserID: strings.TrimSpace(in.PlatformUserID),
		UserEmail:      strings.TrimSpace(in.UserEmail),
	})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) PatchContractor(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contractors.update") {
		return
	}
	existing, err := g.gov.GetContractor(r.Context(), pathSegmentAfter(r, "contractors"))
	if g.handleErr(w, err) {
		return
	}
	var p models.GovContractorPatch
	if err := decodeJSON(r, &p); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if p.Name != nil {
		existing.Name = *p.Name
	}
	if p.Contact != nil {
		existing.Contact = *p.Contact
	}
	if p.PlatformUserID != nil {
		existing.PlatformUserID = strings.TrimSpace(*p.PlatformUserID)
	}
	if p.UserEmail != nil {
		existing.UserEmail = strings.TrimSpace(*p.UserEmail)
	}
	updated, err := g.gov.UpdateContractor(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteContractor(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contractors.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteContractor(r.Context(), pathSegmentAfter(r, "contractors"))) {
		return
	}
	views.NoContent(w)
}

// ----- Progress reports -----

func (g *GovernanceController) ListContractReports(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "progressreports.read") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	list, err := g.gov.ListProgressReports(r.Context(), c.ID)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) ListReportsByPeriod(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "progressreports.read") {
		return
	}
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		views.Error(w, http.StatusBadRequest, "period query parameter is required")
		return
	}
	list, err := g.gov.ListProgressReportsByPeriod(r.Context(), period)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// UpsertContractReport creates or replaces a contract's report for a period.
func (g *GovernanceController) UpsertContractReport(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "progressreports.update") {
		return
	}
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return
	}
	var in models.ProgressReportInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Period) == "" {
		views.Error(w, http.StatusBadRequest, "period is required")
		return
	}
	rep := models.ProgressReport{
		ID:                 models.NewGovID("PRG"),
		ContractID:         c.ID,
		Period:             strings.TrimSpace(in.Period),
		Progress:           clampProgress(in.Progress),
		ExecutionStatus:    in.ExecutionStatus,
		CurrentActivity:    in.CurrentActivity,
		Accomplishments:    in.Accomplishments,
		Challenges:         in.Challenges,
		Interventions:      in.Interventions,
		Responsible:        in.Responsible,
		TargetDate:         in.TargetDate,
		ProposedStart:      in.ProposedStart,
		ProposedCompletion: in.ProposedCompletion,
		Duration:           in.Duration,
		PlannedNext:        in.PlannedNext,
		PlannedProgress:    clampProgress(in.PlannedProgress),
	}
	saved, err := g.gov.UpsertProgressReport(r.Context(), rep)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, saved)
}

func (g *GovernanceController) DeleteReport(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "progressreports.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteProgressReport(r.Context(), pathSegmentAfter(r, "reports"))) {
		return
	}
	views.NoContent(w)
}

// ----- Valuations -----

func (g *GovernanceController) ListValuations(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "valuations.read") {
		return
	}
	list, err := g.gov.ListValuations(r.Context(), strings.TrimSpace(r.URL.Query().Get("period")))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetValuation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "valuations.read") {
		return
	}
	v, err := g.gov.GetValuation(r.Context(), pathSegmentAfter(r, "valuations"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, v)
}

func (g *GovernanceController) CreateValuation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "valuations.create") {
		return
	}
	var in models.ValuationInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.ContractorName) == "" {
		views.Error(w, http.StatusBadRequest, "contractorName is required")
		return
	}
	created, err := g.gov.CreateValuation(r.Context(), models.Valuation{
		ID:                       models.NewGovID("VAL"),
		ContractorID:             in.ContractorID,
		ContractorName:           strings.TrimSpace(in.ContractorName),
		Period:                   in.Period,
		ContractSum:              in.ContractSum,
		AmountPaid:               in.AmountPaid,
		VerifiedValueOwed:        in.VerifiedValueOwed,
		ConsultantRecommendation: in.ConsultantRecommendation,
		CEOApproval:              in.CEOApproval,
		Remarks:                  in.Remarks,
		VerifiedDate:             in.VerifiedDate,
	})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) UpdateValuation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "valuations.update") {
		return
	}
	existing, err := g.gov.GetValuation(r.Context(), pathSegmentAfter(r, "valuations"))
	if g.handleErr(w, err) {
		return
	}
	var in models.ValuationInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	existing.ContractorID = in.ContractorID
	if strings.TrimSpace(in.ContractorName) != "" {
		existing.ContractorName = strings.TrimSpace(in.ContractorName)
	}
	existing.Period = in.Period
	existing.ContractSum = in.ContractSum
	existing.AmountPaid = in.AmountPaid
	existing.VerifiedValueOwed = in.VerifiedValueOwed
	existing.ConsultantRecommendation = in.ConsultantRecommendation
	existing.CEOApproval = in.CEOApproval
	existing.Remarks = in.Remarks
	existing.VerifiedDate = in.VerifiedDate
	updated, err := g.gov.UpdateValuation(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteValuation(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "valuations.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteValuation(r.Context(), pathSegmentAfter(r, "valuations"))) {
		return
	}
	views.NoContent(w)
}

// ----- Challenges register -----

func (g *GovernanceController) ListChallenges(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "challenges.read") {
		return
	}
	list, err := g.gov.ListChallenges(r.Context(), strings.TrimSpace(r.URL.Query().Get("period")))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetChallenge(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "challenges.read") {
		return
	}
	c, err := g.gov.GetChallenge(r.Context(), pathSegmentAfter(r, "challenges"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

func (g *GovernanceController) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "challenges.create") {
		return
	}
	var in models.ChallengeInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Period) == "" {
		views.Error(w, http.StatusBadRequest, "period is required")
		return
	}
	if strings.TrimSpace(in.Description) == "" {
		views.Error(w, http.StatusBadRequest, "description is required")
		return
	}
	created, err := g.gov.CreateChallenge(r.Context(), models.Challenge{
		ID:          models.NewGovID("GCHL"),
		Period:      strings.TrimSpace(in.Period),
		Category:    in.Category,
		Description: strings.TrimSpace(in.Description),
		Affected:    in.Affected,
		Priority:    valOr(in.Priority, "Medium"),
		Action:      in.Action,
		Owner:       in.Owner,
	})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "challenges.update") {
		return
	}
	existing, err := g.gov.GetChallenge(r.Context(), pathSegmentAfter(r, "challenges"))
	if g.handleErr(w, err) {
		return
	}
	var in models.ChallengeInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	existing.Category = in.Category
	if strings.TrimSpace(in.Description) != "" {
		existing.Description = strings.TrimSpace(in.Description)
	}
	existing.Affected = in.Affected
	if strings.TrimSpace(in.Priority) != "" {
		existing.Priority = in.Priority
	}
	existing.Action = in.Action
	existing.Owner = in.Owner
	updated, err := g.gov.UpdateChallenge(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteChallenge(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "challenges.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteChallenge(r.Context(), pathSegmentAfter(r, "challenges"))) {
		return
	}
	views.NoContent(w)
}

// ----- Action-item tracker -----

func (g *GovernanceController) ListActionItems(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "actionitems.read") {
		return
	}
	list, err := g.gov.ListActionItems(r.Context(), strings.TrimSpace(r.URL.Query().Get("period")))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (g *GovernanceController) GetActionItem(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "actionitems.read") {
		return
	}
	a, err := g.gov.GetActionItem(r.Context(), pathSegmentAfter(r, "action-items"))
	if g.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, a)
}

func (g *GovernanceController) CreateActionItem(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "actionitems.create") {
		return
	}
	var in models.ActionItemInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Period) == "" {
		views.Error(w, http.StatusBadRequest, "period is required")
		return
	}
	if strings.TrimSpace(in.Text) == "" {
		views.Error(w, http.StatusBadRequest, "text is required")
		return
	}
	created, err := g.gov.CreateActionItem(r.Context(), models.ActionItem{
		ID:       models.NewGovID("GACT"),
		Period:   strings.TrimSpace(in.Period),
		Priority: valOr(in.Priority, "Medium"),
		Text:     strings.TrimSpace(in.Text),
		Party:    in.Party,
		Target:   in.Target,
		Status:   valOr(in.Status, "Pending"),
	})
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, created)
}

func (g *GovernanceController) UpdateActionItem(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "actionitems.update") {
		return
	}
	existing, err := g.gov.GetActionItem(r.Context(), pathSegmentAfter(r, "action-items"))
	if g.handleErr(w, err) {
		return
	}
	var in models.ActionItemInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Priority) != "" {
		existing.Priority = in.Priority
	}
	if strings.TrimSpace(in.Text) != "" {
		existing.Text = strings.TrimSpace(in.Text)
	}
	existing.Party = in.Party
	existing.Target = in.Target
	if strings.TrimSpace(in.Status) != "" {
		existing.Status = in.Status
	}
	updated, err := g.gov.UpdateActionItem(r.Context(), *existing)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

func (g *GovernanceController) DeleteActionItem(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "actionitems.delete") {
		return
	}
	if g.handleErr(w, g.gov.DeleteActionItem(r.Context(), pathSegmentAfter(r, "action-items"))) {
		return
	}
	views.NoContent(w)
}

// valOr returns v trimmed, or fallback when v is blank.
func valOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

// ----- Executive summary rollup -----

func (g *GovernanceController) MonthlySummaryReport(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "reports.read") {
		return
	}
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	contracts, err := g.gov.ListContracts(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	var reports []models.ProgressReport
	if period != "" {
		reports, err = g.gov.ListProgressReportsByPeriod(r.Context(), period)
		if err != nil {
			views.WriteError(w, err)
			return
		}
	}
	views.JSON(w, http.StatusOK, models.BuildMonthlySummary(period, contracts, reports))
}

// ----- Excel export (regenerate the MR workbook) -----

// ExportMonthlyReportXLSX streams a five-section .xlsx workbook (Executive
// Summary, Tracker, Contractors verified) rebuilt from the stored data for a
// period.
func (g *GovernanceController) ExportMonthlyReportXLSX(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "reports.read") {
		return
	}
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		views.Error(w, http.StatusBadRequest, "period query parameter is required")
		return
	}

	contracts, err := g.gov.ListContracts(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	reports, err := g.gov.ListProgressReportsByPeriod(r.Context(), period)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	valuations, err := g.gov.ListValuations(r.Context(), period)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	reportByContract := make(map[string]models.ProgressReport, len(reports))
	for _, rep := range reports {
		reportByContract[rep.ContractID] = rep
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	f.SetSheetName("Sheet1", "Executive Summary")

	// Executive Summary
	sum := models.BuildMonthlySummary(period, contracts, reports)
	setRow(f, "Executive Summary", 1, "INSPIRE AFRICA GROUP – EXECUTIVE SUMMARY", period)
	setRow(f, "Executive Summary", 2, "Total Contracts", "Completed", "Ongoing", "Halted", "Paused", "Not Started", "Total Value", "Total Received")
	setRow(f, "Executive Summary", 3,
		strconv.Itoa(sum.TotalContracts), strconv.Itoa(sum.Completed), strconv.Itoa(sum.Ongoing),
		strconv.Itoa(sum.Halted), strconv.Itoa(sum.Paused), strconv.Itoa(sum.NotStarted),
		strconv.FormatInt(sum.TotalValue, 10), strconv.FormatInt(sum.TotalReceived, 10))
	setRow(f, "Executive Summary", 5, "Contractor", "Total", "Completed", "Ongoing", "Halted", "Paused", "Not Started", "Avg Progress")
	row := 6
	for _, cs := range sum.ByContractor {
		setRow(f, "Executive Summary", row, cs.Contractor, strconv.Itoa(cs.Total), strconv.Itoa(cs.Completed),
			strconv.Itoa(cs.Ongoing), strconv.Itoa(cs.Halted), strconv.Itoa(cs.Paused), strconv.Itoa(cs.NotStarted),
			strconv.Itoa(cs.AvgProgress)+"%")
		row++
	}

	// Tracker
	_, _ = f.NewSheet("Tracker")
	setRow(f, "Tracker", 1, "S/N", "Contractor", "Contract Details", "Contract Amount (UGX)", "Variation (UGX)",
		"Received Amount (UGX)", "Progress (%)", "Status", "Current Activity", "Accomplishments", "Challenges",
		"Interventions Required", "Planned Completion Date", "Planned Activities")
	tr := 2
	for i, c := range contracts {
		rep := reportByContract[c.ID]
		setRow(f, "Tracker", tr,
			strconv.Itoa(i+1), c.Contractor, c.Name,
			strconv.FormatInt(c.Value, 10), strconv.FormatInt(c.VariationTotal, 10),
			strconv.FormatInt(c.Received, 10), strconv.Itoa(c.Progress)+"%", string(c.ExecutionStatus),
			rep.CurrentActivity, rep.Accomplishments, rep.Challenges, rep.Interventions,
			c.PlannedCompletion, rep.PlannedNext)
		tr++
	}

	// Contractors verified
	_, _ = f.NewSheet("Contractors verified")
	setRow(f, "Contractors verified", 1, "S/No", "Contractor Name", "Contract Sum (UGX)", "Amount Paid to Date (UGX)",
		"Verified Value Owed (UGX)", "Consultant Recommendation", "CEO Approval", "Remarks")
	vr := 2
	for i, v := range valuations {
		setRow(f, "Contractors verified", vr,
			strconv.Itoa(i+1), v.ContractorName, strconv.FormatInt(v.ContractSum, 10),
			strconv.FormatInt(v.AmountPaid, 10), strconv.FormatInt(v.VerifiedValueOwed, 10),
			strconv.FormatInt(v.ConsultantRecommendation, 10), strconv.FormatInt(v.CEOApproval, 10), v.Remarks)
		vr++
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="monthly-report-`+period+`.xlsx"`)
	if err := f.Write(w); err != nil {
		views.WriteError(w, err)
		return
	}
}

// setRow writes a sequence of cell values starting at column A of the given row.
func setRow(f *excelize.File, sheet string, row int, values ...string) {
	for i, v := range values {
		cellRef, err := excelize.CoordinatesToCellName(i+1, row)
		if err != nil {
			continue
		}
		_ = f.SetCellStr(sheet, cellRef, v)
	}
}
