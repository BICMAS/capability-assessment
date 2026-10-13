package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "bicmas-learn-dev"

// Claims are the internal token claims. An enterprise OIDC adapter maps its
// token into this shape; services do not depend on the development issuer.
type Claims struct {
	PersonID       string
	OrganisationID string
	Roles          []string
	ManagerOf      []string
}

func (c Claims) HasRole(role string) bool {
	for _, candidate := range c.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

type registeredClaims struct {
	jwt.RegisteredClaims
	PersonID       string   `json:"personId"`
	OrganisationID string   `json:"organisationId"`
	Roles          []string `json:"roles"`
	ManagerOf      []string `json:"managerOf"`
}

func Verify(secret []byte, raw string) (Claims, error) {
	if len(secret) == 0 {
		return Claims{}, errors.New("JWT secret is required")
	}
	parsed, err := jwt.ParseWithClaims(raw, &registeredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(Issuer))
	if err != nil {
		return Claims{}, err
	}
	tokenClaims, ok := parsed.Claims.(*registeredClaims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}
	if tokenClaims.PersonID == "" || tokenClaims.OrganisationID == "" || len(tokenClaims.Roles) == 0 {
		return Claims{}, errors.New("token is missing claims")
	}
	managerOf := tokenClaims.ManagerOf
	if managerOf == nil {
		managerOf = []string{}
	}
	return Claims{
		PersonID:       tokenClaims.PersonID,
		OrganisationID: tokenClaims.OrganisationID,
		Roles:          append([]string{}, tokenClaims.Roles...),
		ManagerOf:      append([]string{}, managerOf...),
	}, nil
}

func Bearer(header string) (string, bool) {
	scheme, raw, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || raw == "" || strings.Contains(raw, " ") {
		return "", false
	}
	return raw, true
}

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := Bearer(r.Header.Get("Authorization"))
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			claims, err := Verify(secret, raw)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}
