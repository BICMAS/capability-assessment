package auth

import "errors"

// OIDCConfig is the production identity contract. A future adapter must validate
// issuer, audience, expiry, signing keys (JWKS), and the required internal claims
// before mapping into Claims. Roles and manager scope must come from the identity
// provider or a trusted directory lookup — never from client-supplied fields.
type OIDCConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

// ErrOIDCNotConfigured is returned when production authentication is required
// but no real OIDC verifier has been wired.
var ErrOIDCNotConfigured = errors.New("production OIDC verifier is not configured")

// NewOIDCVerifier would construct a production Verifier. It is intentionally
// unimplemented: do not substitute a stub that accepts unsigned or demo tokens.
func NewOIDCVerifier(cfg OIDCConfig) (Verifier, error) {
	if stringsEmpty(cfg.Issuer) || stringsEmpty(cfg.Audience) || stringsEmpty(cfg.JWKSURL) {
		return nil, ErrOIDCNotConfigured
	}
	return nil, ErrOIDCNotConfigured
}

// RejectingVerifier implements Verifier and rejects every token. Used when
// development login is disabled and no production verifier is available.
type RejectingVerifier struct{}

func (RejectingVerifier) Verify(string) (Claims, error) {
	return Claims{}, errors.New("no token verifier is configured")
}

func stringsEmpty(value string) bool {
	return value == ""
}
