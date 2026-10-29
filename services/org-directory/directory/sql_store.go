package directory

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bicmas-academy/learn/services/org-directory/db"
)

type SQLStore struct {
	database *sql.DB
	queries  *db.Queries
}

func NewSQLStore(database *sql.DB) SQLStore {
	return SQLStore{database: database, queries: db.New(database)}
}

func (s SQLStore) ListDepartments(ctx context.Context, organisationID string) ([]Department, error) {
	rows, err := s.queries.ListDepartments(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	departments := make([]Department, 0, len(rows))
	for _, row := range rows {
		departments = append(departments, Department{
			ID:             row.ID,
			OrganisationID: row.OrganisationID,
			Name:           row.Name,
		})
	}
	return departments, nil
}

func (s SQLStore) ListPeople(ctx context.Context, organisationID string) ([]Person, error) {
	rows, err := s.queries.ListPeople(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(rows))
	for _, row := range rows {
		people = append(people, Person{
			ID:             row.ID,
			OrganisationID: row.OrganisationID,
			Name:           row.Name,
			DepartmentName: row.DepartmentName,
		})
	}
	return people, nil
}

func (s SQLStore) ListRoleAssignments(ctx context.Context, organisationID string) ([]RoleAssignment, error) {
	rows, err := s.queries.ListRoleAssignments(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	assignments := make([]RoleAssignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, RoleAssignment{
			ID:             row.ID,
			OrganisationID: row.OrganisationID,
			PersonID:       row.PersonID,
			PersonName:     row.PersonName,
			Title:          row.Title,
			CreatedAt:      row.CreatedAt,
		})
	}
	return assignments, nil
}

func (s SQLStore) Profile(ctx context.Context, organisationID, personID string) (Profile, error) {
	person, err := s.queries.GetPerson(ctx, db.GetPersonParams{
		OrganisationID: organisationID,
		ID:             personID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	titles, err := s.queries.ListRoleTitlesForPerson(ctx, db.ListRoleTitlesForPersonParams{
		OrganisationID: organisationID,
		PersonID:       personID,
	})
	if err != nil {
		return Profile{}, err
	}
	roles := make([]RoleTitle, 0, len(titles))
	for _, title := range titles {
		roles = append(roles, RoleTitle{ID: title.ID, Title: title.Title})
	}
	return Profile{
		Person: Person{
			ID:             person.ID,
			OrganisationID: person.OrganisationID,
			Name:           person.Name,
			DepartmentName: person.DepartmentName,
		},
		Roles: roles,
	}, nil
}

func (s SQLStore) CreateRoleAssignment(ctx context.Context, organisationID, personID, title string, createdAt time.Time) (RoleAssignment, error) {
	profile, err := s.Profile(ctx, organisationID, personID)
	if err != nil {
		return RoleAssignment{}, err
	}
	id, err := newID()
	if err != nil {
		return RoleAssignment{}, err
	}
	if err := s.queries.InsertRoleAssignment(ctx, db.InsertRoleAssignmentParams{
		ID:             id,
		OrganisationID: organisationID,
		PersonID:       personID,
		Title:          title,
		CreatedAt:      createdAt,
	}); err != nil {
		return RoleAssignment{}, err
	}
	return RoleAssignment{
		ID:             id,
		OrganisationID: organisationID,
		PersonID:       personID,
		PersonName:     profile.Name,
		Title:          title,
		CreatedAt:      createdAt,
	}, nil
}
