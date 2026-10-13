package levels

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bicmas-academy/learn/services/token"
)

const httpSecret = "dev-only-not-a-production-secret"

func TestHTTPSelfAssessmentKeepsTheCalculatedLevel(t *testing.T) {
	store := NewMemory()
	explanation := saveRegression(t, store, nil)
	handler := NewRouter(func(context.Context) error { return nil }, []byte(httpSecret), store, &recordingPublisher{})

	body := map[string]any{
		"personId":           regressionPersonID,
		"frameworkVersionId": regressionFrameworkID,
		"capabilityCode":     regressionCapability,
		"source":             SourceSelfAssessment,
		"percentage":         91,
		"evidenceIds":        []string{"evidence-self"},
		"requiredLevel":      4.0,
	}
	response := postJSON(t, handler, "/attempts", eliClaims(), body)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["currentLevel"] != "2.8" || payload["gap"] != "1.2" || payload["status"] != StatusProvisional {
		t.Fatalf("payload = %#v", payload)
	}
	if payload["selfAssessmentPercentage"] != float64(91) {
		t.Fatalf("selfAssessmentPercentage = %#v", payload["selfAssessmentPercentage"])
	}
	if payload["id"] != explanation.ID {
		t.Fatalf("id = %#v", payload["id"])
	}
}

func TestHTTPHasNoWriterForAIInterviewOrSimulation(t *testing.T) {
	store := NewMemory()
	handler := NewRouter(func(context.Context) error { return nil }, []byte(httpSecret), store, &recordingPublisher{})
	for _, source := range []string{SourceAIInterview, SourceSimulation} {
		response := postJSON(t, handler, "/attempts", eliClaims(), map[string]any{
			"personId":           regressionPersonID,
			"frameworkVersionId": regressionFrameworkID,
			"capabilityCode":     regressionCapability,
			"source":             source,
			"percentage":         68,
			"evidenceIds":        []string{"evidence-1"},
			"requiredLevel":      4.0,
		})
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d", source, response.Code)
		}
	}
}

func TestHTTPManagerOverrideIsAudited(t *testing.T) {
	store := NewMemory()
	explanation := saveRegression(t, store, nil)
	publisher := &recordingPublisher{}
	handler := NewRouter(func(context.Context) error { return nil }, []byte(httpSecret), store, publisher)

	response := postJSON(t, handler, "/levels/"+explanation.ID+"/overrides", mayaClaims(), map[string]any{
		"reason":   "Observed a live negotiation",
		"newValue": 3.1,
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["actorPersonId"] != mayaID || payload["reason"] != "Observed a live negotiation" || payload["previousValue"] != "2.8" || payload["newValue"] != "3.1" {
		t.Fatalf("override = %#v", payload)
	}
	if publisher.event.Type != "level.overridden" || publisher.event.ActorPersonID != mayaID || publisher.event.OverrideID != payload["id"] || publisher.event.OrganisationID != regressionOrganisationID {
		t.Fatalf("event = %#v", publisher.event)
	}

	loaded, err := store.Get(context.Background(), regressionOrganisationID, regressionPersonID, regressionFrameworkID, regressionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentLevel.String() != "2.8" {
		t.Fatalf("currentLevel = %s", loaded.CurrentLevel)
	}
}

type recordingPublisher struct {
	event LevelOverridden
}

func (r *recordingPublisher) PublishLevelOverridden(_ context.Context, event LevelOverridden) error {
	r.event = event
	return nil
}

func postJSON(t *testing.T, handler http.Handler, path string, claims token.Claims, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func eliClaims() token.Claims {
	return token.Claims{
		PersonID:       regressionPersonID,
		OrganisationID: regressionOrganisationID,
		Roles:          []string{"employee"},
		ManagerOf:      []string{},
	}
}

const mayaID = "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33"

func mayaClaims() token.Claims {
	return token.Claims{
		PersonID:       mayaID,
		OrganisationID: regressionOrganisationID,
		Roles:          []string{"manager"},
		ManagerOf:      []string{regressionPersonID},
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
	}).SignedString([]byte(httpSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
