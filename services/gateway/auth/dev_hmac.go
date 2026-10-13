package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const devIssuer = "bicmas-learn-dev"

// TokenIssuer issues development tokens. Production sign-in is an OIDC adapter
// in front of Verifier, not this issuer.
type TokenIssuer interface {
	Issue(personKey string) (string, error)
}

type jwtClaims struct {
	jwt.RegisteredClaims
	PersonID       string   `json:"personId"`
	OrganisationID string   `json:"organisationId"`
	Roles          []string `json:"roles"`
	ManagerOf      []string `json:"managerOf"`
}

// DevHMAC issues and verifies development JWTs signed with a shared secret.
type DevHMAC struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewDevHMAC(secret []byte, ttl time.Duration) (*DevHMAC, error) {
	if len(secret) == 0 {
		return nil, errors.New("development JWT secret is required")
	}
	if ttl <= 0 {
		return nil, errors.New("development JWT lifetime must be positive")
	}
	return &DevHMAC{
		secret: append([]byte(nil), secret...),
		ttl:    ttl,
		now:    time.Now,
	}, nil
}

func (d *DevHMAC) Issue(personKey string) (string, error) {
	person, ok := developmentPeople()[personKey]
	if !ok {
		return "", ErrUnknownPerson
	}
	issuedAt := d.now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   person.PersonID,
			Issuer:    devIssuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(d.ttl)),
		},
		PersonID:       person.PersonID,
		OrganisationID: person.OrganisationID,
		Roles:          append([]string{}, person.Roles...),
		ManagerOf:      append([]string{}, person.ManagerOf...),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(d.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (d *DevHMAC) Verify(raw string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return d.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(devIssuer))
	if err != nil {
		return Claims{}, err
	}
	tokenClaims, ok := parsed.Claims.(*jwtClaims)
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
