package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEditApproveAndCopyNegotiationPlan(t *testing.T) {
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
	created, err := store.CreateGeneratedPlan(context.Background(), PlanInput{
		OrganisationID: organisationID,
		AssignmentID:   assignment.ID,
		Skills: []SkillInput{{
			RequirementID:     requirementID,
			CapabilityID:      capabilityID,
			RequiredLevel:     4,
			RecommendedMethod: MethodAvatarRoleplay,
			Rationale:         ReasonNegotiation,
			Instructions:      "Assess Negotiation at required level 4.",
			RequiresSMEReview: true,
			AssessmentItems:   []ItemInput{{Prompt: "Assess Negotiation at required level 4."}},
			Rubric:            []RubricInput{{Level: 4, Descriptor: "Reaches a commercial agreement with the buyer."}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	publisher := &reviewPublisher{}
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, publisher, nil, nil, nil, ProfileBridge{})
	token := sign(t, adaClaims())
	skillID := created.Skills[0].ID

	edited := jsonCall(t, handler, http.MethodPatch, "/plans/"+created.ID+"/skills/"+skillID, token, `{
		"recommendedMethod":"case_study",
		"instructions":"Run a written negotiation case.",
		"assessmentItems":[{"prompt":"Trade a concession for margin."}],
		"rubric":[{"level":4,"descriptor":"Trades concessions for commercial value."}]
	}`, http.StatusOK)
	if edited.Skills[0].RecommendedMethod != "case_study" || edited.Skills[0].EditedByPersonID != adaID {
		t.Fatalf("edited skill = %+v", edited.Skills[0])
	}
	if edited.Skills[0].Instructions != "Run a written negotiation case." || edited.Skills[0].AssessmentItems[0].Prompt != "Trade a concession for margin." || edited.Skills[0].Rubric[0].Descriptor != "Trades concessions for commercial value." {
		t.Fatalf("edited content = %+v", edited.Skills[0])
	}

	approved := jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/approval", token, "", http.StatusOK)
	if approved.Status != StatusApproved || approved.ApprovedByPersonID != adaID || approved.ApprovedAt == "" || approved.VersionNumber != 1 {
		t.Fatalf("approved = %+v", approved)
	}
	if len(publisher.approved) != 1 || publisher.approved[0].Type != "assessment.plan_approved" || publisher.approved[0].PlanID != created.ID || publisher.approved[0].ActorPersonID != adaID || publisher.approved[0].PersonID != eliID || publisher.approved[0].AssignmentID != assignment.ID {
		t.Fatalf("event = %+v", publisher.approved)
	}

	jsonCall(t, handler, http.MethodPatch, "/plans/"+created.ID+"/skills/"+skillID, token, `{
		"recommendedMethod":"knowledge_test",
		"instructions":"This edit must not save.",
		"assessmentItems":[{"prompt":"This edit must not save."}],
		"rubric":[{"level":4,"descriptor":"This edit must not save."}]
	}`, http.StatusConflict)

	stored, err := store.GetPlan(context.Background(), organisationID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusApproved || stored.Skills[0].RecommendedMethod != "case_study" || stored.Skills[0].Instructions != "Run a written negotiation case." {
		t.Fatalf("version 1 changed after approval: %+v", stored.Skills[0])
	}

	next := jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/versions", token, "", http.StatusCreated)
	if next.ID == created.ID || next.VersionNumber != 2 || next.Status != StatusDraft || next.Skills[0].RecommendedMethod != "case_study" {
		t.Fatalf("version 2 = %+v", next)
	}
	original, err := store.GetPlan(context.Background(), organisationID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if original.Status != StatusApproved || original.VersionNumber != 1 || original.ApprovedByPersonID != adaID || original.Skills[0].Instructions != "Run a written negotiation case." || original.Skills[0].Rubric[0].Descriptor != "Trades concessions for commercial value." {
		t.Fatalf("version 1 = %+v skill %+v", original, original.Skills[0])
	}
}

func TestRequestChangesRequiresAReason(t *testing.T) {
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
	created, err := store.CreateGeneratedPlan(context.Background(), PlanInput{
		OrganisationID: organisationID,
		AssignmentID:   assignment.ID,
		Skills:         negotiationSkill(),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, &reviewPublisher{}, nil, nil, nil, ProfileBridge{})
	token := sign(t, adaClaims())
	jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/submission", token, "", http.StatusOK)
	jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/change-requests", token, `{}`, http.StatusBadRequest)
	changed := jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/change-requests", token, `{"reason":"The negotiation rubric needs a concession anchor."}`, http.StatusOK)
	if changed.Status != StatusChangesRequested || changed.ChangeReason != "The negotiation rubric needs a concession anchor." {
		t.Fatalf("changes = %+v", changed)
	}
	edited := jsonCall(t, handler, http.MethodPatch, "/plans/"+created.ID+"/skills/"+created.Skills[0].ID, token, `{
		"recommendedMethod":"ai_avatar_roleplay",
		"instructions":"Revise the negotiation case.",
		"assessmentItems":[{"prompt":"Revise the negotiation case."}],
		"rubric":[{"level":3,"descriptor":"Agrees a next step the buyer accepts."}]
	}`, http.StatusOK)
	if edited.Status != StatusChangesRequested || edited.Skills[0].EditedByPersonID != adaID {
		t.Fatalf("edit during changes = %+v", edited)
	}
	jsonCall(t, handler, http.MethodPatch, "/plans/"+created.ID+"/skills/"+created.Skills[0].ID, sign(t, mayaClaims()), `{
		"recommendedMethod":"case_study",
		"instructions":"A manager cannot edit this skill.",
		"assessmentItems":[{"prompt":"A manager cannot edit this skill."}],
		"rubric":[{"level":3,"descriptor":"Agrees a next step the buyer accepts."}]
	}`, http.StatusForbidden)
}

type reviewedPlan struct {
	ID                 string `json:"id"`
	VersionNumber      int    `json:"versionNumber"`
	Status             string `json:"status"`
	ApprovedByPersonID string `json:"approvedByPersonId"`
	ApprovedAt         string `json:"approvedAt"`
	ChangeReason       string `json:"changeReason"`
	Skills             []struct {
		RecommendedMethod string `json:"recommendedMethod"`
		Instructions      string `json:"instructions"`
		EditedByPersonID  string `json:"editedByPersonId"`
		AssessmentItems   []struct {
			Prompt string `json:"prompt"`
		} `json:"assessmentItems"`
		Rubric []struct {
			Descriptor string `json:"descriptor"`
		} `json:"rubric"`
	} `json:"skills"`
}

func jsonCall(t *testing.T, handler http.Handler, method, path, token, body string, status int) reviewedPlan {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s status = %d body = %s", method, path, response.Code, response.Body.String())
	}
	if status == http.StatusConflict || status == http.StatusBadRequest || status == http.StatusForbidden {
		return reviewedPlan{}
	}
	var plan reviewedPlan
	if err := json.Unmarshal(response.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	return plan
}

type reviewPublisher struct {
	approved []PlanApproved
}

func (p *reviewPublisher) PublishAssessmentAssigned(context.Context, AssessmentAssigned) error {
	return nil
}

func (p *reviewPublisher) PublishAssessmentCompleted(context.Context, AssessmentCompleted) error {
	return nil
}

func (p *reviewPublisher) PublishPlanApproved(_ context.Context, event PlanApproved) error {
	p.approved = append(p.approved, event)
	return nil
}
