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
    status,
    prompt_version,
    model_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetGeneratedPlan :one
SELECT id, organisation_id, assignment_id, version_number, status
FROM plans
WHERE organisation_id = $1 AND assignment_id = $2 AND status = 'generated';

-- name: GetPlan :one
SELECT id, organisation_id, assignment_id, version_number, status, change_reason, approved_by_person_id, approved_at, prompt_version, model_name
FROM plans
WHERE organisation_id = $1 AND id = $2;

-- name: GetPlanForUpdate :one
SELECT id, organisation_id, assignment_id, version_number, status, change_reason, approved_by_person_id, approved_at, prompt_version, model_name
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

-- name: InsertSession :exec
INSERT INTO assessment_sessions (
    id,
    organisation_id,
    plan_id,
    person_id,
    status
) VALUES (
    $1, $2, $3, $4, 'assigned'
);

-- name: GetSession :one
SELECT id, organisation_id, plan_id, person_id, status, started_at, submitted_at, confirmed_at
FROM assessment_sessions
WHERE organisation_id = $1 AND id = $2;

-- name: GetSessionByPlan :one
SELECT id, organisation_id, plan_id, person_id, status, started_at, submitted_at, confirmed_at
FROM assessment_sessions
WHERE organisation_id = $1 AND plan_id = $2;

-- name: ListSessionsByPerson :many
SELECT id, organisation_id, plan_id, person_id, status, started_at, submitted_at, confirmed_at
FROM assessment_sessions
WHERE organisation_id = $1 AND person_id = $2
ORDER BY id ASC;

-- name: ConfirmSession :exec
UPDATE assessment_sessions
SET confirmed_at = $3
WHERE organisation_id = $1 AND id = $2 AND status = 'submitted' AND confirmed_at IS NULL;

-- name: InsertSessionProposal :exec
INSERT INTO session_proposals (
    id,
    organisation_id,
    session_id,
    plan_skill_id,
    requirement_id,
    capability_id,
    required_level,
    recommended_method,
    evidence_source,
    band_levels,
    percentage
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: ListSessionProposals :many
SELECT plan_skill_id, requirement_id, capability_id, required_level, recommended_method, evidence_source, band_levels, percentage
FROM session_proposals
WHERE organisation_id = $1 AND session_id = $2
ORDER BY id ASC;

-- name: UpdateSessionProgress :exec
UPDATE assessment_sessions
SET status = $3,
    started_at = $4,
    submitted_at = $5
WHERE organisation_id = $1 AND id = $2;

-- name: InsertSessionAnswer :exec
INSERT INTO session_answers (
    id,
    organisation_id,
    session_id,
    assessment_item_id,
    answer
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListSessionAnswers :many
SELECT assessment_item_id, answer
FROM session_answers
WHERE organisation_id = $1 AND session_id = $2;

-- name: UpdateSessionAnswer :exec
UPDATE session_answers
SET answer = $4
WHERE organisation_id = $1 AND session_id = $2 AND assessment_item_id = $3;

-- name: ListPlanItems :many
SELECT
    plan_skills.id AS skill_id,
    plan_skills.requirement_id,
    plan_skills.capability_id,
    plan_skills.required_level,
    plan_skills.recommended_method,
    plan_skills.instructions,
    assessment_items.id AS item_id,
    assessment_items.prompt
FROM assessment_items
JOIN plan_skills ON plan_skills.id = assessment_items.plan_skill_id
    AND plan_skills.organisation_id = assessment_items.organisation_id
WHERE assessment_items.organisation_id = $1
  AND plan_skills.plan_id = $2
ORDER BY plan_skills.position ASC, assessment_items.position ASC;
