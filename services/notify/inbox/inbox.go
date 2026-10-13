package inbox

import (
	"context"
	"time"
)

type Item struct {
	ID             string
	OrganisationID string
	PersonID       string
	Title          string
	Body           string
	CreatedAt      time.Time
}

type Store interface {
	List(ctx context.Context, organisationID, personID string) ([]Item, error)
}
