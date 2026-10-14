package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/assessment-gap/levels"
	"github.com/bicmas-academy/learn/services/assessment-plan/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func TestEliSubmitsTheApprovedNegotiationSession(t *testing.T) {
	store := NewMemory()
	assignment := assignEli(t, store)
	created, err := store.CreateGeneratedPlan(context.Background(), PlanInput{
		OrganisationID: organisationID,
		AssignmentID:   assignment.ID,
		Skills: []SkillInput{{
			RequirementID:     requirementID,
			CapabilityID:      capabilityID,
			RequiredLevel:     4,
			RecommendedMethod: MethodAvatarRoleplay,
			Rationale:         ReasonNegotiation,
			Instructions:      "Run the negotiation as a live buyer conversation.",
			RequiresSMEReview: true,
			AssessmentItems:   []ItemInput{{Prompt: "Ask Eli to trade a concession with the buyer."}},
			Rubric:            []RubricInput{{Level: 4, Descriptor: "Trades a concession only when the buyer gives commercial value."}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, scriptMatcher{level: 4}, nil)
	ada := sign(t, adaClaims())
	eli := sign(t, eliClaims())
	maya := sign(t, mayaClaims())

	postSession(t, handler, ada, created.ID, http.StatusConflict)
	if sessions := listSessions(t, handler, eli); len(sessions) != 0 {
		t.Fatalf("unapproved plan created %d sessions", len(sessions))
	}

	jsonCall(t, handler, http.MethodPost, "/plans/"+created.ID+"/approval", ada, "", http.StatusOK)
	sessions := listSessions(t, handler, eli)
	if len(sessions) != 1 || sessions[0].PersonID != eliID || sessions[0].Status != SessionAssigned || sessions[0].PlanID != created.ID {
		t.Fatalf("sessions = %+v", sessions)
	}
	if len(listSessions(t, handler, maya)) != 0 {
		t.Fatal("Maya received Eli's session")
	}
	body := getSession(t, handler, eli, sessions[0].ID, http.StatusOK)
	if strings.Contains(body, "rubric") || strings.Contains(body, "commercial value") || strings.Contains(body, "requiredLevel") || strings.Contains(body, "percent") {
		t.Fatalf("session exposed the plan draft: %s", body)
	}
	if !strings.Contains(body, "Run the negotiation as a live buyer conversation.") || !strings.Contains(body, "Ask Eli to trade a concession with the buyer.") {
		t.Fatalf("session = %s", body)
	}
	if mayaView := getSession(t, handler, maya, sessions[0].ID, http.StatusOK); strings.Contains(mayaView, "rubric") || strings.Contains(mayaView, "percentage") {
		t.Fatalf("proposal appeared before submit: %s", mayaView)
	}
	other := mayaClaims()
	other.ManagerOf = nil
	getSession(t, handler, sign(t, other), sessions[0].ID, http.StatusForbidden)

	var session openapi.Session
	if err := json.Unmarshal([]byte(body), &session); err != nil {
		t.Fatal(err)
	}
	call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/submission", eli, "", http.StatusConflict)
	call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/start", eli, "", http.StatusOK)
	answer := `{"answers":[{"itemId":"` + session.Skills[0].Items[0].ID + `","answer":"I traded a later delivery for a higher price."}]}`
	call(t, handler, http.MethodPut, "/assessment-sessions/"+session.ID+"/answers", eli, answer, http.StatusOK)
	submitted := call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/submission", eli, "", http.StatusOK)
	if !strings.Contains(submitted, `"submitted"`) || !strings.Contains(submitted, "I traded a later delivery for a higher price.") || strings.Contains(submitted, "percentage") || strings.Contains(submitted, "ai_interview") {
		t.Fatalf("submitted = %s", submitted)
	}
	call(t, handler, http.MethodPut, "/assessment-sessions/"+session.ID+"/answers", eli, answer, http.StatusConflict)
	proposed := call(t, handler, http.MethodGet, "/assessment-sessions?personId="+eliID, maya, "", http.StatusOK)
	if !strings.Contains(proposed, `"percentage":100`) || !strings.Contains(proposed, `"source":"ai_interview"`) || !strings.Contains(proposed, `"bandLevels":[4]`) {
		t.Fatalf("proposal = %s", proposed)
	}
}

type scriptMatcher struct {
	level int
	raw   string
	err   error
}

func (s scriptMatcher) Match(_ context.Context, request MatchRequest) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.raw != "" {
		return []byte(s.raw), nil
	}
	skills := make([]map[string]any, 0, len(request.Skills))
	for _, skill := range request.Skills {
		levels := make([]int, len(skill.Answers))
		for i := range levels {
			levels[i] = s.level
		}
		skills = append(skills, map[string]any{"requirementId": skill.RequirementID, "levels": levels})
	}
	return json.Marshal(map[string]any{"skills": skills})
}

func TestPercentageIsTheMeanBandOverFour(t *testing.T) {
	got, err := percentageFromBands([]int{1, 2})
	if err != nil || got != 38 {
		t.Fatalf("percentage = %d err = %v", got, err)
	}
	full, err := percentageFromBands([]int{4})
	if err != nil || full != 100 {
		t.Fatalf("full = %d err = %v", full, err)
	}
}

func TestABandOutsideTheRubricDoesNotStoreAProposal(t *testing.T) {
	store := NewMemory()
	assignment := assignEli(t, store)
	created := approveNegotiation(t, store, assignment.ID)
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, scriptMatcher{
		raw: `{"skills":[{"requirementId":"` + requirementID + `","levels":[1],"descriptor":"invented"}]}`,
	}, nil)
	eli := sign(t, eliClaims())
	session := startEli(t, handler, eli, created.ID)
	call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/submission", eli, "", http.StatusBadRequest)
	if sessions := listSessions(t, handler, eli); len(sessions) != 1 || sessions[0].Status != SessionStarted {
		t.Fatalf("sessions = %+v", sessions)
	}
}

func TestMayaOrAdaConfirmsTheProposalIntoAssessmentGap(t *testing.T) {
	for _, confirmer := range []token.Claims{mayaClaims(), adaClaims()} {
		store := NewMemory()
		assignment := assignEli(t, store)
		created := approveNegotiation(t, store, assignment.ID)
		var recorded []byte
		gapStore := levels.NewMemory()
		inner := levels.NewRouter(nil, []byte(httpSecret), gapStore, nil)
		gap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/attempts" && recorded == nil {
				body, _ := io.ReadAll(r.Body)
				recorded = body
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
			inner.ServeHTTP(w, r)
		}))
		t.Cleanup(gap.Close)
		handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, scriptMatcher{level: 4}, HTTPGap{BaseURL: gap.URL, Client: gap.Client()})
		eli := sign(t, eliClaims())
		session := startEli(t, handler, eli, created.ID)
		call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/submission", eli, "", http.StatusOK)
		if recorded != nil {
			t.Fatal("submit wrote to assessment-gap")
		}
		levelURL := "/levels?personId=" + eliID + "&frameworkVersionId=" + publishedVersionID + "&capabilityCode=" + capabilityID
		gapCall(t, gap, levelURL, eli, http.StatusNotFound)
		call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/confirmation", eli, "", http.StatusForbidden)
		unrelated := mayaClaims()
		unrelated.ManagerOf = nil
		call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/confirmation", sign(t, unrelated), "", http.StatusForbidden)
		confirmed := call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/confirmation", sign(t, confirmer), "", http.StatusOK)
		if !strings.Contains(confirmed, `"currentLevel":"4.0"`) || !strings.Contains(confirmed, `"gap":"0.0"`) || !strings.Contains(confirmed, `"percentage":100`) {
			t.Fatalf("confirmed = %s", confirmed)
		}
		if !strings.Contains(string(recorded), `"source":"ai_interview"`) || !strings.Contains(string(recorded), `"percentage":100`) || !strings.Contains(string(recorded), `"requiredLevel":4`) || !strings.Contains(string(recorded), session.ID) || !strings.Contains(string(recorded), capabilityID) {
			t.Fatalf("attempt = %s", recorded)
		}
		eliView := getSession(t, handler, eli, session.ID, http.StatusOK)
		if !strings.Contains(eliView, `"percentage":100`) || strings.Contains(eliView, "verified") || strings.Contains(eliView, "rubric") {
			t.Fatalf("session = %s", eliView)
		}
		explained := gapCall(t, gap, levelURL, eli, http.StatusOK)
		if !strings.Contains(explained, `"aiInterviewPercentage":100`) || !strings.Contains(explained, `"currentLevel":"4.0"`) || !strings.Contains(explained, `"gap":"0.0"`) {
			t.Fatalf("explanation = %s", explained)
		}
		call(t, handler, http.MethodPost, "/assessment-sessions/"+session.ID+"/confirmation", sign(t, confirmer), "", http.StatusConflict)
	}
}

