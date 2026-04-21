package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SPLITTY_ENV", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_PRIVATE_KEY", "")

	cfg := Load()

	if cfg.Env != EnvDevelopment {
		t.Errorf("expected Env=%q, got %q", EnvDevelopment, cfg.Env)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("expected empty DatabaseURL, got %q", cfg.DatabaseURL)
	}
	if cfg.JWTPrivateKey != "" {
		t.Errorf("expected empty JWTPrivateKey, got %q", cfg.JWTPrivateKey)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SPLITTY_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/splitty")
	t.Setenv("JWT_PRIVATE_KEY", "test-key")

	cfg := Load()

	if cfg.Env != EnvProduction {
		t.Errorf("expected Env=%q, got %q", EnvProduction, cfg.Env)
	}
	if cfg.DatabaseURL != "postgres://localhost:5432/splitty" {
		t.Errorf("expected DatabaseURL=%q, got %q", "postgres://localhost:5432/splitty", cfg.DatabaseURL)
	}
	if cfg.JWTPrivateKey != "test-key" {
		t.Errorf("expected JWTPrivateKey=%q, got %q", "test-key", cfg.JWTPrivateKey)
	}
}
