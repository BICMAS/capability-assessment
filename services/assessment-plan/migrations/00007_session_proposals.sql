-- +goose Up
ALTER TABLE assessment_sessions
    ADD COLUMN confirmed_at timestamptz,
    ADD CONSTRAINT assessment_sessions_confirmed_after_submit CHECK (confirmed_at IS NULL OR status = 'submitted');

CREATE TABLE session_proposals (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    session_id text NOT NULL REFERENCES assessment_sessions (id),
    plan_skill_id text NOT NULL,
    requirement_id text NOT NULL,
    capability_id text NOT NULL,
    required_level integer NOT NULL,
    recommended_method text NOT NULL,
    evidence_source text NOT NULL,
    band_levels text NOT NULL,
    percentage integer NOT NULL,
    UNIQUE (session_id, plan_skill_id),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (evidence_source IN ('knowledge', 'situational_judgement', 'ai_interview')),
    CHECK (percentage >= 0 AND percentage <= 100),
    CHECK (char_length(band_levels) > 0)
);

CREATE INDEX session_proposals_by_session ON session_proposals (organisation_id, session_id);

-- +goose Down
DROP TABLE session_proposals;
ALTER TABLE assessment_sessions
    DROP CONSTRAINT assessment_sessions_confirmed_after_submit,
    DROP COLUMN confirmed_at;
