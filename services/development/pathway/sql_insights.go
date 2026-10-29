package pathway

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/bicmas-academy/learn/services/development/db"
)

func (s *SQLStore) FindCourse(ctx context.Context, organisationID, capabilityID string, targetLevel int) (Mapping, error) {
	row, err := db.New(s.database).GetCourseMapping(ctx, db.GetCourseMappingParams{
		OrganisationID: organisationID,
		CapabilityID:   capabilityID,
		TargetLevel:    int32(targetLevel),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Mapping{}, ErrNotFound
	}
	if err != nil {
		return Mapping{}, err
	}
	return mappingFromDB(row), nil
}

func (s *SQLStore) SaveResource(ctx context.Context, input ResourceInput) (Resource, error) {
	if err := validateResource(input); err != nil {
		return Resource{}, err
	}
	id, err := newID()
	if err != nil {
		return Resource{}, err
	}
	row := Resource{
		ID: id, OrganisationID: input.OrganisationID, Title: strings.TrimSpace(input.Title),
		SkillID: input.SkillID, TargetLevel: input.TargetLevel, Reference: strings.TrimSpace(input.Reference),
		Rationale: strings.TrimSpace(input.Rationale),
	}
	_, err = s.database.ExecContext(ctx, `
		INSERT INTO learning_resources (id, organisation_id, title, skill_id, target_level, reference_uri, rationale)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		row.ID, row.OrganisationID, row.Title, row.SkillID, row.TargetLevel, row.Reference, row.Rationale)
	if err != nil {
		return Resource{}, err
	}
	return row, nil
}

func (s *SQLStore) ListResources(ctx context.Context, organisationID string) ([]Resource, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, organisation_id, title, skill_id, target_level, reference_uri, rationale
		FROM learning_resources WHERE organisation_id = $1 ORDER BY title`, organisationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resources := make([]Resource, 0)
	for rows.Next() {
		var row Resource
		if err := rows.Scan(&row.ID, &row.OrganisationID, &row.Title, &row.SkillID, &row.TargetLevel, &row.Reference, &row.Rationale); err != nil {
			return nil, err
		}
		resources = append(resources, row)
	}
	return resources, rows.Err()
}

func (s *SQLStore) SaveRecommendation(ctx context.Context, input LearningRecommendation) (LearningRecommendation, error) {
	if input.OrganisationID == "" || input.PersonID == "" || input.SkillID == "" || input.Status != StatusRecommended || strings.TrimSpace(input.Activity) == "" {
		return LearningRecommendation{}, ErrInvalid
	}
	existing, err := s.ListRecommendations(ctx, input.OrganisationID, input.PersonID)
	if err != nil {
		return LearningRecommendation{}, err
	}
	for _, row := range existing {
		if row.SkillID == input.SkillID && row.IntendedLevel == input.IntendedLevel {
			return row, nil
		}
	}
	id, err := newID()
	if err != nil {
		return LearningRecommendation{}, err
	}
	input.ID = id
	_, err = s.database.ExecContext(ctx, `
		INSERT INTO learning_recommendations (
			id, organisation_id, person_id, skill_id, required_level, intended_level, activity, rationale, reference_uri, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		input.ID, input.OrganisationID, input.PersonID, input.SkillID, input.RequiredLevel, input.IntendedLevel,
		input.Activity, input.Rationale, input.Reference, input.Status)
	if err != nil {
		return LearningRecommendation{}, err
	}
	return input, nil
}

func (s *SQLStore) ListRecommendations(ctx context.Context, organisationID, personID string) ([]LearningRecommendation, error) {
	query := `
		SELECT id, organisation_id, person_id, skill_id, required_level, intended_level, activity, rationale, reference_uri, status
		FROM learning_recommendations WHERE organisation_id = $1`
	args := []any{organisationID}
	if personID != "" {
		query += ` AND person_id = $2`
		args = append(args, personID)
	}
	query += ` ORDER BY skill_id`
	rows, err := s.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recommendations := make([]LearningRecommendation, 0)
	for rows.Next() {
		var row LearningRecommendation
		if err := rows.Scan(&row.ID, &row.OrganisationID, &row.PersonID, &row.SkillID, &row.RequiredLevel, &row.IntendedLevel, &row.Activity, &row.Rationale, &row.Reference, &row.Status); err != nil {
			return nil, err
		}
		recommendations = append(recommendations, row)
	}
	return recommendations, rows.Err()
}

func (s *SQLStore) SavePractice(ctx context.Context, input PracticeInput) (PracticeRecord, error) {
	if !input.Record || input.OrganisationID == "" || input.PersonID == "" || strings.TrimSpace(input.Feedback) == "" {
		return PracticeRecord{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return PracticeRecord{}, err
	}
	row := PracticeRecord{
		ID: id, OrganisationID: input.OrganisationID, PersonID: input.PersonID, ScenarioKey: input.ScenarioKey,
		Message: input.Message, Feedback: input.Feedback, Recorded: true, Voice: VoiceUnconfigured,
	}
	_, err = s.database.ExecContext(ctx, `
		INSERT INTO practice_sessions (id, organisation_id, person_id, scenario_key, message, feedback)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		row.ID, row.OrganisationID, row.PersonID, row.ScenarioKey, row.Message, row.Feedback)
	if err != nil {
		return PracticeRecord{}, err
	}
	return row, nil
}

func (s *SQLStore) ListPractice(ctx context.Context, organisationID, personID string) ([]PracticeRecord, error) {
	query := `
		SELECT id, organisation_id, person_id, scenario_key, message, feedback
		FROM practice_sessions WHERE organisation_id = $1`
	args := []any{organisationID}
	if personID != "" {
		query += ` AND person_id = $2`
		args = append(args, personID)
	}
	query += ` ORDER BY id`
	rows, err := s.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]PracticeRecord, 0)
	for rows.Next() {
		var row PracticeRecord
		if err := rows.Scan(&row.ID, &row.OrganisationID, &row.PersonID, &row.ScenarioKey, &row.Message, &row.Feedback); err != nil {
			return nil, err
		}
		row.Recorded = true
		row.Voice = VoiceUnconfigured
		records = append(records, row)
	}
	return records, rows.Err()
}
