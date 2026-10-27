package config

import (
	"strings"
	"testing"
)

func TestLoadDevLoginEnabledLocally(t *testing.T) {
	cfg, err := LoadFrom(envMap(map[string]string{
		EnvAppEnv:          "development",
		EnvDevLoginEnabled: "true",
		EnvDevJWTSecret:    strongSecret(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Production || !cfg.DevLoginEnabled || cfg.DevJWTSecret != strongSecret() {
		t.Fatalf("cfg = %#v", cfg)
	}
}

func TestLoadDevLoginDisabled(t *testing.T) {
	cfg, err := LoadFrom(envMap(map[string]string{
		EnvAppEnv:          "development",
		EnvDevLoginEnabled: "false",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DevLoginEnabled {
		t.Fatal("expected development login disabled")
	}
}

func TestLoadRejectsInvalidBoolean(t *testing.T) {
	_, err := LoadFrom(envMap(map[string]string{
		EnvDevLoginEnabled: "yes",
		EnvDevJWTSecret:    strongSecret(),
	}))
	if err == nil || !strings.Contains(err.Error(), EnvDevLoginEnabled) {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsDevLoginInProduction(t *testing.T) {
	_, err := LoadFrom(envMap(map[string]string{
		EnvAppEnv:          "production",
		EnvDevLoginEnabled: "true",
		EnvDevJWTSecret:    strongSecret(),
	}))
	if err == nil || !strings.Contains(err.Error(), "development login cannot be enabled") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadProductionFailsClosedWithoutOIDC(t *testing.T) {
	_, err := LoadFrom(envMap(map[string]string{
		EnvAppEnv:          "production",
		EnvDevLoginEnabled: "false",
	}))
	if err == nil || !strings.Contains(err.Error(), "production identity provider is not configured") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsMissingAndWeakSecrets(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, err := LoadFrom(envMap(map[string]string{
			EnvDevLoginEnabled: "true",
		}))
		if err == nil || !strings.Contains(err.Error(), "DEV_JWT_SECRET is required") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("too short", func(t *testing.T) {
		_, err := LoadFrom(envMap(map[string]string{
			EnvDevLoginEnabled: "true",
			EnvDevJWTSecret:    "short-secret-value",
		}))
		if err == nil || !strings.Contains(err.Error(), "at least") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("denied", func(t *testing.T) {
		_, err := LoadFrom(envMap(map[string]string{
			EnvDevLoginEnabled: "true",
			EnvDevJWTSecret:    "passwordpasswordpasswordpassword",
		}))
		if err == nil || !strings.Contains(err.Error(), "too weak") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDevLoginDefaultsDisabled(t *testing.T) {
	cfg, err := LoadFrom(envMap(map[string]string{
		EnvAppEnv: "development",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DevLoginEnabled {
		t.Fatal("expected DEV_LOGIN_ENABLED to default to false")
	}
}

func strongSecret() string {
	return "dev-only-not-a-production-secret"
}

func envMap(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
