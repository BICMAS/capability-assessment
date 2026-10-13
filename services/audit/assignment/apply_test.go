package assignment

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresAssignmentOnce(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"assignment.changed","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","personId":"7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22","roleAssignmentId":"6e2a9c80-1b47-4d5f-9c31-8a0f7e2d4b66","occurredAt":"2026-10-11T18:00:00Z"}`)
	if err := Apply(context.Background(), store, payload); err != nil {
		t.Fatal(err)
	}
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
	if row.Action != records.ActionAssignmentChanged || row.EntityType != records.EntityTypeRoleAssignment || row.EntityID != "6e2a9c80-1b47-4d5f-9c31-8a0f7e2d4b66" || row.ActorPersonID != "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22" {
		t.Fatalf("row = %#v", row)
	}
	if !row.OccurredAt.Equal(time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)) {
		t.Fatalf("occurredAt = %s", row.OccurredAt)
	}
}
