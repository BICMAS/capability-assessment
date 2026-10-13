-- +goose Up
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

INSERT INTO organisations (id, name) VALUES
    ('8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', 'BICMAS');

INSERT INTO departments (id, organisation_id, name) VALUES
    ('c4e8a1b2-7d63-4f90-8a15-2b6c9d0e3f71', '8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', 'Sales');

INSERT INTO people (id, organisation_id, name, department_id) VALUES
    ('3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11', '8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', 'Ada Admin', NULL),
    ('1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33', '8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', 'Maya Manager', NULL),
    ('7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22', '8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', 'Eli Employee', 'c4e8a1b2-7d63-4f90-8a15-2b6c9d0e3f71');

INSERT INTO manager_links (organisation_id, person_id, manager_id) VALUES
    ('8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', '7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22', '1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33');

INSERT INTO role_assignments (id, organisation_id, person_id, title, created_at) VALUES
    ('6e2a9c80-1b47-4d5f-9c31-8a0f7e2d4b66', '8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21', '7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22', 'Area Sales Manager', '2026-10-11T18:00:00Z');

-- +goose Down
DROP TABLE role_assignments;
DROP TABLE manager_links;
DROP TABLE people;
DROP TABLE departments;
DROP TABLE organisations;
