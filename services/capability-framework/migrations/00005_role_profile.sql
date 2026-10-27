-- +goose Up
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

-- +goose Down
DROP TABLE role_profile_audit;
DROP TABLE role_profile_comments;
DROP TABLE role_profile_skills;
DROP TABLE role_profile_sources;
DROP TABLE role_profile_versions;
DROP TABLE role_profiles;
DROP TABLE canonical_skills;
DROP TABLE profile_roles;
