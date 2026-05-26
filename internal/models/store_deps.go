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
}

// NewStore creates the model store, optionally hydrated from PostgreSQL.
// First run with an empty DB writes the seed AND reloads it (so DB-generated
// IDs like task_projects.id propagate to the in-memory cache). Subsequent
// runs load the saved snapshot directly.
func NewStore(opts *StoreOptions) *Store {
	s := &Store{}
	if opts != nil {
		s.repo = opts.Repo
	}
	seed(s)

	if s.repo == nil {
		return s
	}

	ctx := context.Background()
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
	// Always reload after seed/IsEmpty so DB-generated surrogates (e.g.
	// task_projects.id) are present in the in-memory state — per-entity
	// mutations key off them.
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
