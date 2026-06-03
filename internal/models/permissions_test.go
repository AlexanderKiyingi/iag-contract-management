package models

import (
	"context"
	"testing"
)

func TestIntersectPermissionSets(t *testing.T) {
	jwt := []string{"contracts.read", "contracts.update", "reports.read"}
	custom := []string{"contracts.read", "contracts.update"}
	got := intersectPermissionSets(jwt, custom)
	if len(got) != 2 {
		t.Fatalf("intersect: got %v want 2", got)
	}
	if intersectPermissionSets(jwt, nil)[0] != "contracts.read" {
		t.Fatal("nil custom should return jwt copy")
	}
	if len(intersectPermissionSets(nil, custom)) != 2 {
		t.Fatal("nil jwt should return custom copy")
	}
}

func TestEnrichSessionFromWorkspaceCustomRole(t *testing.T) {
	s := NewStore(nil)
	crID := "cr-1"
	s.mu.Lock()
	s.Frontend.CustomRoles = []CustomRole{
		{ID: crID, Name: "Ops", Permissions: []string{"contracts.read", "contracts.update", "contracts.delete"}},
	}
	s.Workspace.WorkspaceUsers = []WorkspaceUser{{Email: "u@test.com", CustomRoleID: &crID}}
	s.mu.Unlock()

	sess := Session{
		Email:       "u@test.com",
		Role:        "manager",
		Permissions: []string{"contracts.read", "contracts.update", "reports.read"},
	}
	enriched := s.EnrichSessionFromWorkspace(sess)
	if len(enriched.Permissions) != 2 {
		t.Fatalf("enriched perms %v want view+change only", enriched.Permissions)
	}
}

func TestPermissionContextFromJWTSession(t *testing.T) {
	s := NewStore(nil)
	admin := Session{
		Email:       "admin@inspireafrica.test",
		Role:        "super_admin",
		DisplayName: "Admin",
	}
	ctx := WithRequestSession(context.Background(), admin)
	if !s.CanMutateCtx(ctx) {
		t.Fatal("super_admin should mutate")
	}
	viewer := Session{Email: "v@test.com", Role: "viewer", DisplayName: "V"}
	ctx2 := WithRequestSession(context.Background(), viewer)
	if s.CanMutateCtx(ctx2) {
		t.Fatal("viewer should not mutate")
	}
}
