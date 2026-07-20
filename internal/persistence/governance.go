package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// NextContractNumber returns the next auto-generated governance contract number
// of the form CT-<year>-NNNN, based on the highest existing numeric suffix among
// contracts already using that year's prefix. The number column is UNIQUE, so a
// rare concurrent collision just fails the insert (the caller can retry).
func (s *GovStore) NextContractNumber(ctx context.Context) (string, error) {
	prefix := fmt.Sprintf("CT-%d-", time.Now().UTC().Year())
	var maxN int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(CAST(substring(number from '[0-9]+$') AS INTEGER)), 0)
		FROM gov_contracts WHERE number LIKE $1`, prefix+"%").Scan(&maxN)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, maxN+1), nil
}

// NextRequisitionNo returns the next auto-generated requisition number of the
// form REQ-<year>-NNNN.
func (s *GovStore) NextRequisitionNo(ctx context.Context) (string, error) {
	prefix := fmt.Sprintf("REQ-%d-", time.Now().UTC().Year())
	var maxN int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(CAST(substring(no from '[0-9]+$') AS INTEGER)), 0)
		FROM gov_requisitions WHERE no LIKE $1`, prefix+"%").Scan(&maxN)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, maxN+1), nil
}

// ErrGovNotFound is returned when a governance contract/milestone is absent.
var ErrGovNotFound = errors.New("not found")

// GovStore persists the contract-governance domain (gov_contracts,
// gov_milestones). Nested value objects are stored as JSONB so the rich UI
// shape round-trips without a table per sub-collection.
type GovStore struct {
	pool *pgxpool.Pool
}

func NewGovStore(pool *pgxpool.Pool) *GovStore { return &GovStore{pool: pool} }

func jsonb(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

// ----- Contracts -----

// govContractCols is the canonical gov_contracts column list, shared by the
// list/get SELECTs and the create/update RETURNING clauses so the column order
// stays in lockstep with scanGovContract.
const govContractCols = `id, number, name, contractor, contractor_id, contractor_contact, type,
	start_date, end_date, location, pm, department, value, retention, status,
	execution_status, progress, received, variation_total, planned_completion,
	documents, activity, created_at, updated_at, pm_project_id, conversation_id`

func (s *GovStore) ListContracts(ctx context.Context) ([]models.GovContract, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+govContractCols+` FROM gov_contracts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovContract{}
	for rows.Next() {
		c, err := scanGovContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// GetContract resolves by id or contract number and loads its milestones.
func (s *GovStore) GetContract(ctx context.Context, idOrNumber string) (*models.GovContract, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+govContractCols+` FROM gov_contracts WHERE id = $1 OR number = $1`, idOrNumber)
	c, err := scanGovContract(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGovNotFound
		}
		return nil, err
	}
	ms, err := s.ListMilestones(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Milestones = ms
	return c, nil
}

func (s *GovStore) CreateContract(ctx context.Context, c models.GovContract) (*models.GovContract, error) {
	docs, _ := jsonb(c.Documents)
	act, _ := jsonb(c.Activity)
	exec := string(c.ExecutionStatus)
	if exec == "" {
		exec = string(models.ExecNotStarted)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_contracts
			(id, number, name, contractor, contractor_id, contractor_contact, type, start_date, end_date,
			 location, pm, department, value, retention, status, execution_status, progress, received,
			 variation_total, planned_completion, documents, activity, pm_project_id, conversation_id)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),NULLIF($9,''),$10,$11,$12,$13,$14,$15,$16,$17,$18,
		        $19,NULLIF($20,''),$21::jsonb,$22::jsonb,$23,$24)
		RETURNING `+govContractCols,
		c.ID, c.Number, c.Name, c.Contractor, c.ContractorID, c.ContractorContact, c.Type, c.StartDate, c.EndDate,
		c.Location, c.PM, c.Department, c.Value, c.Retention, string(c.Status), exec, c.Progress, c.Received,
		c.VariationTotal, c.PlannedCompletion, docs, act, c.PMProjectID, c.ConversationID)
	return scanGovContract(row)
}

