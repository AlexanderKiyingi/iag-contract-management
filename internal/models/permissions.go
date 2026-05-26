// Package-internal permission catalogue and metadata. Authoritative permission
// decisions are made by sessionHasPermission in session_access.go using the
// caller's JWT claims; this file just owns the keys, labels, and legacy
// aliases used by the UI catalog endpoint and the permissions-register
// payload posted to the auth service at startup.
package models

import (
	"context"
	"strings"
)

type PermissionModule struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type CrudAction struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type PermissionCatalog struct {
	Modules       []PermissionModule  `json:"modules"`
	Actions       []CrudAction        `json:"actions"`
	AllKeys       []string            `json:"allKeys"`
	LegacyAliases map[string][]string `json:"legacyAliases"`
	RoleLabels    map[string]string   `json:"roleLabels"`
	BuiltinRoles  []string            `json:"builtinRoles"`
}

type PermissionContext struct {
	Role           string   `json:"role"`
	Email          string   `json:"email"`
	CustomRoleID   *string  `json:"customRoleId,omitempty"`
	CustomRoleName string   `json:"customRoleName,omitempty"`
	Permissions    []string `json:"permissions"`
	CanMutate      bool     `json:"canMutate"`
	CanManageRoles bool     `json:"canManageRoles"`
	IsPortal       bool     `json:"isPortal"`
}

type PermissionCheckInput struct {
	Keys []string `json:"keys"`
}

type PermissionCheckResult struct {
	Allowed map[string]bool `json:"allowed"`
}

// PermissionDescriptor mirrors platform-go/serviceauth.Permission and is what
// we post to the auth service's /v1/permissions/register endpoint at startup.
type PermissionDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var (
	permissionModules = []PermissionModule{
		{ID: "contracts", Label: "Contracts"},
		{ID: "zones", Label: "Zones"},
		{ID: "payments", Label: "Payments"},
		{ID: "tasks", Label: "Tasks"},
		{ID: "milestones", Label: "Milestones"},
		{ID: "materials", Label: "Materials"},
		{ID: "users", Label: "Users"},
		{ID: "roles", Label: "Roles"},
		{ID: "audit", Label: "Audit log"},
		{ID: "reports", Label: "Reports"},
		{ID: "insights", Label: "AI & insights"},
	}
	crudActions = []CrudAction{
		{Key: "create", Label: "Create"},
		{Key: "read", Label: "Read"},
		{Key: "update", Label: "Update"},
		{Key: "delete", Label: "Delete"},
	}
	roleLabels = map[string]string{
		"super_admin": "Super admin",
		"admin":       "Administrator",
		"manager":     "Manager",
		"viewer":      "Viewer",
		"contractor":  "Contractor",
	}
	builtinRoles  = []string{"super_admin", "admin", "manager", "viewer", "contractor"}
	legacyAliases = map[string][]string{
		"portfolio.view":    {"contracts.read", "zones.read"},
		"portfolio.edit":    {"contracts.create", "contracts.read", "contracts.update", "zones.read", "zones.update"},
		"portfolio.delete":  {"contracts.delete"},
		"payments.view":     {"payments.read"},
		"tasks.manage":      {"tasks.create", "tasks.read", "tasks.update", "tasks.delete"},
		"milestones.manage": {"milestones.create", "milestones.read", "milestones.update", "milestones.delete"},
		"materials.manage":  {"materials.create", "materials.read", "materials.update", "materials.delete"},
		"users.manage":      {"users.create", "users.read", "users.update", "users.delete"},
		"roles.manage":      {"roles.create", "roles.read", "roles.update", "roles.delete"},
		"audit.view":        {"audit.read"},
		"reports.export":    {"reports.read", "reports.create"},
		"insights.run":      {"insights.read", "insights.update"},
	}
)

var allPermissionKeys = buildAllPermissionKeys()

func buildAllPermissionKeys() []string {
	keys := make([]string, 0, len(permissionModules)*len(crudActions))
	for _, m := range permissionModules {
		for _, a := range crudActions {
			keys = append(keys, m.ID+"."+a.Key)
		}
	}
	return keys
}

// PermissionCatalogData is served at GET /v1/permissions/catalog.
func PermissionCatalogData() PermissionCatalog {
	return PermissionCatalog{
		Modules:       permissionModules,
		Actions:       crudActions,
		AllKeys:       allPermissionKeys,
		LegacyAliases: legacyAliases,
		RoleLabels:    roleLabels,
		BuiltinRoles:  builtinRoles,
	}
}

