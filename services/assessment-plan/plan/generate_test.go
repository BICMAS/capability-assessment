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
	router := NewRouter(nil, []byte(httpSecret), store, HTTPFramework{BaseURL: framework.URL, Client: framework.Client()}, nil, nil)
	token := sign(t, adaClaims())
	first := postGenerate(t, router, token, assignment.ID, http.StatusCreated)
	var plan openapi.Plan
	if err := json.Unmarshal(first.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.VersionNumber != 1 || plan.Status != StatusGenerated || len(plan.Skills) != 5 {
		t.Fatalf("plan = %+v", plan)
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
		if skill.RequirementID == negotiationReq {
			if skill.RubricNeedsReview || len(skill.Rubric) != len(statements) {
				t.Fatalf("negotiation rubric %+v", skill.Rubric)
			}
			for i, statement := range statements {
				if skill.Rubric[i].Descriptor != statement || !strings.Contains(original, statement) {
					t.Fatalf("anchor %d = %q", i, skill.Rubric[i].Descriptor)
				}
			}
			continue
		}
		if !skill.RubricNeedsReview || len(skill.Rubric) != 0 {
			t.Fatalf("missing indicators should need rubric review: %+v", skill)
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
	draft, err := buildGeneratedDraft(negotiationRoleID, version)
	if err != nil {
		t.Fatal(err)
	}
	draft[0].RequirementID = "someone-elses-requirement"
	if err := validateGeneratedDraft(negotiationRoleID, version, draft); err == nil {
		t.Fatal("expected rejection")
	}
}
