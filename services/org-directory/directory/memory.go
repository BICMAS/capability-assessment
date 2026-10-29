package directory

import (
	"context"
	"sync"
	"time"
)

type Memory struct {
	mu          sync.Mutex
	departments []Department
	people      []Person
	assignments []RoleAssignment
	organograms []Organogram
	positions   []Position
	files       []StoredOrganogramFile
}

func (m *Memory) ListDepartments(_ context.Context, organisationID string) ([]Department, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]Department, 0)
	for _, department := range m.departments {
		if department.OrganisationID == organisationID {
			matched = append(matched, department)
		}
	}
	return matched, nil
}

func (m *Memory) ListPeople(_ context.Context, organisationID string) ([]Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]Person, 0)
	for _, person := range m.people {
		if person.OrganisationID == organisationID {
			matched = append(matched, person)
		}
	}
	return matched, nil
}

func (m *Memory) ListRoleAssignments(_ context.Context, organisationID string) ([]RoleAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]RoleAssignment, 0)
	for _, assignment := range m.assignments {
		if assignment.OrganisationID == organisationID {
			matched = append(matched, assignment)
		}
	}
	return matched, nil
}

func (m *Memory) Profile(_ context.Context, organisationID, personID string) (Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.profileLocked(organisationID, personID)
}

func (m *Memory) profileLocked(organisationID, personID string) (Profile, error) {
	for _, person := range m.people {
		if person.OrganisationID != organisationID || person.ID != personID {
			continue
		}
		roles := make([]RoleTitle, 0)
		for _, assignment := range m.assignments {
			if assignment.OrganisationID == organisationID && assignment.PersonID == personID {
				roles = append(roles, RoleTitle{ID: assignment.ID, Title: assignment.Title})
			}
		}
		return Profile{Person: person, Roles: roles}, nil
	}
	return Profile{}, ErrNotFound
}

func (m *Memory) CreateRoleAssignment(_ context.Context, organisationID, personID, title string, createdAt time.Time) (RoleAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	profile, err := m.profileLocked(organisationID, personID)
	if err != nil {
		return RoleAssignment{}, err
	}
	id, err := newID()
	if err != nil {
		return RoleAssignment{}, err
	}
	assignment := RoleAssignment{
		ID:             id,
		OrganisationID: organisationID,
		PersonID:       personID,
		PersonName:     profile.Name,
		Title:          title,
		CreatedAt:      createdAt,
	}
	m.assignments = append(m.assignments, assignment)
	return assignment, nil
}
