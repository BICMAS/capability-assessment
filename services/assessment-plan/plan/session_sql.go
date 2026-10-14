package plan

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bicmas-academy/learn/services/assessment-plan/db"
)

func (s *SQLStore) CreateSession(ctx context.Context, organisationID, planID string) (Session, error) {
	if !present(organisationID) || !present(planID) {
		return Session{}, ErrInvalid
	}
	var created Session
	err := s.withTx(ctx, func(q *db.Queries) error {
		session, err := ensureSession(ctx, q, organisationID, planID)
		if err != nil {
			return err
		}
		created = session
		return nil
	})
	return created, err
}

func (s *SQLStore) ListSessions(ctx context.Context, organisationID, personID string) ([]Session, error) {
	if !present(organisationID) || !present(personID) {
		return nil, ErrInvalid
	}
	q := db.New(s.database)
	rows, err := q.ListSessionsByPerson(ctx, db.ListSessionsByPersonParams{
		OrganisationID: organisationID,
		PersonID:       personID,
	})
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, 0, len(rows))
	for _, row := range rows {
		session, err := sessionFromRow(ctx, q, row)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *SQLStore) GetSession(ctx context.Context, organisationID, sessionID string) (Session, error) {
	return loadSession(ctx, db.New(s.database), organisationID, sessionID)
}

func (s *SQLStore) StartSession(ctx context.Context, organisationID, sessionID, personID string, at time.Time) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	var updated Session
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		if current.PersonID != personID {
			return ErrNotFound
		}
		if current.Status != SessionAssigned {
			return ErrNotEditable
		}
		if err := q.UpdateSessionProgress(ctx, db.UpdateSessionProgressParams{
			OrganisationID: organisationID,
			ID:             sessionID,
			Status:         SessionStarted,
			StartedAt:      nullTime(at.UTC()),
		}); err != nil {
			return err
		}
		loaded, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		updated = loaded
		return nil
	})
	return updated, err
}

func (s *SQLStore) SaveAnswers(ctx context.Context, organisationID, sessionID, personID string, answers []AnswerInput) (Session, error) {
	var updated Session
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		if current.PersonID != personID {
			return ErrNotFound
		}
		if current.Status != SessionStarted {
			return ErrNotEditable
		}
		for i := range answers {
			answers[i].Answer = strings.TrimSpace(answers[i].Answer)
		}
		if err := applyAnswers(&current, answers); err != nil {
			return err
		}
		for _, answer := range answers {
			if err := q.UpdateSessionAnswer(ctx, db.UpdateSessionAnswerParams{
				OrganisationID:   organisationID,
				SessionID:        sessionID,
				AssessmentItemID: answer.ItemID,
				Answer:           answer.Answer,
			}); err != nil {
				return err
			}
		}
		loaded, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		updated = loaded
		return nil
	})
	return updated, err
}

func (s *SQLStore) SubmitSession(ctx context.Context, organisationID, sessionID, personID string, at time.Time, proposals []Proposal) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	var updated Session
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		if current.PersonID != personID {
			return ErrNotFound
		}
		if current.Status != SessionStarted {
			return ErrNotEditable
		}
		if !answersComplete(current) {
			return ErrInvalid
		}
		row, err := loadPlan(ctx, q, organisationID, current.PlanID)
		if err != nil {
			return err
		}
		if err := acceptProposals(row, proposals); err != nil {
			return err
		}
		if err := q.UpdateSessionProgress(ctx, db.UpdateSessionProgressParams{
			OrganisationID: organisationID,
			ID:             sessionID,
			Status:         SessionSubmitted,
			StartedAt:      nullTime(current.StartedAt),
			SubmittedAt:    nullTime(at.UTC()),
		}); err != nil {
			return err
		}
		for _, proposal := range proposals {
			id, err := newID()
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(proposal.BandLevels)
			if err != nil {
				return ErrInvalid
			}
			if err := q.InsertSessionProposal(ctx, db.InsertSessionProposalParams{
				ID:                id,
				OrganisationID:    organisationID,
				SessionID:         sessionID,
				PlanSkillID:       proposal.PlanSkillID,
				RequirementID:     proposal.RequirementID,
				CapabilityID:      proposal.CapabilityID,
				RequiredLevel:     int32(proposal.RequiredLevel),
				RecommendedMethod: proposal.RecommendedMethod,
				EvidenceSource:    proposal.EvidenceSource,
				BandLevels:        string(encoded),
				Percentage:        int32(proposal.Percentage),
			}); err != nil {
				return err
			}
		}
		loaded, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		updated = loaded
		return nil
	})
	return updated, err
}

func (s *SQLStore) MarkSessionConfirmed(ctx context.Context, organisationID, sessionID string, at time.Time) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	var updated Session
	err := s.withTx(ctx, func(q *db.Queries) error {
		current, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		if current.Status != SessionSubmitted || len(current.Proposals) == 0 {
			return ErrNotEditable
		}
		if !current.ConfirmedAt.IsZero() {
			return ErrImmutable
		}
		if err := q.ConfirmSession(ctx, db.ConfirmSessionParams{
			OrganisationID: organisationID,
			ID:             sessionID,
			ConfirmedAt:    nullTime(at.UTC()),
		}); err != nil {
			return err
		}
		loaded, err := loadSession(ctx, q, organisationID, sessionID)
		if err != nil {
			return err
		}
		if loaded.ConfirmedAt.IsZero() {
			return ErrNotEditable
		}
		updated = loaded
		return nil
	})
	return updated, err
}

