package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// ---------------- Zone-works milestone profiles ----------------

const zwProfileCols = `milestone_id, contract_no, value, checklist, scope, deliverables, comments, created_at, updated_at`

func (s *GovStore) GetMilestoneProfile(ctx context.Context, milestoneID string) (*models.MilestoneProfile, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+zwProfileCols+` FROM zw_milestone_profiles WHERE milestone_id=$1`, milestoneID)
	p, err := scanMilestoneProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return p, err
}

func (s *GovStore) UpsertMilestoneProfile(ctx context.Context, p models.MilestoneProfile) (*models.MilestoneProfile, error) {
	chk, _ := jsonb(p.Checklist)
	scope, _ := jsonb(p.Scope)
	deliv, _ := jsonb(p.Deliverables)
	com, _ := jsonb(p.Comments)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO zw_milestone_profiles (milestone_id, contract_no, value, checklist, scope, deliverables, comments)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7::jsonb)
		ON CONFLICT (milestone_id) DO UPDATE SET contract_no=EXCLUDED.contract_no, value=EXCLUDED.value,
			checklist=EXCLUDED.checklist, scope=EXCLUDED.scope, deliverables=EXCLUDED.deliverables,
			comments=EXCLUDED.comments, updated_at=NOW()
		RETURNING `+zwProfileCols,
		p.MilestoneID, p.ContractNo, p.Value, chk, scope, deliv, com)
	return scanMilestoneProfile(row)
}

func scanMilestoneProfile(row pgx.Row) (*models.MilestoneProfile, error) {
	var p models.MilestoneProfile
	var chk, scope, deliv, com []byte
	if err := row.Scan(&p.MilestoneID, &p.ContractNo, &p.Value, &chk, &scope, &deliv, &com,
		&p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Checklist = []models.MilestoneCheckItem{}
	p.Scope = []models.MilestoneScopeTask{}
	p.Deliverables = []models.MilestoneDeliverable{}
	p.Comments = []models.MilestoneComment{}
	_ = json.Unmarshal(chk, &p.Checklist)
	_ = json.Unmarshal(scope, &p.Scope)
	_ = json.Unmarshal(deliv, &p.Deliverables)
	_ = json.Unmarshal(com, &p.Comments)
	return &p, nil
}

// ---------------- Zone-works execution trackers ----------------

const zwTrackerCols = `contract_no, stage, steps, execution_date, created_at, updated_at`

func (s *GovStore) GetExecutionTracker(ctx context.Context, contractNo string) (*models.ExecutionTracker, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+zwTrackerCols+` FROM zw_execution_trackers WHERE contract_no=$1`, contractNo)
	t, err := scanExecutionTracker(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return t, err
}

func (s *GovStore) UpsertExecutionTracker(ctx context.Context, t models.ExecutionTracker) (*models.ExecutionTracker, error) {
	steps, _ := jsonb(t.Steps)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO zw_execution_trackers (contract_no, stage, steps, execution_date)
		VALUES ($1,$2,$3::jsonb,$4)
		ON CONFLICT (contract_no) DO UPDATE SET stage=EXCLUDED.stage, steps=EXCLUDED.steps,
			execution_date=EXCLUDED.execution_date, updated_at=NOW()
		RETURNING `+zwTrackerCols,
		t.ContractNo, t.Stage, steps, t.ExecutionDate)
	return scanExecutionTracker(row)
}

func scanExecutionTracker(row pgx.Row) (*models.ExecutionTracker, error) {
	var t models.ExecutionTracker
	var steps []byte
	if err := row.Scan(&t.ContractNo, &t.Stage, &steps, &t.ExecutionDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	t.Steps = []string{}
	_ = json.Unmarshal(steps, &t.Steps)
	return &t, nil
}
