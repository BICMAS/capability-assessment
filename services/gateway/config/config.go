package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// MinDevSecretLength is the minimum length of DEV_JWT_SECRET when
	// development login is enabled. Shorter values are rejected at startup.
	MinDevSecretLength = 32

	EnvAppEnv          = "APP_ENV"
	EnvDevLoginEnabled = "DEV_LOGIN_ENABLED"
	EnvDevJWTSecret    = "DEV_JWT_SECRET"
)

// Config is the gateway authentication environment boundary.
type Config struct {
	AppEnv          string
	Production      bool
	DevLoginEnabled bool
	DevJWTSecret    string
}

// Load reads authentication settings from the process environment.
// Boolean values must be valid; invalid values are rejected.
// Development login defaults to disabled when DEV_LOGIN_ENABLED is unset.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom reads authentication settings through getenv so tests can inject values.
func LoadFrom(getenv func(string) string) (Config, error) {
	appEnv := strings.TrimSpace(getenv(EnvAppEnv))
	production := isProduction(appEnv)

	enabled, err := parseBoolFlag(getenv(EnvDevLoginEnabled), false)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", EnvDevLoginEnabled, err)
	}

	cfg := Config{
		AppEnv:          appEnv,
		Production:      production,
		DevLoginEnabled: enabled,
		DevJWTSecret:    getenv(EnvDevJWTSecret),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate enforces the fail-closed authentication boundary.
func (c Config) Validate() error {
	if c.Production && c.DevLoginEnabled {
		return errors.New("development login cannot be enabled when APP_ENV is production")
	}
	if c.Production {
		return errors.New("production identity provider is not configured; refuse to start until a real OIDC verifier is wired")
	}
	if !c.DevLoginEnabled {
		return nil
	}
	if err := ValidateDevSecret(c.DevJWTSecret); err != nil {
		return err
	}
	return nil
}

// ValidateDevSecret rejects missing and weak development HMAC secrets.
func ValidateDevSecret(secret string) error {
	if secret == "" {
		return errors.New("DEV_JWT_SECRET is required when development login is enabled")
	}
	if len(secret) < MinDevSecretLength {
		return fmt.Errorf("DEV_JWT_SECRET must be at least %d characters when development login is enabled", MinDevSecretLength)
	}
	if isDeniedDevSecret(secret) {
		return errors.New("DEV_JWT_SECRET is too weak for development login")
	}
	return nil
}

func parseBoolFlag(raw string, defaultValue bool) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
	return value, nil
}

func isProduction(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func isDeniedDevSecret(secret string) bool {
	switch strings.ToLower(strings.TrimSpace(secret)) {
	case "secret", "password", "changeme", "dev", "test", "test-secret", "jwt-secret",
		"passwordpasswordpasswordpassword":
		return true
	default:
		return false
	}
}
