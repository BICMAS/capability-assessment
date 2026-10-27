package events

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/capability-framework/framework"
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
		Name:       StreamName,
		Subjects:   []string{PersonSignedInSubject, AssignmentSubject, FrameworkSubject, LevelSubject, LearningSubject, AssessmentSubject, PlanApprovedSubject, "assessment.completed", "role_profile.published"},
		Duplicates: 2 * time.Hour,
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishFrameworkPublished(ctx context.Context, event framework.FrameworkPublished) error {
	payload, err := json.Marshal(publishedEvent{
		Type:               event.Type,
		OrganisationID:     event.OrganisationID,
		FrameworkVersionID: event.FrameworkVersionID,
		ActorPersonID:      event.ActorPersonID,
		OccurredAt:         event.OccurredAt,
	})
	if err != nil {
		return err
	}
	return p.PublishPayload(ctx, FrameworkSubject, "", "", 0, payload)
}

func (p *Publisher) PublishPayload(ctx context.Context, subject, messageID, correlationID string, schemaVersion int, payload []byte) error {
	msg := &nats.Msg{Subject: subject, Data: payload, Header: nats.Header{}}
	if messageID != "" {
		msg.Header.Set("Nats-Msg-Id", messageID)
	}
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	if schemaVersion > 0 {
		msg.Header.Set("X-Schema-Version", strconv.Itoa(schemaVersion))
	}
	msg.Header.Set("X-Event-Type", subject)
	_, err := p.stream.PublishMsg(ctx, msg)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

type publishedEvent struct {
	Type               string `json:"type"`
	OrganisationID     string `json:"organisationId"`
	FrameworkVersionID string `json:"frameworkVersionId"`
	ActorPersonID      string `json:"actorPersonId"`
	OccurredAt         string `json:"occurredAt"`
}
