package models

import (
	"sort"
	"time"
)

// Monthly Report (MR) domain: the entities behind the Inspire Africa
// Construction Department monthly report. A Contractor is the normalized parent
// of many GovContract work-orders; a ProgressReport is a per-(contract, period)
// snapshot; a Valuation is a contractor-level IPC verification.

// ----- Contractors -----

type GovContractor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact,omitempty"`
	// PlatformUserID / UserEmail bind this contractor to a login so that user is
	// scoped to this contractor's contracts in the portal. Either matches.
	PlatformUserID string    `json:"platformUserId,omitempty"`
	UserEmail      string    `json:"userEmail,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type GovContractorInput struct {
	Name           string `json:"name"`
	Contact        string `json:"contact"`
	PlatformUserID string `json:"platformUserId"`
	UserEmail      string `json:"userEmail"`
}

type GovContractorPatch struct {
	Name           *string `json:"name,omitempty"`
	Contact        *string `json:"contact,omitempty"`
	PlatformUserID *string `json:"platformUserId,omitempty"`
	UserEmail      *string `json:"userEmail,omitempty"`
}

// ----- Progress reports (per contract, per period) -----

type ProgressReport struct {
	ID                 string    `json:"id"`
	ContractID         string    `json:"contractId"`
	Period             string    `json:"period"` // e.g. "2026-05"
	Progress           int       `json:"progress"`
	ExecutionStatus    string    `json:"executionStatus,omitempty"`
	CurrentActivity    string    `json:"currentActivity,omitempty"`
	Accomplishments    string    `json:"accomplishments,omitempty"`
	Challenges         string    `json:"challenges,omitempty"`
	Interventions      string    `json:"interventions,omitempty"`
	Responsible        string    `json:"responsible,omitempty"`
	TargetDate         string    `json:"targetDate,omitempty"`
	ProposedStart      string    `json:"proposedStart,omitempty"`
	ProposedCompletion string    `json:"proposedCompletion,omitempty"`
	Duration           string    `json:"duration,omitempty"`
	PlannedNext        string    `json:"plannedNext,omitempty"`
	PlannedProgress    int       `json:"plannedProgress"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ProgressReportInput struct {
	Period             string `json:"period"`
	Progress           int    `json:"progress"`
	ExecutionStatus    string `json:"executionStatus"`
	CurrentActivity    string `json:"currentActivity"`
	Accomplishments    string `json:"accomplishments"`
	Challenges         string `json:"challenges"`
	Interventions      string `json:"interventions"`
	Responsible        string `json:"responsible"`
	TargetDate         string `json:"targetDate"`
	ProposedStart      string `json:"proposedStart"`
	ProposedCompletion string `json:"proposedCompletion"`
	Duration           string `json:"duration"`
	PlannedNext        string `json:"plannedNext"`
	PlannedProgress    int    `json:"plannedProgress"`
}

// ----- Valuations (the "Contractors verified" sheet) -----

type Valuation struct {
	ID                       string    `json:"id"`
	ContractorID             string    `json:"contractorId,omitempty"`
	ContractorName           string    `json:"contractorName"`
	Period                   string    `json:"period,omitempty"`
	ContractSum              int64     `json:"contractSum"`
	AmountPaid               int64     `json:"amountPaid"`
	VerifiedValueOwed        int64     `json:"verifiedValueOwed"`
	ConsultantRecommendation int64     `json:"consultantRecommendation"`
	CEOApproval              int64     `json:"ceoApproval"`
	Remarks                  string    `json:"remarks,omitempty"`
	VerifiedDate             string    `json:"verifiedDate,omitempty"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type ValuationInput struct {
	ContractorID             string `json:"contractorId"`
	ContractorName           string `json:"contractorName"`
	Period                   string `json:"period"`
	ContractSum              int64  `json:"contractSum"`
	AmountPaid               int64  `json:"amountPaid"`
	VerifiedValueOwed        int64  `json:"verifiedValueOwed"`
	ConsultantRecommendation int64  `json:"consultantRecommendation"`
	CEOApproval              int64  `json:"ceoApproval"`
	Remarks                  string `json:"remarks"`
	VerifiedDate             string `json:"verifiedDate"`
}

// ----- Challenges register (report-level, period-scoped) -----

// Challenge is a cross-cutting issue from the monthly report: an issue category,
// who it affects, the recommended action, and the owner. Not tied to a single
// contract, so it is period-scoped rather than contract-scoped.
type Challenge struct {
	ID          string    `json:"id"`
	Period      string    `json:"period"`
	Seq         int       `json:"seq"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Affected    string    `json:"affected,omitempty"`
	Priority    string    `json:"priority"`
	Action      string    `json:"action,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ChallengeInput struct {
	Period      string `json:"period"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Affected    string `json:"affected"`
	Priority    string `json:"priority"`
	Action      string `json:"action"`
	Owner       string `json:"owner"`
}

