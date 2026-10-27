-- +goose Up
CREATE TABLE outbox_events (
    id text PRIMARY KEY,
    event_type text NOT NULL,
    schema_version integer NOT NULL,
    aggregate_id text NOT NULL,
    occurred_at timestamptz NOT NULL,
    correlation_id text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL,
    attempts integer NOT NULL,
    next_attempt_at timestamptz NOT NULL,
    last_error text NOT NULL,
    published_at timestamptz,
    created_at timestamptz NOT NULL,
    CHECK (schema_version > 0),
    CHECK (attempts >= 0),
    CHECK (status IN ('pending', 'published', 'dead'))
);

CREATE INDEX outbox_events_pending ON outbox_events (next_attempt_at, occurred_at)
    WHERE status = 'pending';

-- +goose Down
DROP TABLE outbox_events;
