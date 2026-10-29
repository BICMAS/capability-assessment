-- +goose Up
DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT c.conname INTO constraint_name
    FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    WHERE t.relname = 'role_profile_versions'
      AND c.contype = 'c'
      AND pg_get_constraintdef(c.oid) LIKE '%profile_in_review%';
    IF constraint_name IS NULL THEN
        RAISE EXCEPTION 'role profile status check was not found';
    END IF;
    EXECUTE format('ALTER TABLE role_profile_versions DROP CONSTRAINT %I', constraint_name);
END $$;

ALTER TABLE role_profile_versions
    ADD CONSTRAINT role_profile_versions_status_check
    CHECK (status IN ('profile_draft', 'profile_in_review', 'profile_published', 'profile_superseded'));

ALTER TABLE role_profile_sources
    ADD COLUMN kind text NOT NULL DEFAULT 'reference';

ALTER TABLE role_profile_sources
    ADD CONSTRAINT role_profile_sources_kind_check
    CHECK (kind IN ('reference', 'job_description'));

CREATE TABLE role_profile_approval_assignments (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    assignee_person_id text NOT NULL,
    assigned_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (profile_version_id, assignee_person_id)
);

CREATE INDEX role_profile_approval_assignments_by_version
    ON role_profile_approval_assignments (organisation_id, profile_version_id);

CREATE TABLE role_profile_approval_decisions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    assignment_id text NOT NULL REFERENCES role_profile_approval_assignments (id),
    actor_person_id text NOT NULL,
    decision text NOT NULL,
    comment text NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (decision IN ('approved', 'rejected', 'changes_requested'))
);

CREATE INDEX role_profile_approval_decisions_by_version
    ON role_profile_approval_decisions (organisation_id, profile_version_id, created_at);

-- +goose Down
DROP TABLE role_profile_approval_decisions;
DROP TABLE role_profile_approval_assignments;
ALTER TABLE role_profile_sources DROP CONSTRAINT role_profile_sources_kind_check;
ALTER TABLE role_profile_sources DROP COLUMN kind;
ALTER TABLE role_profile_versions DROP CONSTRAINT role_profile_versions_status_check;
ALTER TABLE role_profile_versions
    ADD CONSTRAINT role_profile_versions_status_check
    CHECK (status IN ('profile_draft', 'profile_in_review', 'profile_published'));
