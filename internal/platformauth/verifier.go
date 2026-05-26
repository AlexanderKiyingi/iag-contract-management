// Package platformauth wraps platform-go's JWKS verifier so the rest of the
// contract-management codebase can stay on the local Session shape.
package platformauth

import (
	"context"
	"time"

	platformauthclient "github.com/alvor-technologies/iag-platform-go/authclient"
)

// Claims aliases the platform claims type — fields stay identical wire-format.
type Claims = platformauthclient.Claims

// Verifier is a thin wrapper around platform-go's Verifier.
type Verifier struct {
	inner *platformauthclient.Verifier
}

// NewVerifier constructs a Verifier that enforces the supplied audience
// (typically "iag.contract-management").
func NewVerifier(jwksURL, issuer, audience string) *Verifier {
	return &Verifier{
		inner: platformauthclient.NewVerifier(platformauthclient.Options{
			JWKSURL:  jwksURL,
			Issuer:   issuer,
			Audience: audience,
		}),
	}
}

// Refresh fetches the JWKS.
func (v *Verifier) Refresh(ctx context.Context) error { return v.inner.Refresh(ctx) }

// StartRefreshLoop periodically refreshes the JWKS until ctx is cancelled.
func (v *Verifier) StartRefreshLoop(ctx context.Context, interval time.Duration) {
	v.inner.StartRefreshLoop(ctx, interval)
}

// Verify validates a Bearer access token and returns its claims.
func (v *Verifier) Verify(token string) (*Claims, error) { return v.inner.Verify(token) }
