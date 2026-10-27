# Transactional outbox

Capability-framework publication is the first service on this pattern. The HTTP handler does not publish to NATS itself. It asks the domain store to commit the version change and an outbox row in one PostgreSQL transaction. A background relay publishes pending rows to the existing `LEARN_EVENTS` JetStream stream.

## Record

`outbox_events` stores:

| Column | Role |
| --- | --- |
| `id` | Unique event ID. Also sent as `Nats-Msg-Id` so JetStream can drop a duplicate publish after a crash. |
| `event_type` | `framework.published` |
| `schema_version` | `1` |
| `aggregate_id` | Framework version ID |
| `occurred_at` | Occurrence timestamp copied into the payload |
| `correlation_id` | `X-Correlation-Id` or `X-Request-Id`, otherwise a new ID |
| `payload` | JSON body validated against `contracts/events/framework.published.schema.json` |

The public payload stays inside that schema (`additionalProperties: false`). Envelope fields travel as columns and NATS headers (`X-Correlation-Id`, `X-Schema-Version`, `X-Event-Type`), not as extra JSON properties.

## Relay

`outbox.Relay` polls pending rows whose `next_attempt_at` is due, claims them with `FOR UPDATE SKIP LOCKED`, and leases them for 30 seconds. Each attempt calls the broker up to three times with short backoff. The row becomes `published` only after JetStream accepts the message.

A failed attempt increments `attempts`, stores `last_error`, and schedules the next try with exponential backoff capped at 60 seconds. After 8 attempts the row becomes `dead` and is left for an operator. Pending rows survive process restarts because they are committed before any broker call.

Metrics logged each tick: pending count, dead count, oldest pending age, publication failures, and retries.

## Consumer

Audit applies `framework.published` through `published.Apply`. The audit primary key plus `ON CONFLICT DO NOTHING` makes a redelivery a no-op. The in-memory store uses the same key. NATS delivery is not treated as exactly-once for the database write. The audit consumer naks with a one-second delay and stops after 10 deliveries (`MaxDeliver`). The message remains in `LEARN_EVENTS` until stream retention removes it, so an operator can still read it with a new consumer.

## Operations

Migration `services/capability-framework/migrations/00004_outbox.sql` is additive (`CREATE TABLE` only). Services apply goose **up** on startup. Do not run the down migration in production.

Inspect:

```sql
SELECT id, status, attempts, last_error, occurred_at, next_attempt_at
FROM outbox_events
WHERE status IN ('pending', 'dead')
ORDER BY occurred_at;
```

Requeue a dead event after the broker is healthy:

```sql
UPDATE outbox_events
SET status = 'pending', attempts = 0, next_attempt_at = now(), last_error = ''
WHERE id = '<event-id>' AND status = 'dead';
```

If the process stops after commit and before publish, the row stays `pending` (or returns to pending when its lease expires). Restart the capability-framework process; the relay publishes it. No second broker and no distributed transaction is involved.

## Applying the same pattern

org-directory (`assignment.changed`) and assessment-plan (`assessment.assigned`, `assessment.plan_approved`) still publish directly after their own database writes. To follow this pattern in one service at a time:

1. Add an outbox table in that service's database. Do not share capability-framework's table.
2. Insert the outbox row in the same transaction as the state change that the event describes.
3. Validate the payload against that event's schema under `contracts/events/` before insert.
4. Run a relay that publishes to the existing `LEARN_EVENTS` stream and marks the row published only after acknowledgement.
5. Keep the audit consumer idempotent on the natural key (already `ON CONFLICT DO NOTHING`).
6. Dead-letter inside the outbox after bounded attempts. Do not add another broker.
