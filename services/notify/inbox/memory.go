package inbox

import "context"

type Memory struct {
	items []Item
}

func (m *Memory) List(_ context.Context, organisationID, personID string) ([]Item, error) {
	matched := make([]Item, 0)
	for _, item := range m.items {
		if item.OrganisationID == organisationID && item.PersonID == personID {
			matched = append(matched, item)
		}
	}
	return matched, nil
}
