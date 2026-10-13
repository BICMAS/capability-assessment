package events

import (
	"context"
	"encoding/json"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/assessment-gap/levels"
)

const (
	StreamName            = "LEARN_EVENTS"
	PersonSignedInSubject = "person.signed_in"
	AssignmentSubject     = "assignment.changed"
	FrameworkSubject      = "framework.published"
	LevelSubject          = "level.overridden"
	LearningSubject       = "learning.completed"
)

type Publisher struct {
	stream js.JetStream
	conn   *nats.Conn
}

func Open(ctx context.Context, natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	stream, err := js.New(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := stream.CreateOrUpdateStream(ctx, js.StreamConfig{
		Name:     StreamName,
		Subjects: []string{PersonSignedInSubject, AssignmentSubject, FrameworkSubject, LevelSubject, LearningSubject},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishLevelOverridden(ctx context.Context, event levels.LevelOverridden) error {
	payload, err := json.Marshal(overriddenEvent{
		Type:           event.Type,
		OrganisationID: event.OrganisationID,
		OverrideID:     event.OverrideID,
		ActorPersonID:  event.ActorPersonID,
		OccurredAt:     event.OccurredAt,
	})
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, LevelSubject, payload)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

type overriddenEvent struct {
	Type           string `json:"type"`
	OrganisationID string `json:"organisationId"`
	OverrideID     string `json:"overrideId"`
	ActorPersonID  string `json:"actorPersonId"`
	OccurredAt     string `json:"occurredAt"`
}
