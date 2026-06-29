package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// Monthly-report persistence: contractors, per-(contract, period) progress
// reports, and contractor-level IPC valuations. Plain-column tables (no JSONB),
// so they follow the simple scan pattern used by obligations/budgets.

// ---------------- Contractors ----------------

const contractorCols = `id, name, contact, COALESCE(platform_user_id,'') AS platform_user_id, created_at, updated_at`

func (s *GovStore) ListContractors(ctx context.Context) ([]models.GovContractor, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+contractorCols+` FROM gov_contractors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovContractor{}
	for rows.Next() {
		c, err := scanContractor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *GovStore) GetContractor(ctx context.Context, id string) (*models.GovContractor, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+contractorCols+` FROM gov_contractors WHERE id=$1 OR name=$1`, id)
	c, err := scanContractor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return c, err
}

func (s *GovStore) CreateContractor(ctx context.Context, c models.GovContractor) (*models.GovContractor, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_contractors (id, name, contact, platform_user_id)
		VALUES ($1,$2,$3,NULLIF($4,'')) RETURNING `+contractorCols,
		c.ID, c.Name, c.Contact, c.PlatformUserID)
	return scanContractor(row)
}

// UpsertContractorByName inserts a contractor or returns the existing one with
// the same name. Used by the importer to dedupe the workbook's contractor names.
func (s *GovStore) UpsertContractorByName(ctx context.Context, c models.GovContractor) (*models.GovContractor, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_contractors (id, name, contact)
		VALUES ($1,$2,$3)
		ON CONFLICT (name) DO UPDATE SET
			contact = CASE WHEN EXCLUDED.contact <> '' THEN EXCLUDED.contact ELSE gov_contractors.contact END,
			updated_at = NOW()
		RETURNING `+contractorCols,
		c.ID, c.Name, c.Contact)
	return scanContractor(row)
}

func (s *GovStore) UpdateContractor(ctx context.Context, c models.GovContractor) (*models.GovContractor, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_contractors SET name=$2, contact=$3, platform_user_id=NULLIF($4,''), updated_at=NOW()
		WHERE id=$1 RETURNING `+contractorCols,
		c.ID, c.Name, c.Contact, c.PlatformUserID)
	cc, err := scanContractor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return cc, err
}

