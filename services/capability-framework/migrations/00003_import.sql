-- +goose Up
CREATE TABLE framework_imports (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text REFERENCES framework_versions (id),
    object_key text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX framework_imports_by_organisation ON framework_imports (organisation_id);

CREATE TABLE import_rows (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    import_id text NOT NULL REFERENCES framework_imports (id),
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    row_number integer NOT NULL,
    framework_version_label text NOT NULL,
    department text NOT NULL,
    role_code text NOT NULL,
    role_name text NOT NULL,
    capability_code text NOT NULL,
    capability_name text NOT NULL,
    capability_type text NOT NULL,
    definition text NOT NULL,
    required_level text NOT NULL,
    proficiency_levels text NOT NULL,
    behavioural_indicators text NOT NULL,
    assessment_methods text NOT NULL,
    learning_asset_ids text NOT NULL,
    practice_scenario_ids text NOT NULL,
    workplace_mission text NOT NULL,
    weight text NOT NULL,
    governance_status text NOT NULL
);

CREATE INDEX import_rows_by_version ON import_rows (organisation_id, framework_version_id);

CREATE TABLE import_row_errors (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    row_number integer NOT NULL,
    message text NOT NULL
);

CREATE INDEX import_row_errors_by_version ON import_row_errors (organisation_id, framework_version_id);

CREATE TABLE review_proposals (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    kind text NOT NULL,
    status text NOT NULL,
    capability_code text NOT NULL,
    message text NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (kind IN ('duplicate_capability_code', 'missing_required_level')),
    CHECK (status IN ('pending', 'accepted', 'edited', 'rejected'))
);

CREATE INDEX review_proposals_by_version ON review_proposals (organisation_id, framework_version_id, status);

-- +goose Down
DROP TABLE review_proposals;
DROP TABLE import_row_errors;
DROP TABLE import_rows;
DROP TABLE framework_imports;
