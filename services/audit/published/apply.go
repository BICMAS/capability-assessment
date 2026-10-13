package published

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
	FrameworkVersionID string `json:"frameworkVersionId"`
	ActorPersonID      string `json:"actorPersonId"`
	OccurredAt         string `json:"occurredAt"`
}

func Apply(ctx context.Context, store records.Store, payload []byte) error {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode framework.published: %w", err)
	}
	if event.Type != "framework.published" || event.OrganisationID == "" || event.FrameworkVersionID == "" || event.ActorPersonID == "" || event.OccurredAt == "" {
		return fmt.Errorf("framework.published is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.ActorPersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionFrameworkPublished,
		EntityType:     records.EntityTypeFrameworkVersion,
		EntityID:       event.FrameworkVersionID,
		OccurredAt:     occurredAt,
	})
}
