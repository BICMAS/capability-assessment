-- name: InsertCourseMapping :exec
INSERT INTO course_mappings (
    id,
    organisation_id,
    external_course_id,
    capability_id,
    target_level,
    title
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetCourseMapping :one
SELECT id, organisation_id, external_course_id, capability_id, target_level, title
FROM course_mappings
WHERE organisation_id = $1 AND capability_id = $2 AND target_level = $3;

-- name: InsertPathway :exec
INSERT INTO pathways (
    id,
    organisation_id,
    person_id,
    capability_id,
    target_level
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: GetPathwayByID :one
SELECT id, organisation_id, person_id, capability_id, target_level
FROM pathways
WHERE organisation_id = $1 AND id = $2;

-- name: GetPathway :one
SELECT id, organisation_id, person_id, capability_id, target_level
FROM pathways
WHERE organisation_id = $1 AND person_id = $2 AND capability_id = $3;

-- name: InsertPlanItem :exec
INSERT INTO plan_items (
    id,
    organisation_id,
    pathway_id,
    position,
    stage,
    title,
    status,
    deep_link
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: ListPlanItems :many
SELECT id, stage, title, status, deep_link
FROM plan_items
WHERE organisation_id = $1 AND pathway_id = $2
ORDER BY position ASC;

-- name: GetPlanItem :one
SELECT id, organisation_id, pathway_id, stage, title, status
FROM plan_items
WHERE organisation_id = $1 AND id = $2;

-- name: MarkPlanItemComplete :exec
UPDATE plan_items
SET status = 'complete'
WHERE organisation_id = $1 AND id = $2 AND status = 'ready';

-- name: InsertCompletion :exec
INSERT INTO completions (
    id,
    organisation_id,
    plan_item_id,
    person_id,
    occurred_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: GetCompletion :one
SELECT id
FROM completions
WHERE organisation_id = $1 AND plan_item_id = $2;
