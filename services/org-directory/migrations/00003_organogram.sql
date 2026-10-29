-- +goose Up
CREATE TABLE organograms (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    name text NOT NULL,
    created_by_person_id text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organisation_id, name)
);

CREATE INDEX organograms_by_organisation ON organograms (organisation_id, name);

CREATE TABLE positions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    organogram_id text NOT NULL REFERENCES organograms (id),
    department_id text REFERENCES departments (id),
    title text NOT NULL,
    person_id text REFERENCES people (id),
    reports_to_position_id text REFERENCES positions (id),
    created_at timestamptz NOT NULL
);

CREATE INDEX positions_by_organogram ON positions (organisation_id, organogram_id);

-- +goose Down
DROP TABLE positions;
DROP TABLE organograms;
