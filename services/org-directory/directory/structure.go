package directory

import (
	"context"
	"strings"
	"time"

	"github.com/bicmas-academy/learn/services/org-directory/db"
)

func (m *Memory) PublishOrganogram(_ context.Context, organisationID, organogramID string) (Organogram, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for index, organogram := range m.organograms {
		if organogram.OrganisationID == organisationID && organogram.ID == organogramID {
			organogram.Status = OrganogramPublished
			m.organograms[index] = organogram
			return organogram, nil
		}
	}
	return Organogram{}, ErrNotFound
}

func (m *Memory) UpdatePosition(_ context.Context, organisationID, organogramID, positionID string, input PositionInput) (Position, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return Position{}, ErrNotFound
	}
	title := strings.TrimSpace(input.Title)
	if title == "" || (input.DepartmentID != "" && !m.hasDepartment(organisationID, input.DepartmentID)) || (input.PersonID != "" && !m.hasPerson(organisationID, input.PersonID)) {
		return Position{}, ErrInvalid
	}
	if input.ReportsToPositionID != "" && !m.hasPosition(organisationID, organogramID, input.ReportsToPositionID) {
		return Position{}, ErrInvalid
	}
	if reportingCycle(m.chartPositions(organisationID, organogramID), positionID, input.ReportsToPositionID) {
		return Position{}, ErrCycle
	}
	for index, position := range m.positions {
		if position.OrganisationID != organisationID || position.OrganogramID != organogramID || position.ID != positionID {
			continue
		}
		position.Title = title
		position.DepartmentID = input.DepartmentID
		position.PersonID = input.PersonID
		position.ReportsToPositionID = input.ReportsToPositionID
		m.positions[index] = position
		return position, nil
	}
	return Position{}, ErrNotFound
}

func (m *Memory) DeletePosition(_ context.Context, organisationID, organogramID, positionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return ErrNotFound
	}
	positions := m.chartPositions(organisationID, organogramID)
	found := false
	for _, position := range positions {
		if position.ID == positionID {
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	if positionHasReports(positions, positionID) {
		return ErrHasReports
	}
	kept := make([]Position, 0, len(m.positions))
	for _, position := range m.positions {
		if position.OrganisationID == organisationID && position.ID == positionID {
			continue
		}
		kept = append(kept, position)
	}
	m.positions = kept
	return nil
}

func (m *Memory) ImportOrganogramFile(_ context.Context, organisationID, organogramID, actorID, fileName string, body []byte, mapping SheetMapping, createdAt time.Time) (ImportResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return ImportResult{}, ErrNotFound
	}
	prepared, err := prepareImport(fileName, body, mapping, m.chartPositions(organisationID, organogramID), m.departmentsFor(organisationID))
	if err != nil {
		return ImportResult{}, err
	}
	created, err := m.insertPlanned(organisationID, organogramID, prepared.positions, createdAt)
	if err != nil {
		return ImportResult{}, err
	}
	file, err := m.storeFile(organisationID, organogramID, actorID, fileName, body, prepared, createdAt)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{File: file, Accepted: created, Rejected: prepared.rejected, Headers: prepared.headers}, nil
}

func (m *Memory) ListOrganogramFiles(_ context.Context, organisationID, organogramID string) ([]OrganogramFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasOrganogram(organisationID, organogramID) {
		return nil, ErrNotFound
	}
	files := make([]OrganogramFile, 0)
	for _, file := range m.files {
		if file.OrganisationID == organisationID && file.OrganogramID == organogramID {
			files = append(files, file.OrganogramFile)
		}
	}
	return files, nil
}

func (m *Memory) ReadOrganogramFile(_ context.Context, organisationID, organogramID, fileID string) (StoredOrganogramFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, file := range m.files {
		if file.OrganisationID == organisationID && file.OrganogramID == organogramID && file.ID == fileID {
			copied := file
			copied.Body = append([]byte{}, file.Body...)
			return copied, nil
		}
	}
	return StoredOrganogramFile{}, ErrNotFound
}

