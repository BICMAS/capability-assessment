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
