package plan

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bicmas-academy/learn/services/assessment-plan/db"
)

type SQLStore struct {
	database *sql.DB
}

func NewSQLStore(database *sql.DB) *SQLStore {
	return &SQLStore{database: database}
}

func (s *SQLStore) CreateAssignment(ctx context.Context, input AssignmentInput) (Assignment, error) {
	input = trimAssignment(input)
	if err := validateAssignment(input); err != nil {
		return Assignment{}, err
	}
	var row Assignment
	err := s.withTx(ctx, func(q *db.Queries) error {
		existing, err := q.GetAssignmentByKey(ctx, db.GetAssignmentByKeyParams{
			OrganisationID:     input.OrganisationID,
			PersonID:           input.PersonID,
			FrameworkVersionID: input.FrameworkVersionID,
			RoleID:             input.RoleID,
		})
		if err == nil {
			row = assignmentFromDB(existing)
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		reason := sql.NullString{}
		if input.ExceptionReason != "" {
			reason = sql.NullString{String: input.ExceptionReason, Valid: true}
		}
		if err := q.InsertAssignment(ctx, db.InsertAssignmentParams{
			ID:                 id,
			OrganisationID:     input.OrganisationID,
			PersonID:           input.PersonID,
			FrameworkVersionID: input.FrameworkVersionID,
			RoleID:             input.RoleID,
			AssignedByPersonID: input.AssignedByPersonID,
			AssignedAt:         input.AssignedAt.UTC(),
			Status:             AssignmentStatus,
			ExceptionReason:    reason,
		}); err != nil {
			if !isUnique(err) {
				return err
			}
		}
		stored, err := q.GetAssignmentByKey(ctx, db.GetAssignmentByKeyParams{
			OrganisationID:     input.OrganisationID,
			PersonID:           input.PersonID,
			FrameworkVersionID: input.FrameworkVersionID,
			RoleID:             input.RoleID,
		})
		if err != nil {
			return err
		}
		row = assignmentFromDB(stored)
		return nil
	})
	return row, err
}

func (s *SQLStore) FindAssignment(ctx context.Context, organisationID, personID, frameworkVersionID, roleID string) (Assignment, error) {
	row, err := db.New(s.database).GetAssignmentByKey(ctx, db.GetAssignmentByKeyParams{
		OrganisationID:     organisationID,
		PersonID:           personID,
		FrameworkVersionID: frameworkVersionID,
		RoleID:             roleID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, err
	}
	return assignmentFromDB(row), nil
}

func (s *SQLStore) GetAssignment(ctx context.Context, organisationID, assignmentID string) (Assignment, error) {
	row, err := db.New(s.database).GetAssignment(ctx, db.GetAssignmentParams{
		OrganisationID: organisationID,
		ID:             assignmentID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, err
	}
	return assignmentFromDB(row), nil
}

func (s *SQLStore) CreateGeneratedPlan(ctx context.Context, input PlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.AssignmentID) || len(input.Skills) == 0 {
		return Plan{}, ErrInvalid
	}
	var created Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetAssignment(ctx, db.GetAssignmentParams{
			OrganisationID: input.OrganisationID,
			ID:             input.AssignmentID,
		}); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := q.GetGeneratedPlan(ctx, db.GetGeneratedPlanParams{
			OrganisationID: input.OrganisationID,
			AssignmentID:   input.AssignmentID,
		}); err == nil {
			return ErrAlreadyGenerated
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertPlan(ctx, db.InsertPlanParams{
			ID:             id,
			OrganisationID: input.OrganisationID,
			AssignmentID:   input.AssignmentID,
			VersionNumber:  1,
			Status:         StatusGenerated,
			PromptVersion:  nullString(input.PromptVersion),
			ModelName:      nullString(input.ModelName),
		}); err != nil {
			return err
		}
		if err := insertSkills(ctx, q, input.OrganisationID, id, input.Skills); err != nil {
			return err
		}
		loaded, err := loadPlan(ctx, q, input.OrganisationID, id)
		if err != nil {
			return err
		}
		created = loaded
		return nil
	})
	if isGeneratedConflict(err) || isPlanVersionConflict(err) {
		return Plan{}, ErrAlreadyGenerated
	}
	return created, err
}

func (s *SQLStore) CreatePlan(ctx context.Context, input PlanInput) (Plan, error) {
	created, err := s.createPlan(ctx, input)
	if isPlanVersionConflict(err) {
		return s.createPlan(ctx, input)
	}
	return created, err
}

func (s *SQLStore) createPlan(ctx context.Context, input PlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.AssignmentID) {
		return Plan{}, ErrInvalid
	}
	if err := validateSkills(input.Skills); err != nil {
		return Plan{}, err
	}
	var created Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetAssignment(ctx, db.GetAssignmentParams{
			OrganisationID: input.OrganisationID,
			ID:             input.AssignmentID,
		}); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		version, err := q.NextPlanVersion(ctx, db.NextPlanVersionParams{
			OrganisationID: input.OrganisationID,
			AssignmentID:   input.AssignmentID,
		})
		if err != nil {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertPlan(ctx, db.InsertPlanParams{
			ID:             id,
			OrganisationID: input.OrganisationID,
			AssignmentID:   input.AssignmentID,
			VersionNumber:  version,
			Status:         StatusDraft,
		}); err != nil {
			return err
		}
		if err := insertSkills(ctx, q, input.OrganisationID, id, input.Skills); err != nil {
			return err
		}
		loaded, err := loadPlan(ctx, q, input.OrganisationID, id)
		if err != nil {
			return err
		}
		created = loaded
		return nil
	})
	return created, err
}

