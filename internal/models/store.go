package models

import (
	"context"
	"log/slog"
	"sync"
)

// logPersistError records a repo failure that is being collapsed into
// ErrPersistFailed for the HTTP layer. Centralised so operators get
// consistent logging on every cache/DB divergence.
func logPersistError(err error) {
	slog.Error("persist failed", "error", err)
}

// Store is the MVC model layer: entities plus optional PostgreSQL persistence.
// Mutation methods write to the repo first; the in-memory state is updated
// only on persist success, so the cache never diverges from durable storage.
type Store struct {
	mu        sync.RWMutex
	Workspace Workspace
	Frontend  FrontendStore
	repo      Repository
}

// persistCtx returns the background context mutations use to write to the
// repo. Today we use Background; once mutation methods accept a ctx (see
// follow-up issue) this becomes the request ctx.
func (s *Store) persistCtx() context.Context { return context.Background() }

// hasRepo reports whether the store has a backing repository (true in prod,
// false in unit tests that exercise the in-memory store directly).
func (s *Store) hasRepo() bool { return s.repo != nil }

// GetWorkspace returns the in-memory workspace UNFILTERED. Use
// GetWorkspaceForSession in handlers — only the bulk PUT path and the
// persistence layer should ever see the raw struct.
func (s *Store) GetWorkspace() Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Workspace
}

// GetFrontend returns the in-memory frontend store UNFILTERED. See
// GetFrontendForSession.
func (s *Store) GetFrontend() FrontendStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Frontend
}

// GetWorkspaceForSession returns the workspace projected through the caller's
// permissions and contractor scope.
func (s *Store) GetWorkspaceForSession(sess Session) Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return FilterWorkspace(s.Workspace, sess)
}

// GetFrontendForSession returns the frontend store projected through the
// caller's permissions.
func (s *Store) GetFrontendForSession(sess Session) FrontendStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return FilterFrontend(s.Frontend, sess)
}

func (s *Store) ListContracts() []Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Contract, len(s.Workspace.Contracts))
	copy(out, s.Workspace.Contracts)
	return out
}

func (s *Store) GetContract(no string) (_ Contract, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.Workspace.Contracts {
		if c.No == no {
			return c, nil
		}
	}
	return Contract{}, ErrNotFound
}

func (s *Store) insertContract(c Contract) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.Workspace.Contracts {
		if existing.No == c.No {
			return ErrConflict
		}
	}
	if s.hasRepo() {
		if err := s.repo.InsertContract(s.persistCtx(), c); err != nil {
			return wrapPersist(err)
		}
	}
	s.Workspace.Contracts = append(s.Workspace.Contracts, c)
	s.persistZoneCountsForLocked(c.Zone)
	return nil
}

func (s *Store) updateContract(no string, fn func(*Contract) error) (Contract, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Workspace.Contracts {
		if s.Workspace.Contracts[i].No != no {
			continue
		}
		// Apply mutation to a copy first so a repo failure doesn't leave the
		// cache half-updated.
		updated := s.Workspace.Contracts[i]
		oldZone := updated.Zone
		if err := fn(&updated); err != nil {
			return Contract{}, err
		}
		if s.hasRepo() {
			if err := s.repo.UpdateContract(s.persistCtx(), updated); err != nil {
				return Contract{}, wrapPersist(err)
			}
		}
		s.Workspace.Contracts[i] = updated
		if updated.Zone != oldZone {
			s.persistZoneCountsForLocked(oldZone, updated.Zone)
		} else {
			s.persistZoneCountsForLocked(updated.Zone)
		}
		return s.Workspace.Contracts[i], nil
	}
	return Contract{}, ErrNotFound
}

func (s *Store) deleteContract(no string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.Workspace.Contracts {
		if c.No != no {
			continue
		}
		if s.hasRepo() {
			if err := s.repo.DeleteContract(s.persistCtx(), no); err != nil {
				return wrapPersist(err)
			}
		}
		s.Workspace.Contracts = append(s.Workspace.Contracts[:i], s.Workspace.Contracts[i+1:]...)
		s.persistZoneCountsForLocked(c.Zone)
		return nil
	}
	return ErrNotFound
}

// persistZoneCountsForLocked recomputes the in-memory zone.Contracts counter
// for every affected zone and, if a repo is wired, writes the new count(s)
// to the zones table. Caller must hold s.mu.Lock.
//
// Failures are logged but do not roll back the in-memory update: the counts
// are derived data and the next mutation will overwrite them.
func (s *Store) persistZoneCountsForLocked(zones ...string) {
	recalcZoneCounts(&s.Workspace)
	if !s.hasRepo() {
		return
	}
	seen := map[string]struct{}{}
	for _, code := range zones {
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		for _, z := range s.Workspace.Zones {
			if z.Code != code {
				continue
			}
			_ = s.repo.UpdateZoneCount(s.persistCtx(), code, z.Contracts)
			break
		}
	}
}

// wrapPersist normalises repo errors. ErrNotFound and ErrConflict from the
// repo are returned as-is so the HTTP layer renders the right status.
// Anything else collapses to ErrPersistFailed; the underlying error is
// logged so operators see the actual database failure.
func wrapPersist(err error) error {
	if err == nil {
		return nil
	}
	if err == ErrNotFound || err == ErrConflict || err == ErrValidation {
		return err
	}
	logPersistError(err)
	return ErrPersistFailed
}

func (s *Store) nextContractNo() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return nextContractNo(s.Workspace.Contracts)
}
