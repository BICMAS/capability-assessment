package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bicmas-academy/learn/services/assessment-plan/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

const (
	organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	mayaID         = "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33"
	eliID          = "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22"
	versionID      = "4c1e8a22-0b55-4d6f-9a18-2e7c5b0d3f64"
	roleID         = "b33a1d90-6e24-4c58-8f17-9a0b2c4d5e61"
	requirementID  = "c44b2e01-7f35-4d69-9a28-0b1c3d5e6f72"
	capabilityID   = "09ac779b-f2ba-4b78-83c2-4844b0a6e690"
	httpSecret     = "dev-only-not-a-production-secret"
)

func TestApprovedPlanStaysUnchanged(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	assignment := assign(t, store)
	created := draftPlan(t, store, assignment.ID)

	if created.VersionNumber != 1 || created.Status != StatusDraft || len(created.Skills) != 1 {
		t.Fatalf("plan = %#v", created)
	}
	skill := created.Skills[0]
	if skill.RequirementID != requirementID || skill.CapabilityID != capabilityID || skill.RequiredLevel != 3 {
		t.Fatalf("skill = %#v", skill)
	}
	if skill.RecommendedMethod != "manager_assessment" || len(skill.AssessmentItems) != 1 || len(skill.Rubric) != 1 {
		t.Fatalf("skill content = %#v", skill)
	}

	approved, err := store.ReplacePlan(ctx, ReplacePlanInput{
		OrganisationID: organisationID,
		PlanID:         created.ID,
		Status:         StatusApproved,
		Skills:         negotiationSkill(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusApproved || approved.VersionNumber != 1 {
		t.Fatalf("approved = %#v", approved)
	}

	changed := negotiationSkill()
	changed[0].Rationale = "A different rationale for the same requirement."
	_, err = store.ReplacePlan(ctx, ReplacePlanInput{
		OrganisationID: organisationID,
		PlanID:         created.ID,
		Status:         StatusChangesRequested,
		Skills:         changed,
	})
	if !errors.Is(err, ErrImmutable) {
		t.Fatalf("replace error = %v", err)
	}

	again, err := store.GetPlan(ctx, organisationID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != StatusApproved || again.Skills[0].Rationale != approved.Skills[0].Rationale {
		t.Fatalf("approved plan changed: %#v", again)
	}
}

func TestHealthDoesNotCreateRecords(t *testing.T) {
	store := &countingStore{}
	handler := NewRouter(func(context.Context) error { return nil }, []byte(httpSecret), store, nil, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if store.writes != 0 {
		t.Fatalf("writes = %d", store.writes)
	}
}

func TestReplaceApprovedPlanConflicts(t *testing.T) {
	store := NewMemory()
	assignment := assign(t, store)
	created := draftPlan(t, store, assignment.ID)
	if _, err := store.ReplacePlan(context.Background(), ReplacePlanInput{
		OrganisationID: organisationID,
		PlanID:         created.ID,
		Status:         StatusApproved,
		Skills:         negotiationSkill(),
	}); err != nil {
		t.Fatal(err)
	}

	body := openapi.ReplacePlan{Status: StatusGenerated, Skills: skillJSON()}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/plans/"+created.ID, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+sign(t, mayaClaims()))
	response := httptest.NewRecorder()
	NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, nil, nil).ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func assign(t *testing.T, store Store) Assignment {
	t.Helper()
	reason := "Coverage gap for a published role."
	row, err := store.CreateAssignment(context.Background(), AssignmentInput{
		OrganisationID:     organisationID,
		PersonID:           eliID,
		FrameworkVersionID: versionID,
		RoleID:             roleID,
		AssignedByPersonID: mayaID,
		AssignedAt:         time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC),
		ExceptionReason:    reason,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != AssignmentStatus || row.ExceptionReason != reason || row.AssignedByPersonID != mayaID {
		t.Fatalf("assignment = %#v", row)
	}
	return row
}

func draftPlan(t *testing.T, store Store, assignmentID string) Plan {
	t.Helper()
	row, err := store.CreatePlan(context.Background(), PlanInput{
		OrganisationID: organisationID,
		AssignmentID:   assignmentID,
		Skills:         negotiationSkill(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func negotiationSkill() []SkillInput {
	return []SkillInput{{
		RequirementID:     requirementID,
		CapabilityID:      capabilityID,
		RequiredLevel:     3,
		RecommendedMethod: "manager_assessment",
		Rationale:         "The role requires a manager assessment at level 3.",
		Instructions:      "Ask Eli to handle a price objection and score the response.",
		AssessmentItems:   []ItemInput{{Prompt: "Handle a price objection from a regular buyer."}},
		Rubric:            []RubricInput{{Level: 3, Descriptor: "Agrees a next step the buyer accepts."}},
	}}
}

func skillJSON() []openapi.SkillInput {
	return []openapi.SkillInput{{
		RequirementID:     requirementID,
		CapabilityID:      capabilityID,
		RequiredLevel:     3,
		RecommendedMethod: "manager_assessment",
		Rationale:         "The role requires a manager assessment at level 3.",
		Instructions:      "Ask Eli to handle a price objection and score the response.",
		AssessmentItems:   []openapi.AssessmentItemInput{{Prompt: "Handle a price objection from a regular buyer."}},
		Rubric:            []openapi.RubricBandInput{{Level: 3, Descriptor: "Agrees a next step the buyer accepts."}},
	}}
}

func mayaClaims() token.Claims {
	return token.Claims{
		PersonID:       mayaID,
		OrganisationID: organisationID,
		Roles:          []string{"manager"},
		ManagerOf:      []string{eliID},
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

type countingStore struct {
	writes int
}

func (s *countingStore) CreateAssignment(context.Context, AssignmentInput) (Assignment, error) {
	s.writes++
	return Assignment{}, nil
}

func (s *countingStore) FindAssignment(context.Context, string, string, string, string) (Assignment, error) {
	s.writes++
	return Assignment{}, ErrNotFound
}

func (s *countingStore) GetAssignment(context.Context, string, string) (Assignment, error) {
	s.writes++
	return Assignment{}, ErrNotFound
}

func (s *countingStore) CreateGeneratedPlan(context.Context, PlanInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) CreatePlan(context.Context, PlanInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) GetPlan(context.Context, string, string) (Plan, error) {
	s.writes++
	return Plan{}, ErrNotFound
}

func (s *countingStore) EditSkill(context.Context, EditSkillInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) SubmitPlan(context.Context, ReviewInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) RequestChanges(context.Context, ReviewInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) ApprovePlan(context.Context, ReviewInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) CopyApprovedPlan(context.Context, string, string) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) ReplacePlan(context.Context, ReplacePlanInput) (Plan, error) {
	s.writes++
	return Plan{}, nil
}

func (s *countingStore) CreateSession(context.Context, string, string) (Session, error) {
	s.writes++
	return Session{}, nil
}

func (s *countingStore) ListSessions(context.Context, string, string) ([]Session, error) {
	s.writes++
	return nil, nil
}

func (s *countingStore) GetSession(context.Context, string, string) (Session, error) {
	s.writes++
	return Session{}, ErrNotFound
}

func (s *countingStore) StartSession(context.Context, string, string, string, time.Time) (Session, error) {
	s.writes++
	return Session{}, nil
}

func (s *countingStore) SaveAnswers(context.Context, string, string, string, []AnswerInput) (Session, error) {
	s.writes++
	return Session{}, nil
}

func (s *countingStore) SubmitSession(context.Context, string, string, string, time.Time, []Proposal) (Session, error) {
	s.writes++
	return Session{}, nil
}

func (s *countingStore) MarkSessionConfirmed(context.Context, string, string, time.Time) (Session, error) {
	s.writes++
	return Session{}, nil
}
