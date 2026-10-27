-- name: InsertProfileRole :exec
INSERT INTO profile_roles (id, organisation_id, name, created_at)
VALUES ($1, $2, $3, $4);

-- name: GetProfileRole :one
SELECT id, organisation_id, name, created_at
FROM profile_roles
WHERE organisation_id = $1 AND id = $2;

-- name: InsertCanonicalSkill :exec
INSERT INTO canonical_skills (id, organisation_id, name, category, definition, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetCanonicalSkill :one
SELECT id, organisation_id, name, category, definition, created_at
FROM canonical_skills
WHERE organisation_id = $1 AND id = $2;

-- name: InsertRoleProfile :exec
INSERT INTO role_profiles (id, organisation_id, role_id, created_by_person_id, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRoleProfile :one
SELECT
    p.id,
    p.organisation_id,
    p.role_id,
    r.name AS role_name,
    p.created_by_person_id,
    p.created_at
FROM role_profiles p
JOIN profile_roles r ON r.id = p.role_id AND r.organisation_id = p.organisation_id
WHERE p.organisation_id = $1 AND p.id = $2;

-- name: InsertRoleProfileVersion :exec
INSERT INTO role_profile_versions (
    id,
    organisation_id,
    role_profile_id,
    version_number,
    revision,
    status,
    purpose,
    responsibilities,
    business_outcomes,
    decision_scope,
    industry,
    region,
    team_size,
    reporting_level,
    framework_version_id,
    ai_draft,
    ai_model,
    ai_source_context_version,
    ai_generated_at,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
);

-- name: GetRoleProfileVersion :one
SELECT
    id,
    organisation_id,
    role_profile_id,
    version_number,
    revision,
    status,
    purpose,
    responsibilities,
    business_outcomes,
    decision_scope,
    industry,
    region,
    team_size,
    reporting_level,
    framework_version_id,
    ai_draft,
    ai_model,
    ai_source_context_version,
    ai_generated_at,
    created_at
FROM role_profile_versions
WHERE organisation_id = $1 AND id = $2;

-- name: ListRoleProfileVersionIDs :many
SELECT id
FROM role_profile_versions
WHERE organisation_id = $1 AND role_profile_id = $2
ORDER BY version_number ASC;

-- name: LatestRoleProfileVersionID :one
SELECT id
FROM role_profile_versions
WHERE organisation_id = $1 AND role_profile_id = $2
ORDER BY version_number DESC
LIMIT 1;

-- name: MaxRoleProfileVersionNumber :one
SELECT COALESCE(MAX(version_number), 0)::int AS version_number
FROM role_profile_versions
WHERE organisation_id = $1 AND role_profile_id = $2;

-- name: UpdateRoleProfileContext :execrows
UPDATE role_profile_versions
SET
    purpose = $3,
    responsibilities = $4,
    business_outcomes = $5,
    decision_scope = $6,
    industry = $7,
    region = $8,
    team_size = $9,
    reporting_level = $10,
    revision = revision + 1
WHERE id = $1
  AND organisation_id = $2
  AND revision = $11
  AND status = 'profile_draft';

-- name: TouchRoleProfileDraft :execrows
UPDATE role_profile_versions
SET revision = revision + 1
WHERE id = $1
  AND organisation_id = $2
  AND revision = $3
  AND status = 'profile_draft';

-- name: TransitionRoleProfileVersion :execrows
UPDATE role_profile_versions
SET status = $4, revision = revision + 1
WHERE id = $1
  AND organisation_id = $2
  AND revision = $3
  AND status = $5;

-- name: SetRoleProfileAIDraft :execrows
UPDATE role_profile_versions
SET
    purpose = $3,
    ai_draft = $4,
    ai_model = $5,
    ai_source_context_version = $6,
    ai_generated_at = $7
WHERE id = $1
  AND organisation_id = $2
  AND status = 'profile_draft'
  AND purpose = ''
  AND ai_draft = '';

-- name: DeleteRoleProfileSources :exec
DELETE FROM role_profile_sources
WHERE organisation_id = $1 AND profile_version_id = $2;

-- name: InsertRoleProfileSource :exec
INSERT INTO role_profile_sources (id, organisation_id, profile_version_id, title, reference_uri)
VALUES ($1, $2, $3, $4, $5);

-- name: ListRoleProfileSources :many
SELECT id, title, reference_uri
FROM role_profile_sources
WHERE organisation_id = $1 AND profile_version_id = $2
ORDER BY title ASC, id ASC;

-- name: InsertRoleProfileSkill :exec
INSERT INTO role_profile_skills (
    id,
    organisation_id,
    profile_version_id,
    skill_id,
    category,
    definition,
    required_proficiency,
    priority,
    rationale,
    provenance,
    position
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: ListRoleProfileSkills :many
SELECT
    a.id,
    a.skill_id,
    s.name AS skill_name,
    a.category,
    a.definition,
    a.required_proficiency,
    a.priority,
    a.rationale,
    a.provenance,
    a.position
FROM role_profile_skills a
JOIN canonical_skills s ON s.id = a.skill_id AND s.organisation_id = a.organisation_id
WHERE a.organisation_id = $1 AND a.profile_version_id = $2
ORDER BY a.position ASC;

-- name: UpdateRoleProfileSkill :exec
UPDATE role_profile_skills
SET
    definition = $4,
    required_proficiency = $5,
    priority = $6,
    rationale = $7,
    provenance = $8
WHERE id = $1
  AND organisation_id = $2
  AND profile_version_id = $3;

-- name: DeleteRoleProfileSkill :execrows
DELETE FROM role_profile_skills
WHERE id = $1 AND organisation_id = $2 AND profile_version_id = $3;

-- name: UpdateRoleProfileSkillPosition :exec
UPDATE role_profile_skills
SET position = $4
WHERE id = $1 AND organisation_id = $2 AND profile_version_id = $3;

-- name: InsertRoleProfileComment :exec
INSERT INTO role_profile_comments (id, organisation_id, profile_version_id, author_person_id, body, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListRoleProfileComments :many
SELECT c.id, c.profile_version_id, c.author_person_id, c.body, c.created_at
FROM role_profile_comments c
JOIN role_profile_versions v ON v.id = c.profile_version_id AND v.organisation_id = c.organisation_id
WHERE c.organisation_id = $1 AND v.role_profile_id = $2
ORDER BY c.created_at ASC, c.id ASC;

-- name: InsertRoleProfileAudit :exec
INSERT INTO role_profile_audit (
    id,
    organisation_id,
    role_profile_id,
    profile_version_id,
    actor_person_id,
    action,
    occurred_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: ListRoleProfileAudit :many
SELECT id, profile_version_id, actor_person_id, action, occurred_at
FROM role_profile_audit
WHERE organisation_id = $1 AND role_profile_id = $2
ORDER BY occurred_at ASC, id ASC;
