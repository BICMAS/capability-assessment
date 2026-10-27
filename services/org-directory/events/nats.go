package events

import (
	"context"
	"encoding/json"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/org-directory/directory"
)

const (
	StreamName            = "LEARN_EVENTS"
	AssignmentSubject     = "assignment.changed"
	PersonSignedInSubject = "person.signed_in"
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
		Name:     StreamName,
		Subjects: []string{PersonSignedInSubject, AssignmentSubject, FrameworkSubject, LevelSubject, LearningSubject, AssessmentSubject, PlanApprovedSubject, "assessment.completed", "role_profile.published"},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishAssignmentChanged(ctx context.Context, event directory.AssignmentChanged) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, AssignmentSubject, payload)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}
