-- name: InsertFramework :exec
INSERT INTO frameworks (id, organisation_id, name, created_at)
VALUES ($1, $2, $3, $4);

-- name: InsertFrameworkVersion :exec
INSERT INTO framework_versions (
    id,
    organisation_id,
    framework_id,
    version_number,
    status,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: MaxVersionNumber :one
SELECT COALESCE(MAX(version_number), 0)::int AS version_number
FROM framework_versions
WHERE organisation_id = $1 AND framework_id = $2;

-- name: GetFrameworkVersion :one
SELECT
    v.id,
    v.organisation_id,
    v.framework_id,
    f.name AS framework_name,
    v.version_number,
    v.status
FROM framework_versions v
JOIN frameworks f ON f.id = v.framework_id AND f.organisation_id = v.organisation_id
WHERE v.organisation_id = $1 AND v.id = $2;

-- name: ListFrameworkVersions :many
SELECT
    v.id,
    v.organisation_id,
    v.framework_id,
    f.name AS framework_name,
    v.version_number,
    v.status
FROM framework_versions v
JOIN frameworks f ON f.id = v.framework_id AND f.organisation_id = v.organisation_id
WHERE v.organisation_id = $1
ORDER BY v.version_number ASC;

-- name: PublishVersion :execrows
UPDATE framework_versions
SET status = 'published'
WHERE id = $1
  AND organisation_id = $2
  AND status IN ('draft', 'in_review', 'approved');

-- name: InsertRole :exec
INSERT INTO roles (id, organisation_id, framework_version_id, name)
VALUES ($1, $2, $3, $4);

-- name: ListRolesForVersion :many
SELECT id, name
FROM roles
WHERE organisation_id = $1 AND framework_version_id = $2
ORDER BY name ASC;

-- name: RenameRole :exec
UPDATE roles AS r
SET name = $1
WHERE r.id = $2
  AND r.organisation_id = $3
  AND r.framework_version_id = $4
  AND EXISTS (
      SELECT 1
      FROM framework_versions v
      WHERE v.id = r.framework_version_id
        AND v.organisation_id = r.organisation_id
        AND v.status IN ('draft', 'in_review', 'approved')
  );

-- name: InsertCapability :exec
INSERT INTO capabilities (id, organisation_id, framework_version_id, name)
VALUES ($1, $2, $3, $4);

-- name: ListCapabilitiesForVersion :many
SELECT id, name
FROM capabilities
WHERE organisation_id = $1 AND framework_version_id = $2
ORDER BY name ASC;

-- name: GetProficiencyLevel :one
SELECT id, capability_id, level
FROM proficiency_levels
WHERE organisation_id = $1 AND capability_id = $2 AND level = $3;

-- name: InsertProficiencyLevel :exec
INSERT INTO proficiency_levels (id, organisation_id, capability_id, level)
VALUES ($1, $2, $3, $4);

-- name: ListProficiencyLevelsForVersion :many
SELECT pl.id, pl.capability_id, pl.level
FROM proficiency_levels pl
JOIN capabilities c ON c.id = pl.capability_id AND c.organisation_id = pl.organisation_id
WHERE pl.organisation_id = $1 AND c.framework_version_id = $2
ORDER BY pl.level ASC;

-- name: InsertBehaviouralIndicator :exec
INSERT INTO behavioural_indicators (id, organisation_id, proficiency_level_id, statement)
VALUES ($1, $2, $3, $4);

-- name: UpdateBehaviouralIndicator :exec
UPDATE behavioural_indicators
SET statement = $1
WHERE id = $2
  AND organisation_id = $3;

-- name: MarkVersionApproved :exec
UPDATE framework_versions
SET status = 'approved'
WHERE id = $1
  AND organisation_id = $2
  AND status IN ('draft', 'in_review');

-- name: ListBehaviouralIndicatorsForVersion :many
SELECT bi.id, bi.proficiency_level_id, bi.statement
FROM behavioural_indicators bi
JOIN proficiency_levels pl ON pl.id = bi.proficiency_level_id AND pl.organisation_id = bi.organisation_id
JOIN capabilities c ON c.id = pl.capability_id AND c.organisation_id = pl.organisation_id
WHERE bi.organisation_id = $1 AND c.framework_version_id = $2
ORDER BY bi.statement ASC;

-- name: InsertRoleCapabilityRequirement :exec
INSERT INTO role_capability_requirements (
    id,
    organisation_id,
    framework_version_id,
    role_id,
    capability_id,
    required_level
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: ListRequirementsForVersion :many
SELECT id, role_id, capability_id, required_level
FROM role_capability_requirements
WHERE organisation_id = $1 AND framework_version_id = $2
ORDER BY required_level ASC;
