package directory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bicmas-academy/learn/services/org-directory/db"
)

func (m *Memory) CreateOrganogram(_ context.Context, organisationID, name, actorID string, createdAt time.Time) (Organogram, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name = strings.TrimSpace(name)
	if organisationID == "" || name == "" || actorID == "" {
		return Organogram{}, ErrInvalid
	}
	for _, existing := range m.organograms {
		if existing.OrganisationID == organisationID && existing.Name == name {
			return Organogram{}, ErrDuplicate
		}
	}
	id, err := newID()
	if err != nil {
		return Organogram{}, err
	}
	created := Organogram{ID: id, OrganisationID: organisationID, Name: name, Status: OrganogramDraft, CreatedByPersonID: actorID, CreatedAt: createdAt}
	m.organograms = append(m.organograms, created)
	return created, nil
}

func (m *Memory) ListOrganograms(_ context.Context, organisationID string) ([]Organogram, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]Organogram, 0)
	for _, organogram := range m.organograms {
		if organogram.OrganisationID == organisationID {
			matched = append(matched, organogram)
		}
	}
	return matched, nil
}

func (m *Memory) CreatePosition(_ context.Context, organisationID, organogramID string, input PositionInput, createdAt time.Time) (Position, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return Position{}, ErrNotFound
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Position{}, ErrInvalid
	}
	if input.DepartmentID != "" && !m.hasDepartment(organisationID, input.DepartmentID) {
		return Position{}, ErrInvalid
	}
	if input.PersonID != "" && !m.hasPerson(organisationID, input.PersonID) {
		return Position{}, ErrInvalid
	}
	if input.ReportsToPositionID != "" && !m.hasPosition(organisationID, organogramID, input.ReportsToPositionID) {
		return Position{}, ErrInvalid
	}
	if reportingCycle(m.positions, "", input.ReportsToPositionID) {
		return Position{}, ErrCycle
	}
	id, err := newID()
	if err != nil {
		return Position{}, err
	}
	created := Position{
		ID:                  id,
		OrganisationID:      organisationID,
		OrganogramID:        organogramID,
		DepartmentID:        input.DepartmentID,
		Title:               title,
		PersonID:            input.PersonID,
		ReportsToPositionID: input.ReportsToPositionID,
		CreatedAt:           createdAt,
	}
	m.positions = append(m.positions, created)
	return created, nil
}

func (m *Memory) ListPositions(_ context.Context, organisationID, organogramID string) ([]Position, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return nil, ErrNotFound
	}
	matched := make([]Position, 0)
	for _, position := range m.positions {
		if position.OrganisationID == organisationID && position.OrganogramID == organogramID {
			matched = append(matched, position)
		}
	}
	return matched, nil
}

func (m *Memory) hasOrganogram(organisationID, organogramID string) bool {
	for _, organogram := range m.organograms {
		if organogram.OrganisationID == organisationID && organogram.ID == organogramID {
			return true
		}
	}
	return false
}

func (m *Memory) hasDepartment(organisationID, departmentID string) bool {
	for _, department := range m.departments {
		if department.OrganisationID == organisationID && department.ID == departmentID {
			return true
		}
	}
	return false
}

func (m *Memory) hasPerson(organisationID, personID string) bool {
	for _, person := range m.people {
		if person.OrganisationID == organisationID && person.ID == personID {
			return true
		}
	}
	return false
}

func (m *Memory) hasPosition(organisationID, organogramID, positionID string) bool {
	for _, position := range m.positions {
		if position.OrganisationID == organisationID && position.OrganogramID == organogramID && position.ID == positionID {
			return true
		}
	}
	return false
}

