package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// ----- Payments -----

func (s *GovStore) CreatePayment(ctx context.Context, p models.GovPayment) (*models.GovPayment, error) {
	hist, _ := jsonb(p.History)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_payments (id, milestone_id, contract_id, amount, retention, payable, stage, status, history)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)
		RETURNING id, milestone_id, contract_id, amount, retention, payable, stage, status, history, created_at, updated_at`,
		p.ID, p.MilestoneID, p.ContractID, p.Amount, p.Retention, p.Payable, p.Stage, p.Status, hist)
	return scanPayment(row)
}

func (s *GovStore) GetPayment(ctx context.Context, id string) (*models.GovPayment, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, milestone_id, contract_id, amount, retention, payable, stage, status, history, created_at, updated_at
		FROM gov_payments WHERE id = $1`, id)
	p, err := scanPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return p, err
}

func (s *GovStore) GetPaymentByMilestone(ctx context.Context, milestoneID string) (*models.GovPayment, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, milestone_id, contract_id, amount, retention, payable, stage, status, history, created_at, updated_at
		FROM gov_payments WHERE milestone_id = $1`, milestoneID)
	p, err := scanPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return p, err
}

func (s *GovStore) UpdatePayment(ctx context.Context, p models.GovPayment) (*models.GovPayment, error) {
	hist, _ := jsonb(p.History)
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_payments SET stage=$2, status=$3, history=$4::jsonb, updated_at=NOW()
		WHERE id=$1
		RETURNING id, milestone_id, contract_id, amount, retention, payable, stage, status, history, created_at, updated_at`,
		p.ID, p.Stage, p.Status, hist)
	pp, err := scanPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return pp, err
}

// ListPayments returns the payment queue across all milestones, newest first,
// with optional contract and status filters (the basis for the finance payment
// dashboard). An empty filter matches everything.
func (s *GovStore) ListPayments(ctx context.Context, contractID, status string) ([]models.GovPayment, error) {
	q := `SELECT id, milestone_id, contract_id, amount, retention, payable, stage, status, history, created_at, updated_at
		FROM gov_payments`
	args := []any{}
	conds := []string{}
	if contractID != "" {
		args = append(args, contractID)
		conds = append(conds, "contract_id = $"+strconv.Itoa(len(args)))
	}
	if status != "" {
		args = append(args, status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovPayment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanPayment(row pgx.Row) (*models.GovPayment, error) {
	var p models.GovPayment
	var hist []byte
	if err := row.Scan(&p.ID, &p.MilestoneID, &p.ContractID, &p.Amount, &p.Retention, &p.Payable,
		&p.Stage, &p.Status, &hist, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.History = []models.PaymentStep{}
	_ = json.Unmarshal(hist, &p.History)
	return &p, nil
}

// ----- Variations -----

func (s *GovStore) CreateVariation(ctx context.Context, v models.GovVariation) (*models.GovVariation, error) {
	app, _ := jsonb(v.Approvals)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_variations
			(id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb)
		RETURNING id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals, created_at, updated_at`,
		v.ID, v.ContractID, v.Number, v.Title, v.Amount, v.ExtensionDays, v.Description, v.Reason, v.Impact, v.Status, v.Stage, app)
	return scanVariation(row)
}

func (s *GovStore) GetVariation(ctx context.Context, id string) (*models.GovVariation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals, created_at, updated_at
		FROM gov_variations WHERE id = $1`, id)
	v, err := scanVariation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return v, err
}

func (s *GovStore) ListVariations(ctx context.Context, contractID string) ([]models.GovVariation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals, created_at, updated_at
		FROM gov_variations WHERE contract_id = $1 ORDER BY created_at`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovVariation{}
	for rows.Next() {
		v, err := scanVariation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// ListAllVariations returns variations across all contracts, newest first, with
// optional contract and status filters (the basis for the variations approval
// queue). An empty filter matches everything.
func (s *GovStore) ListAllVariations(ctx context.Context, contractID, status string) ([]models.GovVariation, error) {
	q := `SELECT id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals, created_at, updated_at
		FROM gov_variations`
	args := []any{}
	conds := []string{}
	if contractID != "" {
		args = append(args, contractID)
		conds = append(conds, "contract_id = $"+strconv.Itoa(len(args)))
	}
	if status != "" {
		args = append(args, status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovVariation{}
	for rows.Next() {
		v, err := scanVariation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (s *GovStore) UpdateVariation(ctx context.Context, v models.GovVariation) (*models.GovVariation, error) {
	app, _ := jsonb(v.Approvals)
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_variations SET status=$2, stage=$3, approvals=$4::jsonb, updated_at=NOW()
		WHERE id=$1
		RETURNING id, contract_id, number, title, amount, extension_days, description, reason, impact, status, stage, approvals, created_at, updated_at`,
		v.ID, v.Status, v.Stage, app)
	vv, err := scanVariation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return vv, err
}

// AddContractValue applies a variation's value impact to the contract total.
func (s *GovStore) AddContractValue(ctx context.Context, contractID string, delta int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE gov_contracts SET value = value + $2, updated_at = NOW() WHERE id = $1`, contractID, delta)
	return err
}

func scanVariation(row pgx.Row) (*models.GovVariation, error) {
	var v models.GovVariation
	var app []byte
	if err := row.Scan(&v.ID, &v.ContractID, &v.Number, &v.Title, &v.Amount, &v.ExtensionDays,
		&v.Description, &v.Reason, &v.Impact, &v.Status, &v.Stage, &app, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, err
	}
	v.Approvals = []models.VariationApproval{}
	_ = json.Unmarshal(app, &v.Approvals)
	return &v, nil
}
