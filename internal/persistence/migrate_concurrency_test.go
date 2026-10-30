package persistence

// Runs only when TEST_DATABASE_URL is set, like the integration tests. The role
// must be allowed to CREATE DATABASE: the test migrates a database of its own
// and drops it afterwards, so it starts empty however the suite is ordered.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Replicas booting together each run RunMigrations against the same database.
// Every one must succeed and each migration must be applied exactly once.
func TestRunMigrationsConcurrentReplicas(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("cm_migrate_concurrency_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	}()

	const replicas = 5
	pools := make([]*pgxpool.Pool, replicas)
	for i := range pools {
		cfg, err := pgxpool.ParseConfig(url)
		if err != nil {
			t.Fatal(err)
		}
		cfg.ConnConfig.Database = name
		// Production's tables sit in public (they predate the contracts schema)
		// and 023_contractor_integrity names them public.* explicitly. With
		// Connect's "contracts, public" a fresh database creates them in
		// contracts instead and 023 fails, with or without concurrency, so the
		// test reproduces production's layout.
		cfg.ConnConfig.RuntimeParams["search_path"] = "public"
		p, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()
		pools[i] = p
	}

	var wg sync.WaitGroup
	errs := make([]error, replicas)
	start := make(chan struct{})
	for i := range pools {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = RunMigrations(ctx, pools[i])
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("replica %d: %v", i, err)
		}
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			want++
		}
	}
	var got int
	if err := pools[0].QueryRow(ctx, `SELECT count(*) FROM contracts.schema_migrations`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ledger holds %d migrations, want %d", got, want)
	}
}
