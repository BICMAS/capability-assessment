package roleprofile

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/bicmas-academy/learn/services/token"
)

func TestApprovalFollowsTheConfiguredStages(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	profile := readyProfile(t, store)
	submitted, err := store.Submit(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision)
	if err != nil || submitted.Status != StatusInReview || submitted.WorkflowStage != StageHRReview || submitted.SubmissionCycle != 1 {
		t.Fatalf("submit = %#v %v", submitted, err)
	}
	if _, err := store.Publish(ctx, "org-a", submitted.ID, "leader-1", submitted.Revision); !errors.Is(err, ErrImmutable) {
		t.Fatalf("skip = %v", err)
	}
	assigned, err := store.AssignApproval(ctx, "org-a", submitted.ID, "hr-1", "hr-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", submitted.ID, assigned.ID, "hr-1", DecisionApproved, "Approved", submitted.Revision, false, []string{token.RoleLineManager}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong role = %v", err)
	}
	if _, err := store.DecideApproval(ctx, "org-b", submitted.ID, assigned.ID, "hr-1", DecisionApproved, "Approved", submitted.Revision, false, []string{token.RoleHRLD}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other organisation = %v", err)
	}
	advanced, err := store.DecideApproval(ctx, "org-a", submitted.ID, assigned.ID, "hr-1", DecisionApproved, "Approved", submitted.Revision, false, []string{token.RoleHRLD})
	if err != nil || advanced.Status != StatusInReview || advanced.WorkflowStage != StageManagerReview {
		t.Fatalf("hr approval = %#v %v", advanced, err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", submitted.ID, assigned.ID, "hr-1", DecisionApproved, "Approved", submitted.Revision, false, []string{token.RoleHRLD}); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent = %v", err)
	}
	reloaded, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil || reloaded.Version.WorkflowStage != StageManagerReview || reloaded.Version.Revision != advanced.Revision {
		t.Fatalf("reload = %#v %v", reloaded.Version, err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", advanced.ID, assigned.ID, "hr-1", DecisionApproved, "Again", advanced.Revision, true, []string{token.RoleAdministrator}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate decision = %v", err)
	}
	manager, err := store.AssignApproval(ctx, "org-a", advanced.ID, "hr-1", "manager-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignApproval(ctx, "org-a", advanced.ID, "hr-1", "manager-2"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate assignment = %v", err)
	}
	leaderStage, err := store.DecideApproval(ctx, "org-a", advanced.ID, manager.ID, "manager-1", DecisionApproved, "Approved", advanced.Revision, false, []string{token.RoleLineManager})
	if err != nil || leaderStage.WorkflowStage != StageLeaderApproval {
		t.Fatalf("manager approval = %#v %v", leaderStage, err)
	}
	leader, err := store.AssignApproval(ctx, "org-a", leaderStage.ID, "hr-1", "leader-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", leaderStage.ID, leader.ID, "leader-1", DecisionApproved, "Approved", leaderStage.Revision, false, []string{token.RoleBusinessLeader}); err != nil {
		t.Fatal(err)
	}
	published, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil || published.Version.Status != StatusPublished || published.Version.ApprovedByPersonID != "leader-1" {
		t.Fatalf("published = %#v %v", published.Version, err)
	}
	if _, err := store.AddSkill(ctx, "org-a", published.Version.ID, "hr-1", published.Version.Revision, SkillInput{SkillID: "missing", Category: CategoryDomain, Definition: "Changed", RequiredProficiency: 1, Priority: PriorityOptional, Rationale: "No", Provenance: ProvenanceManual}); !errors.Is(err, ErrImmutable) {
		t.Fatalf("approved edit = %v", err)
	}
	history, err := store.History(ctx, "org-a", profile.ID)
	if err != nil || !hasAction(history.Events, ActionSubmitted) || !hasAction(history.Events, ActionStageApproved) || !hasAction(history.Events, ActionPublished) {
		t.Fatalf("history = %#v %v", history.Events, err)
	}
	awaiting, err := store.ListApprovalQueue(ctx, "org-a", "leader-1", QueueReviewed, []string{token.RoleBusinessLeader})
	if err != nil || len(awaiting) != 1 || awaiting[0].RoleName != "Area Sales Manager" || awaiting[0].Status != StatusPublished {
		t.Fatalf("queue = %#v %v", awaiting, err)
	}
	if _, err := store.ListApprovalQueue(ctx, "org-b", "leader-1", QueueReviewed, []string{token.RoleBusinessLeader}); err != nil || len(mustQueue(t, store, "org-b")) != 0 {
		t.Fatal("other organisation queue")
	}
}

