package app

import (
	"github.com/alvor-technologies/iag-contract-management/internal/config"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

// buildStoreFrom hydrates the model store from an already-connected Postgres
// pool. The caller owns the pool's lifecycle (Close at shutdown).
func buildStoreFrom(_ config.Config, pg *persistence.Postgres) *models.Store {
	return models.NewStore(&models.StoreOptions{Repo: pg})
}
