package framework

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestPublishedVersionStaysUnchanged(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	const organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"

	draft, err := store.CreateDraft(ctx, organisationID, "Sales capability framework")
	if err != nil {
		t.Fatal(err)
	}
	if draft.VersionNumber != 1 || draft.Status != StatusDraft {
		t.Fatalf("draft = %#v", draft)
	}
	role, err := store.AddRole(ctx, organisationID, draft.ID, "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	capability, err := store.AddCapability(ctx, organisationID, draft.ID, "Negotiation")
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := store.AddRequirement(ctx, organisationID, draft.ID, role.ID, capability.ID, 4)
	if err != nil {
		t.Fatal(err)
	}
	if requirement.RequiredLevel != 4 {
		t.Fatalf("requirement = %#v", requirement)
	}
	if _, err := store.AddIndicator(ctx, organisationID, draft.ID, capability.ID, 4, "Reaches a commercial agreement with the buyer."); err != nil {
		t.Fatal(err)
	}

	published, err := store.Publish(ctx, organisationID, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if published.VersionNumber != 1 || published.Status != StatusPublished {
		t.Fatalf("published = %#v", published)
	}

	_, err = store.PatchRole(ctx, organisationID, draft.ID, role.ID, "Changed title")
	if !errors.Is(err, ErrImmutable) {
		t.Fatalf("patch error = %v", err)
	}

	next, err := store.CreateNextDraft(ctx, organisationID, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == published.ID || next.VersionNumber != 2 || next.Status != StatusDraft {
		t.Fatalf("next = %#v", next)
	}
	if len(next.Roles) != 1 || next.Roles[0].Name != "Area Sales Manager" || next.Roles[0].ID == role.ID {
		t.Fatalf("copied roles = %#v", next.Roles)
	}

	again, err := store.GetVersion(ctx, organisationID, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(published, again) {
		t.Fatalf("version 1 changed\nbefore %#v\nafter %#v", published, again)
	}
	if again.Roles[0].Name != "Area Sales Manager" || again.Requirements[0].RequiredLevel != 4 {
		t.Fatalf("version 1 content = %#v", again)
	}
}
