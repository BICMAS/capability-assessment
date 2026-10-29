CREATE TABLE organisations (
    id text PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE departments (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    name text NOT NULL
);

CREATE INDEX departments_by_organisation ON departments (organisation_id, name);

CREATE TABLE people (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    name text NOT NULL,
    department_id text REFERENCES departments (id)
);

CREATE INDEX people_by_organisation ON people (organisation_id, name);

CREATE TABLE manager_links (
    organisation_id text NOT NULL,
    person_id text NOT NULL REFERENCES people (id),
    manager_id text NOT NULL REFERENCES people (id),
    PRIMARY KEY (organisation_id, person_id)
);

CREATE TABLE role_assignments (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    person_id text NOT NULL REFERENCES people (id),
    title text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX role_assignments_by_organisation ON role_assignments (organisation_id, created_at);

CREATE TABLE organograms (
    id text PRIMARY KEY,
    organisation_id text NOT NULL REFERENCES organisations (id),
    name text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
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
