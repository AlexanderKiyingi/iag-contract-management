package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

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

func (s *GovStore) ListContracts(ctx context.Context) ([]models.GovContract, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, number, name, contractor, contractor_contact, type, start_date, end_date,
		       location, pm, department, value, retention, status, documents, activity,
		       created_at, updated_at
		FROM gov_contracts ORDER BY created_at DESC`)
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
	row := s.pool.QueryRow(ctx, `
		SELECT id, number, name, contractor, contractor_contact, type, start_date, end_date,
		       location, pm, department, value, retention, status, documents, activity,
		       created_at, updated_at
		FROM gov_contracts WHERE id = $1 OR number = $1`, idOrNumber)
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
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_contracts
			(id, number, name, contractor, contractor_contact, type, start_date, end_date,
			 location, pm, department, value, retention, status, documents, activity)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb)
		RETURNING id, number, name, contractor, contractor_contact, type, start_date, end_date,
		          location, pm, department, value, retention, status, documents, activity,
		          created_at, updated_at`,
		c.ID, c.Number, c.Name, c.Contractor, c.ContractorContact, c.Type, c.StartDate, c.EndDate,
		c.Location, c.PM, c.Department, c.Value, c.Retention, string(c.Status), docs, act)
	return scanGovContract(row)
}

// UpdateContract writes the full contract row (the controller computes the
// merged value from a patch + transition check).
func (s *GovStore) UpdateContract(ctx context.Context, c models.GovContract) (*models.GovContract, error) {
	docs, _ := jsonb(c.Documents)
	act, _ := jsonb(c.Activity)
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_contracts SET
			name=$2, contractor=$3, contractor_contact=$4, type=$5,
			start_date=NULLIF($6,''), end_date=NULLIF($7,''), location=$8, pm=$9, department=$10,
			value=$11, retention=$12, status=$13, documents=$14::jsonb, activity=$15::jsonb, updated_at=NOW()
		WHERE id=$1
		RETURNING id, number, name, contractor, contractor_contact, type, start_date, end_date,
		          location, pm, department, value, retention, status, documents, activity,
		          created_at, updated_at`,
		c.ID, c.Name, c.Contractor, c.ContractorContact, c.Type, c.StartDate, c.EndDate,
		c.Location, c.PM, c.Department, c.Value, c.Retention, string(c.Status), docs, act)
	cc, err := scanGovContract(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return cc, err
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
	var status string
	var startDate, endDate *string
	var docs, act []byte
	var created, updated time.Time
	if err := row.Scan(&c.ID, &c.Number, &c.Name, &c.Contractor, &c.ContractorContact, &c.Type,
		&startDate, &endDate, &c.Location, &c.PM, &c.Department, &c.Value, &c.Retention, &status,
		&docs, &act, &created, &updated); err != nil {
		return nil, err
	}
	c.Status = models.GovStatus(status)
	if startDate != nil {
		c.StartDate = *startDate
	}
	if endDate != nil {
		c.EndDate = *endDate
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
