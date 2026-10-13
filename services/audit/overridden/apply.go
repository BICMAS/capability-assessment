package overridden

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

type Event struct {
	Type           string `json:"type"`
	OrganisationID string `json:"organisationId"`
	OverrideID     string `json:"overrideId"`
	ActorPersonID  string `json:"actorPersonId"`
	OccurredAt     string `json:"occurredAt"`
}

func Apply(ctx context.Context, store records.Store, payload []byte) error {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode level.overridden: %w", err)
	}
	if event.Type != "level.overridden" || event.OrganisationID == "" || event.OverrideID == "" || event.ActorPersonID == "" || event.OccurredAt == "" {
		return fmt.Errorf("level.overridden is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.ActorPersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionLevelOverridden,
		EntityType:     records.EntityTypeLevelOverride,
		EntityID:       event.OverrideID,
		OccurredAt:     occurredAt,
	})
}
