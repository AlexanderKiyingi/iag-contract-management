package models

import "testing"

func TestGovStatusValid(t *testing.T) {
	for _, s := range []GovStatus{GovDraft, GovActive, GovClosed, GovTerminated} {
		if !s.Valid() {
			t.Fatalf("%q should be valid", s)
		}
	}
	if GovStatus("Bogus").Valid() {
		t.Fatal("Bogus should be invalid")
	}
}

func TestGovTransitions(t *testing.T) {
	allowed := [][2]GovStatus{
		{GovDraft, GovUnderReview}, {GovUnderReview, GovApproved}, {GovApproved, GovActive},
		{GovActive, GovSuspended}, {GovSuspended, GovActive}, {GovActive, GovCompleted},
		{GovCompleted, GovClosed}, {GovActive, GovTerminated},
	}
	for _, tc := range allowed {
		if !tc[0].CanTransitionTo(tc[1]) {
			t.Errorf("expected %s → %s allowed", tc[0], tc[1])
		}
	}
	denied := [][2]GovStatus{
		{GovDraft, GovActive},      // must be reviewed/approved first
		{GovClosed, GovActive},     // terminal
		{GovTerminated, GovActive}, // terminal
		{GovCompleted, GovActive},  // only → Closed
		{GovActive, GovApproved},   // no going back
	}
	for _, tc := range denied {
		if tc[0].CanTransitionTo(tc[1]) {
			t.Errorf("expected %s → %s denied", tc[0], tc[1])
		}
	}
	if !GovActive.CanTransitionTo(GovActive) {
		t.Error("no-op transition should be allowed")
	}
}