// UpdateContract writes the full contract row (the controller computes the
// merged value from a patch + transition check).
func (s *GovStore) UpdateContract(ctx context.Context, c models.GovContract) (*models.GovContract, error) {
	docs, _ := jsonb(c.Documents)
	act, _ := jsonb(c.Activity)
	exec := string(c.ExecutionStatus)
	if exec == "" {
		exec = string(models.ExecNotStarted)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_contracts SET
			name=$2, contractor=$3, contractor_id=NULLIF($4,''), contractor_contact=$5, type=$6,
			start_date=NULLIF($7,''), end_date=NULLIF($8,''), location=$9, pm=$10, department=$11,
			value=$12, retention=$13, status=$14, execution_status=$15, progress=$16, received=$17,
			variation_total=$18, planned_completion=NULLIF($19,''), documents=$20::jsonb, activity=$21::jsonb,
			pm_project_id=$22, conversation_id=$23, updated_at=NOW()
		WHERE id=$1
		RETURNING `+govContractCols,
		c.ID, c.Name, c.Contractor, c.ContractorID, c.ContractorContact, c.Type, c.StartDate, c.EndDate,
		c.Location, c.PM, c.Department, c.Value, c.Retention, string(c.Status), exec, c.Progress, c.Received,
		c.VariationTotal, c.PlannedCompletion, docs, act, c.PMProjectID, c.ConversationID)
	cc, err := scanGovContract(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return cc, err
}

// SetContractConversationID records the chat thread id, but only if one is not
// already set (so a re-run never clobbers an existing thread).
func (s *GovStore) SetContractConversationID(ctx context.Context, id, convID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE gov_contracts SET conversation_id=$2, updated_at=NOW() WHERE id=$1 AND conversation_id=''`,
		id, convID)
	return err
}

