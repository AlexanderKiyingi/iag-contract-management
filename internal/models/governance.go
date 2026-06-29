package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// Governance is the contract-lifecycle domain that backs the Contract
// Governance System UI (contracts, milestones, deliverables, inspections,
// completion reports). It is modeled separately from the legacy zone-works
// Contract so it can carry the full governance shape and 8-state lifecycle.

// GovStatus is the contract governance lifecycle state.
type GovStatus string

const (
	GovDraft       GovStatus = "Draft"
	GovUnderReview GovStatus = "Under Review"
	GovApproved    GovStatus = "Approved"
	GovActive      GovStatus = "Active"
	GovSuspended   GovStatus = "Suspended"
	GovCompleted   GovStatus = "Completed"
	GovClosed      GovStatus = "Closed"
	GovTerminated  GovStatus = "Terminated"
)

// govTransitions encodes the allowed lifecycle moves. Server-side enforcement
// is a key gap the client-only prototype lacks.
var govTransitions = map[GovStatus][]GovStatus{
	GovDraft:       {GovUnderReview, GovTerminated},
	GovUnderReview: {GovApproved, GovDraft, GovTerminated},
	GovApproved:    {GovActive, GovTerminated},
	GovActive:      {GovSuspended, GovCompleted, GovTerminated},
	GovSuspended:   {GovActive, GovTerminated},
	GovCompleted:   {GovClosed},
	GovClosed:      {},
	GovTerminated:  {},
}

// Valid reports whether s is a known status.
func (s GovStatus) Valid() bool {
	_, ok := govTransitions[s]
	return ok
}

// CanTransitionTo reports whether moving from s to next is allowed. A no-op
// (same status) is always allowed.
func (s GovStatus) CanTransitionTo(next GovStatus) bool {
	if s == next {
		return true
	}
	for _, allowed := range govTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// ErrInvalidTransition is returned when a status change violates the lifecycle.
var ErrInvalidTransition = errors.New("invalid contract status transition")

// ExecutionStatus is the operational execution state of a work-order, a
// SEPARATE axis from the GovStatus lifecycle. It mirrors the Status column of
// the Construction Department monthly-report Tracker sheet.
type ExecutionStatus string

const (
	ExecNotStarted ExecutionStatus = "Not Started"
	ExecOngoing    ExecutionStatus = "Ongoing"
	ExecPaused     ExecutionStatus = "Paused"
	ExecHalted     ExecutionStatus = "Halted"
	ExecCompleted  ExecutionStatus = "Completed"
)

var execStatuses = map[ExecutionStatus]bool{
	ExecNotStarted: true, ExecOngoing: true, ExecPaused: true,
	ExecHalted: true, ExecCompleted: true,
}

// Valid reports whether e is a known execution status.
func (e ExecutionStatus) Valid() bool { return execStatuses[e] }

// NormalizeExecutionStatus maps a free-form sheet status string to a canonical
// ExecutionStatus, defaulting to ExecNotStarted when unrecognized.
func NormalizeExecutionStatus(s string) ExecutionStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "completed", "complete", "done", "finished":
		return ExecCompleted
	case "ongoing", "active", "in progress", "in-progress":
		return ExecOngoing
	case "halted", "stopped":
		return ExecHalted
	case "paused", "on hold", "on-hold":
		return ExecPaused
	case "not started", "not-started", "pending", "":
		return ExecNotStarted
	default:
		return ExecNotStarted
	}
}

// MilestoneGovStatus values (free-form but documented for the UI).
const (
	MSPending      = "Pending"
	MSInProgress   = "In Progress"
	MSVerification = "Verification Requested"
	MSInspection   = "Under Inspection"
	MSApproved     = "Approved"
	MSRejected     = "Rejected"
	MSPaid         = "Paid"
	MSCompleted    = "Completed"
)

// ----- nested value objects -----

type GovDoc struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       string `json:"size,omitempty"`
	UploadedBy string `json:"uploadedBy,omitempty"`
	Date       string `json:"date,omitempty"`
	Version    int    `json:"version,omitempty"`
	Cat        string `json:"cat,omitempty"`
	// Key is the object-storage key when the document was uploaded to the S3
	// bucket (empty for legacy/manual entries). Download URLs are presigned
	// on demand from this key.
	Key string `json:"key,omitempty"`
}

type GovActivity struct {
	Date   string `json:"date"`
	Actor  string `json:"actor"`
	Action string `json:"action"`
}

type ScopeItem struct {
	Task   string `json:"task"`
	Desc   string `json:"desc,omitempty"`
	Target string `json:"target,omitempty"`
	Status string `json:"status,omitempty"`
}

type Deliverable struct {
	Name string `json:"name"`
	Done bool   `json:"done"`
}

type ChecklistItem struct {
	Item string `json:"item"`
	Done bool   `json:"done"`
	By   string `json:"by,omitempty"`
	Date string `json:"date,omitempty"`
}

type GovComment struct {
	Author string `json:"author"`
	Role   string `json:"role,omitempty"`
	Text   string `json:"text"`
	Date   string `json:"date,omitempty"`
}