func (s *SQLStore) GetPlan(ctx context.Context, organisationID, planID string) (Plan, error) {
	return loadPlan(ctx, db.New(s.database), organisationID, planID)
}

func (s *SQLStore) ReplacePlan(ctx context.Context, input ReplacePlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.PlanID) || !validStatus(input.Status) {
		return Plan{}, ErrInvalid
	}
	if err := validateSkills(input.Skills); err != nil {
		return Plan{}, err
	}
	var replaced Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := q.GetPlanForUpdate(ctx, db.GetPlanForUpdateParams{
			OrganisationID: input.OrganisationID,
			ID:             input.PlanID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if current.Status == StatusApproved {
			return ErrImmutable
		}
		if err := q.DeleteAssessmentItemsByPlan(ctx, db.DeleteAssessmentItemsByPlanParams{
			OrganisationID: input.OrganisationID,
			PlanID:         input.PlanID,
		}); err != nil {
			return err
		}
		if err := q.DeleteRubricBandsByPlan(ctx, db.DeleteRubricBandsByPlanParams{
			OrganisationID: input.OrganisationID,
			PlanID:         input.PlanID,
		}); err != nil {
			return err
		}
		if err := q.DeletePlanSkillsByPlan(ctx, db.DeletePlanSkillsByPlanParams{
			OrganisationID: input.OrganisationID,
			PlanID:         input.PlanID,
		}); err != nil {
			return err
		}
		if err := insertSkills(ctx, q, input.OrganisationID, input.PlanID, input.Skills); err != nil {
			return err
		}
		if err := q.UpdatePlanStatus(ctx, db.UpdatePlanStatusParams{
			OrganisationID: input.OrganisationID,
			ID:             input.PlanID,
			Status:         input.Status,
		}); err != nil {
			return mapStoreErr(err)
		}
		loaded, err := loadPlan(ctx, q, input.OrganisationID, input.PlanID)
		if err != nil {
			return err
		}
		replaced = loaded
		return nil
	})
	return replaced, err
}

