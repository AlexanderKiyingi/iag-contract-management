package models

import (
	"context"
	"log/slog"
)

// StoreOptions configures the optional PostgreSQL repository.
type StoreOptions struct {
	// Repo is the combined snapshot + per-entity repository (typically the
	// persistence.Postgres). Optional — if nil, the store stays in-memory
	// (dev only).
	Repo Repository

	// SeedOnStartup writes the demo workspace to the DB when it's empty.
	// Defaults to false; the caller (bootstrap) wires this from config so the
	// production environment ships with the seed disabled by default. See
	// config.Config.SeedOnStartup for the rationale (legacy schema drift).
	SeedOnStartup bool
}

// NewStore creates the model store, optionally hydrated from PostgreSQL.
// When SeedOnStartup is true and the DB is empty, the demo workspace is
// written so /v1/bootstrap returns useful sample data on first run. The
// state is always reloaded from the DB after that so DB-generated surrogates
// (e.g. task_projects.id) land in the in-memory cache.
func NewStore(opts *StoreOptions) *Store {
	s := &Store{}
	seedOnStartup := false
	if opts != nil {
		s.repo = opts.Repo
		seedOnStartup = opts.SeedOnStartup
	}
	seed(s)

	if s.repo == nil {
		return s
	}

	ctx := context.Background()
	if seedOnStartup {
		empty, err := s.repo.IsEmpty(ctx)
		if err != nil {
			slog.Warn("postgres empty check failed", "error", err)
			return s
		}
		if empty {
			if err := s.repo.SaveState(ctx, s.Workspace, s.Frontend); err != nil {
				slog.Warn("postgres initial seed write failed", "error", err)
				return s
			}
		}
	}
	if ws, fe, err := s.repo.LoadState(ctx); err != nil {
		slog.Warn("postgres load failed", "error", err)
	} else {
		s.mu.Lock()
		s.Workspace = ws
		s.Frontend = fe
		if s.Frontend.ProfilePhotos == nil {
			s.Frontend.ProfilePhotos = map[string]string{}
		}
		s.mu.Unlock()
	}
	return s
}
