-- +goose Up
CREATE TABLE learning_resources (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    title text NOT NULL,
    skill_id text NOT NULL,
    target_level integer NOT NULL,
    reference_uri text NOT NULL,
    rationale text NOT NULL,
    CHECK (target_level >= 0 AND target_level <= 4),
    CHECK (char_length(title) > 0),
    CHECK (char_length(reference_uri) > 0)
);

CREATE TABLE learning_recommendations (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    skill_id text NOT NULL,
    required_level integer NOT NULL,
    intended_level integer NOT NULL,
    activity text NOT NULL,
    rationale text NOT NULL,
    reference_uri text NOT NULL,
    status text NOT NULL,
    UNIQUE (organisation_id, person_id, skill_id, intended_level),
    CHECK (status = 'recommended'),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (intended_level >= 0 AND intended_level <= 4)
);

CREATE TABLE practice_sessions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    scenario_key text NOT NULL,
    message text NOT NULL,
    feedback text NOT NULL
);

-- +goose Down
DROP TABLE practice_sessions;
DROP TABLE learning_recommendations;
DROP TABLE learning_resources;
