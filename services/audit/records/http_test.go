package records

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/token"
)

func TestListIsOrganisationScopedAndHasNoMutationRoute(t *testing.T) {
	store := &Memory{}
	occurred := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	if err := store.Append(context.Background(), Record{
		ActorPersonID:  "ada",
		OrganisationID: "org-1",
		Action:         ActionPersonSignedIn,
		EntityType:     EntityTypePerson,
		EntityID:       "ada",
		OccurredAt:     occurred,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), Record{
		ActorPersonID:  "eli",
		OrganisationID: "org-2",
		Action:         ActionPersonSignedIn,
		EntityType:     EntityTypePerson,
		EntityID:       "eli",
		OccurredAt:     occurred,
	}); err != nil {
		t.Fatal(err)
	}

	handler := ListHandler(store)
	admin := httptest.NewRequest(http.MethodGet, "/records", nil)
	admin = admin.WithContext(token.WithClaims(admin.Context(), token.Claims{
		PersonID:       "ada",
		OrganisationID: "org-1",
		Roles:          []string{"admin"},
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, admin)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var records []map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0]["actorPersonId"] != "ada" || records[0]["organisationId"] != "org-1" {
		t.Fatalf("records = %#v", records)
	}

	employee := httptest.NewRequest(http.MethodGet, "/records", nil)
	employee = employee.WithContext(token.WithClaims(employee.Context(), token.Claims{
		PersonID:       "eli",
		OrganisationID: "org-1",
		Roles:          []string{"employee"},
	}))
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, employee)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("employee status = %d", denied.Code)
	}
}

func TestRouterHasNoUpdateOrDelete(t *testing.T) {
	router := NewRouter(nil, []byte("secret"), &Memory{})
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost} {
		request := httptest.NewRequest(method, "/records", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code == http.StatusOK {
			t.Fatalf("%s was accepted", method)
		}
	}
}
