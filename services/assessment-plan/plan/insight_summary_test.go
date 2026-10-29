package plan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/token"
)

func TestSummaryKeepsMissingSeparateFromAConfirmedGap(t *testing.T) {
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	subjects := []InsightSubject{
		subject("eli", "version-1", true, "1.0", when),
		subject("maya", "version-1", false, "", when),
	}
	insight, err := BuildInsight(InsightActor{PersonID: "ada", Roles: []string{"admin"}}, InsightRequest{ProfileVersionID: "version-1"}, subjects)
	if err != nil {
		t.Fatal(err)
	}
	if insight.Summary.RequiredRows != 2 || insight.Summary.AssessedRows != 1 || insight.Summary.MissingRows != 1 || insight.Summary.BelowRows != 1 || insight.Summary.MetRows != 0 {
		t.Fatalf("summary = %+v", insight.Summary)
	}
	if len(insight.Summary.CriticalSkillIDs) != 1 || insight.Summary.CriticalSkillIDs[0] != "skill-1" {
		t.Fatalf("critical = %+v", insight.Summary.CriticalSkillIDs)
	}
	if insight.Summary.Method == "" || strings.Contains(insight.Summary.Method, "proficiency zero") == false {
		t.Fatal("gap method was not recorded")
	}
	for _, person := range insight.People {
		if person.PersonID == "maya" && person.Skills[0].State != GapMissing {
			t.Fatalf("missing skill = %+v", person.Skills[0])
		}
	}
}

func TestInsightFiltersByDateRoleAndQuery(t *testing.T) {
	early := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	later := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	subjects := []InsightSubject{
		subject("eli", "version-1", true, "0.0", early),
		subject("maya", "version-1", true, "1.0", later),
	}
	subjects[1].RoleName = "Account Manager"
	subjects[1].Tenures[0].Title = "Account Manager"
	subjects[1].Skills[0].SkillID = "skill-9"
	actor := InsightActor{PersonID: "ada", Roles: []string{"hr"}}
	byRole, err := BuildInsight(actor, InsightRequest{ProfileVersionID: "version-1", RoleName: "Account Manager"}, subjects)
	if err != nil || len(byRole.Population) != 1 || byRole.Population[0] != "maya" {
		t.Fatalf("role = %+v %v", byRole.Population, err)
	}
	byDate, err := BuildInsight(actor, InsightRequest{ProfileVersionID: "version-1", From: later}, subjects)
	if err != nil || len(byDate.Population) != 1 || byDate.Population[0] != "maya" {
		t.Fatalf("date = %+v %v", byDate.Population, err)
	}
	byQuery, err := BuildInsight(actor, InsightRequest{ProfileVersionID: "version-1", Query: "skill-9"}, subjects)
	if err != nil || len(byQuery.People) != 1 || byQuery.People[0].Skills[0].SkillID != "skill-9" {
		t.Fatalf("query = %+v %v", byQuery.People, err)
	}
	if _, err := BuildInsight(actor, InsightRequest{ProfileVersionID: "version-1", From: later, To: early}, subjects); err != ErrInvalid {
		t.Fatalf("range error = %v", err)
	}
}

func TestADraftChangeDoesNotAlterTheApprovedSummary(t *testing.T) {
	store := NewMemory()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	profile := publishedProfile()
	_, baseline, err := CreateFromProfile(context.Background(), store, profileInput(profile, when))
	if err != nil {
		t.Fatal(err)
	}
	draft := profile
	draft.Status = "profile_draft"
	draft.VersionID = "version-2"
	draft.VersionNumber = 2
	draft.Skills[0].RequiredProficiency = 1
	draft.Skills[0].SkillID = "skill-draft"
	if _, err := MapProfileRequirements(draft); err != ErrProfileUnpublished {
		t.Fatalf("draft mapping = %v", err)
	}
	kept, err := store.ProfileBaseline(context.Background(), "org-1", baseline.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	subject := InsightSubject{
		PersonID: kept.PersonID, Department: "Sales", ProfileVersionID: kept.ProfileVersionID, RoleName: kept.RoleName, AssessedAt: when,
		Tenures: []RoleTenure{{PersonID: kept.PersonID, Title: kept.RoleName, StartedAt: when.Add(-time.Hour)}},
	}
	for _, skill := range kept.Skills {
		if skill.Priority != "required" {
			continue
		}
		subject.Skills = append(subject.Skills, InsightSkill{
			SkillID: skill.SkillID, RequiredProficiency: skill.RequiredProficiency, Found: true, CurrentLevel: "3.0", Gap: "0.0",
		})
	}
	insight, err := BuildInsight(InsightActor{PersonID: "ada", Roles: []string{"leader"}}, InsightRequest{ProfileVersionID: kept.ProfileVersionID}, []InsightSubject{subject})
	if err != nil {
		t.Fatal(err)
	}
	if insight.People[0].Skills[0].RequiredProficiency != 3 || insight.People[0].Skills[0].SkillID != "skill-1" || insight.Summary.MetRows != 1 || insight.Summary.BelowRows != 0 {
		t.Fatalf("historical insight = %+v", insight.People)
	}
	other, err := BuildInsight(InsightActor{PersonID: "ada", Roles: []string{"leader"}}, InsightRequest{ProfileVersionID: draft.VersionID}, []InsightSubject{subject})
	if err != nil || len(other.Population) != 0 {
		t.Fatalf("draft version leaked into the approved population %+v %v", other.Population, err)
	}
}

func TestAnotherOrganisationCannotReadTheApprovedInsight(t *testing.T) {
	store := NewMemory()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	profile := publishedProfile()
	profile.OrganisationID = organisationID
	if _, _, err := CreateFromProfile(context.Background(), store, ProfilePlanInput{
		OrganisationID: organisationID, PersonID: "eli", ActorPersonID: "ada", AssignedAt: when,
		Tenures: []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}},
		Profile: profile,
	}); err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, nil, nil, ProfileBridge{
		Levels: fixedLevels{}, Tenures: fixedTenures{rows: []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}}},
	})
	request := httptest.NewRequest(http.MethodGet, "/profile-insights?profileVersionId=version-1", nil)
	request.Header.Set("Authorization", "Bearer "+sign(t, token.Claims{PersonID: "ada", OrganisationID: "org-other", Roles: []string{"admin"}}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), `"eli"`) {
		t.Fatalf("status %d body %s", response.Code, response.Body.String())
	}
}

func TestDateOnlyInsightRangeIncludesTheWholeEndDay(t *testing.T) {
	from, err := parseInsightTime("2026-10-01", false)
	if err != nil {
		t.Fatal(err)
	}
	to, err := parseInsightTime("2026-10-01", true)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)
	subjects := []InsightSubject{subject("eli", "version-1", true, "1.0", when)}
	insight, err := BuildInsight(InsightActor{PersonID: "ada", Roles: []string{"admin"}}, InsightRequest{
		ProfileVersionID: "version-1", From: from, To: to,
	}, subjects)
	if err != nil || len(insight.Population) != 1 {
		t.Fatalf("included = %+v %v", insight.Population, err)
	}
	if _, err := parseInsightTime("01/10/2026", false); err != ErrInvalid {
		t.Fatalf("invalid date = %v", err)
	}
}