func (m *Memory) chartPositions(organisationID, organogramID string) []Position {
	matched := make([]Position, 0)
	for _, position := range m.positions {
		if position.OrganisationID == organisationID && position.OrganogramID == organogramID {
			matched = append(matched, position)
		}
	}
	return matched
}

func (m *Memory) departmentsFor(organisationID string) []Department {
	matched := make([]Department, 0)
	for _, department := range m.departments {
		if department.OrganisationID == organisationID {
			matched = append(matched, department)
		}
	}
	return matched
}

func (m *Memory) insertPlanned(organisationID, organogramID string, planned []plannedPosition, createdAt time.Time) (int, error) {
	existing := titleIndex(m.chartPositions(organisationID, organogramID))
	created := map[string]string{}
	batch := make([]Position, 0, len(planned))
	for _, position := range planned {
		parent, err := resolveParent(existing, created, position)
		if err != nil {
			return 0, err
		}
		id, err := newID()
		if err != nil {
			return 0, err
		}
		batch = append(batch, Position{
			ID:                  id,
			OrganisationID:      organisationID,
			OrganogramID:        organogramID,
			DepartmentID:        position.DepartmentID,
			Title:               position.Title,
			ReportsToPositionID: parent,
			CreatedAt:           createdAt,
		})
		created[strings.ToLower(position.Title)] = id
	}
	m.positions = append(m.positions, batch...)
	return len(batch), nil
}

func (m *Memory) storeFile(organisationID, organogramID, actorID, fileName string, body []byte, prepared preparedImport, createdAt time.Time) (OrganogramFile, error) {
	id, err := newID()
	if err != nil {
		return OrganogramFile{}, err
	}
	file := OrganogramFile{
		ID:                id,
		OrganisationID:    organisationID,
		OrganogramID:      organogramID,
		FileName:          fileName,
		MediaType:         prepared.mediaType,
		ByteSize:          len(body),
		Status:            prepared.status,
		Detail:            prepared.detail,
		CreatedByPersonID: actorID,
		CreatedAt:         createdAt,
	}
	m.files = append(m.files, StoredOrganogramFile{OrganogramFile: file, Body: append([]byte{}, body...)})
	return file, nil
}

func (s SQLStore) PublishOrganogram(ctx context.Context, organisationID, organogramID string) (Organogram, error) {
	row, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID})
	if err != nil {
		return Organogram{}, mapDirectoryWrite(err)
	}
	if err := s.queries.SetOrganogramStatus(ctx, db.SetOrganogramStatusParams{OrganisationID: organisationID, ID: organogramID, Status: OrganogramPublished}); err != nil {
		return Organogram{}, mapDirectoryWrite(err)
	}
	return Organogram{ID: row.ID, OrganisationID: row.OrganisationID, Name: row.Name, Status: OrganogramPublished, CreatedByPersonID: row.CreatedByPersonID, CreatedAt: row.CreatedAt}, nil
}