func (s *SQLStore) EditSkill(ctx context.Context, input EditSkillInput) (Plan, error) {
	input = trimSkillEdit(input)
	if err := validateSkillEdit(input); err != nil {
		return Plan{}, err
	}
	var edited Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := q.GetPlanForUpdate(ctx, db.GetPlanForUpdateParams{
			OrganisationID: input.OrganisationID,
			ID:             input.PlanID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := ensureSkillEditable(current.Status); err != nil {
			return err
		}
		skills, err := q.ListPlanSkills(ctx, db.ListPlanSkillsParams{OrganisationID: input.OrganisationID, PlanID: input.PlanID})
		if err != nil {
			return err
		}
		found := false
		for _, skill := range skills {
			if skill.ID == input.SkillID {
				found = true
				break
			}
		}
		if !found {
			return ErrNotFound
		}
		if err := q.UpdatePlanSkillContent(ctx, db.UpdatePlanSkillContentParams{
			OrganisationID:    input.OrganisationID,
			PlanID:            input.PlanID,
			ID:                input.SkillID,
			RecommendedMethod: input.RecommendedMethod,
			Instructions:      input.Instructions,
			EditedByPersonID:  nullString(input.EditorPersonID),
		}); err != nil {
			return err
		}
		if err := q.DeleteAssessmentItemsBySkill(ctx, db.DeleteAssessmentItemsBySkillParams{
			OrganisationID: input.OrganisationID,
			PlanSkillID:    input.SkillID,
		}); err != nil {
			return err
		}
		if err := q.DeleteRubricBandsBySkill(ctx, db.DeleteRubricBandsBySkillParams{
			OrganisationID: input.OrganisationID,
			PlanSkillID:    input.SkillID,
		}); err != nil {
			return err
		}
		for position, item := range input.AssessmentItems {
			itemID, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertAssessmentItem(ctx, db.InsertAssessmentItemParams{
				ID:             itemID,
				OrganisationID: input.OrganisationID,
				PlanSkillID:    input.SkillID,
				Position:       int32(position + 1),
				Prompt:         item.Prompt,
			}); err != nil {
				return err
			}
		}
		for _, band := range input.Rubric {
			bandID, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertRubricBand(ctx, db.InsertRubricBandParams{
				ID:             bandID,
				OrganisationID: input.OrganisationID,
				PlanSkillID:    input.SkillID,
				Level:          int32(band.Level),
				Descriptor:     band.Descriptor,
			}); err != nil {
				return err
			}
		}
		loaded, err := loadPlan(ctx, q, input.OrganisationID, input.PlanID)
		if err != nil {
			return err
		}
		edited = loaded
		return nil
	})
	return edited, err
}

func (s *SQLStore) SubmitPlan(ctx context.Context, input ReviewInput) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) {
		return Plan{}, ErrInvalid
	}
	return s.review(ctx, input, func(status string) (string, sql.NullString, sql.NullString, sql.NullTime, error) {
		if err := ensureSubmittable(status); err != nil {
			return "", sql.NullString{}, sql.NullString{}, sql.NullTime{}, err
		}
		return StatusUnderReview, sql.NullString{}, sql.NullString{}, sql.NullTime{}, nil
	})
}

func (s *SQLStore) RequestChanges(ctx context.Context, input ReviewInput) (Plan, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) || !shortText(input.Reason, 280) {
		return Plan{}, ErrInvalid
	}
	return s.review(ctx, input, func(status string) (string, sql.NullString, sql.NullString, sql.NullTime, error) {
		if err := ensureReviewable(status); err != nil {
			return "", sql.NullString{}, sql.NullString{}, sql.NullTime{}, err
		}
		return StatusChangesRequested, nullString(input.Reason), sql.NullString{}, sql.NullTime{}, nil
	})
}

func (s *SQLStore) ApprovePlan(ctx context.Context, input ReviewInput) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) || input.At.IsZero() {
		return Plan{}, ErrInvalid
	}
	return s.review(ctx, input, func(status string) (string, sql.NullString, sql.NullString, sql.NullTime, error) {
		if err := ensureApprovable(status); err != nil {
			return "", sql.NullString{}, sql.NullString{}, sql.NullTime{}, err
		}
		return StatusApproved, sql.NullString{}, nullString(input.ActorPersonID), nullTime(input.At.UTC()), nil
	})
}

