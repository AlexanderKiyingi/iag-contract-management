package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/platformauth"
)

// ContractorLookup resolves a caller's contractor-supervisor binding (if any).
// Returns the supervisor name and true if the email is a contractor; false otherwise.
type ContractorLookup interface {
	ContractorSupervisor(ctx context.Context, email string) (string, bool, error)
}

// GinPlatformAuth verifies the inbound Bearer against the supplied platform
// verifier (which enforces aud=iag.contract-management) and stores a
// derived models.Session on the request context.
//
// Public endpoints: /health, /health/live, /health/ready. Everything else
// requires a valid token. Notably: /bootstrap is NO LONGER public.
func GinPlatformAuth(v *platformauth.Verifier, lookup ContractorLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := v.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		sess := SessionFromClaims(c.Request.Context(), claims, lookup)
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
	return path == "/health" ||
		path == "/health/live" ||
		path == "/health/ready" ||
		path == "/v1/health" ||
		path == "/v1/health/live" ||
		path == "/v1/health/ready"
}
