package persistence

import (
	"context"
	"fmt"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// ListMilestonesDueSoon returns milestones due within the next withinDays (inclusive)
// that are not marked Complete. due_date is stored as YYYY-MM-DD text.
func (p *Postgres) ListMilestonesDueSoon(ctx context.Context, withinDays int) ([]models.Milestone, error) {
	if withinDays < 0 {
		withinDays = 0
	}
	rows, err := p.Pool.Query(ctx, `
		SELECT id, title, due_date, zone_code, status, owner
		FROM milestones
		WHERE lower(status) <> 'complete'
		  AND due_date ~ '^\d{4}-\d{2}-\d{2}$'
		  AND due_date::date >= CURRENT_DATE
		  AND due_date::date <= CURRENT_DATE + ($1::int * INTERVAL '1 day')
		ORDER BY due_date, id`,
		withinDays,
	)
	if err != nil {
		return nil, fmt.Errorf("list milestones due soon: %w", err)
	}
	defer rows.Close()

	var out []models.Milestone
	for rows.Next() {
		var m models.Milestone
		if err := rows.Scan(&m.ID, &m.Title, &m.Due, &m.Zone, &m.Status, &m.Owner); err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// WasMilestoneReminderSent reports whether a reminder was already recorded for this milestone/due pair.
func (p *Postgres) WasMilestoneReminderSent(ctx context.Context, milestoneID, due string) (bool, error) {
	key := milestoneReminderKey(milestoneID, due)
	var exists bool
	err := p.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM app_meta WHERE key = $1)`,
		key,
	).Scan(&exists)
	return exists, err
}

// MarkMilestoneReminderSent records that a due-soon reminder was emitted.
func (p *Postgres) MarkMilestoneReminderSent(ctx context.Context, milestoneID, due string) error {
	key := milestoneReminderKey(milestoneID, due)
	_, err := p.Pool.Exec(ctx,
		`INSERT INTO app_meta (key, value) VALUES ($1, '"sent"'::jsonb)
		 ON CONFLICT (key) DO NOTHING`,
		key,
	)
	return err
}

func milestoneReminderKey(milestoneID, due string) string {
	return "milestone_reminder:" + milestoneID + ":" + due
}
