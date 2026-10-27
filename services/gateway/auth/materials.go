package auth

import (
	"fmt"
	"time"

	"github.com/bicmas-academy/learn/services/gateway/config"
)

// Materials are the issuer and verifier selected for the current environment.
// Issuer is nil when development login must not be registered.
type Materials struct {
	Issuer   TokenIssuer
	Verifier Verifier
}

// NewMaterials selects authentication components from validated config.
// Callers must pass a Config that already passed Validate.
func NewMaterials(cfg config.Config) (Materials, error) {
	if cfg.Production {
		if cfg.DevLoginEnabled {
			return Materials{}, fmt.Errorf("development login cannot be enabled in production")
		}
		verifier, err := NewOIDCVerifier(OIDCConfig{})
		if err != nil {
			return Materials{}, fmt.Errorf("production authentication: %w", err)
		}
		return Materials{Verifier: verifier}, nil
	}
	if cfg.DevLoginEnabled {
		dev, err := NewDevHMAC([]byte(cfg.DevJWTSecret), 12*time.Hour)
		if err != nil {
			return Materials{}, err
		}
		return Materials{Issuer: dev, Verifier: dev}, nil
	}
	return Materials{Verifier: RejectingVerifier{}}, nil
}
