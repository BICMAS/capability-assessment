package pathway

import (
	"context"
	"strings"
	"sync"
)

type Memory struct {
	baseURL         string
	mu              sync.Mutex
	mappings        map[mappingKey]Mapping
	pathways        map[pathwayKey]Pathway
	byItem          map[string]pathwayKey
	completions     map[string]CompletionInput
	resources       []Resource
	recommendations []LearningRecommendation
	practice        []PracticeRecord
}

type mappingKey struct {
	OrganisationID string
	CapabilityID   string
	TargetLevel    int
}

type pathwayKey struct {
	OrganisationID string
	PersonID       string
	CapabilityID   string
}

func NewMemory(baseURL string) *Memory {
	return &Memory{
		baseURL:         baseURL,
		mappings:        map[mappingKey]Mapping{},
		pathways:        map[pathwayKey]Pathway{},
		byItem:          map[string]pathwayKey{},
		completions:     map[string]CompletionInput{},
		resources:       []Resource{},
		recommendations: []LearningRecommendation{},
		practice:        []PracticeRecord{},
	}
}

func (m *Memory) MapCourse(_ context.Context, input MappingInput) (Mapping, error) {
	if err := validateMapping(input); err != nil {
		return Mapping{}, err
	}
	key := mappingKey{OrganisationID: input.OrganisationID, CapabilityID: input.CapabilityID, TargetLevel: input.TargetLevel}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.mappings[key]; ok {
		return existing, nil
	}
	id, err := newID()
	if err != nil {
		return Mapping{}, err
	}
	row := Mapping{
		ID:               id,
		OrganisationID:   input.OrganisationID,
		ExternalCourseID: input.ExternalCourseID,
		CapabilityID:     input.CapabilityID,
		TargetLevel:      input.TargetLevel,
		Title:            input.Title,
	}
	m.mappings[key] = row
	return row, nil
}

func (m *Memory) CreatePathway(_ context.Context, input PathwayInput) (Pathway, error) {
	if err := validatePathway(input); err != nil {
		return Pathway{}, err
	}
	key := pathwayKey{OrganisationID: input.OrganisationID, PersonID: input.PersonID, CapabilityID: input.CapabilityID}
	mappingKey := mappingKey{OrganisationID: input.OrganisationID, CapabilityID: input.CapabilityID, TargetLevel: input.TargetLevel}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.pathways[key]; ok {
		return existing, nil
	}
	mapping, ok := m.mappings[mappingKey]
	if !ok {
		return Pathway{}, ErrNotFound
	}
	pathway, err := m.build(key, mapping)
	if err != nil {
		return Pathway{}, err
	}
	m.pathways[key] = pathway
	for _, item := range pathway.Items {
		m.byItem[item.ID] = key
	}
	return pathway, nil
}

func (m *Memory) GetPathway(_ context.Context, organisationID, personID, capabilityID string) (Pathway, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pathway, ok := m.pathways[pathwayKey{OrganisationID: organisationID, PersonID: personID, CapabilityID: capabilityID}]
	if !ok {
		return Pathway{}, ErrNotFound
	}
	return pathway, nil
}

