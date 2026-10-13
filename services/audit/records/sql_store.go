package records

import (
	"context"

	"github.com/bicmas-academy/learn/services/audit/db"
)

type SQLStore struct {
	queries *db.Queries
}

func NewSQLStore(queries *db.Queries) SQLStore {
	return SQLStore{queries: queries}
}

func (s SQLStore) Append(ctx context.Context, record Record) error {
	return s.queries.AppendRecord(ctx, db.AppendRecordParams{
		ActorPersonID:  record.ActorPersonID,
		OrganisationID: record.OrganisationID,
		Action:         record.Action,
		EntityType:     record.EntityType,
		EntityID:       record.EntityID,
		OccurredAt:     record.OccurredAt,
	})
}

func (s SQLStore) List(ctx context.Context, organisationID string) ([]Record, error) {
	rows, err := s.queries.ListRecordsByOrganisation(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, Record{
			ActorPersonID:  row.ActorPersonID,
			OrganisationID: row.OrganisationID,
			Action:         row.Action,
			EntityType:     row.EntityType,
			EntityID:       row.EntityID,
			OccurredAt:     row.OccurredAt,
		})
	}
	return records, nil
}
