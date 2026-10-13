package directory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bicmas-academy/learn/services/token"
)

const (
	organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	adaID          = "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11"
	eliID          = "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22"
	mayaID         = "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33"
	secret         = "test-secret"
)

func TestAdminListsDirectoryAndCreatePublishes(t *testing.T) {
	store := seeded()
	publisher := &recordingPublisher{}
	router := NewRouter(nil, []byte(secret), store, publisher)

	departments := getJSON(t, router, "/departments", adminClaims())
	if departments[0]["name"] != "Sales" || departments[0]["organisationId"] != organisationID {
		t.Fatalf("departments = %#v", departments)
	}

	people := getJSON(t, router, "/people", adminClaims())
	if len(people) != 3 {
		t.Fatalf("people = %#v", people)
	}

	assignments := getJSON(t, router, "/role-assignments", adminClaims())
	if len(assignments) != 1 || assignments[0]["personName"] != "Eli Employee" || assignments[0]["title"] != "Area Sales Manager" {
		t.Fatalf("assignments = %#v", assignments)
	}

	body := bytes.NewBufferString(`{"personId":"` + eliID + `","title":"Regional Lead"}`)
	request := httptest.NewRequest(http.MethodPost, "/role-assignments", body)
	request.Header.Set("Authorization", "Bearer "+sign(t, adminClaims()))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", response.Code, response.Body.String())
	}
	if len(publisher.events) != 1 || publisher.events[0].Type != "assignment.changed" || publisher.events[0].PersonID != eliID || publisher.events[0].OrganisationID != organisationID {
		t.Fatalf("events = %#v", publisher.events)
	}
}

func TestManagerListsOnlyManagedPeople(t *testing.T) {
	router := NewRouter(nil, []byte(secret), seeded(), &recordingPublisher{})
	people := getJSON(t, router, "/people", token.Claims{
		PersonID:       mayaID,
		OrganisationID: organisationID,
		Roles:          []string{"manager"},
		ManagerOf:      []string{eliID},
	})
	if len(people) != 1 || people[0]["name"] != "Eli Employee" || people[0]["id"] != eliID {
		t.Fatalf("people = %#v", people)
	}
}

func TestEmployeeCannotReadAnotherProfile(t *testing.T) {
	router := NewRouter(nil, []byte(secret), seeded(), &recordingPublisher{})
	request := httptest.NewRequest(http.MethodGet, "/people/"+mayaID, nil)
	request.Header.Set("Authorization", "Bearer "+sign(t, token.Claims{
		PersonID:       eliID,
		OrganisationID: organisationID,
		Roles:          []string{"employee"},
		ManagerOf:      []string{},
	}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "Maya Manager") || strings.Contains(response.Body.String(), mayaID) {
		t.Fatalf("body leaked Maya: %s", response.Body.String())
	}
	listRequest := httptest.NewRequest(http.MethodGet, "/people", nil)
	listRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusForbidden || strings.Contains(listResponse.Body.String(), "Maya Manager") {
		t.Fatalf("list status = %d body = %s", listResponse.Code, listResponse.Body.String())
	}

	profile := getJSONObject(t, router, "/profile", token.Claims{
		PersonID:       eliID,
		OrganisationID: organisationID,
		Roles:          []string{"employee"},
		ManagerOf:      []string{},
	})
	roles, ok := profile["roles"].([]any)
	if !ok || len(roles) != 1 {
		t.Fatalf("profile = %#v", profile)
	}
	role, ok := roles[0].(map[string]any)
	if !ok || role["title"] != "Area Sales Manager" || profile["name"] != "Eli Employee" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestListsAreScopedToTheTokenOrganisation(t *testing.T) {
	router := NewRouter(nil, []byte(secret), seeded(), &recordingPublisher{})
	departments := getJSON(t, router, "/departments", token.Claims{
		PersonID:       adaID,
		OrganisationID: "other-org",
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	})
	if len(departments) != 0 {
		t.Fatalf("departments = %#v", departments)
	}
}

type recordingPublisher struct {
	events []AssignmentChanged
}

func (p *recordingPublisher) PublishAssignmentChanged(_ context.Context, event AssignmentChanged) error {
	p.events = append(p.events, event)
	return nil
}

func seeded() *Memory {
	return &Memory{
		departments: []Department{{
			ID:             "c4e8a1b2-7d63-4f90-8a15-2b6c9d0e3f71",
			OrganisationID: organisationID,
			Name:           "Sales",
		}},
		people: []Person{
			{ID: adaID, OrganisationID: organisationID, Name: "Ada Admin"},
			{ID: eliID, OrganisationID: organisationID, Name: "Eli Employee", DepartmentName: "Sales"},
			{ID: mayaID, OrganisationID: organisationID, Name: "Maya Manager"},
		},
		assignments: []RoleAssignment{{
			ID:             "6e2a9c80-1b47-4d5f-9c31-8a0f7e2d4b66",
			OrganisationID: organisationID,
			PersonID:       eliID,
			PersonName:     "Eli Employee",
			Title:          "Area Sales Manager",
			CreatedAt:      time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC),
		}},
	}
}

func adminClaims() token.Claims {
	return token.Claims{
		PersonID:       adaID,
		OrganisationID: organisationID,
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	}
}

func getJSON(t *testing.T, handler http.Handler, path string, claims token.Claims) []map[string]any {
	t.Helper()
	var payload []map[string]any
	decode(t, handler, path, claims, &payload)
	return payload
}

func getJSONObject(t *testing.T, handler http.Handler, path string, claims token.Claims) map[string]any {
	t.Helper()
	var payload map[string]any
	decode(t, handler, path, claims, &payload)
	return payload
}

func decode(t *testing.T, handler http.Handler, path string, claims token.Claims, payload any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), payload); err != nil {
		t.Fatal(err)
	}
}

func sign(t *testing.T, claims token.Claims) string {
	t.Helper()
	type registered struct {
		jwt.RegisteredClaims
		PersonID       string   `json:"personId"`
		OrganisationID string   `json:"organisationId"`
		Roles          []string `json:"roles"`
		ManagerOf      []string `json:"managerOf"`
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, registered{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    token.Issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		PersonID:       claims.PersonID,
		OrganisationID: claims.OrganisationID,
		Roles:          claims.Roles,
		ManagerOf:      append([]string{}, claims.ManagerOf...),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
