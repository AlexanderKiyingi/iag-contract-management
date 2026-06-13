package models

import "testing"

func TestPaymentPayableAndStages(t *testing.T) {
	p := NewPayment("P1", "M1", "C1", 100_000_000, 5)
	if p.Payable != 95_000_000 {
		t.Fatalf("payable = %d, want 95000000", p.Payable)
	}
	if p.Stage != 0 || p.Status != "PM Approval" {
		t.Fatalf("initial stage/status = %d/%q", p.Stage, p.Status)
	}
	// PM Approval
	if _, auth, paid, err := p.Advance("pm", "d"); err != nil || auth || paid {
		t.Fatalf("stage0: auth=%v paid=%v err=%v", auth, paid, err)
	}
	// Finance Review
	p.Advance("fin", "d")
	// Payment Authorization → authorizes
	if _, auth, _, _ := p.Advance("mgmt", "d"); !auth {
		t.Fatal("completing Payment Authorization should authorize")
	}
	// Paid
	_, _, paid, _ := p.Advance("fin", "d")
	if !paid || p.Status != "Paid" {
		t.Fatalf("final: paid=%v status=%q", paid, p.Status)
	}
	if _, _, _, err := p.Advance("x", "d"); err != ErrWorkflowComplete {
		t.Fatalf("advance past paid should error, got %v", err)
	}
	if p.History[0].By != "pm" || p.History[2].By != "mgmt" {
		t.Fatalf("history not recorded: %+v", p.History)
	}
}

func TestVariationApprovalChain(t *testing.T) {
	v := NewVariation("V1", "C1", "VAR-1", "Extra works", 50_000_000, 15, "", "", "", "John", "d")
	if v.Status != "Pending" || v.Stage != 1 {
		t.Fatalf("initial status/stage = %q/%d", v.Status, v.Stage)
	}
	if v.Approvals[0].By != "John" {
		t.Fatal("raiser should be recorded as first approver")
	}
	// Dept, Procurement, Management
	v.Advance("dept", "d")
	v.Advance("proc", "d")
	approved, _ := v.Advance("mgmt", "d")
	if !approved || v.Status != "Approved" {
		t.Fatalf("should be approved after 4 stages: %q", v.Status)
	}
	if _, err := v.Advance("x", "d"); err != ErrWorkflowComplete {
		t.Fatalf("advance after approval should error, got %v", err)
	}
}

func TestVariationReject(t *testing.T) {
	v := NewVariation("V2", "C1", "VAR-2", "X", 0, 0, "", "", "", "John", "d")
	v.Reject()
	if v.Status != "Rejected" {
		t.Fatalf("status = %q", v.Status)
	}
	if _, err := v.Advance("x", "d"); err != ErrWorkflowComplete {
		t.Fatal("cannot advance a rejected variation")
	}
}
