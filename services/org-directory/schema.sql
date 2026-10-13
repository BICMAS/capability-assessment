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
