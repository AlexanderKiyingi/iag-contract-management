package router

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	platformmw "github.com/alvor-technologies/iag-platform-go/middleware"

	"github.com/alvor-technologies/iag-contract-management/internal/app"
	"github.com/alvor-technologies/iag-contract-management/internal/config"
	"github.com/alvor-technologies/iag-contract-management/internal/middleware"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/platformauth"
)

// New builds the Gin HTTP engine with all API routes under /v1.
// Pre-cutover the routes were mounted at both "" and "/v1"; post-cutover
// /v1 is the single canonical prefix the gateway expects.
func New(
	cfg config.Config,
	pg *persistence.Postgres,
	verifier *platformauth.Verifier,
	contractors middleware.ContractorLookup,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	mvc := app.NewMVC(cfg, pg)

	r := gin.New()
	// Trust only the upstream proxies operators explicitly listed. Without
	// this Gin would either trust all proxies (older default) or trust none
	// (newer default) — both wrong for "behind the platform gateway".
	if len(cfg.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.TrustedProxies)
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(otelgin.Middleware(cfg.ServiceName))
	r.Use(platformmw.RequestID())
	r.Use(middleware.GinRecovery())
	r.Use(middleware.GinSecurityHeaders())
	r.Use(middleware.GinCORS(cfg))
	r.Use(middleware.GinBodyLimit(cfg.MaxBodyBytes))
	if cfg.RequestTimeout > 0 {
		r.Use(middleware.GinTimeout(cfg.RequestTimeout))
	}
	// Auth runs BEFORE the rate limiter so an unauthenticated flood cannot
	// fill the limiter's per-IP map and amplify the small leak there.
	r.Use(middleware.GinPlatformAuth(verifier, contractors))
	r.Use(middleware.GinRateLimit(cfg.RateLimitPerMin))
	// Request log runs last so handler-applied status codes are visible.
	r.Use(middleware.GinLogger())

	wrap := gin.WrapF
	// Root probes match other platform services (/ready) for gateway upstream
	// checks and Railway healthchecks. Canonical API paths remain under /v1.
	r.GET("/health", wrap(mvc.Health.Check))
	r.GET("/health/live", wrap(mvc.Health.Live))
	r.GET("/health/ready", wrap(mvc.Health.Ready))
	r.GET("/ready", wrap(mvc.Health.Ready))

	registerRoutes(r.Group("/v1"), mvc)
	return r
}