func (m *Memory) Complete(_ context.Context, input CompletionInput) (Pathway, bool, error) {
	if input.OrganisationID == "" || input.PersonID == "" || input.PlanItemID == "" || input.CompletedAt.IsZero() {
		return Pathway{}, false, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.byItem[input.PlanItemID]
	if !ok || key.OrganisationID != input.OrganisationID {
		return Pathway{}, false, ErrNotFound
	}
	pathway := m.pathways[key]
	if pathway.PersonID != input.PersonID {
		return Pathway{}, false, ErrInvalid
	}
	index := -1
	for i := range pathway.Items {
		if pathway.Items[i].ID == input.PlanItemID {
			index = i
			break
		}
	}
	if index < 0 || pathway.Items[index].Stage != StageLearn {
		return Pathway{}, false, ErrInvalid
	}
	if _, done := m.completions[input.PlanItemID]; done {
		return pathway, false, nil
	}
	m.completions[input.PlanItemID] = input
	pathway.Items[index].Status = StatusComplete
	m.pathways[key] = pathway
	return pathway, true, nil
}

func (m *Memory) completionCount(planItemID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.completions[planItemID]; ok {
		return 1
	}
	return 0
}

func (m *Memory) build(key pathwayKey, mapping Mapping) (Pathway, error) {
	id, err := newID()
	if err != nil {
		return Pathway{}, err
	}
	learnID, err := newID()
	if err != nil {
		return Pathway{}, err
	}
	link, err := learnLink(m.baseURL, mapping.ExternalCourseID, learnID)
	if err != nil {
		return Pathway{}, err
	}
	items := []PlanItem{{
		ID:       learnID,
		Stage:    StageLearn,
		Title:    mapping.Title,
		Status:   StatusReady,
		DeepLink: link,
	}}
	for _, stage := range []struct{ stage, title string }{
		{StagePractise, "Practise"},
		{StageApply, "Apply"},
		{StageValidate, "Validate"},
	} {
		itemID, err := newID()
		if err != nil {
			return Pathway{}, err
		}
		items = append(items, PlanItem{ID: itemID, Stage: stage.stage, Title: stage.title, Status: StatusUpcoming})
	}
	return Pathway{
		ID:             id,
		OrganisationID: key.OrganisationID,
		PersonID:       key.PersonID,
		CapabilityID:   key.CapabilityID,
		TargetLevel:    mapping.TargetLevel,
		Items:          items,
	}, nil
}

func (m *Memory) FindCourse(_ context.Context, organisationID, capabilityID string, targetLevel int) (Mapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mapping, ok := m.mappings[mappingKey{OrganisationID: organisationID, CapabilityID: capabilityID, TargetLevel: targetLevel}]
	if !ok {
		return Mapping{}, ErrNotFound
	}
	return mapping, nil
}

func (m *Memory) SaveResource(_ context.Context, input ResourceInput) (Resource, error) {
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = append(m.resources, row)
	return row, nil
}

func (m *Memory) ListResources(_ context.Context, organisationID string) ([]Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]Resource, 0)
	for _, row := range m.resources {
		if row.OrganisationID == organisationID {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (m *Memory) SaveRecommendation(_ context.Context, input LearningRecommendation) (LearningRecommendation, error) {
	if input.OrganisationID == "" || input.PersonID == "" || input.SkillID == "" || input.Status != StatusRecommended || strings.TrimSpace(input.Activity) == "" {
		return LearningRecommendation{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.recommendations {
		if existing.OrganisationID == input.OrganisationID && existing.PersonID == input.PersonID && existing.SkillID == input.SkillID && existing.IntendedLevel == input.IntendedLevel {
			return existing, nil
		}
	}
	id, err := newID()
	if err != nil {
		return LearningRecommendation{}, err
	}
	input.ID = id
	m.recommendations = append(m.recommendations, input)
	return input, nil
}

func (m *Memory) ListRecommendations(_ context.Context, organisationID, personID string) ([]LearningRecommendation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]LearningRecommendation, 0)
	for _, row := range m.recommendations {
		if row.OrganisationID != organisationID {
			continue
		}
		if personID != "" && row.PersonID != personID {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (m *Memory) SavePractice(_ context.Context, input PracticeInput) (PracticeRecord, error) {
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.practice = append(m.practice, row)
	return row, nil
}

func (m *Memory) ListPractice(_ context.Context, organisationID, personID string) ([]PracticeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]PracticeRecord, 0)
	for _, row := range m.practice {
		if row.OrganisationID == organisationID && (personID == "" || row.PersonID == personID) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func validateMapping(input MappingInput) error {
	if input.OrganisationID == "" || strings.TrimSpace(input.ExternalCourseID) == "" || input.CapabilityID == "" || strings.TrimSpace(input.Title) == "" {
		return ErrInvalid
	}
	if input.TargetLevel < 0 || input.TargetLevel > 4 {
		return ErrInvalid
	}
	return nil
}

func validatePathway(input PathwayInput) error {
	if input.OrganisationID == "" || input.PersonID == "" || input.CapabilityID == "" {
		return ErrInvalid
	}
	if input.TargetLevel < 0 || input.TargetLevel > 4 {
		return ErrInvalid
	}
	return nil
}
