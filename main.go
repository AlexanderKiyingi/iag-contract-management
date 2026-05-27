package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	platformotel "github.com/alvor-technologies/iag-platform-go/otel"
	platformserviceauth "github.com/alvor-technologies/iag-platform-go/serviceauth"

	"github.com/alvor-technologies/iag-contract-management/internal/config"
	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/platformauth"
	"github.com/alvor-technologies/iag-contract-management/internal/router"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("config invalid", "error", err)
		os.Exit(1)
	}

	// Structured JSON logs in production so the platform log pipeline can
	// parse fields; text in dev for easier eyeballing.
	initLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// OpenTelemetry → otel-collector:4317 (non-blocking dial).
	tp, err := platformotel.Init(ctx, platformotel.Config{
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
	})
	if err != nil {
		slog.Warn("otel disabled", "error", err)
	} else {
		defer func() {
			sc, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			_ = tp.Shutdown(sc)
		}()
	}

	// Inbound verifier — every request must carry aud=cfg.Audience.
	// In production we refuse to start until the JWKS fetch succeeds, so a
	// misconfigured JWKS_URL fails fast instead of serving 401s for 15 minutes
	// until the background refresh loop recovers.
	verifier := platformauth.NewVerifier(cfg.JWKSURL, cfg.JWTIssuer, cfg.Audience)
	jwksCtx, jwksCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := verifier.Refresh(jwksCtx); err != nil {
		jwksCancel()
		if cfg.IsProduction() {
			slog.Error("initial jwks fetch failed in production", "error", err, "jwks_url", cfg.JWKSURL)
			os.Exit(1)
		}
		slog.Warn("initial jwks fetch failed (non-prod, continuing)", "error", err)
	} else {
		jwksCancel()
	}
	verifier.StartRefreshLoop(ctx, 15*time.Minute)

	// One Postgres pool, shared by the model store, health probe, and the
	// contractor lookup the auth middleware needs.
	pg, err := persistence.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	// Best-effort: register our permission catalogue with the auth service.
	// Spawned in the background with exponential backoff so a stuck or
	// late-arriving auth service doesn't block boot AND doesn't leave us
	// permanently un-registered after a transient outage.
	if cfg.ServiceClientSecret != "" {
		go registerPermissionsLoop(ctx, cfg)
	} else {
		slog.Warn("SERVICE_CLIENT_SECRET unset — skipping permissions registration")
	}

	eventBus := events.NewFromEnv()
	defer func() { _ = eventBus.Close() }()
	if eventBus.Enabled() {
		slog.Info("event bus enabled", "topic", events.TopicCommercial)
	}

	engine := router.New(cfg, pg, verifier, pg, eventBus)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("contract-management listening",
			"addr", addr,
			"audience", cfg.Audience,
			"env", cfg.Environment,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", fmt.Errorf("listen: %w", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")

	shutdownCtx, sCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer sCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("shutdown", "error", err)
	}
	// Stop background goroutines (jwks refresh, permissions retry).
	cancel()

	// Pacify unused references that only the smoke-test wires up.
	_ = models.PermissionDescriptors
}

// initLogger installs a default slog handler appropriate for the env.
func initLogger(cfg config.Config) {
	level := slog.LevelInfo
	if !cfg.IsProduction() {
		level = slog.LevelDebug
	}
	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler).With("service", cfg.ServiceName))
}

// registerPermissionsLoop publishes this service's permission catalogue to
// the authentication service, retrying with exponential backoff (capped at 5
// minutes) until success or ctx cancellation. Without this, a transient
// auth-service outage at boot would leave the service permanently un-
// registered until the pod is bounced.
func registerPermissionsLoop(ctx context.Context, cfg config.Config) {
	saClient := platformserviceauth.NewClient(platformserviceauth.Options{
		TokenURL:     cfg.AuthTokenURL,
		ClientID:     cfg.ServiceClientID,
		ClientSecret: cfg.ServiceClientSecret,
		Audience:     "iag.authentication",
	})
	descriptors := models.PermissionDescriptors()
	perms := make([]platformserviceauth.Permission, 0, len(descriptors))
	for _, d := range descriptors {
		perms = append(perms, platformserviceauth.Permission{
			Name:        d.Name,
			Description: d.Description,
		})
	}

	backoff := time.Second
	const maxBackoff = 5 * time.Minute
	for {
		regCtx, c := context.WithTimeout(ctx, 10*time.Second)
		err := platformserviceauth.RegisterPermissions(regCtx, saClient, cfg.JWTIssuer, "contract-management", perms)
		c()
		if err == nil {
			slog.Info("permissions registered", "count", len(perms))
			return
		}
		slog.Warn("permissions register failed — will retry", "error", err, "next_attempt_in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
