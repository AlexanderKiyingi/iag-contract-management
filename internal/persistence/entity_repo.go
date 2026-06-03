// Per-row CRUD against the operational tables. Replaces the previous
// "snapshot rewrite" pattern (SaveState DELETEs every row + reinserts) for
// the per-entity REST endpoints, so concurrent writes no longer trample
// each other and audit history is preserved across mutations.
//
// SaveState/LoadState remain on the snapshot path (initial seed and the
// super_admin-only bulk PUTs).
package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// errOrNotFound returns ErrNotFound when the command affected zero rows,
// the underlying error otherwise.
func errOrNotFound(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// --- Contracts ----------------------------------------------------------

func (p *Postgres) InsertContract(ctx context.Context, c models.Contract) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO contracts (contract_no, name, zone_code, contract_sum, paid, balance,
			progress, status, priority, workers, supervisor, remarks, created_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		c.No, c.Name, c.Zone, c.Cs, c.Paid, c.Bal, c.Prog, c.Status, c.Pri,
		c.Workers, c.Sup, c.Remarks, c.Created)
	return err
}

func (p *Postgres) UpdateContract(ctx context.Context, c models.Contract) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE contracts SET name=$2, zone_code=$3, contract_sum=$4, paid=$5, balance=$6,
			progress=$7, status=$8, priority=$9, workers=$10, supervisor=$11, remarks=$12,
			created_on=$13
		WHERE contract_no=$1`,
		c.No, c.Name, c.Zone, c.Cs, c.Paid, c.Bal, c.Prog, c.Status, c.Pri,
		c.Workers, c.Sup, c.Remarks, c.Created)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteContract(ctx context.Context, no string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM contracts WHERE contract_no=$1`, no)
	return errOrNotFound(tag, err)
}

// --- Zones (only count maintenance — full zone seed is owned by SaveState) -

func (p *Postgres) UpdateZone(ctx context.Context, z models.Zone) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE zones SET name=$2, description=$3, supervisor=$4, color=$5
		WHERE code=$1`,
		z.Code, z.Name, z.Desc, z.Sup, z.Color)
	return errOrNotFound(tag, err)
}

func (p *Postgres) UpdateZoneCount(ctx context.Context, code string, count int) error {
	tag, err := p.Pool.Exec(ctx,
		`UPDATE zones SET contract_count=$2 WHERE code=$1`, code, count)
	return errOrNotFound(tag, err)
}

// --- Engineers ----------------------------------------------------------

func (p *Postgres) InsertEngineer(ctx context.Context, e models.Engineer) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO engineers (id, name, role, zone_code, phone, email, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.Name, e.Role, e.Zone, e.Phone, e.Email, e.Active)
	return err
}

func (p *Postgres) UpdateEngineer(ctx context.Context, e models.Engineer) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE engineers SET name=$2, role=$3, zone_code=$4, phone=$5, email=$6, active=$7
		WHERE id=$1`,
		e.ID, e.Name, e.Role, e.Zone, e.Phone, e.Email, e.Active)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteEngineer(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM engineers WHERE id=$1`, id)
	return errOrNotFound(tag, err)
}

// --- Workspace users ----------------------------------------------------

func (p *Postgres) InsertWorkspaceUser(ctx context.Context, u models.WorkspaceUser) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO workspace_users (id, email, display_name, role, status, custom_role_id)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.Email, u.DisplayName, u.Role, u.Status, u.CustomRoleID)
	return err
}

func (p *Postgres) UpdateWorkspaceUser(ctx context.Context, u models.WorkspaceUser) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE workspace_users SET email=$2, display_name=$3, role=$4, status=$5, custom_role_id=$6
		WHERE id=$1`,
		u.ID, u.Email, u.DisplayName, u.Role, u.Status, u.CustomRoleID)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteWorkspaceUser(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM workspace_users WHERE id=$1`, id)
	return errOrNotFound(tag, err)
}

// --- Milestones ---------------------------------------------------------

func (p *Postgres) InsertMilestone(ctx context.Context, m models.Milestone) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO milestones (id, title, due_date, zone_code, status, owner)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		m.ID, m.Title, m.Due, m.Zone, m.Status, m.Owner)
	return err
}

func (p *Postgres) UpdateMilestone(ctx context.Context, m models.Milestone) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE milestones SET title=$2, due_date=$3, zone_code=$4, status=$5, owner=$6
		WHERE id=$1`,
		m.ID, m.Title, m.Due, m.Zone, m.Status, m.Owner)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteMilestone(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM milestones WHERE id=$1`, id)
	return errOrNotFound(tag, err)
}

// --- Materials ----------------------------------------------------------

func (p *Postgres) InsertMaterial(ctx context.Context, m models.MaterialEntry) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO materials (id, item, zone_code, quantity, unit, entry_date, supplier)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.Item, m.Zone, m.Qty, m.Unit, m.Date, m.Supplier)
	return err
}

func (p *Postgres) UpdateMaterial(ctx context.Context, m models.MaterialEntry) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE materials SET item=$2, zone_code=$3, quantity=$4, unit=$5, entry_date=$6, supplier=$7
		WHERE id=$1`,
		m.ID, m.Item, m.Zone, m.Qty, m.Unit, m.Date, m.Supplier)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteMaterial(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM materials WHERE id=$1`, id)
	return errOrNotFound(tag, err)
}

// --- Custom roles -------------------------------------------------------

func (p *Postgres) InsertCustomRole(ctx context.Context, r models.CustomRole) error {
	perms, _ := json.Marshal(r.Permissions)
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO custom_roles (id, name, description, permissions, template)
		VALUES ($1,$2,$3,$4,$5)`,
		r.ID, r.Name, r.Description, perms, r.Template)
	return err
}

