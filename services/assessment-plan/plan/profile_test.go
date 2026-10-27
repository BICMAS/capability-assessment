package plan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/assessment-gap/levels"
	"github.com/bicmas-academy/learn/services/token"
)

func TestProfileMappingSnapshotsTheApprovedVersion(t *testing.T) {
	store := NewMemory()
	profile := publishedProfile()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	plan, baseline, err := CreateFromProfile(context.Background(), store, ProfilePlanInput{
		OrganisationID: "org-1",
		PersonID:       "eli",
		ActorPersonID:  "ada",
		AssignedAt:     when,
		Tenures:        []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}},
		Profile:        profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusGenerated || plan.PromptVersion != ProfileMapPrompt || len(plan.Skills) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.Skills[0].CapabilityID != "skill-1" || plan.Skills[0].RequiredLevel != 3 || plan.Skills[0].RequirementID != "assignment-1" {
		t.Fatalf("mapped skill = %+v", plan.Skills[0])
	}
	if baseline.ProfileVersionID != "version-1" || baseline.Skills[0].RequiredProficiency != 3 {
		t.Fatalf("baseline = %+v", baseline)
	}
	submitted, err := store.SubmitPlan(context.Background(), ReviewInput{OrganisationID: "org-1", PlanID: plan.ID, ActorPersonID: "ada"})
	if err != nil || submitted.Status != StatusUnderReview {
		t.Fatalf("submit = %+v %v", submitted, err)
	}

	profile.Skills[0].RequiredProficiency = 1
	profile.Skills[0].Name = "Changed after approval"
	again, kept, err := CreateFromProfile(context.Background(), store, ProfilePlanInput{
		OrganisationID: "org-1",
		PersonID:       "eli",
		ActorPersonID:  "ada",
		AssignedAt:     when,
		Tenures:        []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}},
		Profile:        profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != plan.ID || again.Skills[0].RequiredLevel != 3 || kept.Skills[0].RequiredProficiency != 3 || kept.Skills[0].Name != "Territory planning" {
		t.Fatalf("edited profile rewrote plan %+v baseline %+v", again.Skills, kept.Skills)
	}
}

