-- name: InsertFrameworkImport :exec
INSERT INTO framework_imports (
    id,
    organisation_id,
    framework_version_id,
    object_key,
    created_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: InsertImportRow :exec
INSERT INTO import_rows (
    id,
    organisation_id,
    import_id,
    framework_version_id,
    row_number,
    framework_version_label,
    department,
    role_code,
    role_name,
    capability_code,
    capability_name,
    capability_type,
    definition,
    required_level,
    proficiency_levels,
    behavioural_indicators,
    assessment_methods,
    learning_asset_ids,
    practice_scenario_ids,
    workplace_mission,
    weight,
    governance_status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
);

-- name: InsertImportRowError :exec
INSERT INTO import_row_errors (
    id,
    organisation_id,
    framework_version_id,
    row_number,
    message
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListImportRowErrorsForVersion :many
SELECT row_number, message
FROM import_row_errors
WHERE organisation_id = $1 AND framework_version_id = $2
ORDER BY row_number ASC, message ASC;

-- name: InsertReviewProposal :exec
INSERT INTO review_proposals (
    id,
    organisation_id,
    framework_version_id,
    kind,
    status,
    capability_code,
    message,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: ListReviewProposalsForVersion :many
SELECT id, kind, status, capability_code, message
FROM review_proposals
WHERE organisation_id = $1 AND framework_version_id = $2
ORDER BY created_at ASC;

-- name: GetReviewProposal :one
SELECT id, kind, status, capability_code, message
FROM review_proposals
WHERE organisation_id = $1 AND framework_version_id = $2 AND id = $3;

-- name: DecideReviewProposal :exec
UPDATE review_proposals
SET status = $1, message = $2
WHERE id = $3
  AND organisation_id = $4
  AND framework_version_id = $5
  AND status = 'pending';

-- name: MarkVersionInReview :exec
UPDATE framework_versions
SET status = 'in_review'
WHERE id = $1
  AND organisation_id = $2
  AND status = 'draft';
