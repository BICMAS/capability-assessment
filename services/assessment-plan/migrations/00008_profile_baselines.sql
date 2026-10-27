-- +goose Up
CREATE TABLE profile_plan_baselines (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_id text NOT NULL UNIQUE REFERENCES plans (id),
    assignment_id text NOT NULL,
    person_id text NOT NULL,
    profile_id text NOT NULL,
    profile_version_id text NOT NULL,
    version_number integer NOT NULL,
    role_id text NOT NULL,
    role_name text NOT NULL,
    skills text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organisation_id, person_id, profile_version_id),
    CHECK (version_number > 0),
    CHECK (char_length(profile_version_id) > 0),
    CHECK (char_length(skills) > 0)
);

-- +goose Down
DROP TABLE profile_plan_baselines;
