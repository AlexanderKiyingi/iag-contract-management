package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/platformauth"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

// ContractorLookup resolves a caller's contractor-supervisor binding (if any).
// Returns the supervisor name and true if the email is a contractor; false otherwise.
type ContractorLookup interface {
	ContractorSupervisor(ctx context.Context, email string) (string, bool, error)
	// GovContractorLinked reports whether a governance contractor is bound to
	// the platform user id (JWT subject).
	GovContractorLinked(ctx context.Context, userID string) (bool, error)
}

// GinPlatformAuth verifies the inbound Bearer against the supplied platform
// verifier (which enforces aud=iag.contract-management) and stores a
// derived models.Session on the request context.
//
// Public endpoints: /ready, /health, /health/live, /health/ready (root and
// /v1). Everything else requires a valid token. Notably: /bootstrap is NO LONGER public.
func GinPlatformAuth(v *platformauth.Verifier, lookup ContractorLookup, store *models.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Browsers cannot set an Authorization header on a WebSocket, so the
		// workspace socket carries the access token as a ?token= query param.
		// The gateway also injects the header for WS upgrades, but accepting the
		// query form here keeps direct (non-gateway) connections working too.
		var token string
		header := c.GetHeader("Authorization")
		switch {
		case strings.HasPrefix(header, "Bearer "):
			token = strings.TrimPrefix(header, "Bearer ")
		case strings.EqualFold(c.GetHeader("Upgrade"), "websocket"):
			token = c.Query("token")
		}
		if token == "" {
			apierr.Unauthorized(c, "missing bearer token")
			return
		}
		claims, err := v.Verify(token)
		if err != nil {
			apierr.Unauthorized(c, "invalid or expired token")
			return
		}

		sess := SessionFromClaims(c.Request.Context(), claims, lookup)
		if store != nil {
			sess = store.EnrichSessionFromWorkspace(sess)
		}
		c.Request = c.Request.WithContext(models.WithRequestSession(c.Request.Context(), sess))
		c.Next()
	}
}

// SessionFromClaims projects platform claims onto the contract-management
// Session shape. Role is derived from platform groups; a "contractor"
// promotion is applied ONLY when the JWT-derived role is the default
// "viewer" — admins and managers who happen to also be in the contractor
// table keep their elevated role. ContractorSup is still populated either
// way so contractor-scoping logic works for any role that needs it.
func SessionFromClaims(ctx context.Context, claims *platformauth.Claims, lookup ContractorLookup) models.Session {
	displayName := claims.Name
	if displayName == "" {
		displayName = claims.Email
	}
	sess := models.Session{
		Email:       claims.Email,
		Role:        roleFromGroups(claims.Groups, claims.IsSuperuser, claims.IsStaff),
		DisplayName: displayName,
		Permissions: claims.Permissions,
		UserID:      claims.Subject,
	}
	if lookup != nil && sess.Email != "" {
		if sup, ok, _ := lookup.ContractorSupervisor(ctx, sess.Email); ok {
			s := sup
			sess.ContractorSup = &s
			if sess.Role == "viewer" {
				sess.Role = "contractor"
			}
		}
	}
	// A user bound to a governance contractor is a portal contractor even
	// without a legacy contractor_supervisors row.
	if lookup != nil && sess.Role == "viewer" && claims.Subject != "" {
		if linked, _ := lookup.GovContractorLinked(ctx, claims.Subject); linked {
			sess.Role = "contractor"
		}
	}
	return sess
}

func roleFromGroups(groups []string, isSuperuser, isStaff bool) string {
	for _, g := range groups {
		switch g {
		case "superadmin":
			return "super_admin"
		case "admin":
			return "admin"
		case "manager":
			return "manager"
		case "viewer":
			return "viewer"
		case "staff":
			// platform "staff" maps to contract-management "manager" — staff
			// users get write access to operational entities but not roles.
			return "manager"
		}
	}
	if isSuperuser {
		return "super_admin"
	}
	if isStaff {
		return "manager"
	}
	return "viewer"
}

func isPublicPath(path string) bool {
	path = strings.TrimRight(path, "/")
	return path == "/ready" ||
		path == "/health" ||
		path == "/health/live" ||
		path == "/health/ready" ||
		path == "/v1/health" ||
		path == "/v1/health/live" ||
		path == "/v1/health/ready"
}
