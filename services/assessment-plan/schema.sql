CREATE TABLE assignments (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    framework_version_id text NOT NULL,
    role_id text NOT NULL,
    assigned_by_person_id text NOT NULL,
    assigned_at timestamptz NOT NULL,
    status text NOT NULL,
    exception_reason text,
    UNIQUE (organisation_id, person_id, framework_version_id, role_id),
    CHECK (status = 'assigned'),
    CHECK (char_length(person_id) > 0),
    CHECK (char_length(framework_version_id) > 0),
    CHECK (char_length(role_id) > 0),
    CHECK (char_length(assigned_by_person_id) > 0),
    CHECK (exception_reason IS NULL OR (char_length(exception_reason) > 0 AND char_length(exception_reason) <= 280))
);

CREATE INDEX assignments_by_person ON assignments (organisation_id, person_id);

CREATE TABLE plans (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    assignment_id text NOT NULL REFERENCES assignments (id),
    version_number integer NOT NULL,
    status text NOT NULL,
    change_reason text,
    approved_by_person_id text,
    approved_at timestamptz,
    prompt_version text,
    model_name text,
    UNIQUE (assignment_id, version_number),
    CHECK (version_number > 0),
    CHECK (status IN ('draft', 'generated', 'under_review', 'changes_requested', 'approved')),
    CHECK (change_reason IS NULL OR (char_length(change_reason) > 0 AND char_length(change_reason) <= 280)),
    CHECK (approved_by_person_id IS NULL OR char_length(approved_by_person_id) > 0),
    CHECK (
        (status = 'approved' AND approved_by_person_id IS NOT NULL AND approved_at IS NOT NULL)
        OR (status <> 'approved' AND approved_by_person_id IS NULL AND approved_at IS NULL)
    ),
    CHECK (prompt_version IS NULL OR (char_length(prompt_version) > 0 AND char_length(prompt_version) <= 80)),
    CHECK (model_name IS NULL OR (char_length(model_name) > 0 AND char_length(model_name) <= 80)),
    CHECK (
        (prompt_version IS NULL AND model_name IS NULL)
        OR (prompt_version IS NOT NULL AND model_name IS NOT NULL)
    )
);

CREATE INDEX plans_by_assignment ON plans (organisation_id, assignment_id, version_number);

CREATE UNIQUE INDEX plans_one_generated_per_assignment ON plans (assignment_id) WHERE status = 'generated';

CREATE TABLE plan_skills (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_id text NOT NULL REFERENCES plans (id),
    position integer NOT NULL,
    requirement_id text NOT NULL,
    capability_id text NOT NULL,
    required_level integer NOT NULL,
    recommended_method text NOT NULL,
    rationale text NOT NULL,
    instructions text NOT NULL,
    requires_sme_review boolean NOT NULL DEFAULT false,
    rubric_needs_review boolean NOT NULL DEFAULT false,
    edited_by_person_id text,
    UNIQUE (plan_id, position),
    UNIQUE (plan_id, requirement_id),
    UNIQUE (plan_id, capability_id),
    CHECK (position > 0),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (char_length(requirement_id) > 0),
    CHECK (char_length(capability_id) > 0),
    CHECK (char_length(recommended_method) > 0 AND char_length(recommended_method) <= 80),
    CHECK (char_length(rationale) > 0 AND char_length(rationale) <= 280),
    CHECK (char_length(instructions) > 0 AND char_length(instructions) <= 8000),
    CHECK (edited_by_person_id IS NULL OR char_length(edited_by_person_id) > 0)
);

CREATE INDEX plan_skills_by_plan ON plan_skills (organisation_id, plan_id, position);

CREATE TABLE assessment_items (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_skill_id text NOT NULL REFERENCES plan_skills (id),
    position integer NOT NULL,
    prompt text NOT NULL,
    UNIQUE (plan_skill_id, position),
    CHECK (position > 0),
    CHECK (char_length(prompt) > 0 AND char_length(prompt) <= 2000)
);

CREATE INDEX assessment_items_by_skill ON assessment_items (organisation_id, plan_skill_id, position);

CREATE TABLE rubric_bands (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_skill_id text NOT NULL REFERENCES plan_skills (id),
    level integer NOT NULL,
    descriptor text NOT NULL,
    UNIQUE (plan_skill_id, level),
    CHECK (level >= 0 AND level <= 4),
    CHECK (char_length(descriptor) > 0 AND char_length(descriptor) <= 2000)
);

CREATE INDEX rubric_bands_by_skill ON rubric_bands (organisation_id, plan_skill_id, level);

CREATE TABLE assessment_sessions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    plan_id text NOT NULL UNIQUE REFERENCES plans (id),
    person_id text NOT NULL,
    status text NOT NULL,
    started_at timestamptz,
    submitted_at timestamptz,
    CHECK (status IN ('assigned', 'started', 'submitted')),
    CHECK (char_length(person_id) > 0),
    CHECK (
        (status = 'assigned' AND started_at IS NULL AND submitted_at IS NULL)
        OR (status = 'started' AND started_at IS NOT NULL AND submitted_at IS NULL)
        OR (status = 'submitted' AND started_at IS NOT NULL AND submitted_at IS NOT NULL)
    ),
    confirmed_at timestamptz,
    CHECK (confirmed_at IS NULL OR status = 'submitted')
);

CREATE INDEX assessment_sessions_by_person ON assessment_sessions (organisation_id, person_id);

CREATE TABLE session_answers (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    session_id text NOT NULL REFERENCES assessment_sessions (id),
    assessment_item_id text NOT NULL REFERENCES assessment_items (id),
    answer text NOT NULL,
    UNIQUE (session_id, assessment_item_id),
    CHECK (char_length(answer) <= 8000)
);

CREATE INDEX session_answers_by_session ON session_answers (organisation_id, session_id);

CREATE TABLE session_proposals (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    session_id text NOT NULL REFERENCES assessment_sessions (id),
    plan_skill_id text NOT NULL,
    requirement_id text NOT NULL,
    capability_id text NOT NULL,
    required_level integer NOT NULL,
    recommended_method text NOT NULL,
    evidence_source text NOT NULL,
    band_levels text NOT NULL,
    percentage integer NOT NULL,
    UNIQUE (session_id, plan_skill_id),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (evidence_source IN ('knowledge', 'situational_judgement', 'ai_interview')),
    CHECK (percentage >= 0 AND percentage <= 100),
    CHECK (char_length(band_levels) > 0)
);

CREATE INDEX session_proposals_by_session ON session_proposals (organisation_id, session_id);
