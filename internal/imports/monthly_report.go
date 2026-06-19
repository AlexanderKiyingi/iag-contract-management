// Package imports ingests the Construction Department monthly-report workbook
// (Inspire_Africa_MR-<month><year>.xlsx) into the governance/monthly-report
// schema. It maps the five sheets onto contractors, work-order contracts,
// per-period progress reports, IPC valuations, and consultancy contracts.
//
// The importer is idempotent: contracts are keyed by a deterministic number
// (MR-<period>-<row>), and progress reports / valuations upsert by their natural
// keys, so re-running against the same workbook updates in place.
package imports

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

// Result counts what the import touched.
type Result struct {
	Contractors int `json:"contractors"`
	Contracts   int `json:"contracts"`
	Reports     int `json:"reports"`
	Valuations  int `json:"valuations"`
	Consultancy int `json:"consultancy"`
}

// ImportWorkbook reads path and loads it into the store under the given period
// (e.g. "2026-05").
func ImportWorkbook(ctx context.Context, gov *persistence.GovStore, path, period string) (Result, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("open workbook: %w", err)
	}
	defer func() { _ = f.Close() }()

	var res Result
	contractorIDs := map[string]string{} // lower(name) -> id

	ensureContractor := func(name string) string {
		name = strings.TrimSpace(name)
		if name == "" {
			return ""
		}
		key := strings.ToLower(name)
		if id, ok := contractorIDs[key]; ok {
			return id
		}
		c, err := gov.UpsertContractorByName(ctx, models.GovContractor{ID: models.NewGovID("CON"), Name: name})
		if err != nil {
			return ""
		}
		contractorIDs[key] = c.ID
		res.Contractors++
		return c.ID
	}

	wp := readWorkProgramme(f) // contractor|details -> work-programme row

	periodCompact := strings.ReplaceAll(period, "-", "")

	// ----- Tracker (the master work-order list) -----
	if rows, err := f.GetRows("Tracker"); err == nil {
		seq := 0
		for i, row := range rows {
			if i < 2 { // rows 0-1 are title + header
				continue
			}
			contractor := cell(row, 1)
			details := cell(row, 2)
			if strings.TrimSpace(contractor) == "" || strings.TrimSpace(details) == "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(details), "NEW CONTRACTS") {
				continue // section divider row
			}
			seq++
			number := fmt.Sprintf("MR-%s-%03d", periodCompact, seq)
			contractorID := ensureContractor(contractor)
			exec := models.NormalizeExecutionStatus(cell(row, 7))

			c := models.GovContract{
				Number:            number,
				Name:              strings.TrimSpace(details),
				Contractor:        strings.TrimSpace(contractor),
				ContractorID:      contractorID,
				Type:              "Works",
				Value:             parseMoney(cell(row, 3)),
				VariationTotal:    parseMoney(cell(row, 4)),
				Received:          parseMoney(cell(row, 5)),
				Progress:          parseProgress(cell(row, 6)),
				ExecutionStatus:   exec,
				Status:            lifecycleFor(exec),
				PlannedCompletion: strings.TrimSpace(cell(row, 12)),
			}
			saved, err := upsertContract(ctx, gov, c)
			if err != nil {
				return res, fmt.Errorf("tracker row %d (%s): %w", i+1, number, err)
			}
			res.Contracts++

			rep := models.ProgressReport{
				ID:              models.NewGovID("PRG"),
				ContractID:      saved.ID,
				Period:          period,
				Progress:        c.Progress,
				ExecutionStatus: string(exec),
				CurrentActivity: strings.TrimSpace(cell(row, 8)),
				Accomplishments: strings.TrimSpace(cell(row, 9)),
				Challenges:      strings.TrimSpace(cell(row, 10)),
				Interventions:   strings.TrimSpace(cell(row, 11)),
				PlannedNext:     strings.TrimSpace(cell(row, 13)),
			}
			if w, ok := wp[matchKey(contractor, details)]; ok {
				rep.ProposedStart = w.proposedStart
				rep.ProposedCompletion = w.proposedCompletion
				rep.Duration = w.duration
				rep.Responsible = w.responsible
				rep.TargetDate = w.targetDate
			}
			if _, err := gov.UpsertProgressReport(ctx, rep); err != nil {
				return res, fmt.Errorf("tracker report %d: %w", i+1, err)
			}
			res.Reports++
		}
	}

	// ----- Contractors verified (IPC valuations) -----
	if rows, err := f.GetRows("Contractors verified"); err == nil {
		existing := map[string]models.Valuation{}
		if list, err := gov.ListValuations(ctx, period); err == nil {
			for _, v := range list {
				existing[strings.ToLower(v.ContractorName)] = v
			}
		}
		for i, row := range rows {
			if i < 2 {
				continue
			}
			name := strings.TrimSpace(cell(row, 1))
			if name == "" || strings.EqualFold(name, "TOTAL") {
				continue
			}
			contractorID := ensureContractor(name)
			v := models.Valuation{
				ContractorID:             contractorID,
				ContractorName:           name,
				Period:                   period,
				ContractSum:              parseMoney(cell(row, 2)),
				AmountPaid:               parseMoney(cell(row, 3)),
				VerifiedValueOwed:        parseMoney(cell(row, 4)),
				ConsultantRecommendation: parseMoney(cell(row, 5)),
				CEOApproval:              parseMoney(cell(row, 6)),
				Remarks:                  strings.TrimSpace(cell(row, 7)),
			}
			if prev, ok := existing[strings.ToLower(name)]; ok {
				v.ID = prev.ID
				if _, err := gov.UpdateValuation(ctx, v); err != nil {
					return res, fmt.Errorf("valuation %q: %w", name, err)
				}
			} else {
				v.ID = models.NewGovID("VAL")
				if _, err := gov.CreateValuation(ctx, v); err != nil {
					return res, fmt.Errorf("valuation %q: %w", name, err)
				}
			}
			res.Valuations++
		}
	}

	// ----- Consultancy Works (consultancy contracts) -----
	if rows, err := f.GetRows("Consultancy Works"); err == nil {
		seq := 0
		for i, row := range rows {
			if i < 2 {
				continue
			}
			project := strings.TrimSpace(cell(row, 0))
			consultant := strings.TrimSpace(cell(row, 1))
			if project == "" {
				continue
			}
			seq++
			number := fmt.Sprintf("MR-%s-CON-%03d", periodCompact, seq)
			contractorID := ensureContractor(consultant)
			actual := parseProgress(cell(row, 3))
			exec := execForProgress(actual)
			c := models.GovContract{
				Number:          number,
				Name:            project,
				Contractor:      consultant,
				ContractorID:    contractorID,
				Type:            "Consultancy",
				Value:           parseMoney(cell(row, 4)),
				Received:        parseMoney(cell(row, 5)),
				Progress:        actual,
				ExecutionStatus: exec,
				Status:          lifecycleFor(exec),
			}
			saved, err := upsertContract(ctx, gov, c)
			if err != nil {
				return res, fmt.Errorf("consultancy row %d: %w", i+1, err)
			}
			res.Consultancy++
			if _, err := gov.UpsertProgressReport(ctx, models.ProgressReport{
				ID:              models.NewGovID("PRG"),
				ContractID:      saved.ID,
				Period:          period,
				Progress:        actual,
				PlannedProgress: parseProgress(cell(row, 2)),
				ExecutionStatus: string(exec),
			}); err != nil {
				return res, fmt.Errorf("consultancy report %d: %w", i+1, err)
			}
		}
	}

	return res, nil
}

