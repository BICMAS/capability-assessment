package learning

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresTheLearner(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"learning.completed","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","personId":"7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22","planItemId":"plan-item-1","occurredAt":"2026-10-11T22:40:00Z"}`)
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
	if row.Action != records.ActionLearningCompleted || row.EntityType != records.EntityTypePlanItem || row.EntityID != "plan-item-1" || row.ActorPersonID != "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22" {
		t.Fatalf("row = %#v", row)
	}
	if !row.OccurredAt.Equal(time.Date(2026, 10, 11, 22, 40, 0, 0, time.UTC)) {
		t.Fatalf("occurredAt = %s", row.OccurredAt)
	}
}
