-- +goose Up
ALTER TABLE role_profile_skills
    ADD COLUMN display_name text NOT NULL DEFAULT '';

CREATE TABLE role_profile_review_changes (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    role_profile_id text NOT NULL REFERENCES role_profiles (id),
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    skill_assignment_id text NOT NULL,
    actor_person_id text NOT NULL,
    fields text NOT NULL,
    rationale text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL
);

CREATE INDEX role_profile_review_changes_by_profile ON role_profile_review_changes (organisation_id, role_profile_id, occurred_at);

-- +goose Down
DROP INDEX role_profile_review_changes_by_profile;
DROP TABLE role_profile_review_changes;

ALTER TABLE role_profile_skills DROP COLUMN display_name;