func (s *GovStore) DeleteContract(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_contracts WHERE id = $1 OR number = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// GovCounts holds the server-computed nav badge counts for the governance UI.
type GovCounts struct {
	Contracts    int `json:"contracts"`
	Requisitions int `json:"requisitions"` // pending approval
	Approvals    int `json:"approvals"`    // pending across payments + variations + requisitions
	Obligations  int `json:"obligations"`  // open (not Met/Waived)
	Milestones   int `json:"milestones"`   // total
	Payments     int `json:"payments"`     // in-flight (not Paid)
	Variations   int `json:"variations"`   // pending
}

// Counts computes all governance nav badge numbers in a single query round-trip.
// "Attention" counts (requisitions/payments/variations/obligations) reflect
// items still in flight; contracts/milestones are totals.
func (s *GovStore) Counts(ctx context.Context) (GovCounts, error) {
	var c GovCounts
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM gov_contracts),
			(SELECT COUNT(*) FROM gov_requisitions WHERE status = 'Pending Approval'),
			(SELECT COUNT(*) FROM gov_obligations WHERE status NOT IN ('Met','Waived')),
			(SELECT COUNT(*) FROM gov_milestones),
			(SELECT COUNT(*) FROM gov_payments WHERE status <> 'Paid'),
			(SELECT COUNT(*) FROM gov_variations WHERE status = 'Pending')`,
	).Scan(&c.Contracts, &c.Requisitions, &c.Obligations, &c.Milestones, &c.Payments, &c.Variations)
	if err != nil {
		return c, err
	}
	// Approvals badge = everything awaiting a human decision.
	c.Approvals = c.Requisitions + c.Payments + c.Variations
	return c, nil
}

// ----- Milestones -----

func (s *GovStore) ListMilestones(ctx context.Context, contractID string) ([]models.GovMilestone, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
		       docs, comments, inspection, completion_report, sort_order
		FROM gov_milestones WHERE contract_id = $1 ORDER BY sort_order, created_at`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovMilestone{}
	for rows.Next() {
		m, err := scanGovMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// ListAllMilestones returns every milestone across all contracts (each carries
// its contractId), so the portfolio Milestones page and nav counts need one
// round-trip instead of a per-contract fan-out.
func (s *GovStore) ListAllMilestones(ctx context.Context) ([]models.GovMilestone, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
		       docs, comments, inspection, completion_report, sort_order
		FROM gov_milestones ORDER BY contract_id, sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovMilestone{}
	for rows.Next() {
		m, err := scanGovMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *GovStore) GetMilestone(ctx context.Context, id string) (*models.GovMilestone, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
		       docs, comments, inspection, completion_report, sort_order
		FROM gov_milestones WHERE id = $1`, id)
	m, err := scanGovMilestone(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return m, err
}

func (s *GovStore) CreateMilestone(ctx context.Context, m models.GovMilestone) (*models.GovMilestone, error) {
	scope, _ := jsonb(m.Scope)
	deliv, _ := jsonb(m.Deliverables)
	chk, _ := jsonb(m.Checklist)
	docs, _ := jsonb(m.Docs)
	com, _ := jsonb(m.Comments)
	insp, _ := jsonb(m.Inspection)
	cr, _ := jsonb(m.CompletionReport)
	// Append to the end of the contract's milestone list.
	var nextOrder int
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order)+1,0) FROM gov_milestones WHERE contract_id=$1`, m.ContractID).Scan(&nextOrder)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_milestones
			(id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
			 docs, comments, inspection, completion_report, sort_order)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7::jsonb,$8::jsonb,$9::jsonb,$10::jsonb,$11::jsonb,$12::jsonb,$13::jsonb,$14)
		RETURNING id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
		          docs, comments, inspection, completion_report, sort_order`,
		m.ID, m.ContractID, m.Name, m.Value, m.TargetDate, m.Status, scope, deliv, chk, docs, com, insp, cr, nextOrder)
	return scanGovMilestone(row)
}

func (s *GovStore) UpdateMilestone(ctx context.Context, m models.GovMilestone) (*models.GovMilestone, error) {
	scope, _ := jsonb(m.Scope)
	deliv, _ := jsonb(m.Deliverables)
	chk, _ := jsonb(m.Checklist)
	docs, _ := jsonb(m.Docs)
	com, _ := jsonb(m.Comments)
	insp, _ := jsonb(m.Inspection)
	cr, _ := jsonb(m.CompletionReport)
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_milestones SET
			name=$2, value=$3, target_date=NULLIF($4,''), status=$5, scope=$6::jsonb,
			deliverables=$7::jsonb, checklist=$8::jsonb, docs=$9::jsonb, comments=$10::jsonb,
			inspection=$11::jsonb, completion_report=$12::jsonb, updated_at=NOW()
		WHERE id=$1
		RETURNING id, contract_id, name, value, target_date, status, scope, deliverables, checklist,
		          docs, comments, inspection, completion_report, sort_order`,
		m.ID, m.Name, m.Value, m.TargetDate, m.Status, scope, deliv, chk, docs, com, insp, cr)
	mm, err := scanGovMilestone(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return mm, err
}

func (s *GovStore) DeleteMilestone(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_milestones WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// ----- scanners -----

func scanGovContract(row pgx.Row) (*models.GovContract, error) {
	var c models.GovContract
	var status, execStatus string
	var contractorID, startDate, endDate, plannedCompletion *string
	var docs, act []byte
	var created, updated time.Time
	if err := row.Scan(&c.ID, &c.Number, &c.Name, &c.Contractor, &contractorID, &c.ContractorContact, &c.Type,
		&startDate, &endDate, &c.Location, &c.PM, &c.Department, &c.Value, &c.Retention, &status,
		&execStatus, &c.Progress, &c.Received, &c.VariationTotal, &plannedCompletion,
		&docs, &act, &created, &updated, &c.PMProjectID, &c.ConversationID); err != nil {
		return nil, err
	}
	c.Status = models.GovStatus(status)
	c.ExecutionStatus = models.ExecutionStatus(execStatus)
	if contractorID != nil {
		c.ContractorID = *contractorID
	}
	if startDate != nil {
		c.StartDate = *startDate
	}
	if endDate != nil {
		c.EndDate = *endDate
	}
	if plannedCompletion != nil {
		c.PlannedCompletion = *plannedCompletion
	}
	c.Documents = []models.GovDoc{}
	c.Activity = []models.GovActivity{}
	_ = json.Unmarshal(docs, &c.Documents)
	_ = json.Unmarshal(act, &c.Activity)
	c.CreatedAt, c.UpdatedAt = created, updated
	return &c, nil
}

func scanGovMilestone(row pgx.Row) (*models.GovMilestone, error) {
	var m models.GovMilestone
	var targetDate *string
	var scope, deliv, chk, docs, com, insp, cr []byte
	if err := row.Scan(&m.ID, &m.ContractID, &m.Name, &m.Value, &targetDate, &m.Status,
		&scope, &deliv, &chk, &docs, &com, &insp, &cr, &m.SortOrder); err != nil {
		return nil, err
	}
	if targetDate != nil {
		m.TargetDate = *targetDate
	}
	m.Scope = []models.ScopeItem{}
	m.Deliverables = []models.Deliverable{}
	m.Checklist = []models.ChecklistItem{}
	m.Docs = []models.GovDoc{}
	m.Comments = []models.GovComment{}
	_ = json.Unmarshal(scope, &m.Scope)
	_ = json.Unmarshal(deliv, &m.Deliverables)
	_ = json.Unmarshal(chk, &m.Checklist)
	_ = json.Unmarshal(docs, &m.Docs)
	_ = json.Unmarshal(com, &m.Comments)
	if len(insp) > 0 && string(insp) != "null" {
		_ = json.Unmarshal(insp, &m.Inspection)
	}
	if len(cr) > 0 && string(cr) != "null" {
		_ = json.Unmarshal(cr, &m.CompletionReport)
	}
	return &m, nil
}