func (s SQLStore) CreateOrganogram(ctx context.Context, organisationID, name, actorID string, createdAt time.Time) (Organogram, error) {
	name = strings.TrimSpace(name)
	if organisationID == "" || name == "" || actorID == "" {
		return Organogram{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Organogram{}, err
	}
	err = s.queries.InsertOrganogram(ctx, db.InsertOrganogramParams{
		ID:                id,
		OrganisationID:    organisationID,
		Name:              name,
		Status:            OrganogramDraft,
		CreatedByPersonID: actorID,
		CreatedAt:         createdAt,
	})
	if err != nil {
		return Organogram{}, mapDirectoryWrite(err)
	}
	return Organogram{ID: id, OrganisationID: organisationID, Name: name, Status: OrganogramDraft, CreatedByPersonID: actorID, CreatedAt: createdAt}, nil
}

func (s SQLStore) ListOrganograms(ctx context.Context, organisationID string) ([]Organogram, error) {
	rows, err := s.queries.ListOrganograms(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	organograms := make([]Organogram, 0, len(rows))
	for _, row := range rows {
		organograms = append(organograms, Organogram{
			ID:                row.ID,
			OrganisationID:    row.OrganisationID,
			Name:              row.Name,
			Status:            row.Status,
			CreatedByPersonID: row.CreatedByPersonID,
			CreatedAt:         row.CreatedAt,
		})
	}
	return organograms, nil
}

func (s SQLStore) CreatePosition(ctx context.Context, organisationID, organogramID string, input PositionInput, createdAt time.Time) (Position, error) {
	if _, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID}); err != nil {
		return Position{}, mapDirectoryWrite(err)
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Position{}, ErrInvalid
	}
	if input.DepartmentID != "" {
		if _, err := s.queries.GetDepartment(ctx, db.GetDepartmentParams{OrganisationID: organisationID, ID: input.DepartmentID}); err != nil {
			if errors.Is(mapDirectoryWrite(err), ErrNotFound) {
				return Position{}, ErrInvalid
			}
			return Position{}, err
		}
	}
	if input.PersonID != "" {
		if _, err := s.queries.GetPerson(ctx, db.GetPersonParams{OrganisationID: organisationID, ID: input.PersonID}); err != nil {
			if errors.Is(mapDirectoryWrite(err), ErrNotFound) {
				return Position{}, ErrInvalid
			}
			return Position{}, err
		}
	}
	if input.ReportsToPositionID != "" {
		position, err := s.queries.GetPosition(ctx, db.GetPositionParams{OrganisationID: organisationID, ID: input.ReportsToPositionID})
		if err != nil {
			if errors.Is(mapDirectoryWrite(err), ErrNotFound) {
				return Position{}, ErrInvalid
			}
			return Position{}, err
		}
		if position.OrganogramID != organogramID {
			return Position{}, ErrInvalid
		}
		positions, err := s.ListPositions(ctx, organisationID, organogramID)
		if err != nil {
			return Position{}, err
		}
		if reportingCycle(positions, "", input.ReportsToPositionID) {
			return Position{}, ErrCycle
		}
	}
	id, err := newID()
	if err != nil {
		return Position{}, err
	}
	created := Position{
		ID:                  id,
		OrganisationID:      organisationID,
		OrganogramID:        organogramID,
		DepartmentID:        input.DepartmentID,
		Title:               title,
		PersonID:            input.PersonID,
		ReportsToPositionID: input.ReportsToPositionID,
		CreatedAt:           createdAt,
	}
	if err := s.queries.InsertPosition(ctx, db.InsertPositionParams{
		ID:                  created.ID,
		OrganisationID:      created.OrganisationID,
		OrganogramID:        created.OrganogramID,
		DepartmentID:        nullString(created.DepartmentID),
		Title:               created.Title,
		PersonID:            nullString(created.PersonID),
		ReportsToPositionID: nullString(created.ReportsToPositionID),
		CreatedAt:           created.CreatedAt,
	}); err != nil {
		return Position{}, mapDirectoryWrite(err)
	}
	return created, nil
}

func (s SQLStore) ListPositions(ctx context.Context, organisationID, organogramID string) ([]Position, error) {
	if _, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID}); err != nil {
		return nil, mapDirectoryWrite(err)
	}
	rows, err := s.queries.ListPositions(ctx, db.ListPositionsParams{OrganisationID: organisationID, OrganogramID: organogramID})
	if err != nil {
		return nil, err
	}
	positions := make([]Position, 0, len(rows))
	for _, row := range rows {
		positions = append(positions, Position{
			ID:                  row.ID,
			OrganisationID:      row.OrganisationID,
			OrganogramID:        row.OrganogramID,
			DepartmentID:        row.DepartmentID.String,
			Title:               row.Title,
			PersonID:            row.PersonID.String,
			ReportsToPositionID: row.ReportsToPositionID.String,
			CreatedAt:           row.CreatedAt,
		})
	}
	return positions, nil
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func mapDirectoryWrite(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrDuplicate
		case "23503":
			return ErrInvalid
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