func approveNegotiation(t *testing.T, store *Memory, assignmentID string) Plan {
	t.Helper()
	created, err := store.CreateGeneratedPlan(context.Background(), PlanInput{
		OrganisationID: organisationID,
		AssignmentID:   assignmentID,
		Skills: []SkillInput{{
			RequirementID:     requirementID,
			CapabilityID:      capabilityID,
			RequiredLevel:     4,
			RecommendedMethod: MethodAvatarRoleplay,
			Rationale:         ReasonNegotiation,
			Instructions:      "Run the negotiation as a live buyer conversation.",
			RequiresSMEReview: true,
			AssessmentItems:   []ItemInput{{Prompt: "Ask Eli to trade a concession with the buyer."}},
			Rubric:            []RubricInput{{Level: 4, Descriptor: "Trades a concession only when the buyer gives commercial value."}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := store.ApprovePlan(context.Background(), ReviewInput{OrganisationID: organisationID, PlanID: created.ID, ActorPersonID: adaID, At: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	return approved
}

func startEli(t *testing.T, handler http.Handler, eli, planID string) openapi.Session {
	t.Helper()
	sessions := listSessions(t, handler, eli)
	if len(sessions) != 1 || sessions[0].PlanID != planID {
		t.Fatalf("sessions = %+v", sessions)
	}
	call(t, handler, http.MethodPost, "/assessment-sessions/"+sessions[0].ID+"/start", eli, "", http.StatusOK)
	answer := `{"answers":[{"itemId":"` + sessions[0].Skills[0].Items[0].ID + `","answer":"I traded a later delivery for a higher price."}]}`
	call(t, handler, http.MethodPut, "/assessment-sessions/"+sessions[0].ID+"/answers", eli, answer, http.StatusOK)
	return sessions[0]
}

func gapCall(t *testing.T, server *httptest.Server, path, token string, status int) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != status {
		t.Fatalf("GET %s = %d body = %s", path, response.StatusCode, body)
	}
	return string(body)
}

func eliClaims() token.Claims {
	return token.Claims{
		PersonID:       eliID,
		OrganisationID: organisationID,
		Roles:          []string{"employee"},
	}
}

func postSession(t *testing.T, handler http.Handler, token, planID string, status int) {
	t.Helper()
	call(t, handler, http.MethodPost, "/plans/"+planID+"/session", token, "", status)
}

func listSessions(t *testing.T, handler http.Handler, token string) []openapi.Session {
	t.Helper()
	body := call(t, handler, http.MethodGet, "/assessment-sessions", token, "", http.StatusOK)
	var sessions []openapi.Session
	if err := json.Unmarshal([]byte(body), &sessions); err != nil {
		t.Fatal(err)
	}
	return sessions
}

func getSession(t *testing.T, handler http.Handler, token, sessionID string, status int) string {
	t.Helper()
	return call(t, handler, http.MethodGet, "/assessment-sessions/"+sessionID, token, "", status)
}

func call(t *testing.T, handler http.Handler, method, path, token, body string, status int) string {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s = %d body = %s", method, path, response.Code, response.Body.String())
	}
	return response.Body.String()
}