func (s SQLStore) UpdatePosition(ctx context.Context, organisationID, organogramID, positionID string, input PositionInput) (Position, error) {
	current, err := s.positionInChart(ctx, organisationID, organogramID, positionID)
	if err != nil {
		return Position{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Position{}, ErrInvalid
	}
	if err := s.knownReferences(ctx, organisationID, organogramID, input); err != nil {
		return Position{}, err
	}
	positions, err := s.ListPositions(ctx, organisationID, organogramID)
	if err != nil {
		return Position{}, err
	}
	if reportingCycle(positions, positionID, input.ReportsToPositionID) {
		return Position{}, ErrCycle
	}
	if err := s.queries.UpdatePosition(ctx, db.UpdatePositionParams{
		OrganisationID:      organisationID,
		OrganogramID:        organogramID,
		ID:                  positionID,
		DepartmentID:        nullString(input.DepartmentID),
		Title:               title,
		PersonID:            nullString(input.PersonID),
		ReportsToPositionID: nullString(input.ReportsToPositionID),
	}); err != nil {
		return Position{}, mapDirectoryWrite(err)
	}
	current.Title = title
	current.DepartmentID = input.DepartmentID
	current.PersonID = input.PersonID
	current.ReportsToPositionID = input.ReportsToPositionID
	return current, nil
}

func (s SQLStore) DeletePosition(ctx context.Context, organisationID, organogramID, positionID string) error {
	if _, err := s.positionInChart(ctx, organisationID, organogramID, positionID); err != nil {
		return err
	}
	positions, err := s.ListPositions(ctx, organisationID, organogramID)
	if err != nil {
		return err
	}
	if positionHasReports(positions, positionID) {
		return ErrHasReports
	}
	if err := s.queries.DeletePosition(ctx, db.DeletePositionParams{OrganisationID: organisationID, OrganogramID: organogramID, ID: positionID}); err != nil {
		return mapDirectoryWrite(err)
	}
	return nil
}

func (s SQLStore) ImportOrganogramFile(ctx context.Context, organisationID, organogramID, actorID, fileName string, body []byte, mapping SheetMapping, createdAt time.Time) (ImportResult, error) {
	if _, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID}); err != nil {
		return ImportResult{}, mapDirectoryWrite(err)
	}
	departments, err := s.ListDepartments(ctx, organisationID)
	if err != nil {
		return ImportResult{}, err
	}
	positions, err := s.ListPositions(ctx, organisationID, organogramID)
	if err != nil {
		return ImportResult{}, err
	}
	prepared, err := prepareImport(fileName, body, mapping, positions, departments)
	if err != nil {
		return ImportResult{}, err
	}
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, err
	}
	defer transaction.Rollback()
	queries := s.queries.WithTx(transaction)
	existing := titleIndex(positions)
	created := map[string]string{}
	for _, position := range prepared.positions {
		parent, err := resolveParent(existing, created, position)
		if err != nil {
			return ImportResult{}, err
		}
		id, err := newID()
		if err != nil {
			return ImportResult{}, err
		}
		if err := queries.InsertPosition(ctx, db.InsertPositionParams{
			ID:                  id,
			OrganisationID:      organisationID,
			OrganogramID:        organogramID,
			DepartmentID:        nullString(position.DepartmentID),
			Title:               position.Title,
			PersonID:            nullString(""),
			ReportsToPositionID: nullString(parent),
			CreatedAt:           createdAt,
		}); err != nil {
			return ImportResult{}, mapDirectoryWrite(err)
		}
		created[strings.ToLower(position.Title)] = id
	}
	fileID, err := newID()
	if err != nil {
		return ImportResult{}, err
	}
	file := OrganogramFile{
		ID:                fileID,
		OrganisationID:    organisationID,
		OrganogramID:      organogramID,
		FileName:          fileName,
		MediaType:         prepared.mediaType,
		ByteSize:          len(body),
		Status:            prepared.status,
		Detail:            prepared.detail,
		CreatedByPersonID: actorID,
		CreatedAt:         createdAt,
	}
	if err := queries.InsertOrganogramFile(ctx, db.InsertOrganogramFileParams{
		ID:                file.ID,
		OrganisationID:    file.OrganisationID,
		OrganogramID:      file.OrganogramID,
		FileName:          file.FileName,
		MediaType:         file.MediaType,
		ByteSize:          int32(file.ByteSize),
		Body:              body,
		Status:            file.Status,
		Detail:            file.Detail,
		CreatedByPersonID: file.CreatedByPersonID,
		CreatedAt:         file.CreatedAt,
	}); err != nil {
		return ImportResult{}, mapDirectoryWrite(err)
	}
	if err := transaction.Commit(); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{File: file, Accepted: len(prepared.positions), Rejected: prepared.rejected, Headers: prepared.headers}, nil
}

