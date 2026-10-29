package roleprofile

import (
	"context"
	"net/http"
	"testing"

	"github.com/bicmas-academy/learn/services/token"
)

func TestPublishingSupersedesThePreviousApprovedVersion(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	first := readyProfile(t, store)
	submitted, err := store.Submit(ctx, "org-a", first.Version.ID, "ada", first.Version.Revision)
	if err != nil {
		t.Fatal(err)
	}
	published := approveAllStages(t, store, submitted)
	if published.Status != StatusPublished || published.ApprovedByPersonID != "ada" || published.ApprovedAt == nil {
		t.Fatalf("publish = %#v", published)
	}
	if published.Sources[0].Kind != SourceJobDescription {
		t.Fatalf("source kind = %#v", published.Sources)
	}

	next, err := store.NextDraft(ctx, "org-a", published.ID, "hr-1", published.Revision)
	if err != nil || next.Status != StatusDraft || next.Sources[0].Kind != SourceJobDescription {
		t.Fatalf("next = %#v %v", next, err)
	}
	review, err := store.Submit(ctx, "org-a", next.ID, "hr-1", next.Revision)
	if err != nil {
		t.Fatal(err)
	}
	claims := token.Claims{PersonID: "hr-1", OrganisationID: "org-a", Roles: []string{token.RoleHRLD}}
	api := newAPI(store, &claims)
	denied, _ := api.call(http.MethodPost, "/role-profile-versions/"+review.ID+"/publish", `{"expectedRevision":`+itoa(review.Revision)+`}`)
	if denied != http.StatusForbidden {
		t.Fatalf("hr publish = %d", denied)
	}
	review = approveUntil(t, store, review, StageLeaderApproval)
	assigned := callJSON[ApprovalAssignment](t, api, http.MethodPost, "/role-profile-versions/"+review.ID+"/approval-assignments", `{"assigneePersonId":"leader-1"}`)
	if assigned.Code != http.StatusCreated || assigned.Body.AssigneePersonID != "leader-1" {
		t.Fatalf("assign = %d %#v", assigned.Code, assigned.Body)
	}

	claims.PersonID = "leader-2"
	claims.Roles = []string{token.RoleBusinessLeader}
	outsider, _ := api.call(http.MethodPost, "/role-profile-versions/"+review.ID+"/approval-decisions", `{"assignmentId":"`+assigned.Body.ID+`","decision":"approved","comment":"","expectedRevision":`+itoa(review.Revision)+`}`)
	if outsider != http.StatusForbidden {
		t.Fatalf("outsider decision = %d", outsider)
	}
	claims.PersonID = "leader-1"
	approved := callJSON[Version](t, api, http.MethodPost, "/role-profile-versions/"+review.ID+"/approval-decisions", `{"assignmentId":"`+assigned.Body.ID+`","decision":"approved","comment":"Approved","expectedRevision":`+itoa(review.Revision)+`}`)
	if approved.Code != http.StatusOK || approved.Body.Status != StatusPublished {
		t.Fatalf("decision = %d %#v", approved.Code, approved.Body)
	}
	versions, err := store.ListVersions(ctx, "org-a", first.ID)
	if err != nil || len(versions) != 2 || versions[0].Status != StatusSuperseded || versions[1].Status != StatusPublished {
		t.Fatalf("versions = %#v %v", versions, err)
	}
	if versions[0].Skills[0].Definition == "" {
		t.Fatal("superseded snapshot lost its skills")
	}

	claims.Roles = []string{token.RoleEmployee}
	employee, _ := api.call(http.MethodGet, "/role-profiles/"+first.ID, "")
	if employee != http.StatusForbidden {
		t.Fatalf("employee read = %d", employee)
	}
	claims.Roles = []string{token.RoleLineManager}
	manager := callJSON[Profile](t, api, http.MethodGet, "/role-profiles/"+first.ID, "")
	if manager.Code != http.StatusOK || manager.Body.Version.Status != StatusPublished {
		t.Fatalf("manager read = %d %#v", manager.Code, manager.Body.Version.Status)
	}
}

func TestChangesRequestedReturnsTheVersionToDraft(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	profile := readyProfile(t, store)
	submitted, err := store.Submit(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := store.AssignApproval(ctx, "org-a", submitted.ID, "hr-1", "hr-1")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := store.DecideApproval(ctx, "org-a", submitted.ID, assignment.ID, "hr-1", DecisionChanges, "Add the region team", submitted.Revision, false, []string{token.RoleHRLD})
	if err != nil || changed.Status != StatusDraft || changed.Revision != submitted.Revision+1 {
		t.Fatalf("changes = %#v %v", changed, err)
	}
	rejectedBase, err := store.Submit(ctx, "org-a", changed.ID, "hr-1", changed.Revision)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := store.DecideApproval(ctx, "org-a", rejectedBase.ID, assignment.ID, "hr-1", DecisionRejected, "Not this version", rejectedBase.Revision, false, []string{token.RoleHRLD})
	if err != nil || rejected.Status != StatusInReview {
		t.Fatalf("rejected = %#v %v", rejected, err)
	}
}

func readyProfile(t *testing.T, store *Memory) Profile {
	t.Helper()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	skill, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Negotiation", Category: CategoryDomain, Definition: "Agrees terms"})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "hr-1", "")
	if err != nil {
		t.Fatal(err)
	}
	teamSize := 6
	updated, err := store.UpdateContext(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision, Context{
		Purpose:          "Own regional revenue",
		Responsibilities: "Lead the sales team",
		BusinessOutcomes: "Revenue and retention",
		DecisionScope:    "Discount to ten percent",
		Industry:         "Logistics",
		Region:           "UK",
		TeamSize:         &teamSize,
		ReportingLevel:   ReportingManager,
		Sources:          []Source{{Title: "Area sales job description", ReferenceURI: "doc:area-sales", Kind: SourceJobDescription}},
	})
	if err != nil {
		t.Fatal(err)
	}
	withSkill, err := store.AddSkill(ctx, "org-a", updated.ID, "hr-1", updated.Revision, SkillInput{
		SkillID:             skill.ID,
		Category:            CategoryDomain,
		Definition:          "Agrees commercial terms",
		RequiredProficiency: 4,
		Priority:            PriorityRequired,
		Rationale:           "Core to the role",
		Provenance:          ProvenanceManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile.Version = withSkill
	return profile
}

func approveAllStages(t *testing.T, store *Memory, version Version) Version {
	t.Helper()
	return approveUntil(t, store, version, "")
}

func approveUntil(t *testing.T, store *Memory, version Version, stopBefore string) Version {
	t.Helper()
	ctx := context.Background()
	current := version
	for current.Status != StatusPublished {
		if stopBefore != "" && current.WorkflowStage == stopBefore {
			return current
		}
		record, err := store.ListApprovals(ctx, current.OrganisationID, current.ID, "ada", nil)
		if err != nil {
			t.Fatal(err)
		}
		var stage WorkflowStep
		for _, item := range record.Steps {
			if item.StageKey == current.WorkflowStage {
				stage = item
			}
		}
		if stage.StageKey == "" {
			t.Fatalf("stage %s is not on the version", current.WorkflowStage)
		}
		assigned, err := store.AssignApproval(ctx, current.OrganisationID, current.ID, "ada", "ada")
		if err != nil {
			t.Fatal(err)
		}
		next, err := store.DecideApproval(ctx, current.OrganisationID, current.ID, assigned.ID, "ada", DecisionApproved, "Approved", current.Revision, false, []string{stage.RequiredRole})
		if err != nil {
			t.Fatal(err)
		}
		current = next
	}
	return current
}
