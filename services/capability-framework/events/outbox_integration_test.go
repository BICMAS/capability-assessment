//go:build integration

package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"
	"github.com/pressly/goose/v3"

	"github.com/bicmas-academy/learn/services/capability-framework/framework"
	"github.com/bicmas-academy/learn/services/capability-framework/migrations"
	"github.com/bicmas-academy/learn/services/capability-framework/outbox"
)

func TestCommittedOutboxEventSurvivesUntilBrokerAck(t *testing.T) {
	database := openIntegrationDB(t)
	store := framework.NewSQLStore(database)
	ctx := context.Background()
	const organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	draft, err := store.CreateDraft(ctx, organisationID, "Relay framework")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"type":"framework.published","organisationId":"8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21","frameworkVersionId":"` + draft.ID + `","actorPersonId":"3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11","occurredAt":"2026-10-11T21:00:00Z"}`)
	if err := outbox.ValidateFrameworkPublished(payload); err != nil {
		t.Fatal(err)
	}
	event := framework.OutboxEvent{
		ID:            "outbox-relay-" + draft.ID,
		EventType:     outbox.FrameworkPublishedType,
		SchemaVersion: framework.PublicationSchemaVersion,
		AggregateID:   draft.ID,
		OccurredAt:    time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC),
		CorrelationID: "corr-relay",
		Payload:       payload,
	}
	if _, err := store.PublishWithEvent(ctx, organisationID, draft.ID, event); err != nil {
		t.Fatal(err)
	}

	metrics := &outbox.Metrics{}
	outbox.NewRelay(outbox.NewSQLStore(database), downBroker{}, metrics, nil).Dispatch(ctx)
	var status string
	var attempts int
	if err := database.QueryRowContext(ctx, `SELECT status, attempts FROM outbox_events WHERE id = $1`, event.ID).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || attempts != 1 {
		t.Fatalf("after outage status=%s attempts=%d", status, attempts)
	}
	if metrics.Snapshot().PublicationFailures == 0 {
		t.Fatal("expected a publication failure metric")
	}

	if _, err := database.ExecContext(ctx, `UPDATE outbox_events SET next_attempt_at = $2 WHERE id = $1`, event.ID, time.Now().UTC().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Fatal("NATS_URL is required for integration tests")
	}
	publisher, err := Open(ctx, natsURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(publisher.Close)
	outbox.NewRelay(outbox.NewSQLStore(database), publisher, metrics, nil).Dispatch(ctx)
	if err := database.QueryRowContext(ctx, `SELECT status FROM outbox_events WHERE id = $1`, event.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "published" {
		t.Fatalf("status = %s", status)
	}

	conn, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Close)
	jet, err := js.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := jet.CreateOrUpdateConsumer(ctx, StreamName, js.ConsumerConfig{
		Name:          "ci-outbox-" + draft.ID[:8],
		FilterSubject: FrameworkSubject,
		AckPolicy:     js.AckExplicitPolicy,
		DeliverPolicy: js.DeliverAllPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := consumer.Fetch(32, js.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	seen := false
	for msg := range msgs.Messages() {
		var decoded map[string]string
		if err := json.Unmarshal(msg.Data(), &decoded); err != nil {
			t.Fatal(err)
		}
		if err := msg.Ack(); err != nil {
			t.Fatal(err)
		}
		if decoded["frameworkVersionId"] == draft.ID {
			seen = true
			body = decoded
		}
	}
	if err := msgs.Error(); err != nil && !seen {
		t.Fatal(err)
	}
	if !seen || body["frameworkVersionId"] != draft.ID || body["type"] != "framework.published" {
		t.Fatalf("message = %#v", body)
	}
}

type downBroker struct{}

func (downBroker) PublishPayload(context.Context, string, string, string, int, []byte) error {
	return errors.New("nats unavailable")
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Fatal("INTEGRATION_DATABASE_URL is required for integration tests")
	}
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("postgres unavailable: %v", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}
