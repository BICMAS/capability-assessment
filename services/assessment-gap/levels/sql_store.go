package levels

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bicmas-academy/learn/services/assessment-gap/db"
)

// SQLStore persists attempts, explanations, and manager overrides.
type SQLStore struct {
	database *sql.DB
}

func NewSQLStore(database *sql.DB) *SQLStore {
	return &SQLStore{database: database}
}

func (s *SQLStore) Calculate(ctx context.Context, input Calculation) (Explanation, error) {
	if err := validateIdentity(input.OrganisationID, input.PersonID, input.FrameworkVersionID, input.CapabilityCode); err != nil {
		return Explanation{}, err
	}
	if err := validateRequired(input.RequiredLevel); err != nil {
		return Explanation{}, err
	}
	for _, percentage := range []*int{
		input.Sources.Knowledge,
		input.Sources.SituationalJudgement,
		input.Sources.AIInterview,
		input.Sources.Simulation,
		input.Sources.ManagerAssessment,
		input.Sources.SelfAssessment,
	} {
		if percentage != nil {
			if err := validatePercentage(*percentage); err != nil {
				return Explanation{}, err
			}
		}
	}
	var explanation Explanation
	err := s.withTx(ctx, func(q *db.Queries) error {
		now := time.Now().UTC()
		for _, source := range []struct {
			name       string
			percentage *int
		}{
			{SourceKnowledge, input.Sources.Knowledge},
			{SourceSituationalJudgement, input.Sources.SituationalJudgement},
			{SourceSelfAssessment, input.Sources.SelfAssessment},
			{SourceManagerAssessment, input.Sources.ManagerAssessment},
		} {
			if source.percentage == nil {
				continue
			}
			evidence, err := evidenceIDs(input.Evidence[source.name])
			if err != nil {
				return err
			}
			if err := insertAttempt(ctx, q, levelIdentity(input), source.name, *source.percentage, evidence, input.RequiredLevel, now); err != nil {
				return err
			}
		}
		existing, err := lookupExplanation(ctx, q, levelIdentity(input))
		if err != nil {
			return err
		}
		columns := Sources{
			AIInterview: input.Sources.AIInterview,
			Simulation:  input.Sources.Simulation,
		}
		if columns.AIInterview == nil {
			columns.AIInterview = existing.AIInterview
		}
		if columns.Simulation == nil {
			columns.Simulation = existing.Simulation
		}
		explanation, err = saveFold(ctx, q, levelIdentity(input), existing.ID, columns, input.RequiredLevel)
		return err
	})
	return explanation, err
}

func (s *SQLStore) Record(ctx context.Context, input AttemptInput) (Explanation, error) {
	if columnOnly(input.Source) {
		return Explanation{}, ErrNoWriter
	}
	if !writable(input.Source) {
		return Explanation{}, ErrInvalid
	}
	if err := validateIdentity(input.OrganisationID, input.PersonID, input.FrameworkVersionID, input.CapabilityCode); err != nil {
		return Explanation{}, err
	}
	if err := validatePercentage(input.Percentage); err != nil {
		return Explanation{}, err
	}
	if err := validateRequired(input.RequiredLevel); err != nil {
		return Explanation{}, err
	}
	evidence, err := evidenceIDs(input.EvidenceIDs)
	if err != nil {
		return Explanation{}, err
	}
	identity := levelIdentityFromAttempt(input)
	var explanation Explanation
	err = s.withTx(ctx, func(q *db.Queries) error {
		if err := insertAttempt(ctx, q, identity, input.Source, input.Percentage, evidence, input.RequiredLevel, time.Now().UTC()); err != nil {
			return err
		}
		existing, err := lookupExplanation(ctx, q, identity)
		if err != nil {
			return err
		}
		columns := Sources{AIInterview: existing.AIInterview, Simulation: existing.Simulation}
		explanation, err = saveFold(ctx, q, identity, existing.ID, columns, input.RequiredLevel)
		return err
	})
	return explanation, err
}

