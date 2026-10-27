//go:build integration

package events

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	nats "github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"

	"github.com/bicmas-academy/learn/services/capability-framework/framework"
)

func TestPublishFrameworkPublishedRoundTrip(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Fatal("NATS_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	publisher, err := Open(ctx, natsURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(publisher.Close)

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
		Durable:       "ci-framework-published",
		FilterSubject: FrameworkSubject,
		AckPolicy:     js.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}

	event := framework.FrameworkPublished{
		Type:               "framework.published",
		OrganisationID:     "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21",
		FrameworkVersionID: "3a1c9e70-2d48-4f16-8b95-0c7e4a1d2f33",
		ActorPersonID:      "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22",
		OccurredAt:         "2026-10-11T18:00:00Z",
	}
	if err := publisher.PublishFrameworkPublished(ctx, event); err != nil {
		t.Fatal(err)
	}

	msgs, err := consumer.Fetch(1, js.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	var got publishedEvent
	var seen bool
	for msg := range msgs.Messages() {
		seen = true
		if err := json.Unmarshal(msg.Data(), &got); err != nil {
			t.Fatal(err)
		}
		if err := msg.Ack(); err != nil {
			t.Fatal(err)
		}
	}
	if err := msgs.Error(); err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("expected a framework.published message")
	}
	if got.Type != event.Type || got.FrameworkVersionID != event.FrameworkVersionID || got.OrganisationID != event.OrganisationID {
		t.Fatalf("got %#v", got)
	}
}