func TestANewProfileVersionKeepsTheHistoricalBaseline(t *testing.T) {
	store := NewMemory()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	first, _, err := CreateFromProfile(context.Background(), store, profileInput(publishedProfile(), when))
	if err != nil {
		t.Fatal(err)
	}
	next := publishedProfile()
	next.VersionID = "version-2"
	next.VersionNumber = 2
	next.Skills[0].RequiredProficiency = 4
	second, secondBaseline, err := CreateFromProfile(context.Background(), store, profileInput(next, when.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID || secondBaseline.ProfileVersionID != "version-2" || second.Skills[0].RequiredLevel != 4 {
		t.Fatalf("second = %+v baseline %+v", second, secondBaseline)
	}
	kept, err := store.ProfileBaseline(context.Background(), "org-1", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if kept.ProfileVersionID != "version-1" || kept.Skills[0].RequiredProficiency != 3 {
		t.Fatalf("historical baseline = %+v", kept)
	}
}

func TestRoleChangeBlocksAPlanOutsideTheTenure(t *testing.T) {
	store := NewMemory()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	_, _, err := CreateFromProfile(context.Background(), store, ProfilePlanInput{
		OrganisationID: "org-1",
		PersonID:       "eli",
		ActorPersonID:  "ada",
		AssignedAt:     when,
		Tenures: []RoleTenure{{
			PersonID:  "eli",
			Title:     "Area Sales Manager",
			StartedAt: when.Add(-48 * time.Hour),
			EndedAt:   when.Add(-time.Hour),
		}, {
			PersonID:  "eli",
			Title:     "Operations Lead",
			StartedAt: when.Add(-time.Hour),
		}},
		Profile: publishedProfile(),
	})
	if err != ErrRolePeriod {
		t.Fatalf("error = %v", err)
	}
	rows, err := store.ListProfileBaselines(context.Background(), "org-1")
	if err != nil || len(rows) != 0 {
		t.Fatalf("baselines = %+v %v", rows, err)
	}
}

func TestMissingAssessmentIsNotProficiencyZero(t *testing.T) {
	missing, err := ClassifyGap("skill-1", 3, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if missing.State != GapMissing || missing.CurrentLevel != "" || missing.Gap != "" {
		t.Fatalf("missing = %+v", missing)
	}
	zero := 0
	explained := levels.Explain(levels.Sources{Knowledge: &zero}, "version-1", 30)
	assessed, err := ClassifyGap("skill-1", 3, true, explained.CurrentLevel.String(), explained.Gap.String())
	if err != nil {
		t.Fatal(err)
	}
	if assessed.State != GapBelow || assessed.CurrentLevel != "0.0" {
		t.Fatalf("assessed zero = %+v calculation %+v", assessed, explained)
	}
	knowledge := 80
	above := levels.Explain(levels.Sources{Knowledge: &knowledge}, "version-1", 30)
	met, err := ClassifyGap("skill-1", 3, true, above.CurrentLevel.String(), above.Gap.String())
	if err != nil || met.State != GapMet {
		t.Fatalf("met = %+v %v gap %s", met, err, above.Gap)
	}
}

func TestInsightNamesTheVersionPopulationAndHidesOtherPeople(t *testing.T) {
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	subjects := []InsightSubject{
		subject("eli", "version-1", true, "1.0", when),
		subject("maya", "version-1", false, "", when),
		subject("lee", "version-1", true, "1.0", when),
	}
	subjects[2].Tenures[0].EndedAt = when.Add(-time.Minute)
	employee, err := BuildInsight(InsightActor{PersonID: "eli", Roles: []string{"employee"}}, InsightRequest{ProfileVersionID: "version-1"}, subjects)
	if err != nil {
		t.Fatal(err)
	}
	if employee.ProfileVersionID != "version-1" || len(employee.Population) != 1 || employee.Population[0] != "eli" {
		t.Fatalf("employee insight = %+v", employee)
	}
	if employee.People[0].Skills[0].State != GapBelow || employee.EmploymentDecision != EmploymentDecision {
		t.Fatalf("employee skill = %+v", employee.People[0])
	}
	_, err = BuildInsight(InsightActor{PersonID: "eli", Roles: []string{"employee"}}, InsightRequest{PersonID: "maya", ProfileVersionID: "version-1"}, subjects)
	if err != ErrForbidden {
		t.Fatalf("error = %v", err)
	}
	manager, err := BuildInsight(InsightActor{PersonID: "maya", Roles: []string{"manager"}, ManagerOf: []string{"eli"}}, InsightRequest{PersonID: "eli", ProfileVersionID: "version-1"}, subjects)
	if err != nil || len(manager.Population) != 1 || manager.Population[0] != "eli" {
		t.Fatalf("manager insight = %+v %v", manager, err)
	}
	admin, err := BuildInsight(InsightActor{PersonID: "ada", Roles: []string{"admin"}}, InsightRequest{Department: "Sales", ProfileVersionID: "version-1"}, subjects)
	if err != nil || len(admin.Population) != 2 {
		t.Fatalf("department insight = %+v %v", admin, err)
	}
	for _, person := range admin.People {
		if person.PersonID == "lee" {
			t.Fatal("role change stayed in the population")
		}
		if person.PersonID == "maya" && person.Skills[0].State != GapMissing {
			t.Fatalf("maya = %+v", person.Skills)
		}
	}
}

type fixedLevels struct{}

func (fixedLevels) ReadLevel(context.Context, string, string, string, string) (LevelView, error) {
	return LevelView{}, nil
}

type fixedTenures struct {
	rows []RoleTenure
}

func (f fixedTenures) ListRoleTenures(context.Context, string) ([]RoleTenure, error) {
	return f.rows, nil
}

func TestProfileInsightHTTPOmitsAScoreWhenNoLevelExists(t *testing.T) {
	store := NewMemory()
	when := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	profile := publishedProfile()
	profile.OrganisationID = organisationID
	_, _, err := CreateFromProfile(context.Background(), store, ProfilePlanInput{
		OrganisationID: organisationID, PersonID: "eli", ActorPersonID: "ada", AssignedAt: when,
		Tenures: []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}},
		Profile: profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(nil, []byte(httpSecret), store, nil, nil, nil, nil, nil, nil, ProfileBridge{
		Levels: fixedLevels{}, Tenures: fixedTenures{rows: []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}}},
	})
	request := httptest.NewRequest(http.MethodGet, "/profile-insights?profileVersionId=version-1&personId=eli", nil)
	request.Header.Set("Authorization", "Bearer "+sign(t, token.Claims{PersonID: "eli", OrganisationID: organisationID, Roles: []string{"employee"}}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), `"currentLevel"`) || !strings.Contains(response.Body.String(), `"missing"`) {
		t.Fatalf("status %d body %s", response.Code, response.Body.String())
	}
}

func publishedProfile() ProfileSource {
	return ProfileSource{
		ProfileID: "profile-1", VersionID: "version-1", VersionNumber: 1, OrganisationID: "org-1",
		RoleID: "role-1", RoleName: "Area Sales Manager", Status: "profile_published",
		Skills: []ProfileSkillSnapshot{{
			AssignmentID: "assignment-1", SkillID: "skill-1", Name: "Territory planning",
			Definition: "Plans coverage", RequiredProficiency: 3, Priority: "required", Rationale: "Owns coverage",
		}, {
			AssignmentID: "assignment-2", SkillID: "skill-2", Name: "Local customs",
			Definition: "Knows the region", RequiredProficiency: 2, Priority: "preferred",
		}},
	}
}

func profileInput(profile ProfileSource, when time.Time) ProfilePlanInput {
	return ProfilePlanInput{
		OrganisationID: "org-1", PersonID: "eli", ActorPersonID: "ada", AssignedAt: when,
		Tenures: []RoleTenure{{PersonID: "eli", Title: "Area Sales Manager", StartedAt: when.Add(-time.Hour)}},
		Profile: profile,
	}
}

func subject(personID, versionID string, found bool, gap string, when time.Time) InsightSubject {
	current := ""
	if found {
		current = "2.0"
	}
	return InsightSubject{
		PersonID: personID, Department: "Sales", ProfileVersionID: versionID, RoleName: "Area Sales Manager", AssessedAt: when,
		Tenures: []RoleTenure{{PersonID: personID, Title: "Area Sales Manager", DepartmentName: "Sales", StartedAt: when.Add(-time.Hour)}},
		Skills:  []InsightSkill{{SkillID: "skill-1", RequiredProficiency: 3, Found: found, CurrentLevel: current, Gap: gap}},
	}
}
