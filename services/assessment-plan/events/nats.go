package events

import (
	"context"
	"encoding/json"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/assessment-plan/plan"
)

const (
	StreamName              = "LEARN_EVENTS"
	PersonSignedInSubject   = "person.signed_in"
	AssignmentSubject       = "assignment.changed"
	FrameworkSubject        = "framework.published"
	LevelSubject            = "level.overridden"
	LearningSubject         = "learning.completed"
	AssessmentSubject       = "assessment.assigned"
	PlanApprovedSubject     = "assessment.plan_approved"
	CompletedSubject        = "assessment.completed"
	ProfilePublishedSubject = "role_profile.published"
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
			CompletedSubject,
			ProfilePublishedSubject,
		},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Publisher{stream: stream, conn: conn}, nil
}

func (p *Publisher) PublishAssessmentAssigned(ctx context.Context, event plan.AssessmentAssigned) error {
	payload, err := json.Marshal(assignedEvent{
		Type:               event.Type,
		OrganisationID:     event.OrganisationID,
		PersonID:           event.PersonID,
		AssignmentID:       event.AssignmentID,
		ActorPersonID:      event.ActorPersonID,
		FrameworkVersionID: event.FrameworkVersionID,
		RoleID:             event.RoleID,
		OccurredAt:         event.OccurredAt,
	})
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, AssessmentSubject, payload)
	return err
}

func (p *Publisher) PublishPlanApproved(ctx context.Context, event plan.PlanApproved) error {
	payload, err := json.Marshal(approvedEvent{
		Type:           event.Type,
		OrganisationID: event.OrganisationID,
		PersonID:       event.PersonID,
		AssignmentID:   event.AssignmentID,
		PlanID:         event.PlanID,
		ActorPersonID:  event.ActorPersonID,
		VersionNumber:  event.VersionNumber,
		OccurredAt:     event.OccurredAt,
	})
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, PlanApprovedSubject, payload)
	return err
}

func (p *Publisher) PublishAssessmentCompleted(ctx context.Context, event plan.AssessmentCompleted) error {
	payload, err := json.Marshal(completedAssessmentEvent{
		Type:             event.Type,
		OrganisationID:   event.OrganisationID,
		PersonID:         event.PersonID,
		SessionID:        event.SessionID,
		PlanID:           event.PlanID,
		ActorPersonID:    event.ActorPersonID,
		ProfileVersionID: event.ProfileVersionID,
		OccurredAt:       event.OccurredAt,
	})
	if err != nil {
		return err
	}
	_, err = p.stream.Publish(ctx, CompletedSubject, payload)
	return err
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

type assignedEvent struct {
	Type               string `json:"type"`
	OrganisationID     string `json:"organisationId"`
	PersonID           string `json:"personId"`
	AssignmentID       string `json:"assignmentId"`
	ActorPersonID      string `json:"actorPersonId"`
	FrameworkVersionID string `json:"frameworkVersionId"`
	RoleID             string `json:"roleId"`
	OccurredAt         string `json:"occurredAt"`
}

type completedAssessmentEvent struct {
	Type             string `json:"type"`
	OrganisationID   string `json:"organisationId"`
	PersonID         string `json:"personId"`
	SessionID        string `json:"sessionId"`
	PlanID           string `json:"planId"`
	ActorPersonID    string `json:"actorPersonId"`
	ProfileVersionID string `json:"profileVersionId,omitempty"`
	OccurredAt       string `json:"occurredAt"`
}

type approvedEvent struct {
	Type           string `json:"type"`
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	AssignmentID   string `json:"assignmentId"`
	PlanID         string `json:"planId"`
	ActorPersonID  string `json:"actorPersonId"`
	VersionNumber  int    `json:"versionNumber"`
	OccurredAt     string `json:"occurredAt"`
}
