package persistence

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	platformdb "github.com/alvor-technologies/iag-platform-go/db"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Postgres, error) {
	// The schema was pinned with an AfterConnect `SET search_path`, which is
	// right against Postgres directly and silently wrong behind PgBouncer in
	// transaction pooling mode: the SET binds to one server connection while the
	// next transaction may be handed another, so this service's tables would be
	// resolved in whatever schema that connection happened to carry. The shared
	// package applies it as a startup parameter, which the pooler tracks per
	// server connection.
	//
	// The DSN is chosen before the config is built, so an explicit argument is
	// honoured even when DATABASE_URL is unset — how tests and CLI tools call
	// this.
	pcfg := platformdb.ConfigFromEnv("contracts, public")
	if strings.TrimSpace(databaseURL) != "" {
		pcfg.URL = databaseURL
	}
	pool, err := platformdb.Connect(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
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
