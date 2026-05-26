package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

func requirePerm(ctx context.Context, store *models.Store, w http.ResponseWriter, perm string) bool {
	if err := store.RequirePermissionCtx(ctx, perm); err != nil {
		views.WriteError(w, err)
		return false
	}
	return true
}

func requireMutate(ctx context.Context, store *models.Store, w http.ResponseWriter) bool {
	if !store.CanMutateCtx(ctx) {
		views.WriteError(w, models.ErrForbidden)
		return false
	}
	return true
}

func requireManageRoles(ctx context.Context, store *models.Store, w http.ResponseWriter) bool {
	if !store.CanManageRolesCtx(ctx) {
		views.WriteError(w, models.ErrForbidden)
		return false
	}
	return true
}

// requireSuperAdmin gates destructive whole-snapshot operations (bulk
// PUT /v1/workspace and PUT /v1/frontend). Permits only role=super_admin —
// admin is NOT enough, since the bulk PUT can overwrite the audit log.
func requireSuperAdmin(ctx context.Context, store *models.Store, w http.ResponseWriter) bool {
	sess := store.SessionFromRequest(ctx)
	if sess.Role != "super_admin" {
		views.WriteError(w, models.ErrForbidden)
		return false
	}
	return true
}

func canEditProfile(ctx context.Context, store *models.Store, email string) bool {
	sess := store.SessionFromRequest(ctx)
	if strings.EqualFold(sess.Email, email) {
		return true
	}
	return store.HasPermissionCtx(ctx, "users.update") || store.CanMutateCtx(ctx)
}
