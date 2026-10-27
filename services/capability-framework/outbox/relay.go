package outbox

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	maxAttempts     = 8
	attemptsPerTick = 3
	claimLimit      = 20
	lease           = 30 * time.Second
	maxBackoff      = 60 * time.Second
	pollInterval    = time.Second
)

type Event struct {
	ID            string
	EventType     string
	SchemaVersion int
	AggregateID   string
	OccurredAt    time.Time
	CorrelationID string
	Payload       []byte
	Attempts      int
}

type Store interface {
	ClaimPending(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]Event, error)
	MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, id string, attempts int, next time.Time, status, lastError string) error
	Stats(ctx context.Context, now time.Time) (Stats, error)
}

type Publisher interface {
	PublishPayload(ctx context.Context, subject, messageID, correlationID string, schemaVersion int, payload []byte) error
}

type Stats struct {
	Pending             int
	Dead                int
	OldestPendingAge    time.Duration
	PublicationFailures int64
	Retries             int64
}

type Metrics struct {
	mu    sync.Mutex
	stats Stats
}

func (m *Metrics) Snapshot() Stats {
	if m == nil {
		return Stats{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

func (m *Metrics) record(stats Stats) {
	m.mu.Lock()
	m.stats = stats
	m.mu.Unlock()
}

type Relay struct {
	store     Store
	publisher Publisher
	metrics   *Metrics
	logger    *slog.Logger
	interval  time.Duration
}

func NewRelay(store Store, publisher Publisher, metrics *Metrics, logger *slog.Logger) *Relay {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &Relay{
		store:     store,
		publisher: publisher,
		metrics:   metrics,
		logger:    logger,
		interval:  pollInterval,
	}
}

func publishSubject(event Event) (string, error) {
	switch event.EventType {
	case FrameworkPublishedType:
		if err := ValidateFrameworkPublished(event.Payload); err != nil {
			return "", err
		}
		return FrameworkPublishedType, nil
	case RoleProfilePublishedType:
		if err := ValidateRoleProfilePublished(event.Payload); err != nil {
			return "", err
		}
		return RoleProfilePublishedType, nil
	default:
		return "", errors.New("unexpected event type")
	}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.Dispatch(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Dispatch(ctx)
		}
	}
}

func (r *Relay) Dispatch(ctx context.Context) {
	now := time.Now().UTC()
	events, err := r.store.ClaimPending(ctx, now, claimLimit, lease)
	if err != nil {
		r.logger.Error("outbox claim failed", "err", err)
		r.refresh(ctx, now, 0, 0)
		return
	}
	var failures int64
	var retries int64
	for _, event := range events {
		if err := r.publishOne(ctx, event); err != nil {
			failures++
			retries += int64(attemptsPerTick)
			status, next, attempts := nextFailure(event.Attempts, time.Now().UTC())
			if markErr := r.store.MarkFailed(ctx, event.ID, attempts, next, status, err.Error()); markErr != nil {
				r.logger.Error("outbox failure update failed", "eventId", event.ID, "err", markErr)
			}
			r.logger.Error("outbox publish failed", "eventId", event.ID, "status", status, "attempts", attempts, "err", err)
			continue
		}
		if err := r.store.MarkPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			failures++
			r.logger.Error("outbox mark published failed", "eventId", event.ID, "err", err)
		}
	}
	r.refresh(ctx, time.Now().UTC(), failures, retries)
}

func (r *Relay) publishOne(ctx context.Context, event Event) error {
	subject, err := publishSubject(event)
	if err != nil {
		return err
	}
	var last error
	backoff := 20 * time.Millisecond
	for attempt := 1; attempt <= attemptsPerTick; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := r.publisher.PublishPayload(callCtx, subject, event.ID, event.CorrelationID, event.SchemaVersion, event.Payload)
		cancel()
		if err == nil {
			return nil
		}
		last = err
		if attempt == attemptsPerTick || ctx.Err() != nil {
			break
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}
	if last == nil {
		last = errors.New("publish failed")
	}
	return last
}

func nextFailure(attempts int, now time.Time) (status string, next time.Time, updated int) {
	updated = attempts + 1
	if updated >= maxAttempts {
		return "dead", now, updated
	}
	delay := time.Second << updated
	if delay > maxBackoff || delay <= 0 {
		delay = maxBackoff
	}
	return "pending", now.Add(delay), updated
}

func (r *Relay) refresh(ctx context.Context, now time.Time, failures, retries int64) {
	stats, err := r.store.Stats(ctx, now)
	if err != nil {
		r.logger.Error("outbox stats failed", "err", err)
		return
	}
	current := r.metrics.Snapshot()
	stats.PublicationFailures = current.PublicationFailures + failures
	stats.Retries = current.Retries + retries
	r.metrics.record(stats)
	r.logger.Info("outbox metrics",
		"pending", stats.Pending,
		"dead", stats.Dead,
		"oldestPendingSeconds", stats.OldestPendingAge.Seconds(),
		"publicationFailures", stats.PublicationFailures,
		"retries", stats.Retries,
	)
}