func (s SQLStore) ListOrganogramFiles(ctx context.Context, organisationID, organogramID string) ([]OrganogramFile, error) {
	if _, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID}); err != nil {
		return nil, mapDirectoryWrite(err)
	}
	rows, err := s.queries.ListOrganogramFiles(ctx, db.ListOrganogramFilesParams{OrganisationID: organisationID, OrganogramID: organogramID})
	if err != nil {
		return nil, err
	}
	files := make([]OrganogramFile, 0, len(rows))
	for _, row := range rows {
		files = append(files, OrganogramFile{
			ID:                row.ID,
			OrganisationID:    row.OrganisationID,
			OrganogramID:      row.OrganogramID,
			FileName:          row.FileName,
			MediaType:         row.MediaType,
			ByteSize:          int(row.ByteSize),
			Status:            row.Status,
			Detail:            row.Detail,
			CreatedByPersonID: row.CreatedByPersonID,
			CreatedAt:         row.CreatedAt,
		})
	}
	return files, nil
}

func (s SQLStore) ReadOrganogramFile(ctx context.Context, organisationID, organogramID, fileID string) (StoredOrganogramFile, error) {
	row, err := s.queries.GetOrganogramFile(ctx, db.GetOrganogramFileParams{OrganisationID: organisationID, OrganogramID: organogramID, ID: fileID})
	if err != nil {
		return StoredOrganogramFile{}, mapDirectoryWrite(err)
	}
	return StoredOrganogramFile{OrganogramFile: OrganogramFile{
		ID:                row.ID,
		OrganisationID:    row.OrganisationID,
		OrganogramID:      row.OrganogramID,
		FileName:          row.FileName,
		MediaType:         row.MediaType,
		ByteSize:          int(row.ByteSize),
		Status:            row.Status,
		Detail:            row.Detail,
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt,
	}, Body: row.Body}, nil
}

func (s SQLStore) positionInChart(ctx context.Context, organisationID, organogramID, positionID string) (Position, error) {
	if _, err := s.queries.GetOrganogram(ctx, db.GetOrganogramParams{OrganisationID: organisationID, ID: organogramID}); err != nil {
		return Position{}, mapDirectoryWrite(err)
	}
	row, err := s.queries.GetPosition(ctx, db.GetPositionParams{OrganisationID: organisationID, ID: positionID})
	if err != nil {
		return Position{}, mapDirectoryWrite(err)
	}
	if row.OrganogramID != organogramID {
		return Position{}, ErrNotFound
	}
	return Position{
		ID:                  row.ID,
		OrganisationID:      row.OrganisationID,
		OrganogramID:        row.OrganogramID,
		DepartmentID:        row.DepartmentID.String,
		Title:               row.Title,
		PersonID:            row.PersonID.String,
		ReportsToPositionID: row.ReportsToPositionID.String,
		CreatedAt:           row.CreatedAt,
	}, nil
}

func (s SQLStore) knownReferences(ctx context.Context, organisationID, organogramID string, input PositionInput) error {
	if input.DepartmentID != "" {
		if _, err := s.queries.GetDepartment(ctx, db.GetDepartmentParams{OrganisationID: organisationID, ID: input.DepartmentID}); err != nil {
			if mapDirectoryWrite(err) == ErrNotFound {
				return ErrInvalid
			}
			return err
		}
	}
	if input.PersonID != "" {
		if _, err := s.queries.GetPerson(ctx, db.GetPersonParams{OrganisationID: organisationID, ID: input.PersonID}); err != nil {
			if mapDirectoryWrite(err) == ErrNotFound {
				return ErrInvalid
			}
			return err
		}
	}
	if input.ReportsToPositionID != "" {
		position, err := s.positionInChart(ctx, organisationID, organogramID, input.ReportsToPositionID)
		if err != nil {
			if err == ErrNotFound {
				return ErrInvalid
			}
			return err
		}
		if position.ID == "" {
			return ErrInvalid
		}
	}
	return nil
}
