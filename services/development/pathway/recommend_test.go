package pathway

import (
	"context"
	"testing"
	"time"
)

func TestVerifiedGapsMapToExistingPathways(t *testing.T) {
	store := NewMemory("https://lms.bicmas.example")
	if _, err := store.MapCourse(context.Background(), MappingInput{
		OrganisationID: organisationID, ExternalCourseID: "course-1", CapabilityID: capabilityID, TargetLevel: 3, Title: "Negotiation Fundamentals",
	}); err != nil {
		t.Fatal(err)
	}
	first, err := RecommendPathways(context.Background(), store, organisationID, []GapSignal{
		{PersonID: eliID, CapabilityID: capabilityID, TargetLevel: 3, State: GapBelow},
		{PersonID: eliID, CapabilityID: "missing-skill", TargetLevel: 3, State: GapMissing},
		{PersonID: eliID, CapabilityID: capabilityID, TargetLevel: 2, State: GapMet},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].CapabilityID != capabilityID || first[0].Status != StatusReady || first[0].EmploymentDecision != EmploymentDecision {
		t.Fatalf("recommendations = %+v", first)
	}
	pathway, err := store.GetPathway(context.Background(), organisationID, eliID, capabilityID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Complete(context.Background(), CompletionInput{
		OrganisationID: organisationID, PersonID: eliID, PlanItemID: pathway.Items[0].ID, CompletedAt: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	again, err := RecommendPathways(context.Background(), store, organisationID, []GapSignal{
		{PersonID: eliID, CapabilityID: capabilityID, TargetLevel: 3, State: GapBelow},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 || again[0].PathwayID != first[0].PathwayID || again[0].Status != StatusComplete {
		t.Fatalf("repeat = %+v", again)
	}
}