type Inspection struct {
	Date           string `json:"date"`
	Inspector      string `json:"inspector"`
	Observations   string `json:"observations,omitempty"`
	Issues         string `json:"issues,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type CompletionReport struct {
	No   string `json:"no"`
	Date string `json:"date"`
	PM   string `json:"pm,omitempty"`
	Dept string `json:"dept,omitempty"`
}

// ----- aggregates -----

type GovMilestone struct {
	ID               string            `json:"id"`
	ContractID       string            `json:"contractId"`
	Name             string            `json:"name"`
	Value            int64             `json:"value"`
	TargetDate       string            `json:"targetDate,omitempty"`
	Status           string            `json:"status"`
	Scope            []ScopeItem       `json:"scope"`
	Deliverables     []Deliverable     `json:"deliverables"`
	Checklist        []ChecklistItem   `json:"checklist"`
	Docs             []GovDoc          `json:"docs"`
	Comments         []GovComment      `json:"comments"`
	Inspection       *Inspection       `json:"inspection,omitempty"`
	CompletionReport *CompletionReport `json:"completionReport,omitempty"`
	SortOrder        int               `json:"-"`
}

type GovContract struct {
	ID                string          `json:"id"`
	Number            string          `json:"number"`
	Name              string          `json:"name"`
	Contractor        string          `json:"contractor,omitempty"`
	ContractorID      string          `json:"contractorId,omitempty"`
	ContractorContact string          `json:"contractorContact,omitempty"`
	Type              string          `json:"type,omitempty"`
	StartDate         string          `json:"startDate,omitempty"`
	EndDate           string          `json:"endDate,omitempty"`
	Location          string          `json:"location,omitempty"`
	PM                string          `json:"pm,omitempty"`
	Department        string          `json:"department,omitempty"`
	Value             int64           `json:"value"`
	Retention         int             `json:"retention"`
	Status            GovStatus       `json:"status"`
	ExecutionStatus   ExecutionStatus `json:"executionStatus,omitempty"`
	Progress          int             `json:"progress"`
	Received          int64           `json:"received"`
	VariationTotal    int64           `json:"variationTotal"`
	PlannedCompletion string          `json:"plannedCompletion,omitempty"`
	Documents         []GovDoc        `json:"documents"`
	Activity          []GovActivity   `json:"activity"`
	Milestones        []GovMilestone  `json:"milestones,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// ----- inputs -----

type GovContractInput struct {
	Number            string          `json:"number"`
	Name              string          `json:"name"`
	Contractor        string          `json:"contractor"`
	ContractorID      string          `json:"contractorId"`
	ContractorContact string          `json:"contractorContact"`
	Type              string          `json:"type"`
	StartDate         string          `json:"startDate"`
	EndDate           string          `json:"endDate"`
	Location          string          `json:"location"`
	PM                string          `json:"pm"`
	Department        string          `json:"department"`
	Value             int64           `json:"value"`
	Retention         int             `json:"retention"`
	Status            GovStatus       `json:"status"`
	ExecutionStatus   ExecutionStatus `json:"executionStatus"`
	Progress          int             `json:"progress"`
	Received          int64           `json:"received"`
	VariationTotal    int64           `json:"variationTotal"`
	PlannedCompletion string          `json:"plannedCompletion"`
	Documents         []GovDoc        `json:"documents"`
}

type GovContractPatch struct {
	Name              *string          `json:"name,omitempty"`
	Contractor        *string          `json:"contractor,omitempty"`
	ContractorID      *string          `json:"contractorId,omitempty"`
	ContractorContact *string          `json:"contractorContact,omitempty"`
	Type              *string          `json:"type,omitempty"`
	StartDate         *string          `json:"startDate,omitempty"`
	EndDate           *string          `json:"endDate,omitempty"`
	Location          *string          `json:"location,omitempty"`
	PM                *string          `json:"pm,omitempty"`
	Department        *string          `json:"department,omitempty"`
	Value             *int64           `json:"value,omitempty"`
	Retention         *int             `json:"retention,omitempty"`
	Status            *GovStatus       `json:"status,omitempty"`
	ExecutionStatus   *ExecutionStatus `json:"executionStatus,omitempty"`
	Progress          *int             `json:"progress,omitempty"`
	Received          *int64           `json:"received,omitempty"`
	VariationTotal    *int64           `json:"variationTotal,omitempty"`
	PlannedCompletion *string          `json:"plannedCompletion,omitempty"`
	Documents         *[]GovDoc        `json:"documents,omitempty"`
}

type GovMilestoneInput struct {
	Name         string          `json:"name"`
	Value        int64           `json:"value"`
	TargetDate   string          `json:"targetDate"`
	Status       string          `json:"status"`
	Scope        []ScopeItem     `json:"scope"`
	Deliverables []Deliverable   `json:"deliverables"`
	Checklist    []ChecklistItem `json:"checklist"`
	Docs         []GovDoc        `json:"docs"`
	Comments     []GovComment    `json:"comments"`
}

type GovMilestonePatch struct {
	Name             *string           `json:"name,omitempty"`
	Value            *int64            `json:"value,omitempty"`
	TargetDate       *string           `json:"targetDate,omitempty"`
	Status           *string           `json:"status,omitempty"`
	Scope            *[]ScopeItem      `json:"scope,omitempty"`
	Deliverables     *[]Deliverable    `json:"deliverables,omitempty"`
	Checklist        *[]ChecklistItem  `json:"checklist,omitempty"`
	Docs             *[]GovDoc         `json:"docs,omitempty"`
	Comments         *[]GovComment     `json:"comments,omitempty"`
	Inspection       *Inspection       `json:"inspection,omitempty"`
	CompletionReport *CompletionReport `json:"completionReport,omitempty"`
}

// NewGovID mints a prefixed random id (e.g. GCT-9F3A1B).
func NewGovID(prefix string) string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return prefix + "-" + strings.ToUpper(hex.EncodeToString(b[:]))
}
