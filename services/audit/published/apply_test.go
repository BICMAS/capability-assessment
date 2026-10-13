package published

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresThePublishActor(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"framework.published","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","frameworkVersionId":"version-1","actorPersonId":"3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11","occurredAt":"2026-10-11T21:00:00Z"}`)
	if err := Apply(context.Background(), store, payload); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List(context.Background(), "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	row := rows[0]
	if row.Action != records.ActionFrameworkPublished || row.EntityType != records.EntityTypeFrameworkVersion || row.EntityID != "version-1" || row.ActorPersonID != "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11" {
		t.Fatalf("row = %#v", row)
	}
	if !row.OccurredAt.Equal(time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)) {
		t.Fatalf("occurredAt = %s", row.OccurredAt)
	}
}
