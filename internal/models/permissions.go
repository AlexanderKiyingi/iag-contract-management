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
		// Contract-governance modules (back the Contract Governance UI). "update"
		// is the approve/advance action for the workflow modules (payments,
		// variations, requisitions).
		{ID: "requisitions", Label: "Requisitions"},
		{ID: "variations", Label: "Variations"},
		{ID: "obligations", Label: "Obligations"},
		{ID: "approvals", Label: "Approval rules"},
		{ID: "templates", Label: "Templates"},
		{ID: "clauses", Label: "Clause library"},
		{ID: "budgets", Label: "Budgets"},
		{ID: "closeout", Label: "Closeout"},
		// Monthly-report modules (back the Construction Department MR workbook).
		{ID: "contractors", Label: "Contractors"},
		{ID: "progressreports", Label: "Progress reports"},
		{ID: "valuations", Label: "Valuations"},
		{ID: "challenges", Label: "Challenges register"},
		{ID: "actionitems", Label: "Action items"},
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
		"portfolio.view":         {"contracts.read", "zones.read"},
		"portfolio.edit":         {"contracts.create", "contracts.read", "contracts.update", "zones.read", "zones.update"},
		"portfolio.delete":       {"contracts.delete"},
		"payments.view":          {"payments.read"},
		"tasks.manage":           {"tasks.create", "tasks.read", "tasks.update", "tasks.delete"},
		"milestones.manage":      {"milestones.create", "milestones.read", "milestones.update", "milestones.delete"},
		"materials.manage":       {"materials.create", "materials.read", "materials.update", "materials.delete"},
		"users.manage":           {"users.create", "users.read", "users.update", "users.delete"},
		"roles.manage":           {"roles.create", "roles.read", "roles.update", "roles.delete"},
		"audit.view":             {"audit.read"},
		"reports.export":         {"reports.read", "reports.create"},
		"insights.run":           {"insights.read", "insights.update"},
		"requisitions.manage":    {"requisitions.create", "requisitions.read", "requisitions.update"},
		"requisitions.approve":   {"requisitions.update"},
		"variations.manage":      {"variations.create", "variations.read", "variations.update"},
		"variations.approve":     {"variations.update"},
		"payments.approve":       {"payments.update"},
		"obligations.manage":     {"obligations.create", "obligations.read", "obligations.update", "obligations.delete"},
		"approvals.manage":       {"approvals.create", "approvals.read", "approvals.update", "approvals.delete"},
		"templates.manage":       {"templates.create", "templates.read", "templates.update", "templates.delete"},
		"clauses.manage":         {"clauses.create", "clauses.read", "clauses.update", "clauses.delete"},
		"budgets.manage":         {"budgets.create", "budgets.read", "budgets.update", "budgets.delete"},
		"closeout.manage":        {"closeout.create", "closeout.read", "closeout.update"},
		"contractors.manage":     {"contractors.create", "contractors.read", "contractors.update", "contractors.delete"},
		"progressreports.manage": {"progressreports.create", "progressreports.read", "progressreports.update", "progressreports.delete"},
		"valuations.manage":      {"valuations.create", "valuations.read", "valuations.update", "valuations.delete"},
		"challenges.manage":      {"challenges.create", "challenges.read", "challenges.update", "challenges.delete"},
		"actionitems.manage":     {"actionitems.create", "actionitems.read", "actionitems.update", "actionitems.delete"},
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

// EnrichSessionFromWorkspace applies workspace custom-role assignments to the
// JWT session. When a custom role is set, effective permissions are the
// intersection of JWT grants and custom-role keys (custom role cannot exceed
// platform grants). Role-based bypasses (super_admin/admin) still apply in
// sessionHasPermission.
func (s *Store) EnrichSessionFromWorkspace(sess Session) Session {
	if sess.Email == "" {
		return sess
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	u := s.getUserByEmail(sess.Email)
	if u == nil {
		return sess
	}
	out := sess
	if u.CustomRoleID != nil {
		if cr := s.getCustomRoleLocked(*u.CustomRoleID); cr != nil && len(cr.Permissions) > 0 {
			custom := NormalizePermissions(cr.Permissions)
			if len(custom) > 0 {
				out.Permissions = intersectPermissionSets(sess.Permissions, custom)
			}
		}
	}
	return out
}

func intersectPermissionSets(jwtPerms, customPerms []string) []string {
	if len(customPerms) == 0 {
		return append([]string{}, jwtPerms...)
	}
	if len(jwtPerms) == 0 {
		return append([]string{}, customPerms...)
	}
	jwtSet := make(map[string]struct{}, len(jwtPerms))
	for _, p := range jwtPerms {
		if p == "*" {
			return append([]string{}, customPerms...)
		}
		jwtSet[p] = struct{}{}
	}
	out := make([]string, 0, len(customPerms))
	for _, p := range customPerms {
		if _, ok := jwtSet[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

// EffectivePermissionsForUser returns the nominal permission template for a
// workspace user (custom role keys or builtin role template). This is
// advisory for admin UIs — runtime enforcement uses JWT ∩ custom role via
// EnrichSessionFromWorkspace on the caller's own session.
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
		// Governance: managers run the operational modules; config (rules,
		// templates, clauses, budgets) is read-only for them.
		add("requisitions", "create", "read", "update")
		add("variations", "create", "read", "update")
		add("obligations", "create", "read", "update", "delete")
		add("closeout", "create", "read", "update")
		add("approvals", "read")
		add("templates", "read")
		add("clauses", "read")
		add("budgets", "read")
		add("contractors", "create", "read", "update", "delete")
		add("progressreports", "create", "read", "update", "delete")
		add("valuations", "create", "read", "update")
		add("challenges", "create", "read", "update", "delete")
		add("actionitems", "create", "read", "update", "delete")
		return out
	case "contractor":
		return []string{"contracts.read", "milestones.read", "obligations.read"}
	}
	return nil
}

// Legacy wrappers kept so call-sites don't need updates.
func (s *Store) HasPermission(key string) bool { return s.HasPermissionCtx(context.Background(), key) }
func (s *Store) CanMutate() bool               { return s.CanMutateCtx(context.Background()) }
func (s *Store) CanManageRoles() bool          { return s.CanManageRolesCtx(context.Background()) }
func (s *Store) CanEditContract(no string) bool {
	return s.CanEditContractCtx(context.Background(), no)
}
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
