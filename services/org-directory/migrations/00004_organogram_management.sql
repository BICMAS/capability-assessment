-- +goose Up
ALTER TABLE organograms
    ADD COLUMN status text NOT NULL DEFAULT 'draft';

ALTER TABLE organograms
    ADD CONSTRAINT organograms_status_check CHECK (status IN ('draft', 'published'));

CREATE TABLE organogram_files (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    organogram_id text NOT NULL REFERENCES organograms (id),
    file_name text NOT NULL,
    media_type text NOT NULL,
    byte_size integer NOT NULL,
    body bytea NOT NULL,
    status text NOT NULL CHECK (status IN ('imported', 'rejected', 'needs_review')),
    detail text NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX organogram_files_by_chart ON organogram_files (organisation_id, organogram_id, created_at);

-- +goose Down
DROP TABLE organogram_files;
ALTER TABLE organograms DROP CONSTRAINT organograms_status_check;
ALTER TABLE organograms DROP COLUMN status;
