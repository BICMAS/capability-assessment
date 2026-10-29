-- name: ListDepartments :many
SELECT id, organisation_id, name
FROM departments
WHERE organisation_id = $1
ORDER BY name ASC;

-- name: ListPeople :many
SELECT
    p.id,
    p.organisation_id,
    p.name,
    COALESCE(d.name, '') AS department_name
FROM people p
LEFT JOIN departments d ON d.id = p.department_id
WHERE p.organisation_id = $1
ORDER BY p.name ASC;

-- name: GetPerson :one
SELECT
    p.id,
    p.organisation_id,
    p.name,
    COALESCE(d.name, '') AS department_name
FROM people p
LEFT JOIN departments d ON d.id = p.department_id
WHERE p.organisation_id = $1 AND p.id = $2;

-- name: ListRoleAssignments :many
SELECT
    r.id,
    r.organisation_id,
    r.person_id,
    p.name AS person_name,
    r.title,
    r.created_at
FROM role_assignments r
JOIN people p ON p.id = r.person_id AND p.organisation_id = r.organisation_id
WHERE r.organisation_id = $1
ORDER BY r.created_at ASC;

-- name: ListRoleTitlesForPerson :many
SELECT id, title
FROM role_assignments
WHERE organisation_id = $1 AND person_id = $2
ORDER BY title ASC;

-- name: InsertRoleAssignment :exec
INSERT INTO role_assignments (
    id,
    organisation_id,
    person_id,
    title,
    created_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: GetDepartment :one
SELECT id, organisation_id, name
FROM departments
WHERE organisation_id = $1 AND id = $2;

-- name: InsertOrganogram :exec
INSERT INTO organograms (id, organisation_id, name, status, created_by_person_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListOrganograms :many
SELECT id, organisation_id, name, status, created_by_person_id, created_at
FROM organograms
WHERE organisation_id = $1
ORDER BY name ASC, id ASC;

-- name: GetOrganogram :one
SELECT id, organisation_id, name, status, created_by_person_id, created_at
FROM organograms
WHERE organisation_id = $1 AND id = $2;

-- name: SetOrganogramStatus :exec
UPDATE organograms
SET status = $3
WHERE organisation_id = $1 AND id = $2;

-- name: InsertPosition :exec
INSERT INTO positions (
    id,
    organisation_id,
    organogram_id,
    department_id,
    title,
    person_id,
    reports_to_position_id,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: ListPositions :many
SELECT id, organisation_id, organogram_id, department_id, title, person_id, reports_to_position_id, created_at
FROM positions
WHERE organisation_id = $1 AND organogram_id = $2
ORDER BY title ASC, id ASC;

-- name: GetPosition :one
SELECT id, organisation_id, organogram_id, department_id, title, person_id, reports_to_position_id, created_at
FROM positions
WHERE organisation_id = $1 AND id = $2;

-- name: UpdatePosition :exec
UPDATE positions
SET department_id = $4,
    title = $5,
    person_id = $6,
    reports_to_position_id = $7
WHERE organisation_id = $1 AND organogram_id = $2 AND id = $3;

-- name: DeletePosition :exec
DELETE FROM positions
WHERE organisation_id = $1 AND organogram_id = $2 AND id = $3;

-- name: InsertOrganogramFile :exec
INSERT INTO organogram_files (
    id,
    organisation_id,
    organogram_id,
    file_name,
    media_type,
    byte_size,
    body,
    status,
    detail,
    created_by_person_id,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: ListOrganogramFiles :many
SELECT id, organisation_id, organogram_id, file_name, media_type, byte_size, status, detail, created_by_person_id, created_at
FROM organogram_files
WHERE organisation_id = $1 AND organogram_id = $2
ORDER BY created_at ASC, id ASC;

-- name: GetOrganogramFile :one
SELECT id, organisation_id, organogram_id, file_name, media_type, byte_size, body, status, detail, created_by_person_id, created_at
FROM organogram_files
WHERE organisation_id = $1 AND organogram_id = $2 AND id = $3;
