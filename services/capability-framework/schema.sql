CREATE TABLE frameworks (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX frameworks_by_organisation ON frameworks (organisation_id, name);

CREATE TABLE framework_versions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_id text NOT NULL REFERENCES frameworks (id),
    version_number integer NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (framework_id, version_number),
    CHECK (status IN ('draft', 'in_review', 'approved', 'published', 'retired')),
    CHECK (version_number > 0)
);

CREATE INDEX framework_versions_by_organisation ON framework_versions (organisation_id, framework_id);

CREATE TABLE roles (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    name text NOT NULL
);

CREATE INDEX roles_by_version ON roles (organisation_id, framework_version_id, name);

CREATE TABLE capabilities (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    name text NOT NULL
);

CREATE INDEX capabilities_by_version ON capabilities (organisation_id, framework_version_id, name);

CREATE TABLE proficiency_levels (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    capability_id text NOT NULL REFERENCES capabilities (id),
    level integer NOT NULL,
    UNIQUE (capability_id, level),
    CHECK (level >= 0 AND level <= 4)
);

CREATE TABLE behavioural_indicators (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    proficiency_level_id text NOT NULL REFERENCES proficiency_levels (id),
    statement text NOT NULL
);

CREATE INDEX behavioural_indicators_by_level ON behavioural_indicators (organisation_id, proficiency_level_id);

CREATE TABLE role_capability_requirements (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    role_id text NOT NULL REFERENCES roles (id),
    capability_id text NOT NULL REFERENCES capabilities (id),
    required_level integer NOT NULL,
    UNIQUE (role_id, capability_id),
    CHECK (required_level >= 0 AND required_level <= 4)
);

CREATE INDEX requirements_by_version ON role_capability_requirements (organisation_id, framework_version_id);

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
