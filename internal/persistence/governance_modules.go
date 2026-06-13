package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// ---------------- Requisitions ----------------

const reqCols = `id, no, title, department, requester, type, procurement_method, supplier,
	estimate, budget_code, urgency, status, stage, approvals, justification, docs,
	linked_contract, created_at, updated_at`

func (s *GovStore) ListRequisitions(ctx context.Context) ([]models.GovRequisition, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+reqCols+` FROM gov_requisitions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovRequisition{}
	for rows.Next() {
		r, err := scanReq(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *GovStore) GetRequisition(ctx context.Context, idOrNo string) (*models.GovRequisition, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+reqCols+` FROM gov_requisitions WHERE id=$1 OR no=$1`, idOrNo)
	r, err := scanReq(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return r, err
}

func (s *GovStore) CreateRequisition(ctx context.Context, r models.GovRequisition) (*models.GovRequisition, error) {
	app, _ := jsonb(r.Approvals)
	docs, _ := jsonb(r.Docs)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_requisitions (id, no, title, department, requester, type, procurement_method,
			supplier, estimate, budget_code, urgency, status, stage, approvals, justification, docs, linked_contract)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$16::jsonb,$17)
		RETURNING `+reqCols,
		r.ID, r.No, r.Title, r.Department, r.Requester, r.Type, r.ProcurementMethod, r.Supplier,
		r.Estimate, r.BudgetCode, r.Urgency, r.Status, r.Stage, app, r.Justification, docs, r.LinkedContract)
	return scanReq(row)
}

func (s *GovStore) UpdateRequisition(ctx context.Context, r models.GovRequisition) (*models.GovRequisition, error) {
	app, _ := jsonb(r.Approvals)
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_requisitions SET status=$2, stage=$3, approvals=$4::jsonb, linked_contract=$5, updated_at=NOW()
		WHERE id=$1 RETURNING `+reqCols,
		r.ID, r.Status, r.Stage, app, r.LinkedContract)
	rr, err := scanReq(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return rr, err
}

func (s *GovStore) DeleteRequisition(ctx context.Context, idOrNo string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_requisitions WHERE id=$1 OR no=$1`, idOrNo)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanReq(row pgx.Row) (*models.GovRequisition, error) {
	var r models.GovRequisition
	var app, docs []byte
	if err := row.Scan(&r.ID, &r.No, &r.Title, &r.Department, &r.Requester, &r.Type, &r.ProcurementMethod,
		&r.Supplier, &r.Estimate, &r.BudgetCode, &r.Urgency, &r.Status, &r.Stage, &app, &r.Justification,
		&docs, &r.LinkedContract, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Approvals = []models.VariationApproval{}
	r.Docs = []models.GovDoc{}
	_ = json.Unmarshal(app, &r.Approvals)
	_ = json.Unmarshal(docs, &r.Docs)
	return &r, nil
}

// ---------------- Obligations ----------------

const obCols = `id, contract_id, type, owner, due_date, frequency, evidence, status, escalation, created_at, updated_at`

func (s *GovStore) ListObligations(ctx context.Context) ([]models.GovObligation, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+obCols+` FROM gov_obligations ORDER BY due_date NULLS LAST, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovObligation{}
	for rows.Next() {
		o, err := scanObligation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (s *GovStore) CreateObligation(ctx context.Context, o models.GovObligation) (*models.GovObligation, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_obligations (id, contract_id, type, owner, due_date, frequency, evidence, status, escalation)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9) RETURNING `+obCols,
		o.ID, o.ContractID, o.Type, o.Owner, o.DueDate, o.Frequency, o.Evidence, o.Status, o.Escalation)
	return scanObligation(row)
}

