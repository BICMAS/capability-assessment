package outbox

import (
	"context"
	"database/sql"
	"time"
)

type SQLStore struct {
	database *sql.DB
}

func NewSQLStore(database *sql.DB) SQLStore {
	return SQLStore{database: database}
}

func (s SQLStore) ClaimPending(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]Event, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, event_type, schema_version, aggregate_id, occurred_at, correlation_id, payload, attempts
		FROM outbox_events
		WHERE status = 'pending' AND next_attempt_at <= $1
		ORDER BY occurred_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	claimed := []Event{}
	ids := []string{}
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.EventType, &event.SchemaVersion, &event.AggregateID, &event.OccurredAt, &event.CorrelationID, &event.Payload, &event.Attempts); err != nil {
			return nil, err
		}
		claimed = append(claimed, event)
		ids = append(ids, event.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	leaseUntil := now.Add(lease)
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE outbox_events SET next_attempt_at = $2 WHERE id = $1 AND status = 'pending'
		`, id, leaseUntil); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s SQLStore) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	_, err := s.database.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = 'published', published_at = $2, last_error = ''
		WHERE id = $1 AND status = 'pending'
	`, id, publishedAt)
	return err
}

func (s SQLStore) MarkFailed(ctx context.Context, id string, attempts int, next time.Time, status, lastError string) error {
	_, err := s.database.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, attempts = $3, next_attempt_at = $4, last_error = $5
		WHERE id = $1 AND status = 'pending'
	`, id, status, attempts, next, lastError)
	return err
}

func (s SQLStore) Stats(ctx context.Context, now time.Time) (Stats, error) {
	var stats Stats
	var oldest sql.NullTime
	err := s.database.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM outbox_events WHERE status = 'pending'),
			(SELECT COUNT(*) FROM outbox_events WHERE status = 'dead'),
			(SELECT MIN(occurred_at) FROM outbox_events WHERE status = 'pending')
	`).Scan(&stats.Pending, &stats.Dead, &oldest)
	if err != nil {
		return Stats{}, err
	}
	if oldest.Valid {
		stats.OldestPendingAge = now.Sub(oldest.Time)
		if stats.OldestPendingAge < 0 {
			stats.OldestPendingAge = 0
		}
	}
	return stats, nil
}
