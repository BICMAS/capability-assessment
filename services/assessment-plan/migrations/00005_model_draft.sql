-- +goose Up
ALTER TABLE plans
    ADD COLUMN prompt_version text,
    ADD COLUMN model_name text;

ALTER TABLE plans
    ADD CONSTRAINT plans_prompt_version_length CHECK (prompt_version IS NULL OR (char_length(prompt_version) > 0 AND char_length(prompt_version) <= 80)),
    ADD CONSTRAINT plans_model_name_length CHECK (model_name IS NULL OR (char_length(model_name) > 0 AND char_length(model_name) <= 80)),
    ADD CONSTRAINT plans_model_draft_pair CHECK (
        (prompt_version IS NULL AND model_name IS NULL)
        OR (prompt_version IS NOT NULL AND model_name IS NOT NULL)
    );

-- +goose Down
ALTER TABLE plans
    DROP CONSTRAINT plans_model_draft_pair,
    DROP CONSTRAINT plans_model_name_length,
    DROP CONSTRAINT plans_prompt_version_length,
    DROP COLUMN model_name,
    DROP COLUMN prompt_version;