func (s *GovStore) UpdateObligation(ctx context.Context, o models.GovObligation) (*models.GovObligation, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE gov_obligations SET type=$2, owner=$3, due_date=NULLIF($4,''), frequency=$5, evidence=$6,
			status=$7, escalation=$8, updated_at=NOW() WHERE id=$1 RETURNING `+obCols,
		o.ID, o.Type, o.Owner, o.DueDate, o.Frequency, o.Evidence, o.Status, o.Escalation)
	oo, err := scanObligation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return oo, err
}

func (s *GovStore) GetObligation(ctx context.Context, id string) (*models.GovObligation, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+obCols+` FROM gov_obligations WHERE id=$1`, id)
	o, err := scanObligation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return o, err
}

func (s *GovStore) DeleteObligation(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_obligations WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

func scanObligation(row pgx.Row) (*models.GovObligation, error) {
	var o models.GovObligation
	var due *string
	if err := row.Scan(&o.ID, &o.ContractID, &o.Type, &o.Owner, &due, &o.Frequency, &o.Evidence,
		&o.Status, &o.Escalation, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	if due != nil {
		o.DueDate = *due
	}
	return &o, nil
}

// ---------------- Approval rules ----------------

func (s *GovStore) ListApprovalRules(ctx context.Context) ([]models.GovApprovalRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, applies, threshold, min_value, max_value, route, status, created_at, updated_at
		FROM gov_approval_rules ORDER BY min_value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovApprovalRule{}
	for rows.Next() {
		var r models.GovApprovalRule
		var route []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Applies, &r.Threshold, &r.MinValue, &r.MaxValue, &route, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Route = []string{}
		_ = json.Unmarshal(route, &r.Route)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *GovStore) UpsertApprovalRule(ctx context.Context, r models.GovApprovalRule) (*models.GovApprovalRule, error) {
	route, _ := jsonb(r.Route)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_approval_rules (id, name, applies, threshold, min_value, max_value, route, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, applies=EXCLUDED.applies, threshold=EXCLUDED.threshold,
			min_value=EXCLUDED.min_value, max_value=EXCLUDED.max_value, route=EXCLUDED.route, status=EXCLUDED.status, updated_at=NOW()
		RETURNING id, name, applies, threshold, min_value, max_value, route, status, created_at, updated_at`,
		r.ID, r.Name, r.Applies, r.Threshold, r.MinValue, r.MaxValue, route, r.Status)
	var out models.GovApprovalRule
	var rb []byte
	if err := row.Scan(&out.ID, &out.Name, &out.Applies, &out.Threshold, &out.MinValue, &out.MaxValue, &rb, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	out.Route = []string{}
	_ = json.Unmarshal(rb, &out.Route)
	return &out, nil
}

