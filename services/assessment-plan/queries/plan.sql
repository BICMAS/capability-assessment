-- name: InsertAssignment :exec
INSERT INTO assignments (
    id,
    organisation_id,
    person_id,
    framework_version_id,
    role_id,
    assigned_by_person_id,
    assigned_at,
    status,
    exception_reason
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: GetAssignment :one
SELECT
    id,
    organisation_id,
    person_id,
    framework_version_id,
    role_id,
    assigned_by_person_id,
    assigned_at,
    status,
    exception_reason
FROM assignments
WHERE organisation_id = $1 AND id = $2;

-- name: GetAssignmentByKey :one
SELECT
    id,
    organisation_id,
    person_id,
    framework_version_id,
    role_id,
    assigned_by_person_id,
    assigned_at,
    status,
    exception_reason
FROM assignments
WHERE organisation_id = $1
  AND person_id = $2
  AND framework_version_id = $3
  AND role_id = $4;

-- name: NextPlanVersion :one
SELECT (COALESCE(MAX(version_number), 0) + 1)::integer AS next_version
FROM plans
WHERE organisation_id = $1 AND assignment_id = $2;

-- name: InsertPlan :exec
INSERT INTO plans (
    id,
    organisation_id,
    assignment_id,
    version_number,
    status
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: GetGeneratedPlan :one
SELECT id, organisation_id, assignment_id, version_number, status
FROM plans
WHERE organisation_id = $1 AND assignment_id = $2 AND status = 'generated';

-- name: GetPlan :one
SELECT id, organisation_id, assignment_id, version_number, status, change_reason, approved_by_person_id, approved_at
FROM plans
WHERE organisation_id = $1 AND id = $2;

-- name: GetPlanForUpdate :one
SELECT id, organisation_id, assignment_id, version_number, status, change_reason, approved_by_person_id, approved_at
FROM plans
WHERE organisation_id = $1 AND id = $2
FOR UPDATE;

-- name: UpdatePlanStatus :exec
UPDATE plans
SET status = $3
WHERE organisation_id = $1 AND id = $2;

-- name: UpdatePlanReview :exec
UPDATE plans
SET status = $3,
    change_reason = $4,
    approved_by_person_id = $5,
    approved_at = $6
WHERE organisation_id = $1 AND id = $2;

-- name: UpdatePlanSkillContent :exec
UPDATE plan_skills
SET recommended_method = $4,
    instructions = $5,
    edited_by_person_id = $6
WHERE organisation_id = $1 AND plan_id = $2 AND id = $3;

-- name: DeleteAssessmentItemsBySkill :exec
DELETE FROM assessment_items
WHERE organisation_id = $1 AND plan_skill_id = $2;

-- name: DeleteRubricBandsBySkill :exec
DELETE FROM rubric_bands
WHERE organisation_id = $1 AND plan_skill_id = $2;

-- name: InsertPlanSkill :exec
INSERT INTO plan_skills (
    id,
    organisation_id,
    plan_id,
    position,
    requirement_id,
    capability_id,
    required_level,
    recommended_method,
    rationale,
    instructions,
    requires_sme_review,
    rubric_needs_review,
    edited_by_person_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: ListPlanSkills :many
SELECT
    id,
    organisation_id,
    plan_id,
    position,
    requirement_id,
    capability_id,
    required_level,
    recommended_method,
    rationale,
    instructions,
    requires_sme_review,
    rubric_needs_review,
    edited_by_person_id
FROM plan_skills
WHERE organisation_id = $1 AND plan_id = $2
ORDER BY position ASC;

-- name: DeleteAssessmentItemsByPlan :exec
DELETE FROM assessment_items
WHERE assessment_items.organisation_id = $1
  AND assessment_items.plan_skill_id IN (
      SELECT plan_skills.id
      FROM plan_skills
      WHERE plan_skills.organisation_id = $1 AND plan_skills.plan_id = $2
  );

-- name: DeleteRubricBandsByPlan :exec
DELETE FROM rubric_bands
WHERE rubric_bands.organisation_id = $1
  AND rubric_bands.plan_skill_id IN (
      SELECT plan_skills.id
      FROM plan_skills
      WHERE plan_skills.organisation_id = $1 AND plan_skills.plan_id = $2
  );

-- name: DeletePlanSkillsByPlan :exec
DELETE FROM plan_skills
WHERE organisation_id = $1 AND plan_id = $2;

-- name: InsertAssessmentItem :exec
INSERT INTO assessment_items (
    id,
    organisation_id,
    plan_skill_id,
    position,
    prompt
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListAssessmentItems :many
SELECT id, prompt
FROM assessment_items
WHERE organisation_id = $1 AND plan_skill_id = $2
ORDER BY position ASC;

-- name: InsertRubricBand :exec
INSERT INTO rubric_bands (
    id,
    organisation_id,
    plan_skill_id,
    level,
    descriptor
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListRubricBands :many
SELECT level, descriptor
FROM rubric_bands
WHERE organisation_id = $1 AND plan_skill_id = $2
ORDER BY level ASC;
