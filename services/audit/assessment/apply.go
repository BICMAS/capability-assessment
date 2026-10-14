package assessment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

type Event struct {
	Type               string `json:"type"`
	OrganisationID     string `json:"organisationId"`
	PersonID           string `json:"personId"`
	AssignmentID       string `json:"assignmentId"`
	ActorPersonID      string `json:"actorPersonId"`
	FrameworkVersionID string `json:"frameworkVersionId"`
	RoleID             string `json:"roleId"`
	OccurredAt         string `json:"occurredAt"`
}

func Apply(ctx context.Context, store records.Store, payload []byte) error {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode assessment.assigned: %w", err)
	}
	if event.Type != "assessment.assigned" || event.OrganisationID == "" || event.PersonID == "" || event.AssignmentID == "" || event.ActorPersonID == "" || event.FrameworkVersionID == "" || event.RoleID == "" || event.OccurredAt == "" {
		return fmt.Errorf("assessment.assigned is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.ActorPersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionAssessmentAssigned,
		EntityType:     records.EntityTypeAssessmentAssignment,
		EntityID:       event.AssignmentID,
		OccurredAt:     occurredAt,
	})
}

type ApprovedEvent struct {
	Type           string `json:"type"`
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	AssignmentID   string `json:"assignmentId"`
	PlanID         string `json:"planId"`
	ActorPersonID  string `json:"actorPersonId"`
	VersionNumber  int    `json:"versionNumber"`
	OccurredAt     string `json:"occurredAt"`
}

func ApplyApproved(ctx context.Context, store records.Store, payload []byte) error {
	var event ApprovedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode assessment.plan_approved: %w", err)
	}
	if event.Type != "assessment.plan_approved" || event.OrganisationID == "" || event.PersonID == "" || event.AssignmentID == "" || event.PlanID == "" || event.ActorPersonID == "" || event.VersionNumber < 1 || event.OccurredAt == "" {
		return fmt.Errorf("assessment.plan_approved is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.ActorPersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionAssessmentPlanApproved,
		EntityType:     records.EntityTypeAssessmentPlan,
		EntityID:       event.PlanID,
		OccurredAt:     occurredAt,
	})
}