func (s *GovStore) DeleteApprovalRule(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_approval_rules WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// ---------------- Templates ----------------

func (s *GovStore) ListTemplates(ctx context.Context) ([]models.GovTemplate, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, type, owner, version, status, clauses, created_at, updated_at FROM gov_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovTemplate{}
	for rows.Next() {
		var t models.GovTemplate
		var cl []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.Owner, &t.Version, &t.Status, &cl, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Clauses = []string{}
		_ = json.Unmarshal(cl, &t.Clauses)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *GovStore) UpsertTemplate(ctx context.Context, t models.GovTemplate) (*models.GovTemplate, error) {
	cl, _ := jsonb(t.Clauses)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_templates (id, name, type, owner, version, status, clauses)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type, owner=EXCLUDED.owner,
			version=EXCLUDED.version, status=EXCLUDED.status, clauses=EXCLUDED.clauses, updated_at=NOW()
		RETURNING id, name, type, owner, version, status, clauses, created_at, updated_at`,
		t.ID, t.Name, t.Type, t.Owner, t.Version, t.Status, cl)
	var out models.GovTemplate
	var clb []byte
	if err := row.Scan(&out.ID, &out.Name, &out.Type, &out.Owner, &out.Version, &out.Status, &clb, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	out.Clauses = []string{}
	_ = json.Unmarshal(clb, &out.Clauses)
	return &out, nil
}

func (s *GovStore) DeleteTemplate(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_templates WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// ---------------- Clauses ----------------

func (s *GovStore) ListClauses(ctx context.Context) ([]models.GovClause, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, title, risk, approved, owner, text, created_at, updated_at FROM gov_clauses ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovClause{}
	for rows.Next() {
		var c models.GovClause
		if err := rows.Scan(&c.ID, &c.Title, &c.Risk, &c.Approved, &c.Owner, &c.Text, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *GovStore) UpsertClause(ctx context.Context, c models.GovClause) (*models.GovClause, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_clauses (id, title, risk, approved, owner, text)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, risk=EXCLUDED.risk, approved=EXCLUDED.approved,
			owner=EXCLUDED.owner, text=EXCLUDED.text, updated_at=NOW()
		RETURNING id, title, risk, approved, owner, text, created_at, updated_at`,
		c.ID, c.Title, c.Risk, c.Approved, c.Owner, c.Text)
	var out models.GovClause
	if err := row.Scan(&out.ID, &out.Title, &out.Risk, &out.Approved, &out.Owner, &out.Text, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *GovStore) DeleteClause(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_clauses WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// ---------------- Budgets ----------------

func (s *GovStore) ListBudgets(ctx context.Context) ([]models.GovBudget, error) {
	rows, err := s.pool.Query(ctx, `SELECT code, name, owner, approved, committed, paid, created_at, updated_at FROM gov_budgets ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GovBudget{}
	for rows.Next() {
		var b models.GovBudget
		if err := rows.Scan(&b.Code, &b.Name, &b.Owner, &b.Approved, &b.Committed, &b.Paid, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *GovStore) UpsertBudget(ctx context.Context, b models.GovBudget) (*models.GovBudget, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_budgets (code, name, owner, approved, committed, paid)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, owner=EXCLUDED.owner, approved=EXCLUDED.approved,
			committed=EXCLUDED.committed, paid=EXCLUDED.paid, updated_at=NOW()
		RETURNING code, name, owner, approved, committed, paid, created_at, updated_at`,
		b.Code, b.Name, b.Owner, b.Approved, b.Committed, b.Paid)
	var out models.GovBudget
	if err := row.Scan(&out.Code, &out.Name, &out.Owner, &out.Approved, &out.Committed, &out.Paid, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *GovStore) DeleteBudget(ctx context.Context, code string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM gov_budgets WHERE code=$1`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGovNotFound
	}
	return nil
}

// ---------------- Closeout ----------------

func (s *GovStore) GetCloseout(ctx context.Context, contractID string) (*models.GovCloseout, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT contract_id, final_account, retention_decision, defects_liability, documents_complete,
			unresolved_variations, final_report, status, updated_at
		FROM gov_closeouts WHERE contract_id=$1`, contractID)
	c, err := scanCloseout(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGovNotFound
	}
	return c, err
}

func (s *GovStore) UpsertCloseout(ctx context.Context, c models.GovCloseout) (*models.GovCloseout, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO gov_closeouts (contract_id, final_account, retention_decision, defects_liability,
			documents_complete, unresolved_variations, final_report, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (contract_id) DO UPDATE SET final_account=EXCLUDED.final_account,
			retention_decision=EXCLUDED.retention_decision, defects_liability=EXCLUDED.defects_liability,
			documents_complete=EXCLUDED.documents_complete, unresolved_variations=EXCLUDED.unresolved_variations,
			final_report=EXCLUDED.final_report, status=EXCLUDED.status, updated_at=NOW()
		RETURNING contract_id, final_account, retention_decision, defects_liability, documents_complete,
			unresolved_variations, final_report, status, updated_at`,
		c.ContractID, c.FinalAccount, c.RetentionDecision, c.DefectsLiability, c.DocumentsComplete,
		c.UnresolvedVariations, c.FinalReport, c.Status)
	return scanCloseout(row)
}

func scanCloseout(row pgx.Row) (*models.GovCloseout, error) {
	var c models.GovCloseout
	if err := row.Scan(&c.ContractID, &c.FinalAccount, &c.RetentionDecision, &c.DefectsLiability,
		&c.DocumentsComplete, &c.UnresolvedVariations, &c.FinalReport, &c.Status, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}
