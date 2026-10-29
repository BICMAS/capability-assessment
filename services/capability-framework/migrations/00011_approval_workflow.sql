-- +goose Up
ALTER TABLE role_profile_versions
    ADD COLUMN submitted_at timestamptz,
    ADD COLUMN submitted_by_person_id text NOT NULL DEFAULT '',
    ADD COLUMN submission_cycle integer NOT NULL DEFAULT 0,
    ADD COLUMN workflow_stage text NOT NULL DEFAULT '',
    ADD COLUMN approved_at timestamptz,
    ADD COLUMN approved_by_person_id text NOT NULL DEFAULT '';

ALTER TABLE role_profile_versions
    ADD CONSTRAINT role_profile_versions_submission_cycle_check
    CHECK (submission_cycle >= 0);

ALTER TABLE role_profile_approval_assignments
    ADD COLUMN stage_key text NOT NULL DEFAULT '';

-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT c.conname INTO constraint_name
    FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    WHERE t.relname = 'role_profile_approval_assignments'
      AND c.contype = 'u';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE role_profile_approval_assignments DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE role_profile_approval_assignments
    ADD CONSTRAINT role_profile_approval_assignments_stage_key
    UNIQUE (profile_version_id, stage_key);

ALTER TABLE role_profile_approval_decisions
    ADD COLUMN stage_key text NOT NULL DEFAULT '',
    ADD COLUMN submission_cycle integer NOT NULL DEFAULT 0;

ALTER TABLE role_profile_approval_decisions
    ADD CONSTRAINT role_profile_approval_decisions_cycle_check
    CHECK (submission_cycle >= 0);

CREATE TABLE role_profile_approval_paths (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    stage_key text NOT NULL,
    label text NOT NULL,
    required_role text NOT NULL,
    position integer NOT NULL,
    UNIQUE (organisation_id, stage_key),
    UNIQUE (organisation_id, position),
    CHECK (position > 0),
    CHECK (required_role IN ('hr', 'manager', 'leader'))
);

CREATE TABLE role_profile_workflow_steps (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    stage_key text NOT NULL,
    label text NOT NULL,
    required_role text NOT NULL,
    position integer NOT NULL,
    UNIQUE (profile_version_id, stage_key),
    UNIQUE (profile_version_id, position),
    CHECK (position > 0),
    CHECK (required_role IN ('hr', 'manager', 'leader'))
);

CREATE INDEX role_profile_workflow_steps_by_version
    ON role_profile_workflow_steps (organisation_id, profile_version_id, position);

-- +goose Down
DROP TABLE role_profile_workflow_steps;
DROP TABLE role_profile_approval_paths;

ALTER TABLE role_profile_approval_decisions
    DROP CONSTRAINT role_profile_approval_decisions_cycle_check;
ALTER TABLE role_profile_approval_decisions
    DROP COLUMN submission_cycle,
    DROP COLUMN stage_key;

ALTER TABLE role_profile_approval_assignments
    DROP CONSTRAINT role_profile_approval_assignments_stage_key;
ALTER TABLE role_profile_approval_assignments
    DROP COLUMN stage_key;
ALTER TABLE role_profile_approval_assignments
    ADD CONSTRAINT role_profile_approval_assignments_assignee
    UNIQUE (profile_version_id, assignee_person_id);

ALTER TABLE role_profile_versions
    DROP CONSTRAINT role_profile_versions_submission_cycle_check;
ALTER TABLE role_profile_versions
    DROP COLUMN approved_by_person_id,
    DROP COLUMN approved_at,
    DROP COLUMN workflow_stage,
    DROP COLUMN submission_cycle,
    DROP COLUMN submitted_by_person_id,
    DROP COLUMN submitted_at;
