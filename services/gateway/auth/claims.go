package auth

import "context"

const (
	RoleAdmin    = "admin"
	RoleEmployee = "employee"
	RoleManager  = "manager"
)

// Claims are the internal token claims. An enterprise OIDC adapter maps its
// token into this shape; callers do not depend on how the token was issued.
type Claims struct {
	PersonID       string   `json:"personId"`
	OrganisationID string   `json:"organisationId"`
	Roles          []string `json:"roles"`
	ManagerOf      []string `json:"managerOf"`
}

func (c Claims) HasRole(role string) bool {
	for _, candidate := range c.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}
