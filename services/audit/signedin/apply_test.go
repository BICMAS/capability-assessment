package signedin

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresOneOrganisationScopedRow(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"person.signed_in","personId":"ada","organisationId":"org-1","occurredAt":"2026-10-11T18:00:00Z"}`)
	if err := Apply(context.Background(), store, payload); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), store, payload); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), store, []byte(`{"type":"person.signed_in","personId":"eli","organisationId":"org-1","occurredAt":"2026-10-11T18:05:00Z"}`)); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), store, []byte(`{"type":"person.signed_in","personId":"other","organisationId":"org-2","occurredAt":"2026-10-11T18:06:00Z"}`)); err != nil {
		t.Fatal(err)
	}

	rows, err := store.List(context.Background(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].ActorPersonID != "ada" || rows[0].Action != records.ActionPersonSignedIn || rows[0].EntityType != records.EntityTypePerson || rows[0].EntityID != "ada" {
		t.Fatalf("first = %#v", rows[0])
	}
	if rows[1].OrganisationID != "org-1" || !rows[1].OccurredAt.Equal(time.Date(2026, 10, 11, 18, 5, 0, 0, time.UTC)) {
		t.Fatalf("second = %#v", rows[1])
	}
}
