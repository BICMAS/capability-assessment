package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/assessment-plan/openapi"
)

func TestChooseMethod(t *testing.T) {
	cases := []struct {
		name   string
		method string
		reason string
	}{
		{name: "Negotiation Knowledge", method: MethodKnowledgeTest, reason: ReasonKnowledge},
		{name: "Negotiation", method: MethodAvatarRoleplay, reason: ReasonNegotiation},
		{name: "Client Communication", method: MethodAvatarRoleplay, reason: ReasonCommunication},
		{name: "Sales Leadership", method: MethodCaseStudy, reason: ReasonDefaultMethod},
	}
	for _, tc := range cases {
		method, reason := chooseMethod(tc.name)
		if method != tc.method || reason != tc.reason {
			t.Fatalf("%s -> %s (%s)", tc.name, method, reason)
		}
	}
}

func TestGenerateFromElisNegotiationAssignment(t *testing.T) {
	const (
		negotiationCapability = "d0e1f2a3-b4c5-4d6e-8f70-1a2b3c4d5e6f"
		leadershipCapability  = "e1f2a3b4-c5d6-4e7f-8091-2a3b4c5d6e70"
		customerCapability    = "f2a3b4c5-d6e7-4f80-9123-3a4b5c6d7e81"
		commercialCapability  = "a3b4c5d6-e7f8-4091-a234-4b5c6d7e8f92"
		executionCapability   = "b4c5d6e7-f809-41a2-b345-5c6d7e8f90a3"
		negotiationReq        = "c5d6e7f8-091a-42b3-c456-6d7e8f90a1b4"
		leadershipReq         = "d6e7f809-1a2b-43c4-d567-7e8f90a1b2c5"
		customerReq           = "e7f8091a-2b3c-44d5-e678-8f90a1b2c3d6"
		commercialReq         = "f8091a2b-3c4d-45e6-f789-90a1b2c3d4e7"
		executionReq          = "091a2b3c-4d5e-46f7-8091-a1b2c3d4e5f8"
	)
	statements := []string{
		"Describes a commercial negotiation.",
		"Prepares a position before the negotiation.",
		"Explores the buyer interests before proposing terms.",
		"Trades concessions for commercial value.",
		"Reaches a commercial agreement with the buyer.",
	}
	indicators := make([]map[string]any, len(statements))
	for i, statement := range statements {
		indicators[i] = map[string]any{"level": i, "indicators": []map[string]string{{"statement": statement}}}
	}
	document := map[string]any{
		"id":     publishedVersionID,
		"status": "published",
		"roles": []map[string]string{
			{"id": negotiationRoleID, "name": "Area Sales Manager"},
			{"id": otherRoleID, "name": "Other Role"},
		},
		"capabilities": []map[string]any{
			{"id": negotiationCapability, "name": "Negotiation", "levels": indicators},
			{"id": leadershipCapability, "name": "Sales Leadership"},
			{"id": customerCapability, "name": "Customer Management"},
			{"id": commercialCapability, "name": "Commercial Acumen"},
			{"id": executionCapability, "name": "Execution Excellence"},
		},
		"requirements": []map[string]any{
			{"id": negotiationReq, "roleId": negotiationRoleID, "capabilityId": negotiationCapability, "requiredLevel": 4},
			{"id": leadershipReq, "roleId": negotiationRoleID, "capabilityId": leadershipCapability, "requiredLevel": 3},
			{"id": customerReq, "roleId": negotiationRoleID, "capabilityId": customerCapability, "requiredLevel": 3},
			{"id": commercialReq, "roleId": negotiationRoleID, "capabilityId": commercialCapability, "requiredLevel": 3},
			{"id": executionReq, "roleId": negotiationRoleID, "capabilityId": executionCapability, "requiredLevel": 3},
			{"id": "11223344-5566-4777-8888-99aabbccddee", "roleId": otherRoleID, "capabilityId": negotiationCapability, "requiredLevel": 2},
		},
	}
	published, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	original := string(published)
	var writes int
	framework := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(published)
	}))
	defer framework.Close()

	store := NewMemory()
	assignment := assignEli(t, store)
	modelText := map[string]string{
		negotiationReq: "Ask Eli to trade a concession with the buyer.",
		leadershipReq:  "Ask Eli to write how the team will run the quarter.",
		customerReq:    "Ask Eli to recover a key account.",
		commercialReq:  "Ask Eli to explain the margin on the deal.",
		executionReq:   "Ask Eli to sequence the delivery plan.",
	}
	drafter := &scriptDrafter{body: modelDraftJSON(t, []modelSkill{
		{RequirementID: negotiationReq, RecommendedMethod: MethodAvatarRoleplay, Instructions: "Run the negotiation as a live buyer conversation.", AssessmentItems: []string{modelText[negotiationReq]}, Rubric: []modelBand{{Level: 4, Descriptor: "Trades a concession only when the buyer gives commercial value."}}},
		{RequirementID: leadershipReq, RecommendedMethod: MethodCaseStudy, Instructions: "Use a written leadership case.", AssessmentItems: []string{modelText[leadershipReq]}, Rubric: []modelBand{{Level: 3, Descriptor: "Sets the team's commercial priorities."}}},
		{RequirementID: customerReq, RecommendedMethod: MethodCaseStudy, Instructions: "Use a written customer case.", AssessmentItems: []string{modelText[customerReq]}, Rubric: []modelBand{{Level: 2, Descriptor: "Recovers the account relationship."}}},
		{RequirementID: commercialReq, RecommendedMethod: MethodCaseStudy, Instructions: "Use a written commercial case.", AssessmentItems: []string{modelText[commercialReq]}, Rubric: []modelBand{{Level: 1, Descriptor: "Explains the margin."}}},
		{RequirementID: executionReq, RecommendedMethod: MethodCaseStudy, Instructions: "Use a written execution case.", AssessmentItems: []string{modelText[executionReq]}, Rubric: []modelBand{{Level: 0, Descriptor: "Sequences the work."}}},
	})}
	router := NewRouter(nil, []byte(httpSecret), store, HTTPFramework{BaseURL: framework.URL, Client: framework.Client()}, nil, nil, drafter, nil, nil)
	token := sign(t, adaClaims())
	first := postGenerate(t, router, token, assignment.ID, http.StatusCreated)
	var plan openapi.Plan
	if err := json.Unmarshal(first.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.VersionNumber != 1 || plan.Status != StatusGenerated || len(plan.Skills) != 5 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.PromptVersion == nil || *plan.PromptVersion != PromptVersion || plan.ModelName == nil || *plan.ModelName != "draft-model" {
		t.Fatalf("provenance = %v %v", plan.PromptVersion, plan.ModelName)
	}
	if !strings.Contains(string(drafter.request), MethodAvatarRoleplay) || !strings.Contains(string(drafter.request), `"recommendedMethod"`) {
		t.Fatalf("constraint = %s", drafter.request)
	}
	wantMethods := map[string]string{
		negotiationReq: MethodAvatarRoleplay,
		leadershipReq:  MethodCaseStudy,
		customerReq:    MethodCaseStudy,
		commercialReq:  MethodCaseStudy,
		executionReq:   MethodCaseStudy,
	}
	for _, skill := range plan.Skills {
		if skill.RecommendedMethod != wantMethods[skill.RequirementID] || skill.Rationale == "" || len(skill.AssessmentItems) != 1 || !skill.RequiresSMEReview {
			t.Fatalf("skill %+v", skill)
		}
		if skill.AssessmentItems[0].Prompt != modelText[skill.RequirementID] || strings.Contains(skill.AssessmentItems[0].Prompt, "Assess ") {
			t.Fatalf("item = %q", skill.AssessmentItems[0].Prompt)
		}
		if skill.RequirementID == negotiationReq {
			if skill.RubricNeedsReview || len(skill.Rubric) != 1 || skill.Rubric[0].Descriptor == statements[4] {
				t.Fatalf("negotiation rubric %+v", skill.Rubric)
			}
			if skill.RecommendedMethod != MethodAvatarRoleplay {
				t.Fatalf("method = %s", skill.RecommendedMethod)
			}
			continue
		}
		if !skill.RubricNeedsReview || len(skill.Rubric) != 1 {
			t.Fatalf("model rubric %+v", skill)
		}
	}
	postGenerate(t, router, token, assignment.ID, http.StatusConflict)
	if _, err := store.GetPlan(context.Background(), organisationID, plan.ID); err != nil {
		t.Fatal(err)
	}
	if writes != 0 || string(published) != original {
		t.Fatalf("framework writes=%d", writes)
	}
	postGenerate(t, router, sign(t, mayaClaims()), assignment.ID, http.StatusForbidden)
}

