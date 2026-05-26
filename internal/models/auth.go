package models

import (
	"context"
	"fmt"
	"time"
)

const auditRetention = 120

// Bootstrap returns the workspace+frontend snapshot the UI mounts. Requires
// a valid platform session on the request context (enforced by middleware).
func (s *Store) BootstrapForRequest(ctx context.Context) BootstrapResponse {
	return s.BootstrapCtx(ctx)
}

func (s *Store) BootstrapCtx(ctx context.Context) BootstrapResponse {
	sess := s.sessionFromCtx(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return BootstrapResponse{
		Session:     sess,
		Workspace:   FilterWorkspace(s.Workspace, sess),
		Frontend:    FilterFrontend(s.Frontend, sess),
		Permissions: s.permissionContextForLocked(sess),
	}
}

func (s *Store) permissionContextForLocked(sess Session) PermissionContext {
	return PermissionContext{
		Role:           sess.Role,
		Email:          sess.Email,
		Permissions:    sess.Permissions,
		CanMutate:      sessionCanMutate(sess),
		CanManageRoles: sessionCanManageRoles(sess),
		IsPortal:       sess.Role == "contractor",
	}
}

// ReplaceWorkspace overwrites the entire workspace via the snapshot path —
// super_admin-only, used by PUT /v1/workspace.
func (s *Store) ReplaceWorkspace(ws Workspace) error {
	s.mu.Lock()
	s.Workspace = ws
	recalcZoneCounts(&s.Workspace)
	wsCopy := s.Workspace
	feCopy := s.Frontend
	s.mu.Unlock()
	if s.hasRepo() {
		if err := s.repo.SaveState(s.persistCtx(), wsCopy, feCopy); err != nil {
			return wrapPersist(err)
		}
	}
	return nil
}

// ReplaceFrontend overwrites the frontend store via the snapshot path —
// super_admin-only, used by PUT /v1/frontend.
//
// The audit log is preserved from the prior state regardless of what the
// caller submits: a super_admin bulk PUT should not be able to silently
// wipe history, even if their submitted snapshot omits the audit array.
func (s *Store) ReplaceFrontend(fe FrontendStore) error {
	s.mu.Lock()
	if fe.ProfilePhotos == nil {
		fe.ProfilePhotos = map[string]string{}
	}
	if fe.Assistance == nil {
		fe.Assistance = []AssistanceMessage{}
	}
	fe.Audit = append([]AuditEntry(nil), s.Frontend.Audit...)
	s.Frontend = fe
	wsCopy := s.Workspace
	feCopy := s.Frontend
	s.mu.Unlock()
	if s.hasRepo() {
		if err := s.repo.SaveState(s.persistCtx(), wsCopy, feCopy); err != nil {
			return wrapPersist(err)
		}
	}
	return nil
}

// appendAudit writes a new audit entry to the DB and updates the in-memory
// ring (capped at auditRetention rows). DB failures are logged but do not
// abort the in-memory append — the caller (often inside an already-running
// mutation) shouldn't fail just because the audit row couldn't be persisted.
func (s *Store) appendAudit(action, detail, user string) {
	now := time.Now()
	// UnixNano gives a virtually-unique surrogate even under concurrent
	// appends; the InsertAuditAndTrim INSERT carries ON CONFLICT DO NOTHING
	// as a final safety net for the extremely rare same-nanosecond collision.
	entry := AuditEntry{
		ID:     fmt.Sprintf("AUD-%d", now.UnixNano()),
		At:     now.Format("2006-01-02 15:04"),
		User:   user,
		Action: action,
		Detail: detail,
	}
	if s.hasRepo() {
		if err := s.repo.InsertAuditAndTrim(s.persistCtx(), entry, auditRetention); err != nil {
			logPersistError(err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	audit := append([]AuditEntry{entry}, s.Frontend.Audit...)
	if len(audit) > auditRetention {
		audit = audit[:auditRetention]
	}
	s.Frontend.Audit = audit
}

// RecordBulkReplace writes a high-severity audit entry whenever a super_admin
// invokes one of the destructive whole-snapshot endpoints (PUT /v1/workspace,
// PUT /v1/frontend).
func (s *Store) RecordBulkReplace(target, user string) {
	if target == "" {
		target = "snapshot"
	}
	if user == "" {
		user = "unknown"
	}
	s.appendAudit("BulkReplace", "PUT /v1/"+target+" by "+user, user)
}
