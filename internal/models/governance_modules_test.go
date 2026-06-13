package models

import "testing"

func TestRequisitionApprovalChain(t *testing.T) {
	rq := NewRequisition("R1", "REQ-1", "Solar backup", 286_000_000, "James", "d")
	if rq.Status != "Pending Approval" || rq.Stage != 1 {
		t.Fatalf("initial status/stage = %q/%d", rq.Status, rq.Stage)
	}
	rq.Advance("dept", "d")
	rq.Advance("fin", "d")
	approved, _ := rq.Advance("mgmt", "d")
	if !approved || rq.Status != "Approved" {
		t.Fatalf("should be approved after 4 stages: %q", rq.Status)
	}
	if _, err := rq.Advance("x", "d"); err != ErrWorkflowComplete {
		t.Fatal("cannot advance an approved requisition")
	}
}

func TestResolveApprovalRoute(t *testing.T) {
	m := func(v int64) *int64 { return &v }
	rules := []GovApprovalRule{
		{ID: "AR-01", Name: "Low", MinValue: 0, MaxValue: m(50_000_000), Route: []string{"PM", "Finance"}, Status: "Active"},
		{ID: "AR-02", Name: "Standard", MinValue: 50_000_000, MaxValue: m(500_000_000), Route: []string{"PM", "Dept", "Finance"}, Status: "Active"},
		{ID: "AR-03", Name: "High", MinValue: 500_000_000, MaxValue: nil, Route: []string{"PM", "Dept", "Procurement", "Management"}, Status: "Active"},
		{ID: "AR-OFF", Name: "Disabled", MinValue: 0, MaxValue: m(1_000_000_000_000), Route: []string{"X"}, Status: "Inactive"},
	}
	cases := map[int64]string{
		20_000_000:  "AR-01",
		120_000_000: "AR-02",
		850_000_000: "AR-03",
		50_000_000:  "AR-02", // boundary: min inclusive
	}
	for v, want := range cases {
		got := ResolveApprovalRoute(rules, v)
		if got == nil || got.ID != want {
			t.Errorf("value %d → %v, want %s", v, got, want)
		}
	}
	// Inactive rule never selected even though its band covers everything.
	if r := ResolveApprovalRoute([]GovApprovalRule{rules[3]}, 5); r != nil {
		t.Errorf("inactive rule should not resolve, got %v", r)
	}
}
