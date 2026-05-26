package middleware

import (
	"context"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// SessionFrom reads the authenticated session from the request context.
func SessionFrom(ctx context.Context) (models.Session, bool) {
	return models.RequestSession(ctx)
}