func (s *SQLStore) review(ctx context.Context, input ReviewInput, next func(string) (string, sql.NullString, sql.NullString, sql.NullTime, error)) (Plan, error) {
	var updated Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := q.GetPlanForUpdate(ctx, db.GetPlanForUpdateParams{
			OrganisationID: input.OrganisationID,
			ID:             input.PlanID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		status, reason, approver, approvedAt, err := next(current.Status)
		if err != nil {
			return err
		}
		if status == StatusApproved {
			reason = current.ChangeReason
		}
		if status != StatusApproved {
			reason = current.ChangeReason
			if status == StatusUnderReview {
				reason = sql.NullString{}
			}
			if status == StatusChangesRequested {
				reason = nullString(input.Reason)
			}
		}
		if err := q.UpdatePlanReview(ctx, db.UpdatePlanReviewParams{
			OrganisationID:     input.OrganisationID,
			ID:                 input.PlanID,
			Status:             status,
			ChangeReason:       reason,
			ApprovedByPersonID: approver,
			ApprovedAt:         approvedAt,
		}); err != nil {
			return err
		}
		if status == StatusApproved {
			if _, err := ensureSession(ctx, q, input.OrganisationID, input.PlanID); err != nil {
				return err
			}
		}
		loaded, err := loadPlan(ctx, q, input.OrganisationID, input.PlanID)
		if err != nil {
			return err
		}
		updated = loaded
		return nil
	})
	return updated, err
}

func (s *SQLStore) CopyApprovedPlan(ctx context.Context, organisationID, planID string) (Plan, error) {
	if !present(organisationID) || !present(planID) {
		return Plan{}, ErrInvalid
	}
	var created Plan
	err := s.withTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetPlanForUpdate(ctx, db.GetPlanForUpdateParams{
			OrganisationID: organisationID,
			ID:             planID,
		}); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		current, err := loadPlan(ctx, q, organisationID, planID)
		if err != nil {
			return err
		}
		if current.Status != StatusApproved {
			return ErrInvalid
		}
		version, err := q.NextPlanVersion(ctx, db.NextPlanVersionParams{
			OrganisationID: organisationID,
			AssignmentID:   current.AssignmentID,
		})
		if err != nil {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertPlan(ctx, db.InsertPlanParams{
			ID:             id,
			OrganisationID: organisationID,
			AssignmentID:   current.AssignmentID,
			VersionNumber:  version,
			Status:         StatusDraft,
			PromptVersion:  nullString(current.PromptVersion),
			ModelName:      nullString(current.ModelName),
		}); err != nil {
			return err
		}
		if err := insertSkills(ctx, q, organisationID, id, skillInputs(current.Skills)); err != nil {
			return err
		}
		loaded, err := loadPlan(ctx, q, organisationID, id)
		if err != nil {
			return err
		}
		created = loaded
		return nil
	})
	return created, err
}

func (s *SQLStore) withTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(db.New(s.database).WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return mapStoreErr(err)
	}
	return tx.Commit()
}

