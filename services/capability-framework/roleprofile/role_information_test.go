package roleprofile

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bicmas-academy/learn/services/token"
)

func TestClarificationFollowsTheRecordedRole(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Regional Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "hr-1", "")
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptProvider{model: "fake-role-model", clarifyErr: errors.New("provider down")}
	workflow := NewWorkflow(store, provider, testLimits(), nil)

	missing, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "hr-1")
	if err != nil || missing.Complete {
		t.Fatalf("missing = complete %v %v", missing.Complete, err)
	}
	var outcomes Question
	for _, question := range missing.Questions {
		if question.Field == "businessOutcomes" {
			outcomes = question
		}
		if question.AuthoredBy != QuestionService {
			t.Fatalf("missing question origin = %s", question.AuthoredBy)
		}
	}
	if outcomes.ID == "" || !strings.Contains(outcomes.Question, "three to five") || !strings.Contains(outcomes.Question, "this role") || outcomes.Reason == "" {
		t.Fatalf("outcomes question = %#v", outcomes)
	}
	if missing.Snapshot.BusinessOutcomes != "" || provider.clarifyCalls != 0 {
		t.Fatalf("invented outcomes %q calls %d", missing.Snapshot.BusinessOutcomes, provider.clarifyCalls)
	}

	size := 6
	saved, err := store.UpdateContext(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision, Context{
		Purpose:          "Own regional revenue and the customer relationships for the territory",
		Responsibilities: "Lead the sales cycle from qualification through renewal with the team",
		BusinessOutcomes: "Grow retained revenue and keep the renewal rate above the plan",
		DecisionScope:    "Approve standard discounts within the published pricing guidance",
		Industry:         "Business services sold to commercial customers",
		Region:           "West Africa operating region",
		TeamSize:         &size,
		ReportingLevel:   ReportingManager,
		OrganogramID:     "chart-1",
		PositionID:       "position-1",
		BusinessUnit:     "Commercial",
		SubIndustry:      "Field sales",
		Sources:          []Source{{Title: "Approved role brief", ReferenceURI: "doc:role-brief", Kind: SourceJobDescription, Body: "Owns the regional sales plan and the renewal book.", Origin: OriginUser}},
	})
	if err != nil || saved.PositionID != "position-1" || saved.Sources[0].Body == "" || saved.Sources[0].Origin != OriginUser {
		t.Fatalf("save = %#v %v", saved, err)
	}
	enough, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "hr-1")
	if err != nil || !enough.Complete || len(enough.Questions) != len(missing.Questions) {
		t.Fatalf("enough = complete %v questions %d %v", enough.Complete, len(enough.Questions), err)
	}

	other, err := store.CreateRole(ctx, "org-a", "Account Manager")
	if err != nil {
		t.Fatal(err)
	}
	ambiguous, err := store.CreateDraft(ctx, "org-a", other.ID, "hr-1", "")
	if err != nil {
		t.Fatal(err)
	}
	short, err := store.UpdateContext(ctx, "org-a", ambiguous.Version.ID, "hr-1", ambiguous.Version.Revision, Context{
		Purpose: "Own the region", Responsibilities: "Lead the team", BusinessOutcomes: "Grow revenue", DecisionScope: "Approve discounts",
		Industry: "Services", Region: "Africa", TeamSize: &size, ReportingLevel: ReportingManager,
		OrganogramID: "chart-1", PositionID: "position-1",
		Sources: []Source{{Title: "Approved role brief", ReferenceURI: "doc:role-brief"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := workflow.Clarify(ctx, "org-a", short.ID, "hr-1")
	if err != nil || provider.clarifyCalls == 0 {
		t.Fatalf("provider failure = %v calls %d", err, provider.clarifyCalls)
	}
	if failed.Snapshot.Purpose != "Own the region" {
		t.Fatalf("purpose changed to %q", failed.Snapshot.Purpose)
	}
	var purpose Question
	for _, question := range failed.Questions {
		if question.AnsweredAt == nil && question.AuthoredBy != QuestionModelUnavailable {
			t.Fatalf("failed question = %#v", question)
		}
		if question.Field == "purpose" {
			purpose = question
		}
	}
	corrected, err := workflow.Answer(ctx, "org-a", short.ID, "hr-1", []AnswerInput{{QuestionID: purpose.ID, Value: "Own regional revenue and the customer relationships"}})
	if err != nil {
		t.Fatal(err)
	}
	again, err := workflow.Answer(ctx, "org-a", short.ID, "hr-1", []AnswerInput{{QuestionID: purpose.ID, Value: "Own regional revenue, retention, and the customer plan"}})
	if err != nil || !strings.Contains(again.Snapshot.Purpose, "retention") || corrected.Snapshot.Purpose == again.Snapshot.Purpose {
		t.Fatalf("correction = %q %v", again.Snapshot.Purpose, err)
	}
}

func TestRoleInformationRejectsInvalidInputAndRestrictedRoles(t *testing.T) {
	store := NewMemory()
	claims := token.Claims{PersonID: "hr-1", OrganisationID: "org-a", Roles: []string{"hr"}}
	api := newAPI(store, &claims)
	role := callJSON[Role](t, api, http.MethodPost, "/profile-roles", `{"name":"Regional Sales Manager"}`)
	created := callJSON[Profile](t, api, http.MethodPost, "/role-profiles", `{"roleId":"`+role.Body.ID+`"}`)
	invalid := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":1,"purpose":"Sell","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"manager","positionId":"position-1","sources":[]}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("position without organogram = %d %s", invalid.Code, invalid.Raw)
	}
	level := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":1,"purpose":"Sell","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"boss","organogramId":"chart-1","positionId":"position-1","sources":[]}`)
	if level.Code != http.StatusBadRequest {
		t.Fatalf("reporting level = %d", level.Code)
	}

	claims.Roles = []string{"employee"}
	denied, _ := api.call(http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/clarification", "")
	if denied != http.StatusForbidden {
		t.Fatalf("employee clarification = %d", denied)
	}
	uploadDenied := postFile(t, api, created.Body.Version.ID, "role.txt", []byte("Owns the region"))
	if uploadDenied != http.StatusForbidden {
		t.Fatalf("employee upload = %d", uploadDenied)
	}
	claims.Roles = []string{"manager"}
	managerDenied, _ := api.call(http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/clarification", "")
	if managerDenied != http.StatusForbidden {
		t.Fatalf("manager clarification = %d", managerDenied)
	}

	claims.Roles = []string{"hr"}
	allowed, _ := api.call(http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/clarification", "")
	if allowed != http.StatusOK {
		t.Fatalf("hr clarification = %d", allowed)
	}
	stored := postFile(t, api, created.Body.Version.ID, "role.txt", []byte("Owns the regional sales plan."))
	if stored != http.StatusCreated {
		t.Fatalf("txt upload = %d", stored)
	}
	mismatched := postFile(t, api, created.Body.Version.ID, "role.pdf", []byte("not a pdf"))
	if mismatched != http.StatusBadRequest {
		t.Fatalf("pdf upload = %d", mismatched)
	}
	current := callJSON[Profile](t, api, http.MethodGet, "/role-profiles/"+created.Body.ID, "")
	if current.Body.Version.Purpose != "" {
		t.Fatalf("upload wrote purpose %q", current.Body.Version.Purpose)
	}
}

func postFile(t *testing.T, api apiClient, versionID, name string, body []byte) int {
	t.Helper()
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/role-profile-versions/"+versionID+"/job-description", &payload)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	api.handler.ServeHTTP(response, request)
	if response.Code == http.StatusCreated && (strings.Contains(response.Body.String(), "model extracted") || !strings.Contains(response.Body.String(), "not")) {
		t.Fatalf("upload body = %s", response.Body.String())
	}
	return response.Code
}

func TestAttachmentNameDropsHeaderInjection(t *testing.T) {
	got := safeAttachmentName("role\r\nSet-Cookie: a\".pdf")
	if strings.ContainsAny(got, "\r\n\";") {
		t.Fatalf("name = %q", got)
	}
	if safeAttachmentName("\r\n") != "download" {
		t.Fatal("empty name was not replaced")
	}
}
