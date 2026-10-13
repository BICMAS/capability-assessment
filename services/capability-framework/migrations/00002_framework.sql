-- +goose Up
CREATE TABLE frameworks (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX frameworks_by_organisation ON frameworks (organisation_id, name);

CREATE TABLE framework_versions (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_id text NOT NULL REFERENCES frameworks (id),
    version_number integer NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (framework_id, version_number),
    CHECK (status IN ('draft', 'in_review', 'approved', 'published', 'retired')),
    CHECK (version_number > 0)
);

CREATE INDEX framework_versions_by_organisation ON framework_versions (organisation_id, framework_id);

CREATE TABLE roles (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    name text NOT NULL
);

CREATE INDEX roles_by_version ON roles (organisation_id, framework_version_id, name);

CREATE TABLE capabilities (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    name text NOT NULL
);

CREATE INDEX capabilities_by_version ON capabilities (organisation_id, framework_version_id, name);

CREATE TABLE proficiency_levels (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    capability_id text NOT NULL REFERENCES capabilities (id),
    level integer NOT NULL,
    UNIQUE (capability_id, level),
    CHECK (level >= 0 AND level <= 4)
);

CREATE TABLE behavioural_indicators (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    proficiency_level_id text NOT NULL REFERENCES proficiency_levels (id),
    statement text NOT NULL
);

CREATE INDEX behavioural_indicators_by_level ON behavioural_indicators (organisation_id, proficiency_level_id);

CREATE TABLE role_capability_requirements (
    id text PRIMARY KEY,
    organisation_id text NOT NULL,
    framework_version_id text NOT NULL REFERENCES framework_versions (id),
    role_id text NOT NULL REFERENCES roles (id),
    capability_id text NOT NULL REFERENCES capabilities (id),
    required_level integer NOT NULL,
    UNIQUE (role_id, capability_id),
    CHECK (required_level >= 0 AND required_level <= 4)
);

CREATE INDEX requirements_by_version ON role_capability_requirements (organisation_id, framework_version_id);

-- +goose StatementBegin
CREATE FUNCTION reject_published_framework_change() RETURNS trigger AS $$
DECLARE
    version_status text;
BEGIN
    IF TG_TABLE_NAME = 'framework_versions' THEN
        IF OLD.status = 'published' THEN
            RAISE EXCEPTION 'published version cannot be changed';
        END IF;
    ELSIF TG_TABLE_NAME = 'roles' OR TG_TABLE_NAME = 'capabilities' THEN
        SELECT status INTO version_status FROM framework_versions WHERE id = OLD.framework_version_id;
    ELSIF TG_TABLE_NAME = 'role_capability_requirements' THEN
        SELECT status INTO version_status FROM framework_versions WHERE id = OLD.framework_version_id;
    ELSIF TG_TABLE_NAME = 'proficiency_levels' THEN
        SELECT v.status INTO version_status
        FROM capabilities c
        JOIN framework_versions v ON v.id = c.framework_version_id
        WHERE c.id = OLD.capability_id;
    ELSIF TG_TABLE_NAME = 'behavioural_indicators' THEN
        SELECT v.status INTO version_status
        FROM proficiency_levels pl
        JOIN capabilities c ON c.id = pl.capability_id
        JOIN framework_versions v ON v.id = c.framework_version_id
        WHERE pl.id = OLD.proficiency_level_id;
    END IF;

    IF version_status = 'published' THEN
        RAISE EXCEPTION 'published version cannot be changed';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER framework_versions_reject_published_change
    BEFORE UPDATE OR DELETE ON framework_versions
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

CREATE TRIGGER roles_reject_published_change
    BEFORE UPDATE OR DELETE ON roles
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

CREATE TRIGGER capabilities_reject_published_change
    BEFORE UPDATE OR DELETE ON capabilities
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

CREATE TRIGGER proficiency_levels_reject_published_change
    BEFORE UPDATE OR DELETE ON proficiency_levels
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

CREATE TRIGGER behavioural_indicators_reject_published_change
    BEFORE UPDATE OR DELETE ON behavioural_indicators
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

CREATE TRIGGER requirements_reject_published_change
    BEFORE UPDATE OR DELETE ON role_capability_requirements
    FOR EACH ROW EXECUTE FUNCTION reject_published_framework_change();

-- +goose Down
DROP TRIGGER requirements_reject_published_change ON role_capability_requirements;
DROP TRIGGER behavioural_indicators_reject_published_change ON behavioural_indicators;
DROP TRIGGER proficiency_levels_reject_published_change ON proficiency_levels;
DROP TRIGGER capabilities_reject_published_change ON capabilities;
DROP TRIGGER roles_reject_published_change ON roles;
DROP TRIGGER framework_versions_reject_published_change ON framework_versions;
DROP FUNCTION reject_published_framework_change();
DROP TABLE role_capability_requirements;
DROP TABLE behavioural_indicators;
DROP TABLE proficiency_levels;
DROP TABLE capabilities;
DROP TABLE roles;
DROP TABLE framework_versions;
DROP TABLE frameworks;
