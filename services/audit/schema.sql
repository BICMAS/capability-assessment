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
