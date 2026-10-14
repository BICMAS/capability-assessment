package plan

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

func (m *Memory) CreateSession(_ context.Context, organisationID, planID string) (Session, error) {
	if !present(organisationID) || !present(planID) {
		return Session{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[planID]
	if !ok || row.OrganisationID != organisationID {
		return Session{}, ErrNotFound
	}
	return m.insertSession(row)
}

func (m *Memory) ListSessions(_ context.Context, organisationID, personID string) ([]Session, error) {
	if !present(organisationID) || !present(personID) {
		return nil, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]Session, 0)
	for _, session := range m.sessions {
		if session.OrganisationID == organisationID && session.PersonID == personID {
			rows = append(rows, copySession(session))
		}
	}
	return rows, nil
}

func (m *Memory) GetSession(_ context.Context, organisationID, sessionID string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionFor(organisationID, sessionID, "")
}

func (m *Memory) StartSession(_ context.Context, organisationID, sessionID, personID string, at time.Time) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.sessionFor(organisationID, sessionID, personID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != SessionAssigned {
		return Session{}, ErrNotEditable
	}
	session.Status = SessionStarted
	session.StartedAt = at.UTC()
	m.sessions[session.ID] = session
	return copySession(session), nil
}

func (m *Memory) SaveAnswers(_ context.Context, organisationID, sessionID, personID string, answers []AnswerInput) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.sessionFor(organisationID, sessionID, personID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != SessionStarted {
		return Session{}, ErrNotEditable
	}
	if err := applyAnswers(&session, answers); err != nil {
		return Session{}, err
	}
	m.sessions[session.ID] = session
	return copySession(session), nil
}

func (m *Memory) SubmitSession(_ context.Context, organisationID, sessionID, personID string, at time.Time, proposals []Proposal) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.sessionFor(organisationID, sessionID, personID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != SessionStarted {
		return Session{}, ErrNotEditable
	}
	if !answersComplete(session) {
		return Session{}, ErrInvalid
	}
	plan, ok := m.plans[session.PlanID]
	if !ok || plan.OrganisationID != organisationID {
		return Session{}, ErrNotFound
	}
	if err := acceptProposals(plan, proposals); err != nil {
		return Session{}, err
	}
	session.Status = SessionSubmitted
	session.SubmittedAt = at.UTC()
	session.Proposals = copyProposals(proposals)
	m.sessions[session.ID] = session
	return copySession(session), nil
}

func (m *Memory) MarkSessionConfirmed(_ context.Context, organisationID, sessionID string, at time.Time) (Session, error) {
	if at.IsZero() {
		return Session{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.sessionFor(organisationID, sessionID, "")
	if err != nil {
		return Session{}, err
	}
	if session.Status != SessionSubmitted || len(session.Proposals) == 0 {
		return Session{}, ErrNotEditable
	}
	if !session.ConfirmedAt.IsZero() {
		return Session{}, ErrImmutable
	}
	session.ConfirmedAt = at.UTC()
	m.sessions[session.ID] = session
	return copySession(session), nil
}

func (m *Memory) insertSession(row Plan) (Session, error) {
	if row.Status != StatusApproved {
		return Session{}, ErrNotApproved
	}
	if id, ok := m.sessionByPlan[row.ID]; ok {
		return copySession(m.sessions[id]), nil
	}
	assignment, ok := m.assignments[row.AssignmentID]
	if !ok || assignment.OrganisationID != row.OrganisationID {
		return Session{}, ErrNotFound
	}
	skills := sessionSkills(row.Skills)
	if len(skills) == 0 {
		return Session{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	session := Session{
		ID:             id,
		OrganisationID: row.OrganisationID,
		PlanID:         row.ID,
		PersonID:       assignment.PersonID,
		Status:         SessionAssigned,
		Skills:         skills,
	}
	m.sessions[id] = session
	m.sessionByPlan[row.ID] = id
	return copySession(session), nil
}

func (m *Memory) sessionFor(organisationID, sessionID, personID string) (Session, error) {
	session, ok := m.sessions[sessionID]
	if !ok || session.OrganisationID != organisationID {
		return Session{}, ErrNotFound
	}
	if personID != "" && session.PersonID != personID {
		return Session{}, ErrNotFound
	}
	return copySession(session), nil
}

func sessionSkills(skills []PlanSkill) []SessionSkill {
	rows := make([]SessionSkill, 0, len(skills))
	for _, skill := range skills {
		if len(skill.AssessmentItems) == 0 {
			continue
		}
		items := make([]SessionItem, 0, len(skill.AssessmentItems))
		for _, item := range skill.AssessmentItems {
			items = append(items, SessionItem{ID: item.ID, Prompt: item.Prompt})
		}
		rows = append(rows, SessionSkill{
			ID:                skill.ID,
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     skill.RequiredLevel,
			RecommendedMethod: skill.RecommendedMethod,
			Instructions:      skill.Instructions,
			Items:             items,
		})
	}
	return rows
}

func applyAnswers(session *Session, answers []AnswerInput) error {
	if len(answers) == 0 {
		return ErrInvalid
	}
	index := map[string]*SessionItem{}
	for skillIndex := range session.Skills {
		for itemIndex := range session.Skills[skillIndex].Items {
			item := &session.Skills[skillIndex].Items[itemIndex]
			index[item.ID] = item
		}
	}
	seen := map[string]struct{}{}
	for _, answer := range answers {
		answer.Answer = strings.TrimSpace(answer.Answer)
		item, ok := index[answer.ItemID]
		if !ok || !present(answer.ItemID) {
			return ErrInvalid
		}
		if _, dup := seen[answer.ItemID]; dup || utf8.RuneCountInString(answer.Answer) > 8000 {
			return ErrInvalid
		}
		seen[answer.ItemID] = struct{}{}
		item.Answer = answer.Answer
	}
	return nil
}

func answersComplete(session Session) bool {
	for _, skill := range session.Skills {
		for _, item := range skill.Items {
			if !present(item.Answer) {
				return false
			}
		}
	}
	return true
}

func copySession(row Session) Session {
	skills := make([]SessionSkill, len(row.Skills))
	for i, skill := range row.Skills {
		skill.Items = append([]SessionItem(nil), skill.Items...)
		skills[i] = skill
	}
	row.Skills = skills
	row.Proposals = copyProposals(row.Proposals)
	return row
}

func copyProposals(rows []Proposal) []Proposal {
	copied := make([]Proposal, len(rows))
	for i, row := range rows {
		row.BandLevels = append([]int(nil), row.BandLevels...)
		copied[i] = row
	}
	return copied
}
