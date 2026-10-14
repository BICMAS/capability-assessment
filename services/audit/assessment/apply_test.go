package assessment

import (
	"context"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

func TestApplyStoresTheAdmin(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"assessment.assigned","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","personId":"7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22","assignmentId":"assignment-1","actorPersonId":"3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11","frameworkVersionId":"version-1","roleId":"role-1","occurredAt":"2026-10-14T08:00:00Z"}`)
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
	if row.Action != records.ActionAssessmentAssigned || row.EntityType != records.EntityTypeAssessmentAssignment || row.EntityID != "assignment-1" || row.ActorPersonID != "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11" {
		t.Fatalf("row = %#v", row)
	}
	if !row.OccurredAt.Equal(time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("occurredAt = %s", row.OccurredAt)
	}
}

func TestApplyApprovedStoresTheApprover(t *testing.T) {
	store := &records.Memory{}
	payload := []byte(`{"type":"assessment.plan_approved","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","personId":"7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22","assignmentId":"assignment-1","planId":"plan-1","actorPersonId":"3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11","versionNumber":1,"occurredAt":"2026-10-14T09:00:00Z"}`)
	if err := ApplyApproved(context.Background(), store, payload); err != nil {
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
	if row.Action != records.ActionAssessmentPlanApproved || row.EntityType != records.EntityTypeAssessmentPlan || row.EntityID != "plan-1" || row.ActorPersonID != "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11" {
		t.Fatalf("row = %#v", row)
	}
}