func (s *GovStore) DeleteContractor(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_contractors WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanContractor(row pgx.Row) (*models.GovContractor, error) {
	var c models.GovContractor
	if err := row.Scan(&c.ID, &c.Name, &c.Contact, &c.PlatformUserID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// GetContractorByUserID resolves the contractor a platform user is bound to.
func (s *GovStore) GetContractorByUserID(ctx context.Context, userID string) (*models.GovContractor, error) {
	if userID == "" {
		return nil, ErrGovNotFound
	}
	row := s.pool.QueryRow(ctx,
		`SELECT `+contractorCols+` FROM gov_contractors WHERE platform_user_id=$1`, userID)
	c, err := scanContractor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return c, err
}

// ---------------- Progress reports ----------------

const progressCols = `id, contract_id, period, progress, execution_status, current_activity,
	accomplishments, challenges, interventions, responsible, target_date, proposed_start,
	proposed_completion, duration, planned_next, planned_progress, created_at, updated_at`

func (s *GovStore) ListProgressReports(ctx context.Context, contractID string) ([]models.ProgressReport, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+progressCols+` FROM gov_progress_reports WHERE contract_id=$1 ORDER BY period DESC`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ProgressReport{}
	for rows.Next() {
		r, err := scanProgressReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ListProgressReportsByPeriod returns every contract's report for one period —
// the basis for the monthly rollup.
func (s *GovStore) ListProgressReportsByPeriod(ctx context.Context, period string) ([]models.ProgressReport, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+progressCols+` FROM gov_progress_reports WHERE period=$1`, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ProgressReport{}
	for rows.Next() {
		r, err := scanProgressReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// UpsertProgressReport creates or replaces the report for a (contract, period).
func (s *GovStore) UpsertProgressReport(ctx context.Context, r models.ProgressReport) (*models.ProgressReport, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_progress_reports
			(id, contract_id, period, progress, execution_status, current_activity, accomplishments,
			 challenges, interventions, responsible, target_date, proposed_start, proposed_completion,
			 duration, planned_next, planned_progress)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (contract_id, period) DO UPDATE SET
			progress=EXCLUDED.progress, execution_status=EXCLUDED.execution_status,
			current_activity=EXCLUDED.current_activity, accomplishments=EXCLUDED.accomplishments,
			challenges=EXCLUDED.challenges, interventions=EXCLUDED.interventions,
			responsible=EXCLUDED.responsible, target_date=EXCLUDED.target_date,
			proposed_start=EXCLUDED.proposed_start, proposed_completion=EXCLUDED.proposed_completion,
			duration=EXCLUDED.duration, planned_next=EXCLUDED.planned_next,
			planned_progress=EXCLUDED.planned_progress, updated_at=NOW()
		RETURNING `+progressCols,
		r.ID, r.ContractID, r.Period, r.Progress, r.ExecutionStatus, r.CurrentActivity, r.Accomplishments,
		r.Challenges, r.Interventions, r.Responsible, r.TargetDate, r.ProposedStart, r.ProposedCompletion,
		r.Duration, r.PlannedNext, r.PlannedProgress)
	return scanProgressReport(row)
}

func (s *GovStore) DeleteProgressReport(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_progress_reports WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanProgressReport(row pgx.Row) (*models.ProgressReport, error) {
	var r models.ProgressReport
	if err := row.Scan(&r.ID, &r.ContractID, &r.Period, &r.Progress, &r.ExecutionStatus, &r.CurrentActivity,
		&r.Accomplishments, &r.Challenges, &r.Interventions, &r.Responsible, &r.TargetDate, &r.ProposedStart,
		&r.ProposedCompletion, &r.Duration, &r.PlannedNext, &r.PlannedProgress, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// ---------------- Valuations ----------------

const valuationCols = `id, contractor_id, contractor_name, period, contract_sum, amount_paid,
	verified_value_owed, consultant_recommendation, ceo_approval, remarks, verified_date,
	created_at, updated_at`

func (s *GovStore) ListValuations(ctx context.Context, period string) ([]models.Valuation, error) {
	q := `SELECT ` + valuationCols + ` FROM gov_valuations`
	args := []any{}
	if period != "" {
		q += ` WHERE period=$1`
		args = append(args, period)
	}
	q += ` ORDER BY contractor_name`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Valuation{}
	for rows.Next() {
		v, err := scanValuation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (s *GovStore) GetValuation(ctx context.Context, id string) (*models.Valuation, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+valuationCols+` FROM gov_valuations WHERE id=$1`, id)
	v, err := scanValuation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return v, err
}

func (s *GovStore) CreateValuation(ctx context.Context, v models.Valuation) (*models.Valuation, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_valuations
			(id, contractor_id, contractor_name, period, contract_sum, amount_paid, verified_value_owed,
			 consultant_recommendation, ceo_approval, remarks, verified_date)
		VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+valuationCols,
		v.ID, v.ContractorID, v.ContractorName, v.Period, v.ContractSum, v.AmountPaid, v.VerifiedValueOwed,
		v.ConsultantRecommendation, v.CEOApproval, v.Remarks, v.VerifiedDate)
	return scanValuation(row)
}

func (s *GovStore) UpdateValuation(ctx context.Context, v models.Valuation) (*models.Valuation, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_valuations SET
			contractor_id=NULLIF($2,''), contractor_name=$3, period=$4, contract_sum=$5, amount_paid=$6,
			verified_value_owed=$7, consultant_recommendation=$8, ceo_approval=$9, remarks=$10,
			verified_date=$11, updated_at=NOW()
		WHERE id=$1 RETURNING `+valuationCols,
		v.ID, v.ContractorID, v.ContractorName, v.Period, v.ContractSum, v.AmountPaid, v.VerifiedValueOwed,
		v.ConsultantRecommendation, v.CEOApproval, v.Remarks, v.VerifiedDate)
	vv, err := scanValuation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return vv, err
}

func (s *GovStore) DeleteValuation(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_valuations WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanValuation(row pgx.Row) (*models.Valuation, error) {
	var v models.Valuation
	var contractorID *string
	if err := row.Scan(&v.ID, &contractorID, &v.ContractorName, &v.Period, &v.ContractSum, &v.AmountPaid,
		&v.VerifiedValueOwed, &v.ConsultantRecommendation, &v.CEOApproval, &v.Remarks, &v.VerifiedDate,
		&v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, err
	}
	if contractorID != nil {
		v.ContractorID = *contractorID
	}
	return &v, nil
}

// ---------------- Challenges ----------------

const challengeCols = `id, period, seq, category, description, affected, priority, action, owner,
	created_at, updated_at`

// ListChallenges returns the challenges register, optionally filtered to a
// single period, ordered by sequence.
func (s *GovStore) ListChallenges(ctx context.Context, period string) ([]models.Challenge, error) {
	q := `SELECT ` + challengeCols + ` FROM gov_challenges`
	args := []any{}
	if period != "" {
		q += ` WHERE period=$1`
		args = append(args, period)
	}
	q += ` ORDER BY period, seq`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Challenge{}
	for rows.Next() {
		c, err := scanChallenge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *GovStore) GetChallenge(ctx context.Context, id string) (*models.Challenge, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+challengeCols+` FROM gov_challenges WHERE id=$1`, id)
	c, err := scanChallenge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return c, err
}

// CreateChallenge appends a challenge to a period, assigning the next sequence
// number for that period so (period, seq) stays unique.
func (s *GovStore) CreateChallenge(ctx context.Context, c models.Challenge) (*models.Challenge, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner)
		VALUES ($1,$2,(SELECT COALESCE(MAX(seq),0)+1 FROM gov_challenges WHERE period=$2),$3,$4,$5,$6,$7,$8)
		RETURNING `+challengeCols,
		c.ID, c.Period, c.Category, c.Description, c.Affected, c.Priority, c.Action, c.Owner)
	return scanChallenge(row)
}

func (s *GovStore) UpdateChallenge(ctx context.Context, c models.Challenge) (*models.Challenge, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_challenges SET
			category=$2, description=$3, affected=$4, priority=$5, action=$6, owner=$7, updated_at=NOW()
		WHERE id=$1 RETURNING `+challengeCols,
		c.ID, c.Category, c.Description, c.Affected, c.Priority, c.Action, c.Owner)
	cc, err := scanChallenge(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return cc, err
}

func (s *GovStore) DeleteChallenge(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_challenges WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanChallenge(row pgx.Row) (*models.Challenge, error) {
	var c models.Challenge
	if err := row.Scan(&c.ID, &c.Period, &c.Seq, &c.Category, &c.Description, &c.Affected,
		&c.Priority, &c.Action, &c.Owner, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// ---------------- Action items ----------------

const actionItemCols = `id, period, seq, priority, text, party, target, status, created_at, updated_at`

// ListActionItems returns the action-item tracker, optionally filtered to a
// single period, ordered by sequence.
func (s *GovStore) ListActionItems(ctx context.Context, period string) ([]models.ActionItem, error) {
	q := `SELECT ` + actionItemCols + ` FROM gov_action_items`
	args := []any{}
	if period != "" {
		q += ` WHERE period=$1`
		args = append(args, period)
	}
	q += ` ORDER BY period, seq`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ActionItem{}
	for rows.Next() {
		a, err := scanActionItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *GovStore) GetActionItem(ctx context.Context, id string) (*models.ActionItem, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+actionItemCols+` FROM gov_action_items WHERE id=$1`, id)
	a, err := scanActionItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return a, err
}

// CreateActionItem appends an action item to a period, assigning the next
// sequence number for that period so (period, seq) stays unique.
func (s *GovStore) CreateActionItem(ctx context.Context, a models.ActionItem) (*models.ActionItem, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status)
		VALUES ($1,$2,(SELECT COALESCE(MAX(seq),0)+1 FROM gov_action_items WHERE period=$2),$3,$4,$5,$6,$7)
		RETURNING `+actionItemCols,
		a.ID, a.Period, a.Priority, a.Text, a.Party, a.Target, a.Status)
	return scanActionItem(row)
}

func (s *GovStore) UpdateActionItem(ctx context.Context, a models.ActionItem) (*models.ActionItem, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_action_items SET
			priority=$2, text=$3, party=$4, target=$5, status=$6, updated_at=NOW()
		WHERE id=$1 RETURNING `+actionItemCols,
		a.ID, a.Priority, a.Text, a.Party, a.Target, a.Status)
	aa, err := scanActionItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return aa, err
}

func (s *GovStore) DeleteActionItem(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_action_items WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanActionItem(row pgx.Row) (*models.ActionItem, error) {
	var a models.ActionItem
	if err := row.Scan(&a.ID, &a.Period, &a.Seq, &a.Priority, &a.Text, &a.Party,
		&a.Target, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}
