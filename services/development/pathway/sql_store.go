package pathway

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bicmas-academy/learn/services/development/db"
)

type SQLStore struct {
	database *sql.DB
	baseURL  string
}

func NewSQLStore(database *sql.DB, baseURL string) *SQLStore {
	return &SQLStore{database: database, baseURL: baseURL}
}

func (s *SQLStore) MapCourse(ctx context.Context, input MappingInput) (Mapping, error) {
	if err := validateMapping(input); err != nil {
		return Mapping{}, err
	}
	var mapping Mapping
	err := s.withTx(ctx, func(q *db.Queries) error {
		existing, err := q.GetCourseMapping(ctx, db.GetCourseMappingParams{
			OrganisationID: input.OrganisationID,
			CapabilityID:   input.CapabilityID,
			TargetLevel:    int32(input.TargetLevel),
		})
		if err == nil {
			mapping = mappingFromDB(existing)
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertCourseMapping(ctx, db.InsertCourseMappingParams{
			ID:               id,
			OrganisationID:   input.OrganisationID,
			ExternalCourseID: input.ExternalCourseID,
			CapabilityID:     input.CapabilityID,
			TargetLevel:      int32(input.TargetLevel),
			Title:            input.Title,
		}); err != nil {
			if !isUnique(err) {
				return err
			}
		}
		stored, err := q.GetCourseMapping(ctx, db.GetCourseMappingParams{
			OrganisationID: input.OrganisationID,
			CapabilityID:   input.CapabilityID,
			TargetLevel:    int32(input.TargetLevel),
		})
		if err != nil {
			return err
		}
		mapping = mappingFromDB(stored)
		return nil
	})
	return mapping, err
}

func (s *SQLStore) CreatePathway(ctx context.Context, input PathwayInput) (Pathway, error) {
	if err := validatePathway(input); err != nil {
		return Pathway{}, err
	}
	var created Pathway
	err := s.withTx(ctx, func(q *db.Queries) error {
		existing, err := lookupPathway(ctx, q, input.OrganisationID, input.PersonID, input.CapabilityID)
		if err == nil {
			created = existing
			return nil
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		mapping, err := q.GetCourseMapping(ctx, db.GetCourseMappingParams{
			OrganisationID: input.OrganisationID,
			CapabilityID:   input.CapabilityID,
			TargetLevel:    int32(input.TargetLevel),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		built, err := buildPathway(s.baseURL, pathwayKey{
			OrganisationID: input.OrganisationID,
			PersonID:       input.PersonID,
			CapabilityID:   input.CapabilityID,
		}, mappingFromDB(mapping))
		if err != nil {
			return err
		}
		if err := q.InsertPathway(ctx, db.InsertPathwayParams{
			ID:             built.ID,
			OrganisationID: built.OrganisationID,
			PersonID:       built.PersonID,
			CapabilityID:   built.CapabilityID,
			TargetLevel:    int32(built.TargetLevel),
		}); err != nil {
			if !isUnique(err) {
				return err
			}
			created, err = lookupPathway(ctx, q, input.OrganisationID, input.PersonID, input.CapabilityID)
			return err
		}
		for position, item := range built.Items {
			link := sql.NullString{}
			if item.DeepLink != "" {
				link = sql.NullString{String: item.DeepLink, Valid: true}
			}
			if err := q.InsertPlanItem(ctx, db.InsertPlanItemParams{
				ID:             item.ID,
				OrganisationID: built.OrganisationID,
				PathwayID:      built.ID,
				Position:       int32(position + 1),
				Stage:          item.Stage,
				Title:          item.Title,
				Status:         item.Status,
				DeepLink:       link,
			}); err != nil {
				return err
			}
		}
		created = built
		return nil
	})
	return created, err
}

func (s *SQLStore) GetPathway(ctx context.Context, organisationID, personID, capabilityID string) (Pathway, error) {
	return lookupPathway(ctx, db.New(s.database), organisationID, personID, capabilityID)
}

func (s *SQLStore) Complete(ctx context.Context, input CompletionInput) (Pathway, bool, error) {
	if input.OrganisationID == "" || input.PersonID == "" || input.PlanItemID == "" || input.CompletedAt.IsZero() {
		return Pathway{}, false, ErrInvalid
	}
	var pathway Pathway
	created := false
	err := s.withTx(ctx, func(q *db.Queries) error {
		item, err := q.GetPlanItem(ctx, db.GetPlanItemParams{OrganisationID: input.OrganisationID, ID: input.PlanItemID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if item.Stage != StageLearn {
			return ErrInvalid
		}
		row, err := q.GetPathwayByID(ctx, db.GetPathwayByIDParams{OrganisationID: input.OrganisationID, ID: item.PathwayID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		stored, err := pathwayFromRow(ctx, q, row)
		if err != nil {
			return err
		}
		if stored.PersonID != input.PersonID {
			return ErrInvalid
		}
		_, err = q.GetCompletion(ctx, db.GetCompletionParams{OrganisationID: input.OrganisationID, PlanItemID: input.PlanItemID})
		if err == nil {
			pathway = stored
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertCompletion(ctx, db.InsertCompletionParams{
			ID:             id,
			OrganisationID: input.OrganisationID,
			PlanItemID:     input.PlanItemID,
			PersonID:       input.PersonID,
			OccurredAt:     input.CompletedAt.UTC(),
		}); err != nil {
			if !isUnique(err) {
				return err
			}
			pathway, err = lookupPathway(ctx, q, stored.OrganisationID, stored.PersonID, stored.CapabilityID)
			return err
		}
		if err := q.MarkPlanItemComplete(ctx, db.MarkPlanItemCompleteParams{OrganisationID: input.OrganisationID, ID: input.PlanItemID}); err != nil {
			return err
		}
		pathway, err = lookupPathway(ctx, q, stored.OrganisationID, stored.PersonID, stored.CapabilityID)
		created = err == nil
		return err
	})
	return pathway, created, err
}

func (s *SQLStore) withTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(db.New(s.database).WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func lookupPathway(ctx context.Context, q *db.Queries, organisationID, personID, capabilityID string) (Pathway, error) {
	row, err := q.GetPathway(ctx, db.GetPathwayParams{
		OrganisationID: organisationID,
		PersonID:       personID,
		CapabilityID:   capabilityID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Pathway{}, ErrNotFound
	}
	if err != nil {
		return Pathway{}, err
	}
	return pathwayFromRow(ctx, q, row)
}

func pathwayFromRow(ctx context.Context, q *db.Queries, row db.Pathway) (Pathway, error) {
	items, err := q.ListPlanItems(ctx, db.ListPlanItemsParams{OrganisationID: row.OrganisationID, PathwayID: row.ID})
	if err != nil {
		return Pathway{}, err
	}
	return Pathway{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		PersonID:       row.PersonID,
		CapabilityID:   row.CapabilityID,
		TargetLevel:    int(row.TargetLevel),
		Items:          itemsFromRows(items),
	}, nil
}

func itemsFromRows(rows []db.ListPlanItemsRow) []PlanItem {
	items := make([]PlanItem, 0, len(rows))
	for _, row := range rows {
		item := PlanItem{ID: row.ID, Stage: row.Stage, Title: row.Title, Status: row.Status}
		if row.DeepLink.Valid {
			item.DeepLink = row.DeepLink.String
		}
		items = append(items, item)
	}
	return items
}

func mappingFromDB(row db.CourseMapping) Mapping {
	return Mapping{
		ID:               row.ID,
		OrganisationID:   row.OrganisationID,
		ExternalCourseID: row.ExternalCourseID,
		CapabilityID:     row.CapabilityID,
		TargetLevel:      int(row.TargetLevel),
		Title:            row.Title,
	}
}

func buildPathway(baseURL string, key pathwayKey, mapping Mapping) (Pathway, error) {
	memory := NewMemory(baseURL)
	return memory.build(key, mapping)
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
