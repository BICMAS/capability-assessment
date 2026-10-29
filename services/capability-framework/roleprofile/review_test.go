package roleprofile

import (
	"context"
	"net/http"
	"testing"

	"github.com/bicmas-academy/learn/services/token"
)

func TestReviewRecordsHumanChangesWithoutRelabellingAnUntouchedSuggestion(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{body: nil}})
	fx.provider.responses[0].body = validGeneration(fx)
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-review", session.ContextVersion, false); err != nil {
		t.Fatal(err)
	}
	fx.workflow.ProcessQueued(ctx)
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	skill := version.Skills[0]
	originalDraft := string(version.AI.Draft)
	unchanged, err := fx.store.UpdateSkill(ctx, "org-a", version.ID, skill.ID, "person-1", version.Revision, SkillEdit{
		Definition: skill.Definition, RequiredProficiency: skill.RequiredProficiency, Priority: skill.Priority, Rationale: skill.Rationale,
	})
	if err != nil || unchanged.Revision != version.Revision || unchanged.Skills[0].Provenance != ProvenanceAISuggested {
		t.Fatalf("unchanged = %#v %v", unchanged.Skills, err)
	}
	name := "Territory design"
	category := CategoryCommunication
	criteria := "Observe a live account review"
	edited, err := fx.store.UpdateSkill(ctx, "org-a", version.ID, skill.ID, "person-1", version.Revision, SkillEdit{
		Definition: skill.Definition, RequiredProficiency: 4, Priority: skill.Priority, Rationale: skill.Rationale,
		DisplayName: &name, Category: &category, Criteria: &criteria,
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Status != StatusDraft || edited.Revision != version.Revision+1 || edited.Skills[0].Provenance != ProvenanceHRAdjusted || edited.Skills[0].DisplayName != name || edited.Skills[0].Category != category || edited.Skills[0].Criteria != criteria || edited.Skills[0].RequiredProficiency != 4 || string(edited.AI.Draft) != originalDraft {
		t.Fatalf("edited = %#v", edited.Skills[0])
	}
	reloaded, err := fx.store.GetVersion(ctx, "org-a", version.ID)
	if err != nil || reloaded.Skills[0].DisplayName != name || reloaded.Skills[0].Criteria != criteria || reloaded.Revision != edited.Revision {
		t.Fatalf("reloaded = %#v %v", reloaded.Skills, err)
	}
	history, err := fx.store.History(ctx, "org-a", version.ProfileID)
	if err != nil || len(history.Changes) != 1 || history.Changes[0].ActorPersonID != "person-1" || history.Changes[0].Rationale != skill.Rationale {
		t.Fatalf("history = %#v %v", history.Changes, err)
	}
	for _, field := range []string{"displayName", "category", "criteria", "requiredProficiency", "provenance"} {
		if !containsField(history.Changes[0].Fields, field) {
			t.Fatalf("fields = %#v", history.Changes[0].Fields)
		}
	}
	if _, err := fx.store.UpdateSkill(ctx, "org-a", version.ID, skill.ID, "person-1", version.Revision, SkillEdit{
		Definition: "A later edit", RequiredProficiency: 2, Priority: PriorityOptional, Rationale: "Stale",
	}); !errorsIsConflict(err) {
		t.Fatalf("stale edit = %v", err)
	}
	if reloaded.Skills[0].Definition != skill.Definition {
		t.Fatal("stale edit changed the skill")
	}
}

func TestMergeKeepsOneSkillAndCopiesMissingReviewFields(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), nil)
	ctx := context.Background()
	first, err := fx.store.AddSkill(ctx, "org-a", fx.version.ID, "person-1", fx.version.Revision, SkillInput{
		SkillID: fx.technical.ID, Category: CategoryTechnical, Definition: "Plans the territory for this role", RequiredProficiency: 2, Priority: PriorityRequired, Provenance: ProvenanceManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.store.AddSkill(ctx, "org-a", fx.version.ID, "person-1", first.Revision, SkillInput{
		SkillID: fx.behaviour.ID, Category: CategoryLeadership, Definition: "Coaches the same territory work", RequiredProficiency: 2, Priority: PriorityPreferred, Rationale: "Covers the same work", Criteria: "Observe the handover", Provenance: ProvenanceManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := fx.store.MergeSkills(ctx, "org-a", fx.version.ID, "person-1", first.Skills[0].ID, second.Skills[1].ID, second.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Status != StatusDraft || len(merged.Skills) != 1 || merged.Skills[0].ID != first.Skills[0].ID || merged.Skills[0].Criteria != "Observe the handover" || merged.Skills[0].Rationale != "Covers the same work" || merged.Skills[0].Provenance != ProvenanceManual || merged.Skills[0].Position != 1 {
		t.Fatalf("merged = %#v", merged.Skills)
	}
	reloaded, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil || len(reloaded.Skills) != 1 || reloaded.Skills[0].Criteria != "Observe the handover" {
		t.Fatalf("reloaded = %#v %v", reloaded.Skills, err)
	}
	history, err := fx.store.History(ctx, "org-a", fx.version.ProfileID)
	if err != nil || len(history.Changes) != 1 || !containsField(history.Changes[0].Fields, "criteria") || !containsField(history.Changes[0].Fields, "rationale") {
		t.Fatalf("changes = %#v %v", history.Changes, err)
	}
	if _, err := fx.store.MergeSkills(ctx, "org-a", fx.version.ID, "person-1", first.Skills[0].ID, second.Skills[1].ID, second.Revision); !errorsIsConflict(err) {
		t.Fatalf("stale merge = %v", err)
	}
}

func TestDraftCommentsAreStoredAndEmployeesCannotEditSkills(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), nil)
	ctx := context.Background()
	note, err := fx.store.Comment(ctx, "org-a", fx.version.ID, "person-1", "Check the territory skill before submission")
	if err != nil || note.AuthorPersonID != "person-1" {
		t.Fatalf("comment = %#v %v", note, err)
	}
	claims := token.Claims{PersonID: "person-1", OrganisationID: "org-a", Roles: []string{"employee"}}
	api := newAPI(fx.store, &claims)
	denied, _ := api.call(http.MethodPost, "/role-profile-versions/"+fx.version.ID+"/skill-edits", `{"expectedRevision":1,"edits":[]}`)
	if denied != http.StatusForbidden {
		t.Fatalf("employee edits = %d", denied)
	}
	mergeDenied, _ := api.call(http.MethodPost, "/role-profile-versions/"+fx.version.ID+"/skill-merges", `{"expectedRevision":1,"keepId":"a","removeId":"b"}`)
	if mergeDenied != http.StatusForbidden {
		t.Fatalf("employee merge = %d", mergeDenied)
	}
}

func containsField(fields []string, field string) bool {
	for _, item := range fields {
		if item == field {
			return true
		}
	}
	return false
}

func errorsIsConflict(err error) bool {
	return err != nil && err.Error() == ErrConflict.Error()
}
