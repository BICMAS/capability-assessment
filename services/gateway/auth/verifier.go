package auth

// Verifier checks a bearer token and returns internal claims.
// Development login issues HMAC tokens. An enterprise OIDC adapter can
// implement Verifier and replace that login without changing middleware.
type Verifier interface {
	Verify(token string) (Claims, error)
}
