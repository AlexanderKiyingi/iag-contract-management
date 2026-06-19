package models

import "testing"

func TestNormalizeExecutionStatus(t *testing.T) {
	cases := map[string]ExecutionStatus{
		"Completed":   ExecCompleted,
		"complete":    ExecCompleted,
		"Ongoing":     ExecOngoing,
		"in progress": ExecOngoing,
		"Halted":      ExecHalted,
		"Paused":      ExecPaused,
		"on hold":     ExecPaused,
		"Not Started": ExecNotStarted,
		"":            ExecNotStarted,
		"gibberish":   ExecNotStarted,
	}
	for in, want := range cases {
		if got := NormalizeExecutionStatus(in); got != want {
			t.Errorf("NormalizeExecutionStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExecutionStatusValid(t *testing.T) {
	for _, s := range []ExecutionStatus{ExecNotStarted, ExecOngoing, ExecPaused, ExecHalted, ExecCompleted} {
		if !s.Valid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if ExecutionStatus("Bogus").Valid() {
		t.Error("Bogus should be invalid")
	}
}

func TestBuildMonthlySummary(t *testing.T) {
	contracts := []GovContract{
		{ID: "a", Contractor: "Matovu", ExecutionStatus: ExecCompleted, Progress: 100, Value: 50, Received: 40},
		{ID: "b", Contractor: "Matovu", ExecutionStatus: ExecHalted, Progress: 20, Value: 30, Received: 10},
		// Period report overrides the contract's current fields below.
		{ID: "c", Contractor: "DOASCORE", ExecutionStatus: ExecNotStarted, Progress: 0, Value: 15, Received: 0},
	}
	reports := []ProgressReport{
		{ContractID: "c", Period: "2026-05", ExecutionStatus: "Ongoing", Progress: 60},
	}
	s := BuildMonthlySummary("2026-05", contracts, reports)

	if s.TotalContracts != 3 {
		t.Fatalf("total = %d, want 3", s.TotalContracts)
	}
	if s.Completed != 1 || s.Halted != 1 || s.Ongoing != 1 {
		t.Errorf("status counts: completed=%d halted=%d ongoing=%d", s.Completed, s.Halted, s.Ongoing)
	}
	if s.TotalValue != 95 || s.TotalReceived != 50 {
		t.Errorf("totals: value=%d received=%d", s.TotalValue, s.TotalReceived)
	}
	if len(s.ByContractor) != 2 || s.ByContractor[0].Contractor != "DOASCORE" {
		t.Fatalf("byContractor sort/group wrong: %+v", s.ByContractor)
	}
	// DOASCORE's only contract uses the report's Ongoing/60.
	if s.ByContractor[0].Ongoing != 1 || s.ByContractor[0].AvgProgress != 60 {
		t.Errorf("DOASCORE rollup wrong: %+v", s.ByContractor[0])
	}
	// Matovu avg = (100+20)/2 = 60.
	if s.ByContractor[1].AvgProgress != 60 {
		t.Errorf("Matovu avg = %d, want 60", s.ByContractor[1].AvgProgress)
	}
}
