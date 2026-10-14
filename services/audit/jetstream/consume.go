package jetstream

import (
	"context"
	"log/slog"
	"time"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/audit/assessment"
	"github.com/bicmas-academy/learn/services/audit/assignment"
	"github.com/bicmas-academy/learn/services/audit/learning"
	"github.com/bicmas-academy/learn/services/audit/overridden"
	"github.com/bicmas-academy/learn/services/audit/published"
	"github.com/bicmas-academy/learn/services/audit/records"
	"github.com/bicmas-academy/learn/services/audit/signedin"
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

type Subscription struct {
	consumes []js.ConsumeContext
	conn     *nats.Conn
}

func Subscribe(ctx context.Context, natsURL string, store records.Store, logger *slog.Logger) (*Subscription, error) {
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
		Subjects: []string{Subject, AssignmentSubject, FrameworkSubject, LevelSubject, LearningSubject, AssessmentSubject, PlanApprovedSubject},
	}); err != nil {
		conn.Close()
		return nil, err
	}
	signedIn, err := consume(stream, ctx, "audit-person-signed-in", Subject, func(applyCtx context.Context, payload []byte) error {
		return signedin.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		conn.Close()
		return nil, err
	}
	changed, err := consume(stream, ctx, "audit-assignment-changed", AssignmentSubject, func(applyCtx context.Context, payload []byte) error {
		return assignment.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		conn.Close()
		return nil, err
	}
	frameworkPublished, err := consume(stream, ctx, "audit-framework-published", FrameworkSubject, func(applyCtx context.Context, payload []byte) error {
		return published.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		changed.Stop()
		conn.Close()
		return nil, err
	}
	levelOverridden, err := consume(stream, ctx, "audit-level-overridden", LevelSubject, func(applyCtx context.Context, payload []byte) error {
		return overridden.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		changed.Stop()
		frameworkPublished.Stop()
		conn.Close()
		return nil, err
	}
	learningCompleted, err := consume(stream, ctx, "audit-learning-completed", LearningSubject, func(applyCtx context.Context, payload []byte) error {
		return learning.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		changed.Stop()
		frameworkPublished.Stop()
		levelOverridden.Stop()
		conn.Close()
		return nil, err
	}
	assessmentAssigned, err := consume(stream, ctx, "audit-assessment-assigned", AssessmentSubject, func(applyCtx context.Context, payload []byte) error {
		return assessment.Apply(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		changed.Stop()
		frameworkPublished.Stop()
		levelOverridden.Stop()
		learningCompleted.Stop()
		conn.Close()
		return nil, err
	}
	planApproved, err := consume(stream, ctx, "audit-assessment-plan-approved", PlanApprovedSubject, func(applyCtx context.Context, payload []byte) error {
		return assessment.ApplyApproved(applyCtx, store, payload)
	}, logger)
	if err != nil {
		signedIn.Stop()
		changed.Stop()
		frameworkPublished.Stop()
		levelOverridden.Stop()
		learningCompleted.Stop()
		assessmentAssigned.Stop()
		conn.Close()
		return nil, err
	}
	return &Subscription{consumes: []js.ConsumeContext{signedIn, changed, frameworkPublished, levelOverridden, learningCompleted, assessmentAssigned, planApproved}, conn: conn}, nil
}

func consume(stream js.JetStream, ctx context.Context, durable, subject string, apply func(context.Context, []byte) error, logger *slog.Logger) (js.ConsumeContext, error) {
	consumer, err := stream.CreateOrUpdateConsumer(ctx, StreamName, js.ConsumerConfig{
		Durable:       durable,
		FilterSubject: subject,
		AckPolicy:     js.AckExplicitPolicy,
	})
	if err != nil {
		return nil, err
	}
	return consumer.Consume(func(msg js.Msg) {
		applyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := apply(applyCtx, msg.Data()); err != nil {
			logger.Error("audit record failed", "subject", subject, "err", err)
			if nakErr := msg.Nak(); nakErr != nil {
				logger.Error("nak failed", "err", nakErr)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			logger.Error("ack failed", "err", err)
		}
	})
}

func (s *Subscription) Close() {
	if s == nil {
		return
	}
	for _, active := range s.consumes {
		active.Stop()
	}
	s.conn.Close()
}
