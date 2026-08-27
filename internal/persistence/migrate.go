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
	// This service owns the `contracts` schema on the shared Railway database.
	// The ledger is schema-qualified so it can never collide with another
	// service's global public.schema_migrations (the collision that caused
	// cross-service boot failures). Connect pins search_path to `contracts,
	// public`.
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS contracts`); err != nil {
		return fmt.Errorf("create schema contracts: %w", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS contracts.schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE contracts.schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure schema_migrations.checksum: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var files []string
	var versions []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, name)
		versions = append(versions, strings.TrimSuffix(name, ".up.sql"))
	}
	sort.Strings(files)

	// One-time cutover from the shared global public.schema_migrations: mark this
	// service's already-applied versions as applied in the per-service ledger so
	// migrations are not re-run against tables that already exist in public.
	// No-op on a fresh database.
	if err := seedFromLegacyLedger(ctx, pool, versions); err != nil {
		return fmt.Errorf("seed from legacy ledger: %w", err)
	}

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM contracts.schema_migrations WHERE version = $1)`,
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
			`INSERT INTO contracts.schema_migrations (version, checksum) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
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

// seedFromLegacyLedger copies this service's shipped migration versions from a
// legacy global public.schema_migrations (if present) into the per-service
// ledger. It references only the `version` column and is idempotent via ON
// CONFLICT, so it is safe to keep running. No-op on a fresh database.
func seedFromLegacyLedger(ctx context.Context, pool *pgxpool.Pool, versions []string) error {
	if len(versions) == 0 {
		return nil
	}
	var hasLegacy bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)`).Scan(&hasLegacy); err != nil {
		return err
	}
	if !hasLegacy {
		return nil
	}

	// public.schema_migrations is a SHARED ledger: every service that predates the
	// per-service cutover wrote its versions into it, unscoped. A version string
	// found there does not necessarily belong to THIS service - names like
	// '0001_initial' are used by several of them. Seeding on a bare name match would
	// stamp a migration as applied that never ran here, and the tables it creates
	// would silently never exist.
	//
	// The cutover this function exists for only makes sense on a database that has
	// actually run this service before, and such a database necessarily has its
	// tables. A database with none is either brand new or has never hosted this
	// service, and its rows in the shared ledger belong to somebody else.
	var hasOwnTables bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'contracts' AND table_name <> 'schema_migrations'
		)`).Scan(&hasOwnTables); err != nil {
		return err
	}
	if !hasOwnTables {
		return nil
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO contracts.schema_migrations (version)
		SELECT version FROM public.schema_migrations WHERE version = ANY($1)
		ON CONFLICT (version) DO NOTHING`, versions)
	return err
}
