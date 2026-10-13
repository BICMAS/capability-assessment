CREATE TABLE inbox_items (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    person_id text NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX inbox_items_by_person ON inbox_items (organisation_id, person_id, created_at);
