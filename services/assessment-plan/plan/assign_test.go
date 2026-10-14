package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bicmas-academy/learn/services/assessment-plan/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

const (
	adaID              = "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11"
	publishedVersionID = "aa11c2d3-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
	draftVersionID     = "bb22c3d4-5e6f-4a7b-8c8d-0e1f2a3b4c5d"
	negotiationRoleID  = "cc33d4e5-6f70-4b8c-9d0e-1f2a3b4c5d6e"
	otherRoleID        = "dd44e5f6-7081-4c9d-8e1f-2a3b4c5d6e7f"
	outsiderID         = "ee55f607-8192-4d0e-9f2a-3b4c5d6e7f80"
)

func TestAdaAssignsEliToThePublishedNegotiationRole(t *testing.T) {
	store := NewMemory()
	publisher := &recordingPublisher{}
	var directoryWrites int
	directory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			directoryWrites++
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		switch strings.TrimPrefix(r.URL.Path, "/people/") {
		case eliID:
			writeProfile(w, eliID, "Area Sales Manager")
		case mayaID:
			writeProfile(w, mayaID, "Team Manager")
		default:
			http.NotFound(w, r)
		}
	}))
	defer directory.Close()
	framework := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/framework-versions/")
		status := "published"
		if id == draftVersionID {
			status = "draft"
		}
		if id != publishedVersionID && id != draftVersionID {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     id,
			"status": status,
			"roles": []map[string]string{{
				"id":   negotiationRoleID,
				"name": "Area Sales Manager",
			}},
			"capabilities": []map[string]string{{
				"id":   capabilityID,
				"name": "Negotiation",
			}},
		})
	}))
	defer framework.Close()

	handler := NewRouter(nil, []byte(httpSecret), store,
		HTTPFramework{BaseURL: framework.URL},
		HTTPDirectory{BaseURL: directory.URL},
		publisher,
	)
	token := sign(t, adaClaims())

	first := postAssign(t, handler, token, publishedVersionID, negotiationRoleID, []openapi.AssignPerson{
		{PersonID: eliID},
		{PersonID: outsiderID},
	})
	if first.Code != http.StatusOK {
		t.Fatalf("assign = %d body = %s", first.Code, first.Body.String())
	}
	results := decodeBatch(t, first)
	if len(results) != 2 || results[0].Outcome != OutcomeAssigned || results[0].PersonID != eliID || results[1].Outcome != OutcomeFailed || results[1].Reason != ReasonOutside {
		t.Fatalf("results = %#v", results)
	}
	if results[0].AssignmentID == nil {
		t.Fatal("eli assignment id is missing")
	}
	eli, err := store.GetAssignment(context.Background(), organisationID, *results[0].AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if eli.FrameworkVersionID != publishedVersionID || eli.RoleID != negotiationRoleID || eli.ExceptionReason != "" || eli.AssignedByPersonID != adaID || eli.PersonID != eliID {
		t.Fatalf("eli = %#v", eli)
	}
	if len(publisher.events) != 1 || publisher.events[0].ActorPersonID != adaID || publisher.events[0].PersonID != eliID || publisher.events[0].Type != "assessment.assigned" {
		t.Fatalf("events = %#v", publisher.events)
	}

	second := postAssign(t, handler, token, publishedVersionID, negotiationRoleID, []openapi.AssignPerson{{PersonID: eliID}})
	skipped := decodeBatch(t, second)
	if second.Code != http.StatusOK || len(skipped) != 1 || skipped[0].Outcome != OutcomeSkipped || skipped[0].AssignmentID == nil || *skipped[0].AssignmentID != eli.ID || len(publisher.events) != 1 {
		t.Fatalf("second = %d %#v events = %d", second.Code, skipped, len(publisher.events))
	}

	unpublished := postAssign(t, handler, token, draftVersionID, negotiationRoleID, []openapi.AssignPerson{{PersonID: eliID}})
	if unpublished.Code != http.StatusConflict {
		t.Fatalf("unpublished = %d body = %s", unpublished.Code, unpublished.Body.String())
	}
	if _, err := store.FindAssignment(context.Background(), organisationID, eliID, draftVersionID, negotiationRoleID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft assignment error = %v", err)
	}

	missingRole := postAssign(t, handler, token, publishedVersionID, otherRoleID, []openapi.AssignPerson{{PersonID: eliID}})
	if missingRole.Code != http.StatusBadRequest {
		t.Fatalf("missing role = %d", missingRole.Code)
	}

	withoutReason := postAssign(t, handler, token, publishedVersionID, negotiationRoleID, []openapi.AssignPerson{{PersonID: mayaID}})
	mayaFailed := decodeBatch(t, withoutReason)
	if withoutReason.Code != http.StatusOK || len(mayaFailed) != 1 || mayaFailed[0].Outcome != OutcomeFailed || mayaFailed[0].Reason != ReasonTitle {
		t.Fatalf("maya without reason = %d %#v", withoutReason.Code, mayaFailed)
	}
	if _, err := store.FindAssignment(context.Background(), organisationID, mayaID, publishedVersionID, negotiationRoleID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("maya stored without an exception: %v", err)
	}

	reason := "Maya is covering the published role."
	withReason := postAssign(t, handler, token, publishedVersionID, negotiationRoleID, []openapi.AssignPerson{{PersonID: mayaID, ExceptionReason: &reason}})
	mayaAssigned := decodeBatch(t, withReason)
	if withReason.Code != http.StatusOK || len(mayaAssigned) != 1 || mayaAssigned[0].Outcome != OutcomeAssigned {
		t.Fatalf("maya with reason = %d %#v", withReason.Code, mayaAssigned)
	}
	if mayaAssigned[0].AssignmentID == nil {
		t.Fatal("maya assignment id is missing")
	}
	maya, err := store.GetAssignment(context.Background(), organisationID, *mayaAssigned[0].AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if maya.ExceptionReason != reason || maya.RoleID != negotiationRoleID || maya.FrameworkVersionID != publishedVersionID {
		t.Fatalf("maya = %#v", maya)
	}
	if directoryWrites != 0 {
		t.Fatalf("directory writes = %d", directoryWrites)
	}
}

func adaClaims() token.Claims {
	return token.Claims{
		PersonID:       adaID,
		OrganisationID: organisationID,
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	}
}

func writeProfile(w http.ResponseWriter, personID, title string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":    personID,
		"roles": []map[string]string{{"id": "role-title", "title": title}},
	})
}

func postAssign(t *testing.T, handler http.Handler, token, versionID, roleID string, people []openapi.AssignPerson) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(openapi.AssignRequest{
		FrameworkVersionID: versionID,
		RoleID:             roleID,
		People:             people,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/assignments", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeBatch(t *testing.T, response *httptest.ResponseRecorder) []openapi.AssignmentResult {
	t.Helper()
	var batch openapi.AssignmentBatch
	if err := json.Unmarshal(response.Body.Bytes(), &batch); err != nil {
		t.Fatal(err)
	}
	return batch.Results
}

type recordingPublisher struct {
	events []AssessmentAssigned
}

func (p *recordingPublisher) PublishPlanApproved(context.Context, PlanApproved) error {
	return nil
}

func (p *recordingPublisher) PublishAssessmentAssigned(_ context.Context, event AssessmentAssigned) error {
	p.events = append(p.events, event)
	return nil
}
