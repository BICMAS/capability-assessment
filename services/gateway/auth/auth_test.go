package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestDevelopmentLoginTokens(t *testing.T) {
	dev := testIssuer(t)
	handler := RequireToken(dev)(RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	t.Run("employee token fails admin role check", func(t *testing.T) {
		token := login(t, dev, "eli")
		claims, err := dev.Verify(token)
		if err != nil {
			t.Fatal(err)
		}
		if claims.PersonID != EliPersonID || claims.HasRole(RoleAdmin) {
			t.Fatalf("claims = %#v", claims)
		}
		eli := developmentPeople()["eli"]
		if eli.ManagerID != MayaPersonID {
			t.Fatalf("eli manager = %s", eli.ManagerID)
		}

		request := httptest.NewRequest(http.MethodGet, "/admin", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("status = %d", response.Code)
		}
	})

	t.Run("maya token lists eli in managerOf", func(t *testing.T) {
		token := login(t, dev, "maya")
		claims, err := dev.Verify(token)
		if err != nil {
			t.Fatal(err)
		}
		if claims.PersonID != MayaPersonID || !slices.Contains(claims.ManagerOf, EliPersonID) {
			t.Fatalf("managerOf = %#v", claims.ManagerOf)
		}
	})
}

func TestMeReturnsSignedInName(t *testing.T) {
	dev := testIssuer(t)
	token := login(t, dev, "ada")
	handler := RequireToken(dev)(http.HandlerFunc(MeHandler))
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var payload meResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Name != "Ada Admin" || payload.OrganisationID != OrganisationID || !slices.Contains(payload.Roles, RoleAdmin) || payload.ManagerOf == nil {
		t.Fatalf("me = %#v", payload)
	}
}

func TestMiddlewareRejectsMissingAndInvalidTokens(t *testing.T) {
	dev := testIssuer(t)
	handler := RequireToken(dev)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	missing := httptest.NewRequest(http.MethodGet, "/me", nil)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missing)
	if missingResponse.Code != http.StatusUnauthorized {
		t.Fatalf("missing status = %d", missingResponse.Code)
	}

	invalid := httptest.NewRequest(http.MethodGet, "/me", nil)
	invalid.Header.Set("Authorization", "Bearer not-a-token")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnauthorized {
		t.Fatalf("invalid status = %d", invalidResponse.Code)
	}
}

func testIssuer(t *testing.T) *DevHMAC {
	t.Helper()
	dev, err := NewDevHMAC([]byte("test-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return dev
}

type recordingPublisher struct {
	events []PersonSignedIn
}

func (p *recordingPublisher) PublishPersonSignedIn(_ context.Context, event PersonSignedIn) error {
	p.events = append(p.events, event)
	return nil
}

func login(t *testing.T, issuer TokenIssuer, person string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"person": person})
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingPublisher{}
	request := httptest.NewRequest(http.MethodPost, "/dev/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	LoginHandler(issuer, publisher).ServeHTTP(response, request)
	if response.Code == http.StatusOK {
		personRecord := developmentPeople()[person]
		if len(publisher.events) != 1 || publisher.events[0].Type != "person.signed_in" || publisher.events[0].OrganisationID != OrganisationID || publisher.events[0].PersonID != personRecord.PersonID {
			t.Fatalf("events = %#v", publisher.events)
		}
	}
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d", response.Code)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token == "" {
		t.Fatal("empty token")
	}
	return payload.Token
}
