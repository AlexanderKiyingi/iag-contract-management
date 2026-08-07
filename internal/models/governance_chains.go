package models

import (
	"errors"
	"strings"

	"github.com/alvor-technologies/iag-platform-go/approvalchain"
)

// Governance workflows run on the shared platform approval engine rather than
// on stage arrays maintained here.
//
// The engine owns sequencing, segregation of duties and the audit trail; this
// file owns only the two chain definitions and the projection back onto the
// wire format the Contracts UI already consumes. Keeping GovPayment.Advance and
// GovVariation.Advance signature-compatible means controllers, persistence and
// the frontend are untouched by the move.
//
// Both chains set NoSuccessiveApprover: the control these workflows have always
// enforced is that whoever cleared the previous stage cannot clear the next,
// which is stronger than four-eyes against the raiser alone.
//
// Desks accept any role. Authorization in this service is by permission —
// governance.update on the controller — not by role name, and inventing role
// patterns here would silently narrow who can approve.
const anyRole = `.+`

var governanceChains = approvalchain.MustRegistry(
	approvalchain.Chain{
		Key:                  "gov.payment",
		Label:                "PM Approval → Finance Review → Payment Authorization → Paid",
		TerminalLabel:        "Paid",
		NoSuccessiveApprover: true,
		Desks:                desksFor(PaymentStages),
	},
	approvalchain.Chain{
		Key:                  "gov.variation",
		Label:                "Project Manager → Department Head → Procurement → Management",
		TerminalLabel:        "Approved",
		NoSuccessiveApprover: true,
		Desks:                desksFor(VariationStages),
	},
	approvalchain.Chain{
		Key:                  "gov.requisition",
		Label:                requisitionChainLabel(),
		TerminalLabel:        "Approved",
		NoSuccessiveApprover: true,
		Desks:                desksFor(RequisitionStages),
	},
)

func requisitionChainLabel() string { return strings.Join(RequisitionStages, " → ") }

var governanceEngine = approvalchain.NewEngine(governanceChains)

// desksFor turns an ordered stage list into desks. The stage label is the desk
// key as well as its name, so the wire format's step strings and the engine's
// desk keys cannot drift apart.
func desksFor(stages []string) []approvalchain.Desk {
	out := make([]approvalchain.Desk, 0, len(stages))
	for _, s := range stages {
		out = append(out, approvalchain.Desk{
			Key:          approvalchain.DeskKey(s),
			Label:        s,
			RolePatterns: []string{anyRole},
			// The raiser clears the first stage of a variation themselves, and
			// these workflows have never restricted a stage to a non-raiser.
			// NoSuccessiveApprover is the control that matters here.
			AllowRequester: true,
		})
	}
	return out
}

// govState rebuilds engine state from a stored workflow: the stage cursor plus
// whatever history has been recorded. Persistence keeps the projected form, so
// the engine's state is derived on each call rather than stored twice.
func govState(chainKey, raisedBy string, stage int, steps []stepRecord) approvalchain.State {
	s := approvalchain.New(chainKey, raisedBy, approvalchain.Options{})
	chain, ok := governanceChains.Get(chainKey)
	if !ok {
		// Unreachable with the three literal keys this package uses, but a
		// missing chain would otherwise nil-deref below.
		return s
	}
	if stage <= 0 {
		s.Status = approvalchain.StatusInFlight
		s.StageIndex = 0
		if len(chain.Desks) > 0 {
			s.Desk = chain.Desks[0].Key
		}
	} else if stage >= len(chain.Desks) {
		s.Status = approvalchain.StatusApproved
		s.StageIndex = len(chain.Desks)
	} else {
		s.Status = approvalchain.StatusInFlight
		s.StageIndex = stage
		s.Desk = chain.Desks[stage].Key
	}

	// Only the stages already completed, and blanks among them kept.
	//
	// Both halves matter. Including the trailing unfilled stages would put an
	// empty actor at the end of history, and the successive-approver rule reads
	// backwards from there — it would find a blank every time and never fire.
	// Dropping blanks among the completed stages is the opposite error: the rule
	// would reach past an unattributed stage to the human before it, and block
	// someone the previous hand-rolled check allowed.
	done := stage
	if done > len(steps) {
		done = len(steps)
	}
	if done < 0 {
		done = 0
	}
	for _, st := range steps[:done] {
		s.History = append(s.History, approvalchain.Step{
			Desk:      approvalchain.DeskKey(st.Step),
			DeskLabel: st.Step,
			Action:    approvalchain.ActionAdvance,
			Actor:     st.By,
		})
	}
	return s
}

// stepRecord is the shape both PaymentStep and VariationApproval share.
type stepRecord struct {
	Step string
	By   string
}

func paymentSteps(in []PaymentStep) []stepRecord {
	out := make([]stepRecord, 0, len(in))
	for _, s := range in {
		out = append(out, stepRecord{Step: s.Step, By: s.By})
	}
	return out
}

func variationSteps(in []VariationApproval) []stepRecord {
	out := make([]stepRecord, 0, len(in))
	for _, s := range in {
		out = append(out, stepRecord{Step: s.Step, By: s.By})
	}
	return out
}

// govAdvance runs one engine advance, returning the stage index it completed
// and whether that completion finished the chain.
func govAdvance(chainKey, raisedBy string, stage int, steps []stepRecord, by string) (completed int, done bool, err error) {
	s := govState(chainKey, raisedBy, stage, steps)
	if s.Status != approvalchain.StatusInFlight {
		return -1, false, ErrWorkflowComplete
	}
	out, err := governanceEngine.Advance(s, approvalchain.Actor{ID: by, Roles: []string{"actor"}}, "")
	if err != nil {
		return -1, false, translateGovErr(err)
	}
	return s.StageIndex, out.Status == approvalchain.StatusApproved, nil
}

// translateGovErr maps engine errors onto the sentinels this package has always
// returned, so callers and their tests keep working.
func translateGovErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, approvalchain.ErrSelfSuccession):
		return ErrSelfSuccession
	default:
		return ErrWorkflowComplete
	}
}
