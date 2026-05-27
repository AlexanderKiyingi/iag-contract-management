package jobs

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

// RunMilestoneReminders publishes due-soon events for milestones within the configured window.
func RunMilestoneReminders(ctx context.Context, pg *persistence.Postgres, bus *events.Bus) (int, error) {
	if pg == nil {
		return 0, nil
	}
	days := milestoneReminderDays()
	list, err := pg.ListMilestonesDueSoon(ctx, days)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, m := range list {
		already, err := pg.WasMilestoneReminderSent(ctx, m.ID, m.Due)
		if err != nil {
			slog.Warn("milestone reminder check failed", "id", m.ID, "err", err)
			continue
		}
		if already {
			continue
		}
		eventID := "milestone-due-" + m.ID + "-" + m.Due
		events.PublishMilestoneDueSoon(ctx, bus, m, eventID)
		if err := pg.MarkMilestoneReminderSent(ctx, m.ID, m.Due); err != nil {
			slog.Warn("milestone reminder mark failed", "id", m.ID, "err", err)
		}
		sent++
	}
	slog.Info("milestone reminders processed", "candidates", len(list), "sent", sent, "within_days", days)
	return sent, nil
}

func milestoneReminderDays() int {
	raw := os.Getenv("MILESTONE_REMINDER_DAYS")
	if raw == "" {
		return 7
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 7
	}
	return n
}
