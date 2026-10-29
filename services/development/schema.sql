CREATE TABLE course_mappings (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    external_course_id text NOT NULL,
    capability_id text NOT NULL,
    target_level integer NOT NULL,
    title text NOT NULL,
    UNIQUE (organisation_id, capability_id, target_level),
    CHECK (target_level >= 0 AND target_level <= 4),
    CHECK (char_length(external_course_id) > 0),
    CHECK (char_length(title) > 0)
);

CREATE TABLE pathways (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    capability_id text NOT NULL,
    target_level integer NOT NULL,
    UNIQUE (organisation_id, person_id, capability_id),
    CHECK (target_level >= 0 AND target_level <= 4)
);

CREATE TABLE plan_items (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    pathway_id text NOT NULL REFERENCES pathways (id),
    position integer NOT NULL,
    stage text NOT NULL,
    title text NOT NULL,
    status text NOT NULL,
    deep_link text,
    UNIQUE (pathway_id, position),
    CHECK (stage IN ('learn', 'practise', 'apply', 'validate')),
    CHECK (status IN ('ready', 'complete', 'upcoming')),
    CHECK (position > 0)
);

CREATE INDEX plan_items_by_pathway ON plan_items (organisation_id, pathway_id, position);

CREATE TABLE learning_resources (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    title text NOT NULL,
    skill_id text NOT NULL,
    target_level integer NOT NULL,
    reference_uri text NOT NULL,
    rationale text NOT NULL,
    CHECK (target_level >= 0 AND target_level <= 4),
    CHECK (char_length(title) > 0),
    CHECK (char_length(reference_uri) > 0)
);

CREATE TABLE learning_recommendations (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    skill_id text NOT NULL,
    required_level integer NOT NULL,
    intended_level integer NOT NULL,
    activity text NOT NULL,
    rationale text NOT NULL,
    reference_uri text NOT NULL,
    status text NOT NULL,
    UNIQUE (organisation_id, person_id, skill_id, intended_level),
    CHECK (status = 'recommended'),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (intended_level >= 0 AND intended_level <= 4)
);

CREATE TABLE practice_sessions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    scenario_key text NOT NULL,
    message text NOT NULL,
    feedback text NOT NULL
);

CREATE TABLE completions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_item_id text NOT NULL UNIQUE REFERENCES plan_items (id),
    person_id text NOT NULL,
    occurred_at timestamptz NOT NULL
);