func ensureSession(ctx context.Context, q *db.Queries, organisationID, planID string) (Session, error) {
	plan, err := loadPlan(ctx, q, organisationID, planID)
	if err != nil {
		return Session{}, err
	}
	if plan.Status != StatusApproved {
		return Session{}, ErrNotApproved
	}
	existing, err := q.GetSessionByPlan(ctx, db.GetSessionByPlanParams{
		OrganisationID: organisationID,
		PlanID:         planID,
	})
	if err == nil {
		return sessionFromRow(ctx, q, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Session{}, err
	}
	assignment, err := q.GetAssignment(ctx, db.GetAssignmentParams{
		OrganisationID: organisationID,
		ID:             plan.AssignmentID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	items, err := q.ListPlanItems(ctx, db.ListPlanItemsParams{
		OrganisationID: organisationID,
		PlanID:         planID,
	})
	if err != nil {
		return Session{}, err
	}
	if len(items) == 0 {
		return Session{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	if err := q.InsertSession(ctx, db.InsertSessionParams{
		ID:             id,
		OrganisationID: organisationID,
		PlanID:         planID,
		PersonID:       assignment.PersonID,
	}); err != nil {
		return Session{}, err
	}
	for _, item := range items {
		answerID, err := newID()
		if err != nil {
			return Session{}, err
		}
		if err := q.InsertSessionAnswer(ctx, db.InsertSessionAnswerParams{
			ID:               answerID,
			OrganisationID:   organisationID,
			SessionID:        id,
			AssessmentItemID: item.ItemID,
			Answer:           "",
		}); err != nil {
			return Session{}, err
		}
	}
	return loadSession(ctx, q, organisationID, id)
}

func loadSession(ctx context.Context, q *db.Queries, organisationID, sessionID string) (Session, error) {
	row, err := q.GetSession(ctx, db.GetSessionParams{OrganisationID: organisationID, ID: sessionID})
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	return sessionFromRow(ctx, q, row)
}

func sessionFromRow(ctx context.Context, q *db.Queries, row db.AssessmentSession) (Session, error) {
	items, err := q.ListPlanItems(ctx, db.ListPlanItemsParams{
		OrganisationID: row.OrganisationID,
		PlanID:         row.PlanID,
	})
	if err != nil {
		return Session{}, err
	}
	answers, err := q.ListSessionAnswers(ctx, db.ListSessionAnswersParams{
		OrganisationID: row.OrganisationID,
		SessionID:      row.ID,
	})
	if err != nil {
		return Session{}, err
	}
	written := map[string]string{}
	for _, answer := range answers {
		written[answer.AssessmentItemID] = answer.Answer
	}
	skills := make([]SessionSkill, 0)
	var current string
	for _, item := range items {
		if current != item.SkillID {
			skills = append(skills, SessionSkill{
				ID:                item.SkillID,
				RequirementID:     item.RequirementID,
				CapabilityID:      item.CapabilityID,
				RequiredLevel:     int(item.RequiredLevel),
				RecommendedMethod: item.RecommendedMethod,
				Instructions:      item.Instructions,
			})
			current = item.SkillID
		}
		skills[len(skills)-1].Items = append(skills[len(skills)-1].Items, SessionItem{
			ID:     item.ItemID,
			Prompt: item.Prompt,
			Answer: written[item.ItemID],
		})
	}
	startedAt := time.Time{}
	if row.StartedAt.Valid {
		startedAt = row.StartedAt.Time.UTC()
	}
	submittedAt := time.Time{}
	if row.SubmittedAt.Valid {
		submittedAt = row.SubmittedAt.Time.UTC()
	}
	confirmedAt := time.Time{}
	if row.ConfirmedAt.Valid {
		confirmedAt = row.ConfirmedAt.Time.UTC()
	}
	stored, err := q.ListSessionProposals(ctx, db.ListSessionProposalsParams{
		OrganisationID: row.OrganisationID,
		SessionID:      row.ID,
	})
	if err != nil {
		return Session{}, err
	}
	proposals := make([]Proposal, 0, len(stored))
	for _, proposal := range stored {
		var levels []int
		if err := json.Unmarshal([]byte(proposal.BandLevels), &levels); err != nil {
			return Session{}, ErrInvalid
		}
		proposals = append(proposals, Proposal{
			PlanSkillID:       proposal.PlanSkillID,
			RequirementID:     proposal.RequirementID,
			CapabilityID:      proposal.CapabilityID,
			RequiredLevel:     int(proposal.RequiredLevel),
			RecommendedMethod: proposal.RecommendedMethod,
			EvidenceSource:    proposal.EvidenceSource,
			BandLevels:        levels,
			Percentage:        int(proposal.Percentage),
		})
	}
	return Session{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		PlanID:         row.PlanID,
		PersonID:       row.PersonID,
		Status:         row.Status,
		StartedAt:      startedAt,
		SubmittedAt:    submittedAt,
		ConfirmedAt:    confirmedAt,
		Proposals:      proposals,
		Skills:         skills,
	}, nil
}
