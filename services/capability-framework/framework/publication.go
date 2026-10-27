package framework

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/outbox"
)

func newPublicationEvent(organisationID, versionID, actorID, correlationID string, occurredAt time.Time) (OutboxEvent, error) {
	if correlationID == "" {
		var err error
		correlationID, err = newID()
		if err != nil {
			return OutboxEvent{}, err
		}
	}
	eventID, err := newID()
	if err != nil {
		return OutboxEvent{}, err
	}
	payload, err := json.Marshal(FrameworkPublished{
		Type:               outbox.FrameworkPublishedType,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		ActorPersonID:      actorID,
		OccurredAt:         occurredAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return OutboxEvent{}, err
	}
	if err := outbox.ValidateFrameworkPublished(payload); err != nil {
		return OutboxEvent{}, err
	}
	return OutboxEvent{
		ID:            eventID,
		EventType:     outbox.FrameworkPublishedType,
		SchemaVersion: PublicationSchemaVersion,
		AggregateID:   versionID,
		OccurredAt:    occurredAt.UTC(),
		CorrelationID: correlationID,
		Payload:       payload,
	}, nil
}

func acceptedPublication(event OutboxEvent, organisationID, versionID string) error {
	if err := event.validate(); err != nil {
		return err
	}
	if event.EventType != outbox.FrameworkPublishedType || event.SchemaVersion != PublicationSchemaVersion || event.AggregateID != versionID {
		return ErrInvalid
	}
	if err := outbox.ValidateFrameworkPublished(event.Payload); err != nil {
		return ErrInvalid
	}
	var body FrameworkPublished
	if err := json.Unmarshal(event.Payload, &body); err != nil {
		return ErrInvalid
	}
	if body.OrganisationID != organisationID || body.FrameworkVersionID != versionID {
		return ErrInvalid
	}
	return nil
}

func correlationID(r *http.Request) string {
	if value := r.Header.Get("X-Correlation-Id"); value != "" {
		return value
	}
	return r.Header.Get("X-Request-Id")
}