// upsertContract creates the contract or, when its number already exists,
// updates the existing row in place (preserving its id and milestones).
func upsertContract(ctx context.Context, gov *persistence.GovStore, c models.GovContract) (*models.GovContract, error) {
	existing, err := gov.GetContract(ctx, c.Number)
	if err == nil && existing != nil {
		c.ID = existing.ID
		c.Documents = existing.Documents
		c.Activity = existing.Activity
		return gov.UpdateContract(ctx, c)
	}
	c.ID = models.NewGovID("GCT")
	c.Documents = []models.GovDoc{}
	c.Activity = []models.GovActivity{{Date: importStamp, Actor: "import", Action: "Imported from monthly report"}}
	return gov.CreateContract(ctx, c)
}

// importStamp is a stable activity label for imported rows (no wall-clock so
// re-imports stay reproducible).
const importStamp = "Imported"

// ----- Work Programme enrichment -----

type wpRow struct {
	proposedStart, proposedCompletion, duration, responsible, targetDate string
}

func readWorkProgramme(f *excelize.File) map[string]wpRow {
	out := map[string]wpRow{}
	rows, err := f.GetRows("Work Programme")
	if err != nil {
		return out
	}
	for i, row := range rows {
		if i < 4 { // title + multiline header occupy the first rows
			continue
		}
		contractor := cell(row, 1)
		details := cell(row, 2)
		if strings.TrimSpace(contractor) == "" || strings.TrimSpace(details) == "" {
			continue
		}
		out[matchKey(contractor, details)] = wpRow{
			proposedStart:      strings.TrimSpace(cell(row, 4)),
			proposedCompletion: strings.TrimSpace(cell(row, 5)),
			duration:           strings.TrimSpace(cell(row, 6)),
			responsible:        strings.TrimSpace(cell(row, 9)),
			targetDate:         strings.TrimSpace(cell(row, 10)),
		}
	}
	return out
}

func matchKey(contractor, details string) string {
	norm := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	return norm(contractor) + "|" + norm(details)
}

// ----- value parsing -----

func cell(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// parseMoney reads a UGX amount, tolerating thousands separators, currency
// noise, "k"/"m" suffixes, and placeholders like TBU/TBD/N/A/Nil (-> 0).
func parseMoney(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}
	mult := 1.0
	if strings.HasSuffix(s, "m") {
		mult, s = 1_000_000, strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "k") {
		mult, s = 1_000, strings.TrimSuffix(s, "k")
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	clean := b.String()
	if clean == "" || clean == "-" || clean == "." {
		return 0
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return int64(v * mult)
}

// parseProgress reads a progress percentage. Accepts "95%", "95", and fractional
// "0.65" (-> 65). Result is clamped to [0,100].
func parseProgress(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	hadPercent := strings.Contains(s, "%")
	s = strings.TrimSuffix(strings.TrimSpace(strings.ReplaceAll(s, "%", "")), ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	if !hadPercent && v > 0 && v <= 1 {
		v *= 100 // fractional form like 0.65
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(v + 0.5)
}

func lifecycleFor(e models.ExecutionStatus) models.GovStatus {
	if e == models.ExecCompleted {
		return models.GovCompleted
	}
	return models.GovActive
}

func execForProgress(p int) models.ExecutionStatus {
	switch {
	case p >= 100:
		return models.ExecCompleted
	case p > 0:
		return models.ExecOngoing
	default:
		return models.ExecNotStarted
	}
}