// ----- Action-item tracker (report-level, period-scoped) -----

// ActionItem is a follow-up from the monthly report: a prioritized task with a
// responsible party, target date, and status.
type ActionItem struct {
	ID        string    `json:"id"`
	Period    string    `json:"period"`
	Seq       int       `json:"seq"`
	Priority  string    `json:"priority"`
	Text      string    `json:"text"`
	Party     string    `json:"party,omitempty"`
	Target    string    `json:"target,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ActionItemInput struct {
	Period   string `json:"period"`
	Priority string `json:"priority"`
	Text     string `json:"text"`
	Party    string `json:"party"`
	Target   string `json:"target"`
	Status   string `json:"status"`
}

// ----- Executive summary rollup (computed, not stored) -----

type ContractorSummary struct {
	Contractor  string `json:"contractor"`
	Total       int    `json:"total"`
	Completed   int    `json:"completed"`
	Ongoing     int    `json:"ongoing"`
	Halted      int    `json:"halted"`
	Paused      int    `json:"paused"`
	NotStarted  int    `json:"notStarted"`
	AvgProgress int    `json:"avgProgress"`
}

type MonthlySummary struct {
	Period         string              `json:"period"`
	TotalContracts int                 `json:"totalContracts"`
	Completed      int                 `json:"completed"`
	Ongoing        int                 `json:"ongoing"`
	Halted         int                 `json:"halted"`
	Paused         int                 `json:"paused"`
	NotStarted     int                 `json:"notStarted"`
	TotalValue     int64               `json:"totalValue"`
	TotalReceived  int64               `json:"totalReceived"`
	ByContractor   []ContractorSummary `json:"byContractor"`
}

// BuildMonthlySummary computes the Executive Summary rollup for a period. For
// each contract it prefers the period's progress report (execution status +
// progress %) when one exists, otherwise the contract's current operational
// fields. reports must already be filtered to the target period.
func BuildMonthlySummary(period string, contracts []GovContract, reports []ProgressReport) MonthlySummary {
	byContract := make(map[string]ProgressReport, len(reports))
	for _, r := range reports {
		byContract[r.ContractID] = r
	}

	sum := MonthlySummary{Period: period}
	groups := map[string]*ContractorSummary{}
	progressTotals := map[string]int{} // contractor -> summed progress (for averaging)

	for _, c := range contracts {
		status := c.ExecutionStatus
		progress := c.Progress
		if r, ok := byContract[c.ID]; ok {
			if r.ExecutionStatus != "" {
				status = NormalizeExecutionStatus(r.ExecutionStatus)
			}
			progress = r.Progress
		}
		if status == "" {
			status = ExecNotStarted
		}

		sum.TotalContracts++
		sum.TotalValue += c.Value
		sum.TotalReceived += c.Received

		name := c.Contractor
		if name == "" {
			name = "Unassigned"
		}
		g, ok := groups[name]
		if !ok {
			g = &ContractorSummary{Contractor: name}
			groups[name] = g
		}
		g.Total++
		progressTotals[name] += progress

		switch status {
		case ExecCompleted:
			sum.Completed++
			g.Completed++
		case ExecOngoing:
			sum.Ongoing++
			g.Ongoing++
		case ExecHalted:
			sum.Halted++
			g.Halted++
		case ExecPaused:
			sum.Paused++
			g.Paused++
		default:
			sum.NotStarted++
			g.NotStarted++
		}
	}

	sum.ByContractor = make([]ContractorSummary, 0, len(groups))
	for name, g := range groups {
		if g.Total > 0 {
			g.AvgProgress = progressTotals[name] / g.Total
		}
		sum.ByContractor = append(sum.ByContractor, *g)
	}
	sort.Slice(sum.ByContractor, func(i, j int) bool {
		return sum.ByContractor[i].Contractor < sum.ByContractor[j].Contractor
	})
	return sum
}
