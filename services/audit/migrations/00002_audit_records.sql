-- +goose Up
CREATE TABLE audit_records (
    actor_person_id text NOT NULL,
    organisation_id text NOT NULL,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    occurred_at timestamptz NOT NULL,
    PRIMARY KEY (
        actor_person_id,
        organisation_id,
        action,
        entity_type,
        entity_id,
        occurred_at
    )
);

CREATE INDEX audit_records_by_organisation ON audit_records (organisation_id, occurred_at);

-- +goose StatementBegin
CREATE FUNCTION audit_records_reject_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit records are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_records_no_update
BEFORE UPDATE ON audit_records
FOR EACH ROW
EXECUTE FUNCTION audit_records_reject_mutation();

CREATE TRIGGER audit_records_no_delete
BEFORE DELETE ON audit_records
FOR EACH ROW
EXECUTE FUNCTION audit_records_reject_mutation();

-- +goose Down
DROP TRIGGER audit_records_no_update ON audit_records;
DROP TRIGGER audit_records_no_delete ON audit_records;
DROP FUNCTION audit_records_reject_mutation();
DROP TABLE audit_records;
