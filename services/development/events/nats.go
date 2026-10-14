package events

import (
	"context"
	"encoding/json"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/development/pathway"
)

const (
	StreamName            = "LEARN_EVENTS"
	PersonSignedInSubject = "person.signed_in"
	AssignmentSubject     = "assignment.changed"
	FrameworkSubject      = "framework.published"
	LevelSubject          = "level.overridden"
	LearningSubject       = "learning.completed"
	AssessmentSubject     = "assessment.assigned"
	PlanApprovedSubject   = "assessment.plan_approved"
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
		Name: StreamName,
		Subjects: []string{
			PersonSignedInSubject,
			AssignmentSubject,
			FrameworkSubject,
			LevelSubject,
			LearningSubject,
			AssessmentSubject,
			PlanApprovedSubject,
		},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishLearningCompleted(ctx context.Context, event pathway.LearningCompleted) error {
	payload, err := json.Marshal(completedEvent{
		Type:           event.Type,
		OrganisationID: event.OrganisationID,
		PersonID:       event.PersonID,
		PlanItemID:     event.PlanItemID,
		OccurredAt:     event.OccurredAt,
	})
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, LearningSubject, payload)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

type completedEvent struct {
	Type           string `json:"type"`
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	PlanItemID     string `json:"planItemId"`
	OccurredAt     string `json:"occurredAt"`
}
