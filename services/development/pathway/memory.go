package pathway

import (
	"context"
	"strings"
	"sync"
)

type Memory struct {
	baseURL     string
	mu          sync.Mutex
	mappings    map[mappingKey]Mapping
	pathways    map[pathwayKey]Pathway
	byItem      map[string]pathwayKey
	completions map[string]CompletionInput
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
		baseURL:     baseURL,
		mappings:    map[mappingKey]Mapping{},
		pathways:    map[pathwayKey]Pathway{},
		byItem:      map[string]pathwayKey{},
		completions: map[string]CompletionInput{},
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
