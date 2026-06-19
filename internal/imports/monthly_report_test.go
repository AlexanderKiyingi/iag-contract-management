package imports

import (
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

func TestParseMoney(t *testing.T) {
	cases := map[string]int64{
		"50,000,000":          50_000_000,
		"170000000":           170_000_000,
		" 28,000,000 ":        28_000_000,
		"10m":                 10_000_000,
		"5M":                  5_000_000,
		"500k":                500_000,
		"":                    0,
		"TBU":                 0,
		"N/A":                 0,
		"Nil":                 0,
		"-":                   0,
		"no payment recieved": 0,
		"UGX 22,000,000":      22_000_000,
	}
	for in, want := range cases {
		if got := parseMoney(in); got != want {
			t.Errorf("parseMoney(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseProgress(t *testing.T) {
	cases := map[string]int{
		"95%": 95, "100%": 100, "60": 60, "0.65": 65, "0.52": 52,
		"": 0, "TBD": 0, "0%": 0, "0.03": 3,
	}
	for in, want := range cases {
		if got := parseProgress(in); got != want {
			t.Errorf("parseProgress(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestWorkbookStatusSplit parses the real workbook (when present) and asserts
// the Tracker maps onto the status split the Executive Summary reports.
func TestWorkbookStatusSplit(t *testing.T) {
	const path = "../../Inspire_Africa_MR-May2026.xlsx"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("workbook not present: %v", err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Tracker")
	if err != nil {
		t.Fatalf("Tracker: %v", err)
	}
	counts := map[models.ExecutionStatus]int{}
	total := 0
	for i, row := range rows {
		if i < 2 {
			continue
		}
		contractor, details := cell(row, 1), cell(row, 2)
		if strings.TrimSpace(contractor) == "" || strings.TrimSpace(details) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(details), "NEW CONTRACTS") {
			continue
		}
		total++
		counts[models.NormalizeExecutionStatus(cell(row, 7))]++
	}
	t.Logf("tracker total=%d completed=%d ongoing=%d halted=%d paused=%d notStarted=%d",
		total, counts[models.ExecCompleted], counts[models.ExecOngoing], counts[models.ExecHalted],
		counts[models.ExecPaused], counts[models.ExecNotStarted])

	if total < 130 || total > 145 {
		t.Errorf("tracker total = %d, expected ~135", total)
	}
	if counts[models.ExecCompleted] < 40 {
		t.Errorf("completed = %d, expected ~46", counts[models.ExecCompleted])
	}
}
