-- +goose Up
CREATE TABLE assessment_sessions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_id text NOT NULL UNIQUE REFERENCES plans (id),
    person_id text NOT NULL,
    status text NOT NULL,
    started_at timestamptz,
    submitted_at timestamptz,
    CHECK (status IN ('assigned', 'started', 'submitted')),
    CHECK (char_length(person_id) > 0),
    CHECK (
        (status = 'assigned' AND started_at IS NULL AND submitted_at IS NULL)
        OR (status = 'started' AND started_at IS NOT NULL AND submitted_at IS NULL)
        OR (status = 'submitted' AND started_at IS NOT NULL AND submitted_at IS NOT NULL)
    )
);

CREATE INDEX assessment_sessions_by_person ON assessment_sessions (organisation_id, person_id);

CREATE TABLE session_answers (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    session_id text NOT NULL REFERENCES assessment_sessions (id),
    assessment_item_id text NOT NULL REFERENCES assessment_items (id),
    answer text NOT NULL,
    UNIQUE (session_id, assessment_item_id),
    CHECK (char_length(answer) <= 8000)
);

CREATE INDEX session_answers_by_session ON session_answers (organisation_id, session_id);

-- +goose Down
DROP TABLE session_answers;
DROP TABLE assessment_sessions;
