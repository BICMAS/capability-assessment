package assignment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

type Event struct {
	Type             string `json:"type"`
	OrganisationID   string `json:"organisationId"`
	PersonID         string `json:"personId"`
	RoleAssignmentID string `json:"roleAssignmentId"`
	OccurredAt       string `json:"occurredAt"`
}

func Apply(ctx context.Context, store records.Store, payload []byte) error {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode assignment.changed: %w", err)
	}
	if event.Type != "assignment.changed" || event.OrganisationID == "" || event.PersonID == "" || event.RoleAssignmentID == "" || event.OccurredAt == "" {
		return fmt.Errorf("assignment.changed is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.PersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionAssignmentChanged,
		EntityType:     records.EntityTypeRoleAssignment,
		EntityID:       event.RoleAssignmentID,
		OccurredAt:     occurredAt,
	})
}
