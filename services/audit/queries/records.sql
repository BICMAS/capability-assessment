-- name: AppendRecord :exec
INSERT INTO audit_records (
    actor_person_id,
    organisation_id,
    action,
    entity_type,
    entity_id,
    occurred_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (
    actor_person_id,
    organisation_id,
    action,
    entity_type,
    entity_id,
    occurred_at
) DO NOTHING;

-- name: ListRecordsByOrganisation :many
SELECT
    actor_person_id,
    organisation_id,
    action,
    entity_type,
    entity_id,
    occurred_at
FROM audit_records
WHERE organisation_id = $1
ORDER BY occurred_at ASC;