// PermissionDescriptors returns this service's full permission catalogue in
// the shape expected by the auth service's /v1/permissions/register endpoint.
func PermissionDescriptors() []PermissionDescriptor {
	out := make([]PermissionDescriptor, 0, len(allPermissionKeys))
	for _, m := range permissionModules {
		for _, a := range crudActions {
			out = append(out, PermissionDescriptor{
				Name:        m.ID + "." + a.Key,
				Description: a.Label + " " + strings.ToLower(m.Label),
			})
		}
	}
	return out
}

// NormalizePermissions resolves legacy aliases against the canonical key set.
// Used by the custom-roles entity (which still lives in the workspace for
// historical reasons; runtime auth is JWT-only).
func NormalizePermissions(perms []string) []string {
	if len(perms) == 0 {
		return []string{}
	}
	for _, p := range perms {
		if p == "*" {
			return append([]string{}, allPermissionKeys...)
		}
	}
	out := make(map[string]struct{})
	for _, p := range perms {
		if aliases, ok := legacyAliases[p]; ok {
			for _, a := range aliases {
				out[a] = struct{}{}
			}
			continue
		}
		for _, k := range allPermissionKeys {
			if k == p {
				out[k] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(out))
	for k := range out {
		result = append(result, k)
	}
	return result
}

// EffectivePermissionsForUser returns a descriptive set of permissions for
// the named workspace user. Post-cutover this is advisory only — the
// authoritative permission set lives in the auth service.
func (s *Store) EffectivePermissionsForUser(userID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.Workspace.WorkspaceUsers {
		u := &s.Workspace.WorkspaceUsers[i]
		if u.ID != userID {
			continue
		}
		if u.CustomRoleID != nil {
			if cr := s.getCustomRoleLocked(*u.CustomRoleID); cr != nil && len(cr.Permissions) > 0 {
				return NormalizePermissions(cr.Permissions), nil
			}
		}
		return permissionsForRole(u.Role), nil
	}
	return nil, ErrNotFound
}

func (s *Store) getCustomRoleLocked(id string) *CustomRole {
	for i := range s.Frontend.CustomRoles {
		if s.Frontend.CustomRoles[i].ID == id {
			return &s.Frontend.CustomRoles[i]
		}
	}
	return nil
}

func (s *Store) getUserByEmail(email string) *WorkspaceUser {
	email = strings.ToLower(strings.TrimSpace(email))
	for i := range s.Workspace.WorkspaceUsers {
		if strings.EqualFold(s.Workspace.WorkspaceUsers[i].Email, email) {
			return &s.Workspace.WorkspaceUsers[i]
		}
	}
	return nil
}

// permissionsForRole is the descriptive (UI-only) view of what a role
// "normally" carries. The actual enforcement is via JWT claims.
func permissionsForRole(role string) []string {
	switch role {
	case "super_admin", "admin":
		return append([]string{}, allPermissionKeys...)
	case "viewer":
		out := make([]string, 0, len(permissionModules))
		for _, m := range permissionModules {
			out = append(out, m.ID+".read")
		}
		return out
	case "manager":
		var out []string
		add := func(module string, actions ...string) {
			for _, a := range actions {
				out = append(out, module+"."+a)
			}
		}
		add("contracts", "create", "read", "update")
		add("zones", "read", "update")
		add("payments", "read")
		add("tasks", "create", "read", "update", "delete")
		add("milestones", "create", "read", "update", "delete")
		add("materials", "create", "read", "update", "delete")
		add("users", "read")
		add("roles", "read")
		add("audit", "read")
		add("reports", "read", "create")
		add("insights", "read", "update")
		return out
	case "contractor":
		return []string{"contracts.read"}
	}
	return nil
}

// Legacy wrappers kept so call-sites don't need updates.
func (s *Store) HasPermission(key string) bool  { return s.HasPermissionCtx(context.Background(), key) }
func (s *Store) CanMutate() bool                { return s.CanMutateCtx(context.Background()) }
func (s *Store) CanManageRoles() bool           { return s.CanManageRolesCtx(context.Background()) }
func (s *Store) CanEditContract(no string) bool { return s.CanEditContractCtx(context.Background(), no) }
func (s *Store) PermissionContext() PermissionContext {
	return s.PermissionContextFor(context.Background())
}
func (s *Store) CheckPermissions(keys []string) map[string]bool {
	return s.CheckPermissionsCtx(context.Background(), keys)
}
func (s *Store) RequirePermission(key string) error {
	return s.RequirePermissionCtx(context.Background(), key)
}

// BuiltinRolesPermissions feeds the GET /v1/permissions/builtin endpoint
// (advisory data for the UI's role picker).
type BuiltinRolePermissions struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func BuiltinRolesPermissions() []BuiltinRolePermissions {
	out := make([]BuiltinRolePermissions, 0, len(builtinRoles))
	for _, role := range builtinRoles {
		out = append(out, BuiltinRolePermissions{
			Role:        role,
			Permissions: permissionsForRole(role),
		})
	}
	return out
}
