package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestRelayMarksPublishedAfterBrokerAck(t *testing.T) {
	store := newMemStore(Event{
		ID: "evt-1", EventType: FrameworkPublishedType, SchemaVersion: 1, AggregateID: "ver-1",
		OccurredAt: time.Now().UTC().Add(-time.Minute), CorrelationID: "corr-1",
		Payload: []byte(`{"type":"framework.published","organisationId":"org","frameworkVersionId":"ver-1","actorPersonId":"ada","occurredAt":"2026-10-11T21:00:00Z"}`),
	})
	broker := &recordingBroker{}
	metrics := &Metrics{}
	NewRelay(store, broker, metrics, nil).Dispatch(context.Background())
	if len(broker.payloads) != 1 || broker.ids[0] != "evt-1" || broker.correlations[0] != "corr-1" {
		t.Fatalf("broker = %#v", broker.ids)
	}
	if store.events["evt-1"].status != "published" {
		t.Fatalf("status = %s", store.events["evt-1"].status)
	}
	if metrics.Snapshot().Pending != 0 {
		t.Fatalf("pending = %d", metrics.Snapshot().Pending)
	}
}

func TestRelayPublishesARoleProfileEventOnItsOwnSubject(t *testing.T) {
	payload := []byte(`{"type":"role_profile.published","organisationId":"org","profileId":"profile-1","profileVersionId":"version-1","versionNumber":2,"actorPersonId":"ada","occurredAt":"2026-10-23T09:00:00Z"}`)
	store := newMemStore(Event{
		ID: "evt-role", EventType: RoleProfilePublishedType, SchemaVersion: 1, AggregateID: "version-1",
		OccurredAt: time.Now().UTC(), CorrelationID: "corr-role", Payload: payload,
	})
	broker := &recordingBroker{}
	NewRelay(store, broker, &Metrics{}, nil).Dispatch(context.Background())
	if len(broker.subjects) != 1 || broker.subjects[0] != RoleProfilePublishedType || store.events["evt-role"].status != "published" {
		t.Fatalf("subjects = %#v status = %s", broker.subjects, store.events["evt-role"].status)
	}
}

func TestRelayKeepsEventWhenBrokerIsDown(t *testing.T) {
	store := newMemStore(Event{
		ID: "evt-2", EventType: FrameworkPublishedType, SchemaVersion: 1, AggregateID: "ver-2",
		OccurredAt: time.Now().UTC(), CorrelationID: "corr-2",
		Payload: []byte(`{"type":"framework.published","organisationId":"org","frameworkVersionId":"ver-2","actorPersonId":"ada","occurredAt":"2026-10-11T21:00:00Z"}`),
	})
	broker := &failingBroker{err: errors.New("nats unavailable")}
	NewRelay(store, broker, &Metrics{}, nil).Dispatch(context.Background())
	got := store.events["evt-2"]
	if got.status != "pending" || got.Attempts != 1 || got.lastError == "" {
		t.Fatalf("event = %+v", got)
	}
	if broker.calls != attemptsPerTick {
		t.Fatalf("calls = %d", broker.calls)
	}
}

func TestRelayDeadLettersAfterBoundedFailures(t *testing.T) {
	store := newMemStore(Event{
		ID: "evt-3", EventType: FrameworkPublishedType, SchemaVersion: 1, AggregateID: "ver-3",
		OccurredAt: time.Now().UTC(), CorrelationID: "corr-3",
		Payload: []byte(`{"type":"framework.published","organisationId":"org","frameworkVersionId":"ver-3","actorPersonId":"ada","occurredAt":"2026-10-11T21:00:00Z"}`),
	})
	relay := NewRelay(store, &failingBroker{err: errors.New("nats unavailable")}, &Metrics{}, nil)
	for i := 0; i < maxAttempts; i++ {
		store.release()
		relay.Dispatch(context.Background())
	}
	if store.events["evt-3"].status != "dead" {
		t.Fatalf("status = %s attempts = %d", store.events["evt-3"].status, store.events["evt-3"].Attempts)
	}
	if store.events["evt-3"].Attempts != maxAttempts {
		t.Fatalf("attempts = %d", store.events["evt-3"].Attempts)
	}
}

func TestValidateFrameworkPublishedRejectsExtraFields(t *testing.T) {
	payload := []byte(`{"type":"framework.published","organisationId":"org","frameworkVersionId":"ver","actorPersonId":"ada","occurredAt":"2026-10-11T21:00:00Z","eventId":"nope"}`)
	if err := ValidateFrameworkPublished(payload); err == nil {
		t.Fatal("expected schema rejection")
	}
}

type stored struct {
	Event
	status    string
	lastError string
	next      time.Time
}

type memStore struct {
	mu     sync.Mutex
	events map[string]*stored
}

func newMemStore(event Event) *memStore {
	return &memStore{events: map[string]*stored{event.ID: {Event: event, status: "pending"}}}
}

func (m *memStore) ClaimPending(_ context.Context, now time.Time, limit int, lease time.Duration) ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	claimed := []Event{}
	for _, event := range m.events {
		if event.status == "pending" && !event.next.After(now) && len(claimed) < limit {
			event.next = now.Add(lease)
			claimed = append(claimed, event.Event)
		}
	}
	return claimed, nil
}

func (m *memStore) MarkPublished(_ context.Context, id string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[id].status = "published"
	return nil
}

func (m *memStore) MarkFailed(_ context.Context, id string, attempts int, next time.Time, status, lastError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	event := m.events[id]
	event.Attempts = attempts
	event.status = status
	event.next = next
	event.lastError = lastError
	return nil
}

func (m *memStore) Stats(_ context.Context, now time.Time) (Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats := Stats{}
	for _, event := range m.events {
		switch event.status {
		case "pending":
			stats.Pending++
			age := now.Sub(event.OccurredAt)
			if age > stats.OldestPendingAge {
				stats.OldestPendingAge = age
			}
		case "dead":
			stats.Dead++
		}
	}
	return stats, nil
}

func (m *memStore) release() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range m.events {
		if event.status == "pending" {
			event.next = time.Time{}
		}
	}
}

type recordingBroker struct {
	subjects     []string
	ids          []string
	correlations []string
	payloads     [][]byte
}

func (b *recordingBroker) PublishPayload(_ context.Context, subject, messageID, correlationID string, _ int, payload []byte) error {
	b.subjects = append(b.subjects, subject)
	b.ids = append(b.ids, messageID)
	b.correlations = append(b.correlations, correlationID)
	b.payloads = append(b.payloads, payload)
	return nil
}

type failingBroker struct {
	err   error
	calls int
}

func (b *failingBroker) PublishPayload(context.Context, string, string, string, int, []byte) error {
	b.calls++
	return b.err
}
