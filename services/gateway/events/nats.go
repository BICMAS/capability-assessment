package events

import (
	"context"
	"encoding/json"
	"errors"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/gateway/auth"
)

const (
	StreamName          = "LEARN_EVENTS"
	Subject             = "person.signed_in"
	AssignmentSubject   = "assignment.changed"
	FrameworkSubject    = "framework.published"
	LevelSubject        = "level.overridden"
	LearningSubject     = "learning.completed"
	AssessmentSubject   = "assessment.assigned"
	PlanApprovedSubject = "assessment.plan_approved"
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
		Subjects: []string{Subject, AssignmentSubject, FrameworkSubject, LevelSubject, LearningSubject, AssessmentSubject, PlanApprovedSubject, "assessment.completed", "role_profile.published"},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishPersonSignedIn(ctx context.Context, event auth.PersonSignedIn) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, Subject, payload)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

func (p *Publisher) Ready() error {
	if p == nil || p.conn == nil || !p.conn.IsConnected() {
		return errors.New("nats unavailable")
	}
	return nil
}
