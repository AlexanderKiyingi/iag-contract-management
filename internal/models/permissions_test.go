package models

import (
	"context"
	"testing"
)

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
