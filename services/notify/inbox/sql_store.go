package inbox

import (
	"context"

	"github.com/bicmas-academy/learn/services/notify/db"
)

type SQLStore struct {
	queries *db.Queries
}

func NewSQLStore(queries *db.Queries) SQLStore {
	return SQLStore{queries: queries}
}

func (s SQLStore) List(ctx context.Context, organisationID, personID string) ([]Item, error) {
	rows, err := s.queries.ListInbox(ctx, db.ListInboxParams{
		OrganisationID: organisationID,
		PersonID:       personID,
	})
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			ID:             row.ID,
			OrganisationID: row.OrganisationID,
			PersonID:       row.PersonID,
			Title:          row.Title,
			Body:           row.Body,
			CreatedAt:      row.CreatedAt,
		})
	}
	return items, nil
}
