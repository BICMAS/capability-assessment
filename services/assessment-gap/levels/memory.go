package levels

import (
	"context"
	"strings"
	"sync"
	"time"
)

type levelKey struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
}

// Memory is the in-process store used by the level tests.
type Memory struct {
	mu           sync.Mutex
	attempts     map[levelKey][]Attempt
	explanations map[levelKey]Explanation
	byID         map[string]levelKey
	overrides    map[string][]Override
}

func NewMemory() *Memory {
	return &Memory{
		attempts:     map[levelKey][]Attempt{},
		explanations: map[levelKey]Explanation{},
		byID:         map[string]levelKey{},
		overrides:    map[string][]Override{},
	}
}

func (m *Memory) Calculate(_ context.Context, input Calculation) (Explanation, error) {
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
	key := levelKey{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		CapabilityCode:     input.CapabilityCode,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
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
			return Explanation{}, err
		}
		if err := m.appendAttempt(key, source.name, *source.percentage, evidence, input.RequiredLevel, now); err != nil {
			return Explanation{}, err
		}
	}
	columns := Sources{
		AIInterview: input.Sources.AIInterview,
		Simulation:  input.Sources.Simulation,
	}
	return m.saveFold(key, columns, input.RequiredLevel)
}

func (m *Memory) Record(_ context.Context, input AttemptInput) (Explanation, error) {
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
	key := levelKey{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		CapabilityCode:     input.CapabilityCode,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.appendAttempt(key, input.Source, input.Percentage, evidence, input.RequiredLevel, time.Now().UTC()); err != nil {
		return Explanation{}, err
	}
	existing := m.explanations[key]
	columns := Sources{AIInterview: existing.AIInterview, Simulation: existing.Simulation}
	return m.saveFold(key, columns, input.RequiredLevel)
}

func (m *Memory) Get(_ context.Context, organisationID, personID, frameworkVersionID, capabilityCode string) (Explanation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	explanation, ok := m.explanations[levelKey{
		OrganisationID:     organisationID,
		PersonID:           personID,
		FrameworkVersionID: frameworkVersionID,
		CapabilityCode:     capabilityCode,
	}]
	if !ok {
		return Explanation{}, ErrNotFound
	}
	return explanation, nil
}

func (m *Memory) GetByID(_ context.Context, organisationID, explanationID string) (Explanation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.byID[explanationID]
	if !ok || key.OrganisationID != organisationID {
		return Explanation{}, ErrNotFound
	}
	return m.explanations[key], nil
}

func (m *Memory) Override(_ context.Context, input OverrideInput) (Override, error) {
	reason := strings.TrimSpace(input.Reason)
	if input.OrganisationID == "" || input.ExplanationID == "" || input.ActorPersonID == "" || reason == "" {
		return Override{}, ErrInvalid
	}
	if err := validateRequired(input.NewValue); err != nil {
		return Override{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.byID[input.ExplanationID]
	if !ok || key.OrganisationID != input.OrganisationID {
		return Override{}, ErrNotFound
	}
	explanation := m.explanations[key]
	previous := explanation.CurrentLevel
	if rows := m.overrides[explanation.ID]; len(rows) > 0 {
		previous = rows[len(rows)-1].NewValue
	}
	id, err := newID()
	if err != nil {
		return Override{}, err
	}
	row := Override{
		ID:             id,
		OrganisationID: input.OrganisationID,
		ExplanationID:  explanation.ID,
		ActorPersonID:  input.ActorPersonID,
		Reason:         reason,
		PreviousValue:  previous,
		NewValue:       input.NewValue,
		CreatedAt:      time.Now().UTC(),
	}
	m.overrides[explanation.ID] = append(m.overrides[explanation.ID], row)
	return row, nil
}

func (m *Memory) appendAttempt(key levelKey, source string, percentage int, evidence []string, required Tenths, createdAt time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	m.attempts[key] = append(m.attempts[key], Attempt{
		ID:                 id,
		OrganisationID:     key.OrganisationID,
		PersonID:           key.PersonID,
		FrameworkVersionID: key.FrameworkVersionID,
		CapabilityCode:     key.CapabilityCode,
		Source:             source,
		Percentage:         percentage,
		EvidenceIDs:        evidence,
		RequiredLevel:      required,
		CreatedAt:          createdAt,
	})
	return nil
}

func (m *Memory) saveFold(key levelKey, columns Sources, required Tenths) (Explanation, error) {
	sources, requiredLevel := fold(m.attempts[key], columns, required)
	explanation := Explain(sources, key.FrameworkVersionID, requiredLevel)
	if existing, ok := m.explanations[key]; ok {
		explanation.ID = existing.ID
	} else {
		id, err := newID()
		if err != nil {
			return Explanation{}, err
		}
		explanation.ID = id
	}
	explanation.OrganisationID = key.OrganisationID
	explanation.PersonID = key.PersonID
	explanation.CapabilityCode = key.CapabilityCode
	m.explanations[key] = explanation
	m.byID[explanation.ID] = key
	return explanation, nil
}

func (m *Memory) attemptsFor(organisationID, personID, frameworkVersionID, capabilityCode string) []Attempt {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.attempts[levelKey{
		OrganisationID:     organisationID,
		PersonID:           personID,
		FrameworkVersionID: frameworkVersionID,
		CapabilityCode:     capabilityCode,
	}]
	copied := make([]Attempt, len(rows))
	copy(copied, rows)
	return copied
}
