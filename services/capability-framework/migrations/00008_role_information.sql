-- +goose Up
ALTER TABLE role_profile_versions
    ADD COLUMN organogram_id text NOT NULL DEFAULT '',
    ADD COLUMN position_id text NOT NULL DEFAULT '',
    ADD COLUMN business_unit text NOT NULL DEFAULT '',
    ADD COLUMN sub_industry text NOT NULL DEFAULT '',
    ADD COLUMN organisational_context text NOT NULL DEFAULT '';

ALTER TABLE role_profile_sources
    ADD COLUMN body text NOT NULL DEFAULT '',
    ADD COLUMN origin text NOT NULL DEFAULT 'user';

ALTER TABLE role_profile_sources
    ADD CONSTRAINT role_profile_sources_origin_check
    CHECK (origin IN ('user', 'imported', 'ai_suggested'));

ALTER TABLE role_profile_questions
    ADD COLUMN authored_by text NOT NULL DEFAULT 'service';

ALTER TABLE role_profile_questions
    ADD CONSTRAINT role_profile_questions_authored_by_check
    CHECK (authored_by IN ('service', 'model', 'service_after_model_error'));

CREATE TABLE role_profile_job_descriptions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    file_name text NOT NULL,
    media_type text NOT NULL,
    byte_size integer NOT NULL,
    body bytea NOT NULL,
    detail text NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (byte_size > 0)
);

CREATE INDEX role_profile_job_descriptions_by_version
    ON role_profile_job_descriptions (organisation_id, profile_version_id);

-- +goose Down
DROP TABLE role_profile_job_descriptions;

ALTER TABLE role_profile_questions DROP CONSTRAINT role_profile_questions_authored_by_check;
ALTER TABLE role_profile_questions DROP COLUMN authored_by;

ALTER TABLE role_profile_sources DROP CONSTRAINT role_profile_sources_origin_check;
ALTER TABLE role_profile_sources DROP COLUMN origin;
ALTER TABLE role_profile_sources DROP COLUMN body;

ALTER TABLE role_profile_versions
    DROP COLUMN organisational_context,
    DROP COLUMN sub_industry,
    DROP COLUMN business_unit,
    DROP COLUMN position_id,
    DROP COLUMN organogram_id;
