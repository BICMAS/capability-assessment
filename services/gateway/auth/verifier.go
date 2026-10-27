package auth

// Verifier checks a bearer token and returns internal claims.
// Development login uses DevHMAC. Production must use a real OIDC-backed
// Verifier (see OIDCConfig and NewOIDCVerifier) without changing middleware.
type Verifier interface {
	Verify(token string) (Claims, error)
}
