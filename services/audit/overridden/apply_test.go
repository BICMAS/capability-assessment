package overridden

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresTheOverrideActor(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"level.overridden","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","overrideId":"override-1","actorPersonId":"1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33","occurredAt":"2026-10-11T21:30:00Z"}`)
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
	if row.Action != records.ActionLevelOverridden || row.EntityType != records.EntityTypeLevelOverride || row.EntityID != "override-1" || row.ActorPersonID != "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33" {
		t.Fatalf("row = %#v", row)
	}
	if !row.OccurredAt.Equal(time.Date(2026, 10, 11, 21, 30, 0, 0, time.UTC)) {
		t.Fatalf("occurredAt = %s", row.OccurredAt)
	}
}
