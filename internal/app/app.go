package app

import (
	"github.com/alvor-technologies/iag-contract-management/internal/config"
	"github.com/alvor-technologies/iag-contract-management/internal/controllers"
	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

// MVC wires model, view, and controller layers.
type MVC struct {
	Config      config.Config
	Model       *models.Store
	Postgres    *persistence.Postgres
	Health      *controllers.HealthController
	Auth        *controllers.AuthController
	Workspace   *controllers.WorkspaceController
	WsRes       *controllers.WorkspaceResourcesController
	Frontend    *controllers.FrontendController
	FeRes       *controllers.FrontendResourcesController
	Contracts   *controllers.ContractController
	Permissions *controllers.PermissionsController
	Uploads     *controllers.UploadsController
	Exports     *controllers.ExportsController
	Admin       *controllers.AdminController
}

// NewMVC wires the dependency graph from a pre-opened Postgres pool. The
// caller owns the pool's lifecycle.
func NewMVC(cfg config.Config, pg *persistence.Postgres, bus *events.Bus) *MVC {
	store := buildStoreFrom(cfg, pg)
	health := controllers.NewHealthController(pg)
	return &MVC{
		Config:      cfg,
		Model:       store,
		Postgres:    pg,
		Health:      health,
		Auth:        controllers.NewAuthController(store, cfg),
		Workspace:   controllers.NewWorkspaceController(store),
		WsRes:       controllers.NewWorkspaceResourcesController(store),
		Frontend:    controllers.NewFrontendController(store),
		FeRes:       controllers.NewFrontendResourcesController(store, bus),
		Contracts:   controllers.NewContractController(store, bus),
		Permissions: controllers.NewPermissionsController(store),
		Uploads:     controllers.NewUploadsController(store),
		Exports:     controllers.NewExportsController(store),
		Admin:       controllers.NewAdminController(store, pg, bus),
	}
}
