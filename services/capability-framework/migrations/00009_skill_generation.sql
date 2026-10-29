-- +goose Up
ALTER TABLE role_profile_skills
    ADD COLUMN criteria text NOT NULL DEFAULT '',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE role_profile_generation_jobs
    ADD COLUMN replace_existing boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE role_profile_generation_jobs DROP COLUMN replace_existing;

ALTER TABLE role_profile_skills
    DROP COLUMN updated_at,
    DROP COLUMN created_at,
    DROP COLUMN criteria;
