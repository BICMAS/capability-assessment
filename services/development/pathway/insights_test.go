package pathway

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bicmas-academy/learn/services/token"
)

func TestRecommendationMatchesAConfirmedGapOnly(t *testing.T) {
	store := NewMemory("https://lms.bicmas.example")
	if _, err := store.SaveResource(context.Background(), ResourceInput{
		OrganisationID: organisationID, Title: "Territory planning practice", SkillID: "skill-1", TargetLevel: 3,
		Reference: "resource-1", Rationale: "Closes the confirmed gap for territory planning.",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := RecommendLearning(context.Background(), store, organisationID, GapSignal{PersonID: eliID, CapabilityID: "skill-1", TargetLevel: 3, State: GapMissing}); err != ErrInvalid {
		t.Fatalf("missing = %v", err)
	}
	if _, err := RecommendLearning(context.Background(), store, organisationID, GapSignal{PersonID: eliID, CapabilityID: "skill-1", TargetLevel: 3, State: GapMet}); err != ErrInvalid {
		t.Fatalf("met = %v", err)
	}
	if _, err := RecommendLearning(context.Background(), store, "org-other", GapSignal{PersonID: eliID, CapabilityID: "skill-1", TargetLevel: 3, State: GapBelow}); err != ErrNotFound {
		t.Fatalf("other organisation = %v", err)
	}
	first, err := RecommendLearning(context.Background(), store, organisationID, GapSignal{PersonID: eliID, CapabilityID: "skill-1", TargetLevel: 3, State: GapBelow})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusRecommended || first.IntendedLevel != 3 || first.Reference != "resource-1" || strings.Contains(first.Status, "enrolled") || strings.Contains(first.Status, "complete") {
		t.Fatalf("recommendation = %+v", first)
	}
	again, err := RecommendLearning(context.Background(), store, organisationID, GapSignal{PersonID: eliID, CapabilityID: "skill-1", TargetLevel: 3, State: GapBelow})
	if err != nil || again.ID != first.ID {
		t.Fatalf("repeat = %+v %v", again, err)
	}
	if _, err := store.MapCourse(context.Background(), MappingInput{
		OrganisationID: organisationID, ExternalCourseID: "course-9", CapabilityID: "skill-2", TargetLevel: 2, Title: "Mapped course",
	}); err != nil {
		t.Fatal(err)
	}
	mapped, err := RecommendLearning(context.Background(), store, organisationID, GapSignal{PersonID: eliID, CapabilityID: "skill-2", TargetLevel: 2, State: GapBelow})
	if err != nil || mapped.Reference != "course-9" || mapped.Status != StatusRecommended || !strings.Contains(mapped.Rationale, "not confirmed") {
		t.Fatalf("mapped = %+v %v", mapped, err)
	}
}

func TestPracticeStoresATranscriptOnlyWhenRequested(t *testing.T) {
	store := NewMemory("https://lms.bicmas.example")
	preview, err := Practise(context.Background(), store, PracticeInput{
		OrganisationID: organisationID, PersonID: eliID, ScenarioKey: "sales_negotiation", Message: "I would hold the approved range.", Record: false,
	})
	if err != nil || preview.Recorded || preview.Voice != VoiceUnconfigured || !strings.Contains(preview.Feedback, PracticePrototype) {
		t.Fatalf("preview = %+v %v", preview, err)
	}
	rows, err := store.ListPractice(context.Background(), organisationID, eliID)
	if err != nil || len(rows) != 0 {
		t.Fatalf("stored preview = %+v %v", rows, err)
	}
	saved, err := Practise(context.Background(), store, PracticeInput{
		OrganisationID: organisationID, PersonID: eliID, ScenarioKey: "conflict_resolution", Message: "I would ask which decision left them out.", Record: true,
	})
	if err != nil || !saved.Recorded || saved.ID == "" {
		t.Fatalf("saved = %+v %v", saved, err)
	}
	rows, err = store.ListPractice(context.Background(), organisationID, eliID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows = %+v %v", rows, err)
	}
	if _, err := Reply("unknown", "hello"); err != ErrInvalid {
		t.Fatalf("unknown scenario = %v", err)
	}
}

func TestEmployeesCannotManageTheCatalogue(t *testing.T) {
	store := NewMemory("https://lms.bicmas.example")
	handler := NewRouter(func(context.Context) error { return nil }, []byte("jwt-secret"), []byte(webhookSecret), store, nil)
	employee := httptest.NewRequest(http.MethodPost, "/learning-resources", bytes.NewBufferString(`{"title":"Course","skillId":"skill-1","targetLevel":3,"reference":"ref","rationale":"Because the gap is confirmed."}`))
	employee.Header.Set("Authorization", "Bearer "+insightToken(t, token.Claims{PersonID: eliID, OrganisationID: organisationID, Roles: []string{"employee"}}))
	employeeResponse := httptest.NewRecorder()
	handler.ServeHTTP(employeeResponse, employee)
	if employeeResponse.Code != http.StatusForbidden {
		t.Fatalf("employee status %d", employeeResponse.Code)
	}
	admin := httptest.NewRequest(http.MethodPost, "/learning-resources", bytes.NewBufferString(`{"title":"Course","skillId":"skill-1","targetLevel":3,"reference":"ref","rationale":"Because the gap is confirmed."}`))
	admin.Header.Set("Authorization", "Bearer "+insightToken(t, token.Claims{PersonID: "ada", OrganisationID: organisationID, Roles: []string{"admin"}}))
	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, admin)
	if adminResponse.Code != http.StatusCreated {
		t.Fatalf("admin status %d body %s", adminResponse.Code, adminResponse.Body.String())
	}
	other := httptest.NewRequest(http.MethodGet, "/learning-resources", nil)
	other.Header.Set("Authorization", "Bearer "+insightToken(t, token.Claims{PersonID: "ada", OrganisationID: "org-other", Roles: []string{"admin"}}))
	otherResponse := httptest.NewRecorder()
	handler.ServeHTTP(otherResponse, other)
	if otherResponse.Code != http.StatusOK || strings.Contains(otherResponse.Body.String(), "skill-1") {
		t.Fatalf("other organisation %d %s", otherResponse.Code, otherResponse.Body.String())
	}
}

func insightToken(t *testing.T, claims token.Claims) string {
	t.Helper()
	type registered struct {
		jwt.RegisteredClaims
		PersonID       string   `json:"personId"`
		OrganisationID string   `json:"organisationId"`
		Roles          []string `json:"roles"`
		ManagerOf      []string `json:"managerOf"`
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, registered{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: token.Issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		PersonID:         claims.PersonID,
		OrganisationID:   claims.OrganisationID,
		Roles:            claims.Roles,
		ManagerOf:        claims.ManagerOf,
	}).SignedString([]byte("jwt-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
