package models

import (
	"errors"
	"testing"
)

// These cover the seams of running the governance workflows on the shared
// approval engine — the places where reconstructing engine state from the
// stored wire format could quietly change behaviour that the pre-existing
// workflow tests would not notice.

// A human clears a stage, an unattributed actor clears the next, then the same
// human clears the one after.
//
// The rule is about the immediately previous stage. An unattributed stage
// breaks the succession, so the human is allowed. Reconstructing history
// without its blank entries reached past that stage and blocked them.
func TestSuccessionLooksOnlyAtTheImmediatelyPreviousStage(t *testing.T) {
	p := NewPayment("P", "M", "C", 1000, 0)
	if _, _, _, err := p.Advance("alice", "d"); err != nil {
		t.Fatalf("stage 0 by alice: %v", err)
	}
	if _, _, _, err := p.Advance("", "d"); err != nil {
		t.Fatalf("stage 1 by an unattributed actor: %v", err)
	}
	if _, _, _, err := p.Advance("alice", "d"); err != nil {
		t.Fatalf("stage 2 by alice: %v — an unattributed stage breaks the succession", err)
	}
}

// The mirror of the above: with no blank between them, the same human is
// blocked. Together these pin the rule to one stage of look-back.
func TestSuccessionBlocksTheImmediatelyPreviousApprover(t *testing.T) {
	p := NewPayment("P", "M", "C", 1000, 0)
	if _, _, _, err := p.Advance("alice", "d"); err != nil {
		t.Fatalf("stage 0: %v", err)
	}
	if _, _, _, err := p.Advance("alice", "d"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("stage 1 by alice again: %v, want ErrSelfSuccession", err)
	}
}

// A variation records its first stage against the raiser, so the raiser is the
// previous approver on the second stage and must not clear it.
func TestVariationRaiserCannotClearTheSecondStage(t *testing.T) {
	v := NewVariation("V", "C", "VO-1", "T", 100, 0, "", "", "", "alice", "d")
	if _, err := v.Advance("alice", "d"); !errors.Is(err, ErrSelfSuccession) {
		t.Fatalf("raiser advancing their own variation: %v, want ErrSelfSuccession", err)
	}
	if _, err := v.Advance("bob", "d"); err != nil {
		t.Fatalf("a different approver: %v", err)
	}
}

// Every chain the package routes must exist in the registry: govState falls
// back to an empty state on an unknown key, which would surface as a confusing
// "workflow already complete" rather than a missing chain.
func TestEveryGovernanceChainIsRegistered(t *testing.T) {
	for key, stages := range map[string][]string{
		"gov.payment":     PaymentStages,
		"gov.variation":   VariationStages,
		"gov.requisition": RequisitionStages,
	} {
		chain, ok := governanceChains.Get(key)
		if !ok {
			t.Fatalf("chain %q is not registered", key)
		}
		if len(chain.Desks) != len(stages) {
			t.Errorf("chain %q has %d desks, want %d — one per stage",
				key, len(chain.Desks), len(stages))
		}
		for i, d := range chain.Desks {
			if string(d.Key) != stages[i] {
				t.Errorf("chain %q desk %d = %q, want %q — desk keys are the stage labels",
					key, i, d.Key, stages[i])
			}
		}
		if !chain.NoSuccessiveApprover {
			t.Errorf("chain %q does not enforce segregation of duties", key)
		}
	}
}

// govState must place the cursor on the stage the stored workflow is actually
// waiting on, including at both ends of the walk.
func TestGovStateTracksTheStoredStage(t *testing.T) {
	steps := make([]stepRecord, len(PaymentStages))
	for i, s := range PaymentStages {
		steps[i] = stepRecord{Step: s}
	}

	for stage := 0; stage < len(PaymentStages); stage++ {
		s := govState("gov.payment", "", stage, steps)
		if got := int(s.StageIndex); got != stage {
			t.Errorf("stage %d: stageIndex = %d", stage, got)
		}
		if string(s.Desk) != PaymentStages[stage] {
			t.Errorf("stage %d: desk = %q, want %q", stage, s.Desk, PaymentStages[stage])
		}
	}

	// Past the end the workflow is complete and accepts no further advance.
	if _, _, err := govAdvance("gov.payment", "", len(PaymentStages), steps, "alice"); !errors.Is(err, ErrWorkflowComplete) {
		t.Errorf("advancing past the last stage: %v, want ErrWorkflowComplete", err)
	}
}
