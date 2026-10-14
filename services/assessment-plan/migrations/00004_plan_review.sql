-- +goose Up
ALTER TABLE plans
    ADD COLUMN change_reason text,
    ADD COLUMN approved_by_person_id text,
    ADD COLUMN approved_at timestamptz;

ALTER TABLE plans
    ADD CONSTRAINT plans_change_reason_length CHECK (change_reason IS NULL OR (char_length(change_reason) > 0 AND char_length(change_reason) <= 280)),
    ADD CONSTRAINT plans_approved_by_length CHECK (approved_by_person_id IS NULL OR char_length(approved_by_person_id) > 0),
    ADD CONSTRAINT plans_approval_pair CHECK (
        (status = 'approved' AND approved_by_person_id IS NOT NULL AND approved_at IS NOT NULL)
        OR (status <> 'approved' AND approved_by_person_id IS NULL AND approved_at IS NULL)
    );

ALTER TABLE plan_skills
    ADD COLUMN edited_by_person_id text,
    ADD CONSTRAINT plan_skills_edited_by_length CHECK (edited_by_person_id IS NULL OR char_length(edited_by_person_id) > 0);

-- +goose Down
ALTER TABLE plan_skills
    DROP CONSTRAINT plan_skills_edited_by_length,
    DROP COLUMN edited_by_person_id;
ALTER TABLE plans
    DROP CONSTRAINT plans_approval_pair,
    DROP CONSTRAINT plans_approved_by_length,
    DROP CONSTRAINT plans_change_reason_length,
    DROP COLUMN approved_at,
    DROP COLUMN approved_by_person_id,
    DROP COLUMN change_reason;
