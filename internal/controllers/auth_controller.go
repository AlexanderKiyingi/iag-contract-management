package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/config"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// AuthController exposes the read-only session surface that survived the
// platform cutover. Login/refresh/logout are owned by the authentication
// service at /api/v1/authentication/oauth/token; calling those endpoints
// here would be confusing, so they're removed.
type AuthController struct {
	model *models.Store
	cfg   config.Config
}

// NewAuthController constructs an AuthController.
func NewAuthController(model *models.Store, cfg config.Config) *AuthController {
	return &AuthController{model: model, cfg: cfg}
}

// Bootstrap returns the full workspace snapshot for the calling user.
// Auth is enforced by the platform middleware; this handler is no longer public.
func (c *AuthController) Bootstrap(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, c.model.BootstrapForRequest(r.Context()))
}

// Session returns just the current session + permission context.
func (c *AuthController) Session(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	views.JSON(w, http.StatusOK, models.SessionResponse{
		Session:     c.model.SessionFromRequest(ctx),
		Permissions: c.model.PermissionContextFor(ctx),
	})
}
