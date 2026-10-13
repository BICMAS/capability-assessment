CREATE TABLE attempts (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    framework_version_id text NOT NULL,
    capability_code text NOT NULL,
    source text NOT NULL,
    percentage integer NOT NULL,
    evidence_ids text NOT NULL,
    required_level_tenths integer NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (source IN (
        'knowledge',
        'situational_judgement',
        'self_assessment',
        'manager_assessment',
        'ai_interview',
        'simulation'
    )),
    CHECK (percentage >= 0 AND percentage <= 100),
    CHECK (required_level_tenths >= 0 AND required_level_tenths <= 40)
);

CREATE INDEX attempts_by_level ON attempts (
    organisation_id,
    person_id,
    framework_version_id,
    capability_code,
    created_at
);

CREATE TABLE explanations (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    framework_version_id text NOT NULL,
    capability_code text NOT NULL,
    knowledge_percentage integer,
    situational_judgement_percentage integer,
    ai_interview_percentage integer,
    simulation_percentage integer,
    manager_assessment_percentage integer,
    self_assessment_percentage integer,
    mean_tenths integer NOT NULL,
    divisor integer NOT NULL,
    required_level_tenths integer NOT NULL,
    current_level_tenths integer NOT NULL,
    gap_tenths integer NOT NULL,
    status text NOT NULL,
    UNIQUE (organisation_id, person_id, framework_version_id, capability_code),
    CHECK (status IN ('provisional', 'verified')),
    CHECK (divisor = 25),
    CHECK (required_level_tenths >= 0 AND required_level_tenths <= 40)
);

CREATE TABLE level_overrides (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    explanation_id text NOT NULL REFERENCES explanations (id),
    actor_person_id text NOT NULL,
    reason text NOT NULL,
    previous_value_tenths integer NOT NULL,
    new_value_tenths integer NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (char_length(reason) > 0),
    CHECK (previous_value_tenths >= 0 AND previous_value_tenths <= 40),
    CHECK (new_value_tenths >= 0 AND new_value_tenths <= 40)
);

CREATE INDEX level_overrides_by_explanation ON level_overrides (
    organisation_id,
    explanation_id,
    created_at
);
