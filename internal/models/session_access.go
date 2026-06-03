package models

import "context"

// sessionFromCtx returns the platform-derived session installed by the auth
// middleware. There is no in-memory fallback: a missing session means an
// unauthenticated request slipped through, and we treat it as an empty
// session (no permissions, no role).
func (s *Store) sessionFromCtx(ctx context.Context) Session {
	if sess, ok := RequestSession(ctx); ok {
		return sess
	}
	return Session{}
}

// RequirePermissionCtx returns ErrForbidden unless the caller carries key.
func (s *Store) RequirePermissionCtx(ctx context.Context, key string) error {
	if !s.HasPermissionCtx(ctx, key) {
		return ErrForbidden
	}
	return nil
}

// HasPermissionCtx returns true if the caller's JWT permissions array contains
// key, or if the caller is a super_admin/admin (which the auth service signals
// either via the corresponding group or via an explicit wildcard).
func (s *Store) HasPermissionCtx(ctx context.Context, key string) bool {
	sess := s.sessionFromCtx(ctx)
	return sessionHasPermission(sess, key)
}

func (s *Store) CanMutateCtx(ctx context.Context) bool {
	sess := s.sessionFromCtx(ctx)
	return sessionCanMutate(sess)
}

func (s *Store) CanManageRolesCtx(ctx context.Context) bool {
	sess := s.sessionFromCtx(ctx)
	return sessionCanManageRoles(sess)
}

func (s *Store) CanEditContractCtx(ctx context.Context, contractNo string) bool {
	sess := s.sessionFromCtx(ctx)
	if sessionCanMutate(sess) {
		return true
	}
	// Contractors are scoped: they can only edit contracts whose supervisor
	// matches their contractor_supervisors mapping.
	if sess.Role != "contractor" || sess.ContractorSup == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.Workspace.Contracts {
		if c.No == contractNo && c.Sup == *sess.ContractorSup {
			return true
		}
	}
	return false
}

// PermissionContextFor builds the front-end-friendly view of what the caller can do.
func (s *Store) PermissionContextFor(ctx context.Context) PermissionContext {
	sess := s.sessionFromCtx(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.permissionContextForLocked(sess)
}

func (s *Store) CheckPermissionsCtx(ctx context.Context, keys []string) map[string]bool {
	sess := s.sessionFromCtx(ctx)
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = sessionHasPermission(sess, k)
	}
	return out
}

func (s *Store) SessionFromRequest(ctx context.Context) Session { return s.sessionFromCtx(ctx) }

func (s *Store) GetSessionCtx(ctx context.Context) Session { return s.sessionFromCtx(ctx) }

func (s *Store) PermissionContextCtx(ctx context.Context) PermissionContext {
	return s.PermissionContextFor(ctx)
}

// sessionHasPermission encapsulates the role+permissions decision so all the
// Has*/Can* helpers stay consistent.
func sessionHasPermission(sess Session, key string) bool {
	if sess.Role == "super_admin" || sess.Role == "admin" {
		return true
	}
	for _, p := range sess.Permissions {
		if p == key || p == "*" {
			return true
		}
	}
	if aliases, ok := legacyAliases[key]; ok {
		for _, alias := range aliases {
			for _, p := range sess.Permissions {
				if p == alias {
					return true
				}
			}
		}
	}
	return false
}

func sessionCanMutate(sess Session) bool {
	switch sess.Role {
	case "super_admin", "admin", "manager":
		return true
	}
	return sessionHasPermission(sess, "contracts.update")
}

func sessionCanManageRoles(sess Session) bool {
	switch sess.Role {
	case "super_admin", "admin":
		return true
	}
	return sessionHasPermission(sess, "roles.update") ||
		sessionHasPermission(sess, "roles.create") ||
		sessionHasPermission(sess, "roles.manage")
}