func TestChangesAndResubmissionKeepTheEarlierDecision(t *testing.T) {
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
	changed, err := store.DecideApproval(ctx, "org-a", submitted.ID, assignment.ID, "hr-1", DecisionChanges, "Add the regional team", submitted.Revision, false, []string{token.RoleHRLD})
	if err != nil || changed.Status != StatusDraft || changed.WorkflowStage != "" {
		t.Fatalf("changes = %#v %v", changed, err)
	}
	record, err := store.ListApprovals(ctx, "org-a", submitted.ID, "hr-1", []string{token.RoleHRLD})
	if err != nil || len(record.Decisions) != 1 || record.Decisions[0].Decision != DecisionChanges || record.Outstanding != "Revise and resubmit" {
		t.Fatalf("record = %#v %v", record, err)
	}
	if _, err := store.AddSkill(ctx, "org-a", changed.ID, "hr-1", changed.Revision, SkillInput{SkillID: profile.Version.Skills[0].SkillID, Category: CategoryTechnical, Definition: "Extra", RequiredProficiency: 2, Priority: PriorityPreferred, Rationale: "Added after review", Provenance: ProvenanceManual}); err == nil {
		t.Fatal("duplicate canonical skill was accepted")
	}
	again, err := store.Submit(ctx, "org-a", changed.ID, "hr-1", changed.Revision)
	if err != nil || again.WorkflowStage != StageHRReview || again.SubmissionCycle != 2 {
		t.Fatalf("resubmit = %#v %v", again, err)
	}
	rejected, err := store.DecideApproval(ctx, "org-a", again.ID, assignment.ID, "hr-1", DecisionRejected, "Stop this version", again.Revision, false, []string{token.RoleHRLD})
	if err != nil || rejected.Status != StatusInReview {
		t.Fatalf("reject = %#v %v", rejected, err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", rejected.ID, assignment.ID, "hr-1", DecisionApproved, "Approved", rejected.Revision, false, []string{token.RoleHRLD}); !errors.Is(err, ErrImmutable) {
		t.Fatalf("approve after reject = %v", err)
	}
	restarted, err := store.Resubmit(ctx, "org-a", rejected.ID, "hr-1", rejected.Revision, []string{token.RoleHRLD})
	if err != nil || restarted.WorkflowStage != StageHRReview || restarted.SubmissionCycle != 3 {
		t.Fatalf("restart = %#v %v", restarted, err)
	}
	if _, err := store.Publish(ctx, "org-a", restarted.ID, "leader-1", restarted.Revision); !errors.Is(err, ErrImmutable) {
		t.Fatalf("publish after resubmit = %v", err)
	}
}

func TestAShorterConfiguredPathCannotRewriteAnOpenReview(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	profile := readyProfile(t, store)
	open, err := store.Submit(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveApprovalPath(ctx, "org-a", "ada", []ApprovalPathStage{{StageKey: StageLeaderApproval, Label: "Business Leader Approval", RequiredRole: "leader", Position: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, "org-a", open.ID, "leader-1", open.Revision); !errors.Is(err, ErrImmutable) {
		t.Fatalf("open review changed path = %v", err)
	}
	if err := store.SaveApprovalPath(ctx, "org-a", "ada", []ApprovalPathStage{{StageKey: "Finance", Label: "Finance", RequiredRole: "leader", Position: 1}}); err == nil {
		t.Fatal("invalid stage key was stored")
	}
	published := approveAllStages(t, store, open)
	draft, err := store.NextDraft(ctx, "org-a", published.ID, "hr-1", published.Revision)
	if err != nil {
		t.Fatal(err)
	}
	short, err := store.Submit(ctx, "org-a", draft.ID, "hr-1", draft.Revision)
	if err != nil || short.WorkflowStage != StageLeaderApproval {
		t.Fatalf("short path = %#v %v", short, err)
	}
	assignment, err := store.AssignApproval(ctx, "org-a", short.ID, "hr-1", "leader-1")
	if err != nil {
		t.Fatal(err)
	}
	approved, err := store.DecideApproval(ctx, "org-a", short.ID, assignment.ID, "leader-1", DecisionApproved, "Approved", short.Revision, false, []string{token.RoleBusinessLeader})
	if err != nil || approved.Status != StatusPublished || approved.VersionNumber != 2 {
		t.Fatalf("short approval = %#v %v", approved, err)
	}
	versions, err := store.ListVersions(ctx, "org-a", profile.ID)
	if err != nil || len(versions) != 2 || versions[0].Status != StatusSuperseded || versions[0].Skills[0].Definition == "" || versions[1].Status != StatusPublished {
		t.Fatalf("versions = %#v %v", versions, err)
	}
}

func TestFinalApprovalRollsBackWhenTheSnapshotCannotBePublished(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	profile := readyProfile(t, store)
	submitted, err := store.Submit(ctx, "org-a", profile.Version.ID, "hr-1", profile.Version.Revision)
	if err != nil {
		t.Fatal(err)
	}
	current := approveUntil(t, store, submitted, StageLeaderApproval)
	for i := range store.versions {
		if store.versions[i].ID == current.ID {
			store.versions[i].Skills = nil
		}
	}
	before := len(store.decisions)
	leader, err := store.AssignApproval(ctx, "org-a", current.ID, "hr-1", "leader-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideApproval(ctx, "org-a", current.ID, leader.ID, "leader-1", DecisionApproved, "Approved", current.Revision, false, []string{token.RoleBusinessLeader}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publish failure = %v", err)
	}
	if len(store.decisions) != before {
		t.Fatalf("decision survived rollback: %d", len(store.decisions))
	}
	reloaded, err := store.GetProfile(ctx, "org-a", profile.ID)
	if err != nil || reloaded.Version.Status != StatusInReview || reloaded.Version.WorkflowStage != StageLeaderApproval || reloaded.Version.Revision != current.Revision {
		t.Fatalf("rolled back = %#v %v", reloaded.Version, err)
	}
}

func TestEmployeesCannotDecideOrReadAnotherOrganisationsQueue(t *testing.T) {
	store := NewMemory()
	profile := readyProfile(t, store)
	claims := token.Claims{PersonID: "eli", OrganisationID: "org-a", Roles: []string{token.RoleEmployee}}
	api := newAPI(store, &claims)
	denied, _ := api.call(http.MethodPost, "/role-profile-versions/"+profile.Version.ID+"/approval-decisions", `{"assignmentId":"","decision":"approved","comment":"","expectedRevision":1}`)
	if denied != http.StatusForbidden {
		t.Fatalf("employee decision = %d", denied)
	}
	queue, _ := api.call(http.MethodGet, "/role-profile-approvals?view=awaiting", "")
	if queue != http.StatusForbidden {
		t.Fatalf("employee queue = %d", queue)
	}
	claims.Roles = []string{token.RoleAdministrator}
	claims.OrganisationID = "org-b"
	missing, _ := api.call(http.MethodGet, "/role-profile-versions/"+profile.Version.ID+"/approvals", "")
	if missing != http.StatusNotFound {
		t.Fatalf("other organisation = %d", missing)
	}
}

func mustQueue(t *testing.T, store *Memory, organisationID string) []ApprovalQueueItem {
	t.Helper()
	items, err := store.ListApprovalQueue(context.Background(), organisationID, "leader-1", QueueReviewed, []string{token.RoleBusinessLeader})
	if err != nil {
		t.Fatal(err)
	}
	return items
}
