-- name: ListInbox :many
SELECT id, organisation_id, person_id, title, body, created_at
FROM inbox_items
WHERE organisation_id = $1 AND person_id = $2
ORDER BY created_at ASC;
