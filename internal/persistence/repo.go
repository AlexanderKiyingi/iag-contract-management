package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/jackc/pgx/v5"
)

// LoadState reads all workspace and frontend entities from indexed PostgreSQL tables.
func (p *Postgres) LoadState(ctx context.Context) (models.Workspace, models.FrontendStore, error) {
	ws := models.Workspace{}
	fe := models.FrontendStore{
		ProfilePhotos: map[string]string{},
		Assistance:    []models.AssistanceMessage{},
		Updates:       []any{},
	}

	rows, err := p.Pool.Query(ctx, `
		SELECT code, name, description, supervisor, contract_sum, paid, balance, color, contract_count
		FROM zones ORDER BY code`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.Code, &z.Name, &z.Desc, &z.Sup, &z.Cs, &z.Paid, &z.Bal, &z.Color, &z.Contracts); err != nil {
			rows.Close()
			return ws, fe, err
		}
		ws.Zones = append(ws.Zones, z)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT contract_no, name, zone_code, contract_sum, paid, balance, progress, status, priority,
		       workers, supervisor, remarks, created_on
		FROM contracts ORDER BY contract_no`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var c models.Contract
		if err := rows.Scan(&c.No, &c.Name, &c.Zone, &c.Cs, &c.Paid, &c.Bal, &c.Prog, &c.Status, &c.Pri,
			&c.Workers, &c.Sup, &c.Remarks, &c.Created); err != nil {
			rows.Close()
			return ws, fe, err
		}
		ws.Contracts = append(ws.Contracts, c)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, name, role, zone_code, phone, email, active FROM engineers ORDER BY id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var e models.Engineer
		if err := rows.Scan(&e.ID, &e.Name, &e.Role, &e.Zone, &e.Phone, &e.Email, &e.Active); err != nil {
			rows.Close()
			return ws, fe, err
		}
		ws.Engineers = append(ws.Engineers, e)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `SELECT id, name FROM contractors ORDER BY id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var c models.Contractor
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			rows.Close()
			return ws, fe, err
		}
		ws.Contractors = append(ws.Contractors, c)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, email, display_name, role, status, custom_role_id
		FROM workspace_users ORDER BY id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var u models.WorkspaceUser
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status, &u.CustomRoleID); err != nil {
			rows.Close()
			return ws, fe, err
		}
		ws.WorkspaceUsers = append(ws.WorkspaceUsers, u)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, title, due_date, zone_code, status, owner FROM milestones ORDER BY due_date, id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var m models.Milestone
		if err := rows.Scan(&m.ID, &m.Title, &m.Due, &m.Zone, &m.Status, &m.Owner); err != nil {
			rows.Close()
			return ws, fe, err
		}
		fe.Milestones = append(fe.Milestones, m)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, item, zone_code, quantity, unit, entry_date, supplier FROM materials ORDER BY entry_date DESC, id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var m models.MaterialEntry
		if err := rows.Scan(&m.ID, &m.Item, &m.Zone, &m.Qty, &m.Unit, &m.Date, &m.Supplier); err != nil {
			rows.Close()
			return ws, fe, err
		}
		fe.Materials = append(fe.Materials, m)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, name, description, permissions, template FROM custom_roles ORDER BY name`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var r models.CustomRole
		var perms []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &perms, &r.Template); err != nil {
			rows.Close()
			return ws, fe, err
		}
		_ = json.Unmarshal(perms, &r.Permissions)
		fe.CustomRoles = append(fe.CustomRoles, r)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `
		SELECT id, logged_at, user_name, action, detail
		FROM audit_entries ORDER BY logged_at_ts DESC LIMIT 120`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var a models.AuditEntry
		if err := rows.Scan(&a.ID, &a.At, &a.User, &a.Action, &a.Detail); err != nil {
			rows.Close()
			return ws, fe, err
		}
		fe.Audit = append(fe.Audit, a)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `SELECT sender, body, sent_at FROM assistance_messages ORDER BY id`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var m models.AssistanceMessage
		if err := rows.Scan(&m.From, &m.Text, &m.At); err != nil {
			rows.Close()
			return ws, fe, err
		}
		fe.Assistance = append(fe.Assistance, m)
	}
	rows.Close()

	rows, err = p.Pool.Query(ctx, `SELECT email, data_url FROM profile_photos`)
	if err != nil {
		return ws, fe, err
	}
	for rows.Next() {
		var email, dataURL string
		if err := rows.Scan(&email, &dataURL); err != nil {
			rows.Close()
			return ws, fe, err
		}
		fe.ProfilePhotos[email] = dataURL
	}
	rows.Close()

	fe.Tasks, err = p.loadTasks(ctx)
	if err != nil {
		return ws, fe, err
	}

	var aiScan []byte
	err = p.Pool.QueryRow(ctx, `SELECT value FROM app_meta WHERE key = 'ai_scan'`).Scan(&aiScan)
	if err == nil && len(aiScan) > 0 && string(aiScan) != "null" {
		var scan any
		if json.Unmarshal(aiScan, &scan) == nil {
			fe.AiScan = scan
		}
	} else if err != nil && err != pgx.ErrNoRows {
		return ws, fe, err
	}

	return ws, fe, nil
}

func (p *Postgres) loadTasks(ctx context.Context) (models.TasksStore, error) {
	var ts models.TasksStore
	rows, err := p.Pool.Query(ctx, `
		SELECT id, sort_order, name, sections FROM task_projects ORDER BY sort_order`)
	if err != nil {
		return ts, err
	}
	type projRow struct {
		id    int
		order int
		proj  models.TaskProject
	}
	var projs []projRow
	for rows.Next() {
		var pr projRow
		var sections []byte
		if err := rows.Scan(&pr.id, &pr.order, &pr.proj.Name, &sections); err != nil {
			rows.Close()
			return ts, err
		}
		_ = json.Unmarshal(sections, &pr.proj.Sections)
		pr.proj.DBID = pr.id
		pr.proj.Tasks = []models.TaskItem{}
		projs = append(projs, pr)
	}
	rows.Close()

	if len(projs) == 0 {
		return ts, nil
	}

	rows, err = p.Pool.Query(ctx, `
		SELECT project_id, id, title, column_key, assignee FROM task_items ORDER BY project_id, id`)
	if err != nil {
		return ts, err
	}
	byProj := map[int][]models.TaskItem{}
	for rows.Next() {
		var projectID int
		var t models.TaskItem
		if err := rows.Scan(&projectID, &t.ID, &t.Title, &t.Col, &t.Assignee); err != nil {
			rows.Close()
			return ts, err
		}
		byProj[projectID] = append(byProj[projectID], t)
	}
	rows.Close()

	for _, pr := range projs {
		pr.proj.Tasks = byProj[pr.id]
		if pr.proj.Tasks == nil {
			pr.proj.Tasks = []models.TaskItem{}
		}
		ts.Projects = append(ts.Projects, pr.proj)
	}
	return ts, nil
}

// SaveState replaces all entity rows in a single transaction (keeps indexes in sync with API state).
func (p *Postgres) SaveState(ctx context.Context, ws models.Workspace, fe models.FrontendStore) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tables := []string{
		"task_items", "task_projects", "audit_entries", "assistance_messages", "profile_photos",
		"materials", "milestones", "custom_roles", "contracts", "engineers", "contractors",
		"workspace_users", "zones",
	}
	for _, t := range tables {
		if _, err := tx.Exec(ctx, `DELETE FROM `+t); err != nil {
			return fmt.Errorf("clear %s: %w", t, err)
		}
	}

	for _, z := range ws.Zones {
		if _, err := tx.Exec(ctx, `
			INSERT INTO zones (code, name, description, supervisor, contract_sum, paid, balance, color, contract_count)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			z.Code, z.Name, z.Desc, z.Sup, z.Cs, z.Paid, z.Bal, z.Color, z.Contracts); err != nil {
			return err
		}
	}
	for _, c := range ws.Contracts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO contracts (contract_no, name, zone_code, contract_sum, paid, balance, progress, status, priority,
				workers, supervisor, remarks, created_on)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			c.No, c.Name, c.Zone, c.Cs, c.Paid, c.Bal, c.Prog, c.Status, c.Pri,
			c.Workers, c.Sup, c.Remarks, c.Created); err != nil {
			return err
		}
	}
	for _, e := range ws.Engineers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO engineers (id, name, role, zone_code, phone, email, active)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			e.ID, e.Name, e.Role, e.Zone, e.Phone, e.Email, e.Active); err != nil {
			return err
		}
	}
	for _, c := range ws.Contractors {
		if _, err := tx.Exec(ctx, `INSERT INTO contractors (id, name) VALUES ($1,$2)`, c.ID, c.Name); err != nil {
			return err
		}
	}
	for _, u := range ws.WorkspaceUsers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO workspace_users (id, email, display_name, role, status, custom_role_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			u.ID, u.Email, u.DisplayName, u.Role, u.Status, u.CustomRoleID); err != nil {
			return err
		}
	}

	for _, m := range fe.Milestones {
		if _, err := tx.Exec(ctx, `
			INSERT INTO milestones (id, title, due_date, zone_code, status, owner)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			m.ID, m.Title, m.Due, m.Zone, m.Status, m.Owner); err != nil {
			return err
		}
	}
	for _, m := range fe.Materials {
		if _, err := tx.Exec(ctx, `
			INSERT INTO materials (id, item, zone_code, quantity, unit, entry_date, supplier)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			m.ID, m.Item, m.Zone, m.Qty, m.Unit, m.Date, m.Supplier); err != nil {
			return err
		}
	}
	for _, r := range fe.CustomRoles {
		perms, _ := json.Marshal(r.Permissions)
		if _, err := tx.Exec(ctx, `
			INSERT INTO custom_roles (id, name, description, permissions, template)
			VALUES ($1,$2,$3,$4,$5)`,
			r.ID, r.Name, r.Description, perms, r.Template); err != nil {
			return err
		}
	}
	for _, a := range fe.Audit {
		ts := time.Now()
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_entries (id, logged_at, logged_at_ts, user_name, action, detail)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			a.ID, a.At, ts, a.User, a.Action, a.Detail); err != nil {
			return err
		}
	}
	for _, m := range fe.Assistance {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assistance_messages (sender, body, sent_at) VALUES ($1,$2,$3)`,
			m.From, m.Text, m.At); err != nil {
			return err
		}
	}
	for email, dataURL := range fe.ProfilePhotos {
		if _, err := tx.Exec(ctx, `INSERT INTO profile_photos (email, data_url) VALUES ($1,$2)`,
			strings.ToLower(email), dataURL); err != nil {
			return err
		}
	}

	for i, pr := range fe.Tasks.Projects {
		sections, _ := json.Marshal(pr.Sections)
		var projectID int
		if err := tx.QueryRow(ctx, `
			INSERT INTO task_projects (sort_order, name, sections) VALUES ($1,$2,$3) RETURNING id`,
			i, pr.Name, sections).Scan(&projectID); err != nil {
			return err
		}
		for _, t := range pr.Tasks {
			if _, err := tx.Exec(ctx, `
				INSERT INTO task_items (id, project_id, title, column_key, assignee)
				VALUES ($1,$2,$3,$4,$5)`,
				t.ID, projectID, t.Title, t.Col, t.Assignee); err != nil {
				return err
			}
		}
	}

	if fe.AiScan != nil {
		raw, _ := json.Marshal(fe.AiScan)
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_meta (key, value) VALUES ('ai_scan', $1::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, string(raw)); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ContractorSupervisor returns the supervisor mapping for the supplied email.
// (supervisor, true, nil) when present; ("", false, nil) when the caller is
// not a contractor in this deployment.
func (p *Postgres) ContractorSupervisor(ctx context.Context, email string) (string, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", false, nil
	}
	var sup string
	err := p.Pool.QueryRow(ctx,
		`SELECT supervisor FROM contractor_supervisors WHERE email = $1`, email,
	).Scan(&sup)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return sup, true, nil
}

// GovContractorLinked reports whether a governance contractor is bound to the
// given platform user id (gov_contractors.platform_user_id). Used to promote a
// linked user to the contractor role so the governance portal is shown.
func (p *Postgres) GovContractorLinked(ctx context.Context, userID, email string) (bool, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	if userID == "" && email == "" {
		return false, nil
	}
	var exists bool
	err := p.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM gov_contractors
		   WHERE (platform_user_id <> '' AND platform_user_id = $1)
		      OR (user_email <> '' AND lower(user_email) = $2))`, userID, email,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// UpsertContractor binds email → supervisor for the contractor portal.
func (p *Postgres) UpsertContractor(ctx context.Context, email, supervisor string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || supervisor == "" {
		return models.ErrValidation
	}
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO contractor_supervisors (email, supervisor) VALUES ($1,$2)
		ON CONFLICT (email) DO UPDATE SET supervisor = EXCLUDED.supervisor
	`, email, supervisor)
	return err
}

// RemoveContractor clears the contractor binding for an email.
func (p *Postgres) RemoveContractor(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	_, err := p.Pool.Exec(ctx,
		`DELETE FROM contractor_supervisors WHERE email = $1`, email)
	return err
}
