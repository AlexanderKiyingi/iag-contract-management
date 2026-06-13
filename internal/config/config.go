package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-platform-go/corsenv"
)

// Config holds runtime knobs for the contract-management service.
// Post-cutover: no local JWT, no AllowMemoryFallback, no demo reset. Auth is
// the platform authentication service via JWKS + audience.
type Config struct {
	ServiceName     string
	Port            string
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	DatabaseURL string

	// Platform auth.
	JWTIssuer string
	JWKSURL   string
	Audience  string

	// Outbound service credentials (for /v1/permissions/register and any
	// future calls to other platform services).
	ServiceClientID     string
	ServiceClientSecret string
	AuthTokenURL        string

	AllowedOrigins  []string
	RateLimitPerMin int
	MaxBodyBytes    int64

	// TrustedProxies lists upstream proxy IPs/CIDRs that Gin should trust
	// when reading X-Forwarded-For. Set to your gateway/load-balancer.
	// Empty list = trust nobody (ClientIP() falls back to RemoteAddr).
	TrustedProxies []string

	// RequestTimeout is the per-request deadline applied by the timeout
	// middleware. Zero disables it.
	RequestTimeout time.Duration

	// SeedOnStartup writes the demo workspace into the DB on first run (when
	// the DB is empty). Defaults to true in non-production envs and false in
	// production, where leaving it on can collide with legacy table shapes
	// (e.g. audit_entries.id BIGINT vs the current schema's TEXT). Set
	// SEED_ON_STARTUP=true|false to override.
	SeedOnStartup bool
}

// Load reads configuration from env.
func Load() Config {
	env := strings.ToLower(envStr("ENVIRONMENT", envStr("APP_ENV", "development")))

	issuer := envStr("JWT_ISSUER", "http://localhost:3001")
	jwksURL := envStr("JWKS_URL", strings.TrimRight(issuer, "/")+"/.well-known/jwks.json")

	// corsenv.Allowlist always falls back to the permissive dev origins when no
	// CORS env var is set (even for an empty fallback). That's fine outside
	// production, but in production an unset allowlist must stay empty so
	// Validate() fails closed rather than silently admitting localhost.
	origins := corsenv.Allowlist(corsenv.DefaultDevOrigins)
	if env == "production" {
		origins = ""
		for _, key := range corsenv.EnvKeys {
			if v := strings.TrimSpace(os.Getenv(key)); v != "" {
				origins = v
				break
			}
		}
	}
	var allowed []string
	for _, o := range strings.Split(origins, ",") {
		if t := strings.TrimSpace(o); t != "" {
			allowed = append(allowed, t)
		}
	}

	var proxies []string
	for _, p := range strings.Split(envStr("TRUSTED_PROXIES", ""), ",") {
		if t := strings.TrimSpace(p); t != "" {
			proxies = append(proxies, t)
		}
	}

	seedOnStartup := env != "production"
	if raw := strings.TrimSpace(os.Getenv("SEED_ON_STARTUP")); raw != "" {
		seedOnStartup = strings.EqualFold(raw, "true")
	}

	return Config{
		ServiceName:     envStr("SERVICE_NAME", "contract-management"),
		Port:            envStr("PORT", "4103"),
		Environment:     env,
		ReadTimeout:     time.Duration(envInt("READ_TIMEOUT_SECONDS", 15)) * time.Second,
		WriteTimeout:    time.Duration(envInt("WRITE_TIMEOUT_SECONDS", 30)) * time.Second,
		ShutdownTimeout: time.Duration(envInt("SHUTDOWN_TIMEOUT_SECONDS", 15)) * time.Second,

		DatabaseURL: envStr("DATABASE_URL", ""),

		JWTIssuer: issuer,
		JWKSURL:   jwksURL,
		Audience:  envStr("AUDIENCE", "iag.contract-management"),

		ServiceClientID:     envStr("SERVICE_CLIENT_ID", "iag-contract-management"),
		ServiceClientSecret: envStr("SERVICE_CLIENT_SECRET", ""),
		AuthTokenURL:        envStr("AUTH_TOKEN_URL", strings.TrimRight(issuer, "/")+"/oauth/token"),

		AllowedOrigins:  allowed,
		RateLimitPerMin: envInt("RATE_LIMIT_PER_MINUTE", 120),
		MaxBodyBytes:    int64(envInt("MAX_BODY_BYTES", 8*1024*1024)),
		TrustedProxies:  proxies,
		RequestTimeout:  time.Duration(envInt("REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
		SeedOnStartup:   seedOnStartup,
	}
}

// Validate enforces production invariants.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Audience == "" {
		return fmt.Errorf("AUDIENCE is required (e.g. iag.contract-management)")
	}
	if c.IsProduction() {
		var missing []string
		if c.ServiceClientSecret == "" {
			missing = append(missing, "SERVICE_CLIENT_SECRET")
		}
		if len(c.AllowedOrigins) == 0 {
			missing = append(missing, "ALLOWED_ORIGINS")
		}
		if len(missing) > 0 {
			return fmt.Errorf("invalid production config: %s", strings.Join(missing, ", "))
		}
	}
	return nil
}

// IsProduction reports whether the env is production.
func (c Config) IsProduction() bool { return c.Environment == "production" }

// GinMode returns the gin mode for this environment.
func (c Config) GinMode() string {
	if c.IsProduction() {
		return "release"
	}
	return "debug"
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
