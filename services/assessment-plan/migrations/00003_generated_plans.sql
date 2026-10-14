-- +goose Up
ALTER TABLE plan_skills
    ADD COLUMN requires_sme_review boolean NOT NULL DEFAULT false,
    ADD COLUMN rubric_needs_review boolean NOT NULL DEFAULT false;

CREATE UNIQUE INDEX plans_one_generated_per_assignment ON plans (assignment_id) WHERE status = 'generated';

-- +goose Down
DROP INDEX plans_one_generated_per_assignment;
ALTER TABLE plan_skills
    DROP COLUMN rubric_needs_review,
    DROP COLUMN requires_sme_review;
