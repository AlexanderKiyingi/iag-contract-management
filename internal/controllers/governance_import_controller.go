package controllers

import (
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/imports"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// maxImportBytes bounds the multipart parse for the MR workbook upload.
const maxImportBytes = 25 << 20 // 25 MiB

// ImportMonthlyReport ingests an uploaded Construction Department monthly-report
// .xlsx workbook into the governance/monthly-report schema for the given period.
// This is the HTTP counterpart of the `import-mr` CLI: multipart field "file"
// carries the workbook, and "period" (or ?period=) is the YYYY-MM reporting
// period. The import is idempotent (contracts keyed by MR-<period>-<row>).
func (g *GovernanceController) ImportMonthlyReport(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "reports.update") {
		return
	}
	if err := r.ParseMultipartForm(maxImportBytes); err != nil {
		views.Error(w, http.StatusBadRequest, "expected a multipart form with an .xlsx file")
		return
	}
	period := strings.TrimSpace(r.FormValue("period"))
	if period == "" {
		period = strings.TrimSpace(r.URL.Query().Get("period"))
	}
	if period == "" {
		views.Error(w, http.StatusBadRequest, "period is required (YYYY-MM)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		views.Error(w, http.StatusBadRequest, "file is required (multipart field \"file\")")
		return
	}
	defer func() { _ = file.Close() }()
	if header != nil && header.Filename != "" && !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		views.Error(w, http.StatusBadRequest, "file must be an .xlsx workbook")
		return
	}

	res, err := imports.ImportWorkbookFromReader(r.Context(), g.gov, file, period)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "import failed: "+err.Error())
		return
	}
	if g.events != nil {
		g.events.PublishCommercial(r.Context(), "contracts.monthly_report.imported", map[string]any{
			"period": period, "contractors": res.Contractors, "contracts": res.Contracts,
			"reports": res.Reports, "valuations": res.Valuations,
		}, period)
	}
	views.JSON(w, http.StatusOK, map[string]any{"period": period, "imported": res})
}
