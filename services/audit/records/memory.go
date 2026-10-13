package records

import (
	"context"
	"slices"
	"sync"
)

type Memory struct {
	mu      sync.Mutex
	records []Record
}

func (m *Memory) Append(_ context.Context, record Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if slices.ContainsFunc(m.records, func(existing Record) bool {
		return existing.ActorPersonID == record.ActorPersonID &&
			existing.OrganisationID == record.OrganisationID &&
			existing.Action == record.Action &&
			existing.EntityType == record.EntityType &&
			existing.EntityID == record.EntityID &&
			existing.OccurredAt.Equal(record.OccurredAt)
	}) {
		return nil
	}
	m.records = append(m.records, record)
	return nil
}

func (m *Memory) List(_ context.Context, organisationID string) ([]Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]Record, 0)
	for _, record := range m.records {
		if record.OrganisationID == organisationID {
			matched = append(matched, record)
		}
	}
	return matched, nil
}
