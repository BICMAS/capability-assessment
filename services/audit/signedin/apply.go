package signedin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bicmas-academy/learn/services/audit/records"
)

type Event struct {
	Type           string `json:"type"`
	PersonID       string `json:"personId"`
	OrganisationID string `json:"organisationId"`
	OccurredAt     string `json:"occurredAt"`
}

func Apply(ctx context.Context, store records.Store, payload []byte) error {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode person.signed_in: %w", err)
	}
	if event.Type != "person.signed_in" || event.PersonID == "" || event.OrganisationID == "" || event.OccurredAt == "" {
		return fmt.Errorf("person.signed_in is missing fields")
	}
	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse occurredAt: %w", err)
	}
	return store.Append(ctx, records.Record{
		ActorPersonID:  event.PersonID,
		OrganisationID: event.OrganisationID,
		Action:         records.ActionPersonSignedIn,
		EntityType:     records.EntityTypePerson,
		EntityID:       event.PersonID,
		OccurredAt:     occurredAt,
	})
}
