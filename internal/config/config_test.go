package config

import (
	"os"
	"testing"
)

func TestProductionRequiresServiceSecret(t *testing.T) {
	os.Clearenv()
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/cm")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	// SERVICE_CLIENT_SECRET deliberately omitted.

	if err := Load().Validate(); err == nil {
		t.Fatal("expected SERVICE_CLIENT_SECRET to be required in production")
	}
}

func TestProductionRequiresAllowedOrigins(t *testing.T) {
	os.Clearenv()
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/cm")
	t.Setenv("SERVICE_CLIENT_SECRET", "abcd1234abcd1234")
	// ALLOWED_ORIGINS deliberately omitted.

	if err := Load().Validate(); err == nil {
		t.Fatal("expected ALLOWED_ORIGINS to be required in production")
	}
}

func TestDatabaseURLRequired(t *testing.T) {
	os.Clearenv()
	t.Setenv("ENVIRONMENT", "development")

	if err := Load().Validate(); err == nil {
		t.Fatal("expected DATABASE_URL to be required")
	}
}

func TestDevelopmentDefaults(t *testing.T) {
	os.Clearenv()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/cm")

	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate dev: %v", err)
	}
	if cfg.Audience != "iag.contract-management" {
		t.Fatalf("audience: %s", cfg.Audience)
	}
	if cfg.Port != "4103" {
		t.Fatalf("port: %s", cfg.Port)
	}
}