func (s *SQLStore) Get(ctx context.Context, organisationID, personID, frameworkVersionID, capabilityCode string) (Explanation, error) {
	row, err := db.New(s.database).GetExplanation(ctx, db.GetExplanationParams{
		OrganisationID:     organisationID,
		PersonID:           personID,
		FrameworkVersionID: frameworkVersionID,
		CapabilityCode:     capabilityCode,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Explanation{}, ErrNotFound
	}
	if err != nil {
		return Explanation{}, err
	}
	return explanationFromDB(row), nil
}

func (s *SQLStore) GetByID(ctx context.Context, organisationID, explanationID string) (Explanation, error) {
	row, err := db.New(s.database).GetExplanationByID(ctx, db.GetExplanationByIDParams{
		OrganisationID: organisationID,
		ID:             explanationID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Explanation{}, ErrNotFound
	}
	if err != nil {
		return Explanation{}, err
	}
	return explanationFromDB(row), nil
}

func (s *SQLStore) Override(ctx context.Context, input OverrideInput) (Override, error) {
	reason := strings.TrimSpace(input.Reason)
	if input.OrganisationID == "" || input.ExplanationID == "" || input.ActorPersonID == "" || reason == "" {
		return Override{}, ErrInvalid
	}
	if err := validateRequired(input.NewValue); err != nil {
		return Override{}, err
	}
	var row Override
	err := s.withTx(ctx, func(q *db.Queries) error {
		existing, err := q.GetExplanationByID(ctx, db.GetExplanationByIDParams{
			OrganisationID: input.OrganisationID,
			ID:             input.ExplanationID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		overrides, err := q.ListOverrides(ctx, db.ListOverridesParams{
			OrganisationID: input.OrganisationID,
			ExplanationID:  existing.ID,
		})
		if err != nil {
			return err
		}
		previous := Tenths(existing.CurrentLevelTenths)
		if len(overrides) > 0 {
			previous = Tenths(overrides[len(overrides)-1].NewValueTenths)
		}
		id, err := newID()
		if err != nil {
			return err
		}
		createdAt := time.Now().UTC()
		if err := q.InsertOverride(ctx, db.InsertOverrideParams{
			ID:                  id,
			OrganisationID:      input.OrganisationID,
			ExplanationID:       existing.ID,
			ActorPersonID:       input.ActorPersonID,
			Reason:              reason,
			PreviousValueTenths: int32(previous),
			NewValueTenths:      int32(input.NewValue),
			CreatedAt:           createdAt,
		}); err != nil {
			return err
		}
		row = Override{
			ID:             id,
			OrganisationID: input.OrganisationID,
			ExplanationID:  existing.ID,
			ActorPersonID:  input.ActorPersonID,
			Reason:         reason,
			PreviousValue:  previous,
			NewValue:       input.NewValue,
			CreatedAt:      createdAt,
		}
		return nil
	})
	return row, err
}

type identity struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
}

func levelIdentity(input Calculation) identity {
	return identity{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		CapabilityCode:     input.CapabilityCode,
	}
}

func levelIdentityFromAttempt(input AttemptInput) identity {
	return identity{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		CapabilityCode:     input.CapabilityCode,
	}
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

func insertAttempt(ctx context.Context, q *db.Queries, key identity, source string, percentage int, evidence []string, required Tenths, createdAt time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	return q.InsertAttempt(ctx, db.InsertAttemptParams{
		ID:                  id,
		OrganisationID:      key.OrganisationID,
		PersonID:            key.PersonID,
		FrameworkVersionID:  key.FrameworkVersionID,
		CapabilityCode:      key.CapabilityCode,
		Source:              source,
		Percentage:          int32(percentage),
		EvidenceIds:         string(encoded),
		RequiredLevelTenths: int32(required),
		CreatedAt:           createdAt,
	})
}

func lookupExplanation(ctx context.Context, q *db.Queries, key identity) (Explanation, error) {
	row, err := q.GetExplanation(ctx, db.GetExplanationParams{
		OrganisationID:     key.OrganisationID,
		PersonID:           key.PersonID,
		FrameworkVersionID: key.FrameworkVersionID,
		CapabilityCode:     key.CapabilityCode,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Explanation{}, nil
	}
	if err != nil {
		return Explanation{}, err
	}
	return explanationFromDB(row), nil
}

func saveFold(ctx context.Context, q *db.Queries, key identity, existingID string, columns Sources, required Tenths) (Explanation, error) {
	rows, err := q.ListAttempts(ctx, db.ListAttemptsParams{
		OrganisationID:     key.OrganisationID,
		PersonID:           key.PersonID,
		FrameworkVersionID: key.FrameworkVersionID,
		CapabilityCode:     key.CapabilityCode,
	})
	if err != nil {
		return Explanation{}, err
	}
	attempts := make([]Attempt, 0, len(rows))
	for _, row := range rows {
		attempts = append(attempts, Attempt{
			Source:        row.Source,
			Percentage:    int(row.Percentage),
			RequiredLevel: Tenths(row.RequiredLevelTenths),
		})
	}
	sources, requiredLevel := fold(attempts, columns, required)
	explanation := Explain(sources, key.FrameworkVersionID, requiredLevel)
	explanation.ID = existingID
	if explanation.ID == "" {
		id, err := newID()
		if err != nil {
			return Explanation{}, err
		}
		explanation.ID = id
	}
	explanation.OrganisationID = key.OrganisationID
	explanation.PersonID = key.PersonID
	explanation.CapabilityCode = key.CapabilityCode
	if err := q.UpsertExplanation(ctx, db.UpsertExplanationParams{
		ID:                             explanation.ID,
		OrganisationID:                 explanation.OrganisationID,
		PersonID:                       explanation.PersonID,
		FrameworkVersionID:             explanation.FrameworkVersionID,
		CapabilityCode:                 explanation.CapabilityCode,
		KnowledgePercentage:            nullPercentage(explanation.Knowledge),
		SituationalJudgementPercentage: nullPercentage(explanation.SituationalJudgement),
		AiInterviewPercentage:          nullPercentage(explanation.AIInterview),
		SimulationPercentage:           nullPercentage(explanation.Simulation),
		ManagerAssessmentPercentage:    nullPercentage(explanation.ManagerAssessment),
		SelfAssessmentPercentage:       nullPercentage(explanation.SelfAssessment),
		MeanTenths:                     int32(explanation.Mean),
		Divisor:                        int32(explanation.Divisor),
		RequiredLevelTenths:            int32(explanation.RequiredLevel),
		CurrentLevelTenths:             int32(explanation.CurrentLevel),
		GapTenths:                      int32(explanation.Gap),
		Status:                         explanation.Status,
	}); err != nil {
		return Explanation{}, err
	}
	return explanation, nil
}

func explanationFromDB(row db.Explanation) Explanation {
	return Explanation{
		ID:                   row.ID,
		OrganisationID:       row.OrganisationID,
		PersonID:             row.PersonID,
		FrameworkVersionID:   row.FrameworkVersionID,
		CapabilityCode:       row.CapabilityCode,
		Knowledge:            percentagePtr(row.KnowledgePercentage),
		SituationalJudgement: percentagePtr(row.SituationalJudgementPercentage),
		AIInterview:          percentagePtr(row.AiInterviewPercentage),
		Simulation:           percentagePtr(row.SimulationPercentage),
		ManagerAssessment:    percentagePtr(row.ManagerAssessmentPercentage),
		SelfAssessment:       percentagePtr(row.SelfAssessmentPercentage),
		Mean:                 Tenths(row.MeanTenths),
		Divisor:              int(row.Divisor),
		RequiredLevel:        Tenths(row.RequiredLevelTenths),
		CurrentLevel:         Tenths(row.CurrentLevelTenths),
		Gap:                  Tenths(row.GapTenths),
		Status:               row.Status,
	}
}

func nullPercentage(value *int) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*value), Valid: true}
}

func percentagePtr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	copied := int(value.Int32)
	return &copied
}