func registerRoutes(g *gin.RouterGroup, mvc *app.MVC) {
	wrap := gin.WrapF

	g.GET("/health", wrap(mvc.Health.Check))
	g.GET("/health/live", wrap(mvc.Health.Live))
	g.GET("/health/ready", wrap(mvc.Health.Ready))

	// Session is the only auth surface this service exposes; login/refresh/logout
	// live on the platform authentication service at /api/v1/authentication/oauth/token.
	g.GET("/bootstrap", wrap(mvc.Auth.Bootstrap))
	g.GET("/auth/session", wrap(mvc.Auth.Session))

	// Workspace snapshot
	g.GET("/workspace", wrap(mvc.Workspace.Get))
	g.PUT("/workspace", wrap(mvc.Workspace.Put))

	// Frontend store snapshot
	g.GET("/frontend", wrap(mvc.Frontend.Get))
	g.PUT("/frontend", wrap(mvc.Frontend.Put))

	// Contracts
	g.GET("/contracts", wrap(mvc.Contracts.List))
	g.POST("/contracts", wrap(mvc.Contracts.Create))
	g.GET("/contracts/:no", wrap(mvc.Contracts.Get))
	g.PATCH("/contracts/:no", wrap(mvc.Contracts.Patch))
	g.PUT("/contracts/:no", wrap(mvc.Contracts.Patch))
	g.DELETE("/contracts/:no", wrap(mvc.Contracts.Delete))

	// Zones
	g.GET("/zones", wrap(mvc.WsRes.ListZones))
	g.GET("/zones/:code", wrap(mvc.WsRes.GetZone))

	// Engineers
	g.GET("/engineers", wrap(mvc.WsRes.ListEngineers))
	g.GET("/engineers/:id", wrap(mvc.WsRes.GetEngineer))
	g.POST("/engineers", wrap(mvc.WsRes.CreateEngineer))
	g.PATCH("/engineers/:id", wrap(mvc.WsRes.PatchEngineer))
	g.DELETE("/engineers/:id", wrap(mvc.WsRes.DeleteEngineer))

	// Users
	g.GET("/users", wrap(mvc.WsRes.ListUsers))
	g.GET("/users/:id", wrap(mvc.WsRes.GetUser))
	g.POST("/users", wrap(mvc.WsRes.CreateUser))
	g.PATCH("/users/:id", wrap(mvc.WsRes.PatchUser))
	g.DELETE("/users/:id", wrap(mvc.WsRes.DeleteUser))

	// Milestones
	g.GET("/milestones", wrap(mvc.FeRes.ListMilestones))
	g.POST("/milestones", wrap(mvc.FeRes.CreateMilestone))
	g.GET("/milestones/:id", wrap(mvc.FeRes.GetMilestone))
	g.PATCH("/milestones/:id", wrap(mvc.FeRes.PatchMilestone))
	g.DELETE("/milestones/:id", wrap(mvc.FeRes.DeleteMilestone))

	// Materials
	g.GET("/materials", wrap(mvc.FeRes.ListMaterials))
	g.POST("/materials", wrap(mvc.FeRes.CreateMaterial))
	g.PATCH("/materials/:id", wrap(mvc.FeRes.PatchMaterial))
	g.DELETE("/materials/:id", wrap(mvc.FeRes.DeleteMaterial))

	// Tasks (projects + tasks)
	g.GET("/projects", wrap(mvc.FeRes.ListProjects))
	g.POST("/projects", wrap(mvc.FeRes.CreateProject))
	g.PATCH("/projects/:index", wrap(mvc.FeRes.PatchProject))
	g.DELETE("/projects/:index", wrap(mvc.FeRes.DeleteProject))
	g.POST("/projects/:index/tasks", wrap(mvc.FeRes.CreateTask))
	g.PATCH("/projects/:index/tasks/:taskId", wrap(mvc.FeRes.PatchTask))
	g.DELETE("/projects/:index/tasks/:taskId", wrap(mvc.FeRes.DeleteTask))

	// Permissions
	g.GET("/permissions/catalog", wrap(mvc.Permissions.Catalog))
	g.GET("/permissions/builtin", wrap(mvc.Permissions.Builtin))
	g.GET("/permissions/me", wrap(mvc.Permissions.Me))
	g.POST("/permissions/check", wrap(mvc.Permissions.Check))
	g.GET("/permissions/users/:id", wrap(mvc.Permissions.UserPermissions))

	// Custom roles (workspace-local; advisory metadata only after cutover)
	g.GET("/roles", wrap(mvc.FeRes.ListRoles))
	g.POST("/roles", wrap(mvc.FeRes.CreateRole))
	g.GET("/roles/:id", wrap(mvc.FeRes.GetRole))
	g.PATCH("/roles/:id", wrap(mvc.FeRes.PatchRole))
	g.DELETE("/roles/:id", wrap(mvc.FeRes.DeleteRole))

	// Audit log
	g.GET("/audit", wrap(mvc.FeRes.ListAudit))
	g.POST("/audit", wrap(mvc.FeRes.AppendAudit))
	g.GET("/audit/:id", wrap(mvc.FeRes.GetAudit))

	// Help
	g.GET("/assistance", wrap(mvc.FeRes.ListAssistance))
	g.POST("/assistance", wrap(mvc.FeRes.PostAssistance))

	// Profile & uploads
	g.GET("/profile/photo", wrap(mvc.Uploads.GetProfile))
	g.PUT("/profile/photo", wrap(mvc.FeRes.PutProfilePhoto))
	g.DELETE("/profile/photo", wrap(mvc.FeRes.DeleteProfilePhoto))
	g.POST("/uploads/profile", wrap(mvc.Uploads.UploadProfile))

	// Insights
	g.PUT("/insights/scan", wrap(mvc.FeRes.PutAiScan))

	// Reports
	g.GET("/exports/contracts.csv", wrap(mvc.Exports.ExportContractsCSV))
}
