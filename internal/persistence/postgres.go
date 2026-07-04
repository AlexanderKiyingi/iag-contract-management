package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	// Resolve unqualified names to this service's own schema first, falling back
	// to public so legacy tables that still live there keep resolving. On the
	// shared Railway database this isolates contract-management from the global
	// public namespace and its single global schema_migrations ledger.
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET search_path TO contracts, public`)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	pg := &Postgres{Pool: pool}
	if err := RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pg, nil
}

func (p *Postgres) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}

func (p *Postgres) Ping(ctx context.Context) error {
	if p == nil || p.Pool == nil {
		return fmt.Errorf("postgres not configured")
	}
	return p.Pool.Ping(ctx)
}

func (p *Postgres) IsEmpty(ctx context.Context) (bool, error) {
	var n int
	err := p.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM contracts`).Scan(&n)
	if err != nil {
		return true, err
	}
	return n == 0, nil
}
