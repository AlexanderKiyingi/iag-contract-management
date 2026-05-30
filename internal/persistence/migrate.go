package persistence

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies every migrations/*.up.sql file in lex order, tracking
// applied versions in schema_migrations. Each file's basename (minus the
// .up.sql suffix) is the version key.
//
// Each file is sent as a single Exec under the simple query protocol so
// PostgreSQL handles multi-statement bodies (semicolons inside DO blocks,
// function bodies, and string literals all work). pgx's default extended
// protocol rejects multi-statement bodies — the first symptom is a confusing
// "column X does not exist" because PostgreSQL parses only the first
// statement of the file and then runs subsequent index DDL against an
// empty table.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Legacy deployments on Railway have schema_migrations with a NOT NULL
	// `checksum` column from an earlier migration tool. Always ensure the
	// column exists (no-op on legacy DBs) and always supply a value on INSERT
	// so fresh and legacy DBs both succeed. Statements are issued separately
	// because pgx's default extended protocol only handles one statement per
	// Exec.
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure schema_migrations.checksum: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check %s: %w", version, err)
		}
		if applied {
			continue
		}

		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		sum := sha256.Sum256(body)
		checksum := hex.EncodeToString(sum[:])

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		// QueryExecModeSimpleProtocol forces the underlying query into the
		// PostgreSQL simple query protocol, which permits multi-statement
		// SQL bodies. Without this pgx defaults to extended protocol
		// (Parse/Bind/Execute) which is one statement per call.
		if _, err := tx.Exec(ctx, string(body), pgx.QueryExecModeSimpleProtocol); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%s exec: %w", version, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			version, checksum,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", version, err)
		}
	}
	return nil
}