func insertSkills(ctx context.Context, q *db.Queries, organisationID, planID string, skills []SkillInput) error {
	for position, skill := range skills {
		skillID, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertPlanSkill(ctx, db.InsertPlanSkillParams{
			ID:                skillID,
			OrganisationID:    organisationID,
			PlanID:            planID,
			Position:          int32(position + 1),
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     int32(skill.RequiredLevel),
			RecommendedMethod: skill.RecommendedMethod,
			Rationale:         skill.Rationale,
			Instructions:      skill.Instructions,
			RequiresSmeReview: skill.RequiresSMEReview,
			RubricNeedsReview: skill.RubricNeedsReview,
			EditedByPersonID:  nullString(skill.EditedByPersonID),
		}); err != nil {
			return err
		}
		for itemPosition, item := range skill.AssessmentItems {
			itemID, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertAssessmentItem(ctx, db.InsertAssessmentItemParams{
				ID:             itemID,
				OrganisationID: organisationID,
				PlanSkillID:    skillID,
				Position:       int32(itemPosition + 1),
				Prompt:         item.Prompt,
			}); err != nil {
				return err
			}
		}
		for _, band := range skill.Rubric {
			bandID, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertRubricBand(ctx, db.InsertRubricBandParams{
				ID:             bandID,
				OrganisationID: organisationID,
				PlanSkillID:    skillID,
				Level:          int32(band.Level),
				Descriptor:     band.Descriptor,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadPlan(ctx context.Context, q *db.Queries, organisationID, planID string) (Plan, error) {
	row, err := q.GetPlan(ctx, db.GetPlanParams{OrganisationID: organisationID, ID: planID})
	if errors.Is(err, sql.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	skillRows, err := q.ListPlanSkills(ctx, db.ListPlanSkillsParams{OrganisationID: organisationID, PlanID: planID})
	if err != nil {
		return Plan{}, err
	}
	skills := make([]PlanSkill, 0, len(skillRows))
	for _, skill := range skillRows {
		items, err := q.ListAssessmentItems(ctx, db.ListAssessmentItemsParams{
			OrganisationID: organisationID,
			PlanSkillID:    skill.ID,
		})
		if err != nil {
			return Plan{}, err
		}
		bands, err := q.ListRubricBands(ctx, db.ListRubricBandsParams{
			OrganisationID: organisationID,
			PlanSkillID:    skill.ID,
		})
		if err != nil {
			return Plan{}, err
		}
		encodedItems := make([]AssessmentItem, 0, len(items))
		for _, item := range items {
			encodedItems = append(encodedItems, AssessmentItem{ID: item.ID, Prompt: item.Prompt})
		}
		encodedBands := make([]RubricBand, 0, len(bands))
		for _, band := range bands {
			encodedBands = append(encodedBands, RubricBand{Level: int(band.Level), Descriptor: band.Descriptor})
		}
		skills = append(skills, PlanSkill{
			ID:                skill.ID,
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     int(skill.RequiredLevel),
			RecommendedMethod: skill.RecommendedMethod,
			Rationale:         skill.Rationale,
			Instructions:      skill.Instructions,
			RequiresSMEReview: skill.RequiresSmeReview,
			RubricNeedsReview: skill.RubricNeedsReview,
			EditedByPersonID:  skill.EditedByPersonID.String,
			AssessmentItems:   encodedItems,
			Rubric:            encodedBands,
		})
	}
	approvedAt := time.Time{}
	if row.ApprovedAt.Valid {
		approvedAt = row.ApprovedAt.Time.UTC()
	}
	return Plan{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		AssignmentID:       row.AssignmentID,
		VersionNumber:      int(row.VersionNumber),
		Status:             row.Status,
		ChangeReason:       row.ChangeReason.String,
		ApprovedByPersonID: row.ApprovedByPersonID.String,
		ApprovedAt:         approvedAt,
		PromptVersion:      row.PromptVersion.String,
		ModelName:          row.ModelName.String,
		Skills:             skills,
	}, nil
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}

func assignmentFromDB(row db.Assignment) Assignment {
	reason := ""
	if row.ExceptionReason.Valid {
		reason = row.ExceptionReason.String
	}
	return Assignment{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		PersonID:           row.PersonID,
		FrameworkVersionID: row.FrameworkVersionID,
		RoleID:             row.RoleID,
		AssignedByPersonID: row.AssignedByPersonID,
		AssignedAt:         row.AssignedAt.UTC(),
		Status:             row.Status,
		ExceptionReason:    reason,
	}
}

func mapStoreErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrInvalid), errors.Is(err, ErrImmutable), errors.Is(err, ErrNotEditable):
		return err
	case isImmutable(err):
		return ErrImmutable
	case isCheck(err):
		return ErrInvalid
	default:
		return err
	}
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isGeneratedConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "plans_one_generated_per_assignment"
}

func isPlanVersionConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "plans_assignment_id_version_number_key"
}

func isCheck(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func isImmutable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "P0001"
}