func (p *Postgres) UpdateCustomRole(ctx context.Context, r models.CustomRole) error {
	perms, _ := json.Marshal(r.Permissions)
	tag, err := p.Pool.Exec(ctx, `
		UPDATE custom_roles SET name=$2, description=$3, permissions=$4, template=$5
		WHERE id=$1`,
		r.ID, r.Name, r.Description, perms, r.Template)
	return errOrNotFound(tag, err)
}

// DeleteCustomRole deletes the role AND clears any workspace_users.custom_role_id
// pointing at it, in a single transaction.
func (p *Postgres) DeleteCustomRole(ctx context.Context, id string) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE workspace_users SET custom_role_id=NULL WHERE custom_role_id=$1`, id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM custom_roles WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return tx.Commit(ctx)
}

// --- Audit log ----------------------------------------------------------

// InsertAuditAndTrim inserts a new audit entry at "now" and prunes the table
// to keep at most `keep` most-recent rows. Performed in a single transaction
// so the pruner can never observe partial state.
func (p *Postgres) InsertAuditAndTrim(ctx context.Context, a models.AuditEntry, keep int) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_entries (id, logged_at, user_name, action, detail)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (id) DO NOTHING`,
		a.ID, a.At, a.User, a.Action, a.Detail); err != nil {
		return err
	}

	if keep > 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM audit_entries
			WHERE id NOT IN (
				SELECT id FROM audit_entries
				ORDER BY logged_at_ts DESC, id DESC
				LIMIT $1
			)`, keep); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// --- Assistance messages ------------------------------------------------

func (p *Postgres) InsertAssistance(ctx context.Context, m models.AssistanceMessage) error {
	_, err := p.Pool.Exec(ctx,
		`INSERT INTO assistance_messages (sender, body, sent_at) VALUES ($1,$2,$3)`,
		m.From, m.Text, m.At)
	return err
}

// --- Profile photos -----------------------------------------------------

func (p *Postgres) UpsertProfilePhoto(ctx context.Context, email, dataURL string) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO profile_photos (email, data_url) VALUES ($1,$2)
		ON CONFLICT (email) DO UPDATE SET data_url=EXCLUDED.data_url`,
		strings.ToLower(strings.TrimSpace(email)), dataURL)
	return err
}

func (p *Postgres) DeleteProfilePhoto(ctx context.Context, email string) error {
	_, err := p.Pool.Exec(ctx,
		`DELETE FROM profile_photos WHERE email=$1`,
		strings.ToLower(strings.TrimSpace(email)))
	return err
}

// --- AI scan blob -------------------------------------------------------

func (p *Postgres) UpsertAiScan(ctx context.Context, scan any) error {
	raw, err := json.Marshal(scan)
	if err != nil {
		return fmt.Errorf("marshal ai scan: %w", err)
	}
	_, err = p.Pool.Exec(ctx, `
		INSERT INTO app_meta (key, value) VALUES ('ai_scan', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, string(raw))
	return err
}

// --- Task projects + items ---------------------------------------------

// InsertTaskProject creates a new project and returns its surrogate id.
// sort_order is computed as max(sort_order)+1 so a prior delete that leaves
// a gap never collides with the UNIQUE constraint.
func (p *Postgres) InsertTaskProject(ctx context.Context, name string, sections []string) (int, error) {
	sectionsJSON, _ := json.Marshal(sections)
	var nextOrder int
	if err := p.Pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order)+1, 0) FROM task_projects`).Scan(&nextOrder); err != nil {
		return 0, err
	}
	var id int
	err := p.Pool.QueryRow(ctx, `
		INSERT INTO task_projects (sort_order, name, sections)
		VALUES ($1,$2,$3) RETURNING id`,
		nextOrder, name, sectionsJSON).Scan(&id)
	return id, err
}

func (p *Postgres) UpdateTaskProject(ctx context.Context, id int, name string, sections []string) error {
	sectionsJSON, _ := json.Marshal(sections)
	tag, err := p.Pool.Exec(ctx, `
		UPDATE task_projects SET name=$2, sections=$3 WHERE id=$1`,
		id, name, sectionsJSON)
	return errOrNotFound(tag, err)
}

// DeleteTaskProject also removes all tasks under it via the FK ON DELETE CASCADE.
func (p *Postgres) DeleteTaskProject(ctx context.Context, id int) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM task_projects WHERE id=$1`, id)
	return errOrNotFound(tag, err)
}

func (p *Postgres) InsertTaskItem(ctx context.Context, projectID int, t models.TaskItem) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO task_items (id, project_id, title, column_key, assignee)
		VALUES ($1,$2,$3,$4,$5)`,
		t.ID, projectID, t.Title, t.Col, t.Assignee)
	return err
}

func (p *Postgres) UpdateTaskItem(ctx context.Context, projectID int, t models.TaskItem) error {
	tag, err := p.Pool.Exec(ctx, `
		UPDATE task_items SET title=$2, column_key=$3, assignee=$4
		WHERE id=$1 AND project_id=$5`,
		t.ID, t.Title, t.Col, t.Assignee, projectID)
	return errOrNotFound(tag, err)
}

func (p *Postgres) DeleteTaskItem(ctx context.Context, projectID int, taskID string) error {
	tag, err := p.Pool.Exec(ctx, `
		DELETE FROM task_items WHERE id=$1 AND project_id=$2`,
		taskID, projectID)
	return errOrNotFound(tag, err)
}

// Compile-time check that *Postgres satisfies models.Repository.
var _ models.Repository = (*Postgres)(nil)
