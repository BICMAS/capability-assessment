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

CREATE TABLE outbox_events (
    id text PRIMARY KEY,
    event_type text NOT NULL,
    schema_version integer NOT NULL,
    aggregate_id text NOT NULL,
    occurred_at timestamptz NOT NULL,
    correlation_id text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL,
    attempts integer NOT NULL,
    next_attempt_at timestamptz NOT NULL,
    last_error text NOT NULL,
    published_at timestamptz,
    created_at timestamptz NOT NULL,
    CHECK (schema_version > 0),
    CHECK (attempts >= 0),
    CHECK (status IN ('pending', 'published', 'dead'))
);

CREATE INDEX outbox_events_pending ON outbox_events (next_attempt_at, occurred_at)
    WHERE status = 'pending';

CREATE TABLE profile_roles (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organisation_id, name)
);

CREATE TABLE canonical_skills (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    name text NOT NULL,
    category text NOT NULL,
    definition text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organisation_id, name),
    CHECK (category IN ('technical', 'leadership', 'domain', 'communication', 'compliance'))
);

CREATE TABLE role_profiles (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    role_id text NOT NULL REFERENCES profile_roles (id),
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organisation_id, role_id)
);

CREATE TABLE role_profile_versions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    role_profile_id text NOT NULL REFERENCES role_profiles (id),
    version_number integer NOT NULL,
    revision integer NOT NULL,
    status text NOT NULL,
    purpose text NOT NULL,
    responsibilities text NOT NULL,
    business_outcomes text NOT NULL,
    decision_scope text NOT NULL,
    industry text NOT NULL,
    region text NOT NULL,
    team_size integer,
    reporting_level text NOT NULL,
    framework_version_id text REFERENCES framework_versions (id),
    ai_draft text NOT NULL,
    ai_model text NOT NULL,
    ai_source_context_version text NOT NULL,
    ai_generated_at timestamptz,
    created_at timestamptz NOT NULL,
    UNIQUE (role_profile_id, version_number),
    CHECK (version_number > 0),
    CHECK (revision > 0),
    CHECK (status IN ('profile_draft', 'profile_in_review', 'profile_published')),
    CHECK (reporting_level IN ('', 'individual', 'team_lead', 'manager', 'director', 'executive')),
    CHECK (team_size IS NULL OR team_size > 0)
);

CREATE INDEX role_profile_versions_by_profile ON role_profile_versions (organisation_id, role_profile_id, version_number);

CREATE TABLE role_profile_sources (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    title text NOT NULL,
    reference_uri text NOT NULL
);

CREATE TABLE role_profile_skills (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    skill_id text NOT NULL REFERENCES canonical_skills (id),
    category text NOT NULL,
    definition text NOT NULL,
    required_proficiency integer NOT NULL,
    priority text NOT NULL,
    rationale text NOT NULL,
    provenance text NOT NULL,
    position integer NOT NULL,
    UNIQUE (profile_version_id, skill_id),
    CHECK (category IN ('technical', 'leadership', 'domain', 'communication', 'compliance')),
    CHECK (required_proficiency >= 0 AND required_proficiency <= 4),
    CHECK (priority IN ('required', 'preferred', 'optional')),
    CHECK (provenance IN ('ai_suggested', 'hr_adjusted', 'imported', 'manually_added')),
    CHECK (position > 0)
);

CREATE INDEX role_profile_skills_by_version ON role_profile_skills (organisation_id, profile_version_id, position);

CREATE TABLE role_profile_comments (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    author_person_id text NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE role_profile_audit (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    role_profile_id text NOT NULL REFERENCES role_profiles (id),
    profile_version_id text NOT NULL,
    actor_person_id text NOT NULL,
    action text NOT NULL,
    occurred_at timestamptz NOT NULL
);

CREATE INDEX role_profile_audit_by_profile ON role_profile_audit (organisation_id, role_profile_id, occurred_at);

CREATE TABLE role_profile_contexts (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    version_number integer NOT NULL,
    content_hash text NOT NULL,
    snapshot text NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (profile_version_id, version_number),
    UNIQUE (profile_version_id, content_hash),
    CHECK (version_number > 0)
);

CREATE TABLE role_profile_questions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    context_version integer NOT NULL,
    round integer NOT NULL,
    field text NOT NULL,
    question text NOT NULL,
    critical boolean NOT NULL,
    answer text NOT NULL,
    asked_actor_id text NOT NULL,
    answered_actor_id text NOT NULL,
    asked_at timestamptz NOT NULL,
    answered_at timestamptz,
    CHECK (round > 0)
);

CREATE INDEX role_profile_questions_by_version ON role_profile_questions (organisation_id, profile_version_id, context_version);

CREATE TABLE role_profile_generation_jobs (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    profile_version_id text NOT NULL REFERENCES role_profile_versions (id),
    context_version integer NOT NULL,
    context_hash text NOT NULL,
    idempotency_key text NOT NULL,
    status text NOT NULL,
    model_id text NOT NULL,
    prompt_version text NOT NULL,
    input_hash text NOT NULL,
    output_hash text NOT NULL,
    latency_ms integer NOT NULL,
    prompt_tokens integer NOT NULL,
    completion_tokens integer NOT NULL,
    attempts integer NOT NULL,
    last_error text NOT NULL,
    unsupported text NOT NULL,
    uncertainty text NOT NULL,
    result text NOT NULL,
    cancel_requested boolean NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    started_at timestamptz,
    finished_at timestamptz,
    UNIQUE (organisation_id, idempotency_key),
    CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'stale', 'cancelled')),
    CHECK (attempts >= 0)
);

CREATE INDEX role_profile_generation_jobs_queued ON role_profile_generation_jobs (created_at)
    WHERE status = 'queued';
