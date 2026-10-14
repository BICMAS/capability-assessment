-- +goose Up
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
    UNIQUE (assignment_id, version_number),
    CHECK (version_number > 0),
    CHECK (status IN ('draft', 'generated', 'under_review', 'changes_requested', 'approved'))
);

CREATE INDEX plans_by_assignment ON plans (organisation_id, assignment_id, version_number);

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
    UNIQUE (plan_id, position),
    UNIQUE (plan_id, requirement_id),
    UNIQUE (plan_id, capability_id),
    CHECK (position > 0),
    CHECK (required_level >= 0 AND required_level <= 4),
    CHECK (char_length(requirement_id) > 0),
    CHECK (char_length(capability_id) > 0),
    CHECK (char_length(recommended_method) > 0 AND char_length(recommended_method) <= 80),
    CHECK (char_length(rationale) > 0 AND char_length(rationale) <= 280),
    CHECK (char_length(instructions) > 0 AND char_length(instructions) <= 8000)
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

-- +goose StatementBegin
CREATE FUNCTION reject_approved_plan_change() RETURNS trigger AS $$
DECLARE
    plan_status text;
    parent_plan_id text;
BEGIN
    IF TG_TABLE_NAME = 'plans' THEN
        IF OLD.status = 'approved' THEN
            RAISE EXCEPTION 'approved plan cannot be changed';
        END IF;
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NEW;
    END IF;

    IF TG_TABLE_NAME = 'plan_skills' THEN
        IF TG_OP = 'DELETE' THEN
            parent_plan_id := OLD.plan_id;
        ELSE
            parent_plan_id := NEW.plan_id;
        END IF;
    ELSE
        IF TG_OP = 'DELETE' THEN
            SELECT plan_id INTO parent_plan_id FROM plan_skills WHERE id = OLD.plan_skill_id;
        ELSE
            SELECT plan_id INTO parent_plan_id FROM plan_skills WHERE id = NEW.plan_skill_id;
        END IF;
    END IF;

    SELECT status INTO plan_status FROM plans WHERE id = parent_plan_id;
    IF plan_status = 'approved' THEN
        RAISE EXCEPTION 'approved plan cannot be changed';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER plans_reject_approved_change
    BEFORE UPDATE OR DELETE ON plans
    FOR EACH ROW EXECUTE FUNCTION reject_approved_plan_change();

CREATE TRIGGER plan_skills_reject_approved_change
    BEFORE INSERT OR UPDATE OR DELETE ON plan_skills
    FOR EACH ROW EXECUTE FUNCTION reject_approved_plan_change();

CREATE TRIGGER assessment_items_reject_approved_change
    BEFORE INSERT OR UPDATE OR DELETE ON assessment_items
    FOR EACH ROW EXECUTE FUNCTION reject_approved_plan_change();

CREATE TRIGGER rubric_bands_reject_approved_change
    BEFORE INSERT OR UPDATE OR DELETE ON rubric_bands
    FOR EACH ROW EXECUTE FUNCTION reject_approved_plan_change();

-- +goose Down
DROP TRIGGER rubric_bands_reject_approved_change ON rubric_bands;
DROP TRIGGER assessment_items_reject_approved_change ON assessment_items;
DROP TRIGGER plan_skills_reject_approved_change ON plan_skills;
DROP TRIGGER plans_reject_approved_change ON plans;
DROP FUNCTION reject_approved_plan_change();
DROP TABLE rubric_bands;
DROP TABLE assessment_items;
DROP TABLE plan_skills;
DROP TABLE plans;
DROP TABLE assignments;
