package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bicmas-academy/learn/services/gateway/config"
)

func TestMaterialsRejectProductionWithoutOIDC(t *testing.T) {
	cfg := config.Config{Production: true, DevLoginEnabled: false}
	_, err := NewMaterials(cfg)
	if err == nil || !strings.Contains(err.Error(), "OIDC") {
		t.Fatalf("error = %v", err)
	}
}

func TestMaterialsRejectDevLoginInProduction(t *testing.T) {
	_, err := NewMaterials(config.Config{
		Production:      true,
		DevLoginEnabled: true,
		DevJWTSecret:    "dev-only-not-a-production-secret",
	})
	if err == nil || !strings.Contains(err.Error(), "development login cannot be enabled") {
		t.Fatalf("error = %v", err)
	}
}

func TestMaterialsDevLoginEnabled(t *testing.T) {
	cfg := config.Config{
		DevLoginEnabled: true,
		DevJWTSecret:    "dev-only-not-a-production-secret",
	}
	materials, err := NewMaterials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if materials.Issuer == nil || materials.Verifier == nil {
		t.Fatal("expected issuer and verifier")
	}
}

func TestMaterialsDevLoginDisabledUsesRejectingVerifier(t *testing.T) {
	materials, err := NewMaterials(config.Config{DevLoginEnabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if materials.Issuer != nil {
		t.Fatal("issuer must be nil when development login is disabled")
	}
	if _, err := materials.Verifier.Verify("anything"); err == nil {
		t.Fatal("expected rejecting verifier")
	}
}

func TestOIDCVerifierNotImplemented(t *testing.T) {
	_, err := NewOIDCVerifier(OIDCConfig{
		Issuer:   "https://idp.example",
		Audience: "bicmas-learn",
		JWKSURL:  "https://idp.example/jwks",
	})
	if err != ErrOIDCNotConfigured {
		t.Fatalf("error = %v", err)
	}
}

func TestExpiredAndMalformedTokens(t *testing.T) {
	dev, err := NewDevHMAC([]byte("dev-only-not-a-production-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("expired", func(t *testing.T) {
		dev.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
		token, err := dev.Issue("eli")
		if err != nil {
			t.Fatal(err)
		}
		dev.now = func() time.Time { return time.Unix(1_700_000_000, 0).Add(2 * time.Hour) }
		if _, err := dev.Verify(token); err == nil {
			t.Fatal("expected expired token to fail")
		}
	})

	t.Run("malformed", func(t *testing.T) {
		dev.now = time.Now
		if _, err := dev.Verify("not.a.jwt"); err == nil {
			t.Fatal("expected malformed token to fail")
		}
	})
}

func TestIncorrectAlgorithmAndIssuer(t *testing.T) {
	secret := []byte("dev-only-not-a-production-secret")
	dev, err := NewDevHMAC(secret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("wrong issuer", func(t *testing.T) {
		claims := jwtClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   EliPersonID,
				Issuer:    "other-issuer",
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			PersonID:       EliPersonID,
			OrganisationID: OrganisationID,
			Roles:          []string{RoleEmployee},
			ManagerOf:      []string{},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dev.Verify(signed); err == nil {
			t.Fatal("expected wrong issuer to fail")
		}
	})

	t.Run("wrong algorithm", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		claims := jwtClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   EliPersonID,
				Issuer:    devIssuer,
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			PersonID:       EliPersonID,
			OrganisationID: OrganisationID,
			Roles:          []string{RoleEmployee},
			ManagerOf:      []string{},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dev.Verify(signed); err == nil {
			t.Fatal("expected wrong algorithm to fail")
		}
	})
}

func TestLoginDoesNotAcceptArbitraryAdminClaims(t *testing.T) {
	dev, err := NewDevHMAC([]byte("dev-only-not-a-production-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, err := dev.Issue("eli")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := dev.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.HasRole(RoleAdmin) || claims.PersonID != EliPersonID {
		t.Fatalf("claims = %#v", claims)
	}
}
