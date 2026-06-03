package controllers

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type ExportsController struct {
	model *models.Store
}

func NewExportsController(model *models.Store) *ExportsController {
	return &ExportsController{model: model}
}

// ExportContractsCSV streams a properly-escaped CSV of all contracts.
// Uses encoding/csv so commas, double-quotes, and newlines in fields like
// `name` or `remarks` are quoted per RFC 4180 instead of breaking row
// alignment (the previous fmt.Fprintf path did).
func (c *ExportsController) ExportContractsCSV(w http.ResponseWriter, r *http.Request) {
	if !c.model.HasPermissionCtx(r.Context(), "contracts.read") &&
		!c.model.HasPermissionCtx(r.Context(), "reports.create") {
		views.WriteError(w, models.ErrForbidden)
		return
	}
	contracts := c.model.ListContractsForSession(r.Context())
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="contracts-export.csv"`)

	cw := csv.NewWriter(w)
	header := []string{"no", "name", "zone", "status", "priority", "prog",
		"cs", "paid", "bal", "workers", "sup", "created"}
	if err := cw.Write(header); err != nil {
		return
	}
	for _, row := range contracts {
		record := []string{
			row.No, row.Name, row.Zone,
			string(row.Status), string(row.Pri),
			strconv.Itoa(row.Prog),
			strconv.FormatInt(row.Cs, 10),
			strconv.FormatInt(row.Paid, 10),
			strconv.FormatInt(row.Bal, 10),
			strconv.Itoa(row.Workers),
			row.Sup, row.Created,
		}
		if err := cw.Write(record); err != nil {
			return
		}
	}
	cw.Flush()
}