func postGenerate(t *testing.T, handler http.Handler, token, assignmentID string, status int) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/assessment-plans", bytes.NewReader([]byte(`{"assignmentId":"`+assignmentID+`"}`)))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	return response
}

func TestGeneratedDraftRejectsARequirementOutsideTheRole(t *testing.T) {
	version := FrameworkVersion{
		Status: "published",
		Requirements: []FrameworkRequirement{{
			ID: "req-1", RoleID: negotiationRoleID, CapabilityID: "cap-1", RequiredLevel: 2,
		}},
		Capabilities: []FrameworkCapability{{ID: "cap-1", Name: "Negotiation"}},
	}
	raw := modelDraftJSON(t, []modelSkill{{
		RequirementID: "someone-elses-requirement", RecommendedMethod: MethodAvatarRoleplay,
		Instructions: "Talk to the buyer.", AssessmentItems: []string{"Open the negotiation."},
		Rubric: []modelBand{{Level: 2, Descriptor: "Opens with a commercial position."}},
	}})
	if _, err := applyModelDraft(negotiationRoleID, version, raw); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestModelDraftThatChangesTheMethodIsRejected(t *testing.T) {
	store, router := negotiationRouter(t, &scriptDrafter{body: modelDraftJSON(t, eliModelSkills(MethodCaseStudy))})
	assignment := assignEli(t, store)
	postGenerate(t, router, sign(t, adaClaims()), assignment.ID, http.StatusBadRequest)
	if len(store.plans) != 0 {
		t.Fatalf("saved %d plans", len(store.plans))
	}
}

func TestModelDraftThatInventsALevelIsRejected(t *testing.T) {
	skills := eliModelSkills(MethodAvatarRoleplay)
	skills[0].Rubric = []modelBand{{Level: 3, Descriptor: "A level the framework does not publish."}}
	store, router := negotiationRouter(t, &scriptDrafter{body: modelDraftJSON(t, skills)})
	assignment := assignEli(t, store)
	postGenerate(t, router, sign(t, adaClaims()), assignment.ID, http.StatusBadRequest)
	if len(store.plans) != 0 {
		t.Fatalf("saved %d plans", len(store.plans))
	}
}

func TestModelDraftThatOmitsASkillIsRejected(t *testing.T) {
	store, router := negotiationRouter(t, &scriptDrafter{body: modelDraftJSON(t, eliModelSkills(MethodAvatarRoleplay)[:4])})
	assignment := assignEli(t, store)
	postGenerate(t, router, sign(t, adaClaims()), assignment.ID, http.StatusBadRequest)
	if len(store.plans) != 0 {
		t.Fatalf("saved %d plans", len(store.plans))
	}
}

func TestInvalidModelJSONDoesNotSaveAPlan(t *testing.T) {
	store, router := negotiationRouter(t, &scriptDrafter{body: []byte("not json")})
	assignment := assignEli(t, store)
	postGenerate(t, router, sign(t, adaClaims()), assignment.ID, http.StatusBadRequest)
	if len(store.plans) != 0 {
		t.Fatalf("saved %d plans", len(store.plans))
	}
}

func TestUnconfiguredProviderDoesNotSaveAPlan(t *testing.T) {
	store, router := negotiationRouter(t, HTTPDrafter{})
	assignment := assignEli(t, store)
	postGenerate(t, router, sign(t, adaClaims()), assignment.ID, http.StatusServiceUnavailable)
	if len(store.plans) != 0 {
		t.Fatalf("saved %d plans", len(store.plans))
	}
}

func assignEli(t *testing.T, store *Memory) Assignment {
	t.Helper()
	assignment, err := store.CreateAssignment(context.Background(), AssignmentInput{
		OrganisationID:     organisationID,
		PersonID:           eliID,
		FrameworkVersionID: publishedVersionID,
		RoleID:             negotiationRoleID,
		AssignedByPersonID: adaID,
		AssignedAt:         time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return assignment
}

func negotiationRouter(t *testing.T, drafter Drafter) (*Memory, http.Handler) {
	t.Helper()
	document := map[string]any{
		"id": publishedVersionID, "status": "published",
		"roles": []map[string]string{{"id": negotiationRoleID, "name": "Area Sales Manager"}},
		"capabilities": []map[string]any{
			{"id": "d0e1f2a3-b4c5-4d6e-8f70-1a2b3c4d5e6f", "name": "Negotiation", "levels": []map[string]any{{"level": 4, "indicators": []map[string]string{{"statement": "Reaches a commercial agreement with the buyer."}}}}},
			{"id": "e1f2a3b4-c5d6-4e7f-8091-2a3b4c5d6e70", "name": "Sales Leadership"},
			{"id": "f2a3b4c5-d6e7-4f80-9123-3a4b5c6d7e81", "name": "Customer Management"},
			{"id": "a3b4c5d6-e7f8-4091-a234-4b5c6d7e8f92", "name": "Commercial Acumen"},
			{"id": "b4c5d6e7-f809-41a2-b345-5c6d7e8f90a3", "name": "Execution Excellence"},
		},
		"requirements": []map[string]any{
			{"id": "c5d6e7f8-091a-42b3-c456-6d7e8f90a1b4", "roleId": negotiationRoleID, "capabilityId": "d0e1f2a3-b4c5-4d6e-8f70-1a2b3c4d5e6f", "requiredLevel": 4},
			{"id": "d6e7f809-1a2b-43c4-d567-7e8f90a1b2c5", "roleId": negotiationRoleID, "capabilityId": "e1f2a3b4-c5d6-4e7f-8091-2a3b4c5d6e70", "requiredLevel": 3},
			{"id": "e7f8091a-2b3c-44d5-e678-8f90a1b2c3d6", "roleId": negotiationRoleID, "capabilityId": "f2a3b4c5-d6e7-4f80-9123-3a4b5c6d7e81", "requiredLevel": 3},
			{"id": "f8091a2b-3c4d-45e6-f789-90a1b2c3d4e7", "roleId": negotiationRoleID, "capabilityId": "a3b4c5d6-e7f8-4091-a234-4b5c6d7e8f92", "requiredLevel": 3},
			{"id": "091a2b3c-4d5e-46f7-8091-a1b2c3d4e5f8", "roleId": negotiationRoleID, "capabilityId": "b4c5d6e7-f809-41a2-b345-5c6d7e8f90a3", "requiredLevel": 3},
		},
	}
	published, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	framework := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("framework %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(published)
	}))
	t.Cleanup(framework.Close)
	store := NewMemory()
	router := NewRouter(nil, []byte(httpSecret), store, HTTPFramework{BaseURL: framework.URL, Client: framework.Client()}, nil, nil, drafter, nil, nil)
	return store, router
}

func eliModelSkills(negotiationMethod string) []modelSkill {
	return []modelSkill{
		{RequirementID: "c5d6e7f8-091a-42b3-c456-6d7e8f90a1b4", RecommendedMethod: negotiationMethod, Instructions: "Run the negotiation as a live buyer conversation.", AssessmentItems: []string{"Ask Eli to trade a concession."}, Rubric: []modelBand{{Level: 4, Descriptor: "Trades a concession for commercial value."}}},
		{RequirementID: "d6e7f809-1a2b-43c4-d567-7e8f90a1b2c5", RecommendedMethod: MethodCaseStudy, Instructions: "Use a written leadership case.", AssessmentItems: []string{"Ask Eli to lead the quarter."}, Rubric: []modelBand{{Level: 3, Descriptor: "Sets the team's priorities."}}},
		{RequirementID: "e7f8091a-2b3c-44d5-e678-8f90a1b2c3d6", RecommendedMethod: MethodCaseStudy, Instructions: "Use a written customer case.", AssessmentItems: []string{"Ask Eli to recover the account."}, Rubric: []modelBand{{Level: 2, Descriptor: "Recovers the account."}}},
		{RequirementID: "f8091a2b-3c4d-45e6-f789-90a1b2c3d4e7", RecommendedMethod: MethodCaseStudy, Instructions: "Use a written commercial case.", AssessmentItems: []string{"Ask Eli to explain the margin."}, Rubric: []modelBand{{Level: 1, Descriptor: "Explains the margin."}}},
		{RequirementID: "091a2b3c-4d5e-46f7-8091-a1b2c3d4e5f8", RecommendedMethod: MethodCaseStudy, Instructions: "Use a written execution case.", AssessmentItems: []string{"Ask Eli to sequence the work."}, Rubric: []modelBand{{Level: 0, Descriptor: "Sequences the work."}}},
	}
}

func modelDraftJSON(t *testing.T, skills []modelSkill) []byte {
	t.Helper()
	raw, err := json.Marshal(modelDraft{Skills: skills})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type scriptDrafter struct {
	body    []byte
	request []byte
}

func (d *scriptDrafter) ModelName() string { return "draft-model" }

func (d *scriptDrafter) Draft(_ context.Context, request DraftRequest) ([]byte, error) {
	raw, err := json.Marshal(request.Requirements)
	if err != nil {
		return nil, err
	}
	d.request = raw
	return d.body, nil
}
