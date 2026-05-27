// Command jobs runs scheduled contract-management maintenance (milestone reminders).
//
//	DATABASE_URL=... EVENT_BUS_ENABLED=true KAFKA_BROKERS=localhost:19092 go run ./cmd/jobs --milestone-reminders
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/jobs"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

func main() {
	milestoneReminders := flag.Bool("milestone-reminders", false, "publish contracts.milestone.due_soon for milestones due within MILESTONE_REMINDER_DAYS")
	flag.Parse()

	if !*milestoneReminders {
		fmt.Fprintln(os.Stderr, "specify --milestone-reminders")
		flag.Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pg, err := persistence.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer pg.Close()

	bus := events.NewFromEnv()
	defer func() { _ = bus.Close() }()

	if _, err := jobs.RunMilestoneReminders(ctx, pg, bus); err != nil {
		log.Fatalf("milestone reminders: %v", err)
	}
}
