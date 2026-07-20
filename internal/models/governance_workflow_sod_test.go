package models

import (
	"errors"
	"testing"
)

// Four-eyes / segregation-of-duties: the actor who completed the previous stage
// of a workflow may not complete the next one. System/empty actors are exempt.

func TestPaymentAdvance_FourEyes(t *testing.T) {
	p := NewPayment("GPAY-1", "MS-1", "C-1", 1000, 10)

	// Stage 0 (PM Approval) by alice — allowed (no prior human actor).
	if _, _, _, err := p.Advance("alice", "2026-07-19"); err != nil {
		t.Fatalf("stage0 by alice: unexpected err %v", err)
	}
	// Stage 1 (Finance Review) by alice again — blocked.
	if _, _, _, err := p.Advance("alice", "2026-07-19"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("stage1 by alice: want ErrSelfSuccession, got %v", err)
	}
	// Case-insensitive: "Alice" is still the same actor.
	if _, _, _, err := p.Advance("Alice", "2026-07-19"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("stage1 by Alice: want ErrSelfSuccession, got %v", err)
	}
	// Stage 1 by bob — allowed.
	if _, _, _, err := p.Advance("bob", "2026-07-19"); err != nil {
		t.Fatalf("stage1 by bob: unexpected err %v", err)
	}
	// Stage 2 (Payment Authorization) by alice — allowed, authorizes disbursement.
	_, authorized, _, err := p.Advance("alice", "2026-07-19")
	if err != nil {
		t.Fatalf("stage2 by alice: %v", err)
	}
	if !authorized {
		t.Fatalf("stage2 completion should authorize disbursement")
	}
	// Stage 3 (Paid) by bob — allowed (differs from alice), marks paid.
	_, _, paid, err := p.Advance("bob", "2026-07-19")
	if err != nil {
		t.Fatalf("stage3 by bob: %v", err)
	}
	if !paid {
		t.Fatalf("stage3 completion should mark paid")
	}
}

func TestPaymentAdvance_SystemActorExempt(t *testing.T) {
	p := NewPayment("GPAY-2", "MS-2", "C-2", 1000, 0)
	for i := 0; i < len(PaymentStages); i++ {
		if _, _, _, err := p.Advance("system", "2026-07-19"); err != nil {
			t.Fatalf("system advance stage %d: %v", i, err)
		}
	}
}

func TestVariationAdvance_FourEyes(t *testing.T) {
	// Stage 0 auto-approved by the raiser (alice).
	v := NewVariation("GVAR-1", "C-1", "V1", "Extra works", 500, 5, "", "", "", "alice", "2026-07-19")
	if _, err := v.Advance("alice", "2026-07-19"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("variation stage1 by raiser alice: want ErrSelfSuccession, got %v", err)
	}
	if _, err := v.Advance("bob", "2026-07-19"); err != nil {
		t.Fatalf("variation stage1 by bob: %v", err)
	}
}

func TestRequisitionAdvance_FourEyes(t *testing.T) {
	rq := NewRequisition("GREQ-1", "REQ-1", "Buy stuff", 1000, "alice", "2026-07-19")
	if _, err := rq.Advance("alice", "2026-07-19"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("requisition stage1 by raiser alice: want ErrSelfSuccession, got %v", err)
	}
	if _, err := rq.Advance("bob", "2026-07-19"); err != nil {
		t.Fatalf("requisition stage1 by bob: %v", err)
	}
}
