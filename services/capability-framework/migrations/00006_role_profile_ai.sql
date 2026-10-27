-- +goose Up
CREATE TABLE role_profile_contexts (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    version_number integer NOT NULL,
    content_hash text NOT NULL,
    snapshot text NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (profile_version_id, version_number),
    UNIQUE (profile_version_id, content_hash),
    CHECK (version_number > 0)
);

CREATE TABLE role_profile_questions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    context_version integer NOT NULL,
    round integer NOT NULL,
    field text NOT NULL,
    question text NOT NULL,
    critical boolean NOT NULL,
    answer text NOT NULL,
    asked_actor_id text NOT NULL,
    answered_actor_id text NOT NULL,
    asked_at timestamptz NOT NULL,
    answered_at timestamptz,
    CHECK (round > 0)
);

CREATE INDEX role_profile_questions_by_version ON role_profile_questions (organisation_id, profile_version_id, context_version);

CREATE TABLE role_profile_generation_jobs (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    context_version integer NOT NULL,
    context_hash text NOT NULL,
    idempotency_key text NOT NULL,
    status text NOT NULL,
    model_id text NOT NULL,
    prompt_version text NOT NULL,
    input_hash text NOT NULL,
    output_hash text NOT NULL,
    latency_ms integer NOT NULL,
    prompt_tokens integer NOT NULL,
    completion_tokens integer NOT NULL,
    attempts integer NOT NULL,
    last_error text NOT NULL,
    unsupported text NOT NULL,
    uncertainty text NOT NULL,
    result text NOT NULL,
    cancel_requested boolean NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    started_at timestamptz,
    finished_at timestamptz,
    UNIQUE (organisation_id, idempotency_key),
    CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'stale', 'cancelled')),
    CHECK (attempts >= 0)
);

CREATE INDEX role_profile_generation_jobs_queued ON role_profile_generation_jobs (created_at)
    WHERE status = 'queued';

-- +goose Down
DROP TABLE role_profile_generation_jobs;
DROP TABLE role_profile_questions;
DROP TABLE role_profile_contexts;
