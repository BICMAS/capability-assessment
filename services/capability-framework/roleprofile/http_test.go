package roleprofile

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/token"
)

func TestRoleProfileDraftLifecycle(t *testing.T) {
	store := NewMemory()
	claims := token.Claims{PersonID: "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11", OrganisationID: "org-a", Roles: []string{"admin"}}
	api := newAPI(store, &claims)

	status, _ := api.call(http.MethodPost, "/profile-roles", `{"name":""}`)
	if status != http.StatusBadRequest {
		t.Fatalf("empty role = %d", status)
	}
	role := callJSON[Role](t, api, http.MethodPost, "/profile-roles", `{"name":"Area Sales Manager"}`)
	if role.Code != http.StatusCreated || role.Body.OrganisationID != "org-a" || role.Body.Name != "Area Sales Manager" {
		t.Fatalf("role = %d %#v", role.Code, role.Body)
	}
	skillStatus, skillBody := api.call(http.MethodPost, "/canonical-skills", `{"name":"Negotiation","category":"magic","definition":"Agrees terms"}`)
	if skillStatus != http.StatusBadRequest || errorCode(t, skillBody) != "invalid" {
		t.Fatalf("category = %d %s", skillStatus, skillBody)
	}
	negotiation := callJSON[Skill](t, api, http.MethodPost, "/canonical-skills", `{"name":"Negotiation","category":"domain","definition":"Agrees commercial terms"}`)
	planning := callJSON[Skill](t, api, http.MethodPost, "/canonical-skills", `{"name":"Territory planning","category":"technical","definition":"Plans coverage"}`)

	missing := callJSON[Profile](t, api, http.MethodPost, "/role-profiles", `{"roleId":"missing"}`)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing role = %d", missing.Code)
	}
	unknownFramework := callJSON[Profile](t, api, http.MethodPost, "/role-profiles", `{"roleId":"`+role.Body.ID+`","frameworkVersionId":"fw-missing"}`)
	if unknownFramework.Code != http.StatusNotFound {
		t.Fatalf("framework = %d", unknownFramework.Code)
	}
	store.NoteFrameworkVersion("org-a", "fw-1")
	created := callJSON[Profile](t, api, http.MethodPost, "/role-profiles", `{"roleId":"`+role.Body.ID+`","frameworkVersionId":"fw-1"}`)
	if created.Code != http.StatusCreated || created.Body.Version.Status != StatusDraft || created.Body.Version.Revision != 1 || created.Body.Version.FrameworkVersionID != "fw-1" {
		t.Fatalf("draft = %d %#v", created.Code, created.Body.Version)
	}
	if created.Body.Version.VersionNumber != 1 || len(created.Body.Version.Skills) != 0 {
		t.Fatalf("version = %#v", created.Body.Version)
	}

	claims.OrganisationID = "org-b"
	other := callJSON[Profile](t, api, http.MethodGet, "/role-profiles/"+created.Body.ID, "")
	if other.Code != http.StatusNotFound || errorCode(t, other.Raw) != "not_found" {
		t.Fatalf("cross tenant = %d %s", other.Code, other.Raw)
	}
	claims.OrganisationID = "org-a"

	protected := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":1,"status":"profile_published","purpose":"Sell","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount to 10%","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"manager","sources":[]}`)
	if protected.Code != http.StatusBadRequest || errorCode(t, protected.Raw) != "invalid" {
		t.Fatalf("status write = %d %s", protected.Code, protected.Raw)
	}
	stale := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":9,"purpose":"Sell","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount to 10%","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"manager","sources":[{"title":"Job description","referenceUri":"doc:jd"}]}`)
	if stale.Code != http.StatusConflict || errorCode(t, stale.Raw) != "conflict" {
		t.Fatalf("stale = %d %s", stale.Code, stale.Raw)
	}
	updated := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":1,"purpose":"Sell","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount to 10%","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"manager","sources":[{"title":"Job description","referenceUri":"doc:jd"}]}`)
	if updated.Code != http.StatusOK || updated.Body.Revision != 2 || updated.Body.Purpose != "Sell" || len(updated.Body.Sources) != 1 {
		t.Fatalf("context = %d %#v", updated.Code, updated.Body)
	}

	badLevel := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/skills", `{"expectedRevision":2,"skillId":"`+negotiation.Body.ID+`","category":"domain","definition":"Agrees terms","requiredProficiency":9,"priority":"required","rationale":"Core","provenance":"manually_added"}`)
	if badLevel.Code != http.StatusBadRequest {
		t.Fatalf("proficiency = %d", badLevel.Code)
	}
	badCategory := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/skills", `{"expectedRevision":2,"skillId":"`+negotiation.Body.ID+`","category":"technical","definition":"Agrees terms","requiredProficiency":3,"priority":"required","rationale":"Core","provenance":"manually_added"}`)
	if badCategory.Code != http.StatusBadRequest {
		t.Fatalf("mismatched category = %d", badCategory.Code)
	}
	withSkill := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/skills", `{"expectedRevision":2,"skillId":"`+negotiation.Body.ID+`","category":"domain","definition":"Agrees terms with the buyer","requiredProficiency":3,"priority":"required","rationale":"Core","provenance":"manually_added"}`)
	if withSkill.Code != http.StatusCreated || withSkill.Body.Revision != 3 || len(withSkill.Body.Skills) != 1 || withSkill.Body.Skills[0].Position != 1 {
		t.Fatalf("add = %d %#v", withSkill.Code, withSkill.Body.Skills)
	}
	duplicate := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/skills", `{"expectedRevision":3,"skillId":"`+negotiation.Body.ID+`","category":"domain","definition":"Again","requiredProficiency":3,"priority":"required","rationale":"Core","provenance":"imported"}`)
	if duplicate.Code != http.StatusConflict || errorCode(t, duplicate.Raw) != "duplicate" {
		t.Fatalf("duplicate = %d %s", duplicate.Code, duplicate.Raw)
	}
	both := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/skills", `{"expectedRevision":3,"skillId":"`+planning.Body.ID+`","category":"technical","definition":"Plans the territory","requiredProficiency":2,"priority":"preferred","rationale":"Coverage","provenance":"imported"}`)
	if both.Code != http.StatusCreated || len(both.Body.Skills) != 2 {
		t.Fatalf("second skill = %d %#v", both.Code, both.Body.Skills)
	}
	ordered := callJSON[Version](t, api, http.MethodPut, "/role-profile-versions/"+created.Body.Version.ID+"/skill-order", `{"expectedRevision":4,"skillAssignmentIds":["`+both.Body.Skills[1].ID+`","`+both.Body.Skills[0].ID+`"]}`)
	if ordered.Code != http.StatusOK || ordered.Body.Skills[0].SkillID != planning.Body.ID || ordered.Body.Skills[0].Position != 1 || ordered.Body.Skills[1].Position != 2 {
		t.Fatalf("order = %#v", ordered.Body.Skills)
	}
	edited := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID+"/skills/"+ordered.Body.Skills[1].ID, `{"expectedRevision":5,"definition":"Agrees the commercial terms","requiredProficiency":4,"priority":"required","rationale":"Raised after review"}`)
	if edited.Code != http.StatusOK || edited.Body.Skills[1].RequiredProficiency != 4 || edited.Body.Skills[1].Provenance != ProvenanceManual {
		t.Fatalf("edit = %#v", edited.Body.Skills)
	}
	removed := callJSON[Version](t, api, http.MethodDelete, "/role-profile-versions/"+created.Body.Version.ID+"/skills/"+edited.Body.Skills[0].ID+"?expectedRevision=6", "")
	if removed.Code != http.StatusOK || len(removed.Body.Skills) != 1 || removed.Body.Skills[0].Position != 1 || removed.Body.Skills[0].SkillID != negotiation.Body.ID {
		t.Fatalf("remove = %#v", removed.Body.Skills)
	}

	early := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/publish", `{"expectedRevision":7}`)
	if early.Code != http.StatusConflict || errorCode(t, early.Raw) != "immutable" {
		t.Fatalf("publish draft = %d %s", early.Code, early.Raw)
	}
	submitted := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/submit", `{"expectedRevision":7}`)
	if submitted.Code != http.StatusOK || submitted.Body.Status != StatusInReview || submitted.Body.Revision != 8 {
		t.Fatalf("submit = %d %#v", submitted.Code, submitted.Body)
	}
	blocked := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID+"/skills/"+submitted.Body.Skills[0].ID, `{"expectedRevision":8,"definition":"Changed in review","requiredProficiency":4,"priority":"required","rationale":"No"}`)
	if blocked.Code != http.StatusConflict || errorCode(t, blocked.Raw) != "immutable" {
		t.Fatalf("edit in review = %d %s", blocked.Code, blocked.Raw)
	}
	note := callJSON[Comment](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/comments", `{"body":"Ready for the sales director"}`)
	if note.Code != http.StatusCreated || note.Body.AuthorPersonID != claims.PersonID {
		t.Fatalf("comment = %d %#v", note.Code, note.Body)
	}
	published := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/publish", `{"expectedRevision":8}`)
	if published.Code != http.StatusOK || published.Body.Status != StatusPublished {
		t.Fatalf("publish = %d %#v", published.Code, published.Body)
	}
	recorded := store.Publications()
	if len(recorded) != 1 || !strings.Contains(string(recorded[0]), `"type":"role_profile.published"`) || !strings.Contains(string(recorded[0]), published.Body.ID) {
		t.Fatalf("publication = %s", recorded)
	}
	frozen := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+created.Body.Version.ID, `{"expectedRevision":9,"purpose":"Changed","responsibilities":"Own the territory","businessOutcomes":"Revenue","decisionScope":"Discount to 10%","industry":"Logistics","region":"UK","teamSize":8,"reportingLevel":"manager","sources":[]}`)
	if frozen.Code != http.StatusConflict || errorCode(t, frozen.Raw) != "immutable" {
		t.Fatalf("edit published = %d %s", frozen.Code, frozen.Raw)
	}

	next := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+created.Body.Version.ID+"/next-draft", `{"expectedRevision":9}`)
	if next.Code != http.StatusCreated || next.Body.VersionNumber != 2 || next.Body.Status != StatusDraft || next.Body.Revision != 1 || next.Body.Skills[0].Definition != "Agrees the commercial terms" {
		t.Fatalf("next = %d %#v", next.Code, next.Body)
	}
	custom := callJSON[Version](t, api, http.MethodPatch, "/role-profile-versions/"+next.Body.ID+"/skills/"+next.Body.Skills[0].ID, `{"expectedRevision":1,"definition":"Agrees terms for enterprise accounts","requiredProficiency":4,"priority":"required","rationale":"Segment focus"}`)
	if custom.Code != http.StatusOK {
		t.Fatalf("customise = %d %s", custom.Code, custom.Raw)
	}
	versions := callJSON[[]Version](t, api, http.MethodGet, "/role-profiles/"+created.Body.ID+"/versions", "")
	if versions.Code != http.StatusOK || len(versions.Body) != 2 || versions.Body[0].Status != StatusPublished || versions.Body[0].Skills[0].Definition != "Agrees the commercial terms" {
		t.Fatalf("history versions = %#v", versions.Body)
	}
	diff := callJSON[Comparison](t, api, http.MethodGet, "/role-profiles/"+created.Body.ID+"/compare?from="+versions.Body[0].ID+"&to="+versions.Body[1].ID, "")
	if diff.Code != http.StatusOK || len(diff.Body.Changed) != 1 || diff.Body.Changed[0].FromProficiency != 4 || diff.Body.Changed[0].SkillID != negotiation.Body.ID {
		t.Fatalf("compare = %#v", diff.Body)
	}
	record := callJSON[History](t, api, http.MethodGet, "/role-profiles/"+created.Body.ID+"/history", "")
	if record.Code != http.StatusOK || !hasAction(record.Body.Events, ActionPublished) || len(record.Body.Comments) != 1 {
		t.Fatalf("history = %#v", record.Body)
	}
	claims.Roles = []string{"employee"}
	denied, deniedBody := api.call(http.MethodGet, "/role-profiles/"+created.Body.ID+"/history", "")
	if denied != http.StatusForbidden || errorCode(t, deniedBody) != "forbidden" {
		t.Fatalf("employee = %d %s", denied, deniedBody)
	}
}

func TestAIDraftIsPreservedWhenTheProfileIsCustomised(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Planner")
	if err != nil {
		t.Fatal(err)
	}
	skill, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Forecasting", Category: CategoryTechnical, Definition: "Forecasts demand"})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "ada", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AttachAIDraft(ctx, "org-a", profile.Version.ID, "system", AIDraft{
		Model:                "fake-model",
		SourceContextVersion: "context-1",
		GeneratedAt:          time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC),
		Purpose:              "Plan the network",
		Skills:               []SuggestedSkill{{SkillID: skill.ID, Definition: "Builds the forecast", RequiredProficiency: 3, Priority: PriorityRequired, Rationale: "Suggested"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	missing, err := store.AttachAIDraft(ctx, "org-a", profile.Version.ID, "system", AIDraft{
		Model:                "fake-model",
		SourceContextVersion: "context-2",
		GeneratedAt:          time.Date(2026, 10, 11, 22, 0, 0, 0, time.UTC),
		Purpose:              "Replacement",
		Skills:               []SuggestedSkill{{SkillID: "missing", Definition: "Nope", RequiredProficiency: 1, Priority: PriorityOptional}},
	})
	if err == nil || missing.Purpose == "Replacement" {
		t.Fatal("second draft overwrote the original")
	}
	current, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	original := string(current.Version.AI.Draft)
	if current.Version.Purpose != "Plan the network" || current.Version.AI.SourceContextVersion != "context-1" || len(current.Version.Skills) != 1 {
		t.Fatalf("draft = %#v", current.Version)
	}
	updated, err := store.UpdateSkill(ctx, "org-a", current.Version.ID, current.Version.Skills[0].ID, "ada", current.Version.Revision, SkillEdit{
		Definition:          "Builds the weekly forecast",
		RequiredProficiency: 4,
		Priority:            PriorityRequired,
		Rationale:           "HR adjustment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Skills[0].Provenance != ProvenanceHRAdjusted || updated.Skills[0].Definition != "Builds the weekly forecast" || string(updated.AI.Draft) != original {
		t.Fatalf("customised = %#v draft %s", updated.Skills[0], updated.AI.Draft)
	}
	reloaded, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(reloaded.Version.AI.Draft) != original || reloaded.Version.Skills[0].Provenance != ProvenanceHRAdjusted {
		t.Fatalf("reloaded = %#v", reloaded.Version)
	}
}

func TestAttachAIDraftRollsBackWhenASkillIsMissing(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Planner")
	if err != nil {
		t.Fatal(err)
	}
	skill, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Forecasting", Category: CategoryTechnical, Definition: "Forecasts demand"})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "ada", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AttachAIDraft(ctx, "org-a", profile.Version.ID, "system", AIDraft{
		Model:                "fake-model",
		SourceContextVersion: "context-1",
		GeneratedAt:          time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC),
		Purpose:              "Plan the network",
		Skills: []SuggestedSkill{
			{SkillID: skill.ID, Definition: "Builds the forecast", RequiredProficiency: 3, Priority: PriorityRequired},
			{SkillID: "missing", Definition: "Unknown", RequiredProficiency: 1, Priority: PriorityOptional},
		},
	})
	if err == nil {
		t.Fatal("expected missing skill to fail")
	}
	current, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version.Purpose != "" || current.Version.Revision != 1 || len(current.Version.Skills) != 0 || len(current.Version.AI.Draft) != 0 {
		t.Fatalf("rolled back = %#v", current.Version)
	}
}

func TestRoleProfileRequiresAnAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/role-profiles/profile-1", nil)
	response := httptest.NewRecorder()
	getProfile(NewMemory()).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
}

type apiResult[T any] struct {
	Code int
	Body T
	Raw  []byte
}

type apiClient struct {
	handler http.Handler
}

func newAPI(store Store, claims *token.Claims) apiClient {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(token.WithClaims(r.Context(), *claims)))
		})
	})
	Mount(router, store, nil)
	return apiClient{handler: router}
}

func (c apiClient) call(method, path, body string) (int, []byte) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, request)
	return response.Code, response.Body.Bytes()
}

func callJSON[T any](t *testing.T, api apiClient, method, path, body string) apiResult[T] {
	t.Helper()
	status, raw := api.call(method, path, body)
	return decodeJSON[T](t, status, raw)
}

func decodeJSON[T any](t *testing.T, status int, raw []byte) apiResult[T] {
	t.Helper()
	var body T
	if status >= 200 && status < 300 {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return apiResult[T]{Code: status, Body: body, Raw: raw}
}

func errorCode(t *testing.T, raw []byte) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("error body %s: %v", raw, err)
	}
	return body["error"]
}

func hasAction(events []AuditEvent, action string) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}
