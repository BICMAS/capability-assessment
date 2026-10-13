-- name: InsertAttempt :exec
INSERT INTO attempts (
    id,
    organisation_id,
    person_id,
    framework_version_id,
    capability_code,
    source,
    percentage,
    evidence_ids,
    required_level_tenths,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: ListAttempts :many
SELECT
    id,
    source,
    percentage,
    evidence_ids,
    required_level_tenths,
    created_at
FROM attempts
WHERE organisation_id = $1
  AND person_id = $2
  AND framework_version_id = $3
  AND capability_code = $4
ORDER BY created_at ASC, id ASC;

-- name: GetExplanation :one
SELECT
    id,
    organisation_id,
    person_id,
    framework_version_id,
    capability_code,
    knowledge_percentage,
    situational_judgement_percentage,
    ai_interview_percentage,
    simulation_percentage,
    manager_assessment_percentage,
    self_assessment_percentage,
    mean_tenths,
    divisor,
    required_level_tenths,
    current_level_tenths,
    gap_tenths,
    status
FROM explanations
WHERE organisation_id = $1
  AND person_id = $2
  AND framework_version_id = $3
  AND capability_code = $4;

-- name: GetExplanationByID :one
SELECT
    id,
    organisation_id,
    person_id,
    framework_version_id,
    capability_code,
    knowledge_percentage,
    situational_judgement_percentage,
    ai_interview_percentage,
    simulation_percentage,
    manager_assessment_percentage,
    self_assessment_percentage,
    mean_tenths,
    divisor,
    required_level_tenths,
    current_level_tenths,
    gap_tenths,
    status
FROM explanations
WHERE organisation_id = $1 AND id = $2;

-- name: UpsertExplanation :exec
INSERT INTO explanations (
    id,
    organisation_id,
    person_id,
    framework_version_id,
    capability_code,
    knowledge_percentage,
    situational_judgement_percentage,
    ai_interview_percentage,
    simulation_percentage,
    manager_assessment_percentage,
    self_assessment_percentage,
    mean_tenths,
    divisor,
    required_level_tenths,
    current_level_tenths,
    gap_tenths,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
ON CONFLICT (organisation_id, person_id, framework_version_id, capability_code)
DO UPDATE SET
    knowledge_percentage = EXCLUDED.knowledge_percentage,
    situational_judgement_percentage = EXCLUDED.situational_judgement_percentage,
    ai_interview_percentage = EXCLUDED.ai_interview_percentage,
    simulation_percentage = EXCLUDED.simulation_percentage,
    manager_assessment_percentage = EXCLUDED.manager_assessment_percentage,
    self_assessment_percentage = EXCLUDED.self_assessment_percentage,
    mean_tenths = EXCLUDED.mean_tenths,
    divisor = EXCLUDED.divisor,
    required_level_tenths = EXCLUDED.required_level_tenths,
    current_level_tenths = EXCLUDED.current_level_tenths,
    gap_tenths = EXCLUDED.gap_tenths,
    status = EXCLUDED.status;

-- name: InsertOverride :exec
INSERT INTO level_overrides (
    id,
    organisation_id,
    explanation_id,
    actor_person_id,
    reason,
    previous_value_tenths,
    new_value_tenths,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: ListOverrides :many
SELECT
    id,
    actor_person_id,
    reason,
    previous_value_tenths,
    new_value_tenths,
    created_at
FROM level_overrides
WHERE organisation_id = $1 AND explanation_id = $2
ORDER BY created_at ASC, id ASC;
