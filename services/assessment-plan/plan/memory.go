package plan

import (
	"context"
	"strings"
	"sync"
)

type assignmentKey struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	RoleID             string
}

type Memory struct {
	mu          sync.Mutex
	assignments map[string]Assignment
	byKey       map[assignmentKey]string
	plans       map[string]Plan
	versions    map[string]int
}

func NewMemory() *Memory {
	return &Memory{
		assignments: map[string]Assignment{},
		byKey:       map[assignmentKey]string{},
		plans:       map[string]Plan{},
		versions:    map[string]int{},
	}
}

func (m *Memory) CreateAssignment(_ context.Context, input AssignmentInput) (Assignment, error) {
	input = trimAssignment(input)
	if err := validateAssignment(input); err != nil {
		return Assignment{}, err
	}
	key := assignmentKey{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		RoleID:             input.RoleID,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byKey[key]; ok {
		return m.assignments[id], nil
	}
	id, err := newID()
	if err != nil {
		return Assignment{}, err
	}
	row := Assignment{
		ID:                 id,
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		RoleID:             input.RoleID,
		AssignedByPersonID: input.AssignedByPersonID,
		AssignedAt:         input.AssignedAt.UTC(),
		Status:             AssignmentStatus,
		ExceptionReason:    input.ExceptionReason,
	}
	m.assignments[id] = row
	m.byKey[key] = id
	return row, nil
}

func (m *Memory) FindAssignment(_ context.Context, organisationID, personID, frameworkVersionID, roleID string) (Assignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byKey[assignmentKey{
		OrganisationID:     organisationID,
		PersonID:           personID,
		FrameworkVersionID: frameworkVersionID,
		RoleID:             roleID,
	}]
	if !ok {
		return Assignment{}, ErrNotFound
	}
	return m.assignments[id], nil
}

func (m *Memory) GetAssignment(_ context.Context, organisationID, assignmentID string) (Assignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.assignments[assignmentID]
	if !ok || row.OrganisationID != organisationID {
		return Assignment{}, ErrNotFound
	}
	return row, nil
}

func (m *Memory) CreateGeneratedPlan(_ context.Context, input PlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.AssignmentID) {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	assignment, ok := m.assignments[input.AssignmentID]
	if !ok || assignment.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	for _, existing := range m.plans {
		if existing.AssignmentID == input.AssignmentID && existing.Status == StatusGenerated {
			return Plan{}, ErrAlreadyGenerated
		}
	}
	if m.versions[input.AssignmentID] != 0 {
		return Plan{}, ErrAlreadyGenerated
	}
	id, err := newID()
	if err != nil {
		return Plan{}, err
	}
	skills, err := materialiseSkills(input.Skills)
	if err != nil {
		return Plan{}, err
	}
	row := Plan{
		ID:             id,
		OrganisationID: input.OrganisationID,
		AssignmentID:   input.AssignmentID,
		VersionNumber:  1,
		Status:         StatusGenerated,
		Skills:         skills,
	}
	m.plans[id] = row
	m.versions[input.AssignmentID] = 1
	return copyPlan(row), nil
}

func (m *Memory) CreatePlan(_ context.Context, input PlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.AssignmentID) {
		return Plan{}, ErrInvalid
	}
	if err := validateSkills(input.Skills); err != nil {
		return Plan{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	assignment, ok := m.assignments[input.AssignmentID]
	if !ok || assignment.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	id, err := newID()
	if err != nil {
		return Plan{}, err
	}
	version := m.versions[input.AssignmentID] + 1
	skills, err := materialiseSkills(input.Skills)
	if err != nil {
		return Plan{}, err
	}
	row := Plan{
		ID:             id,
		OrganisationID: input.OrganisationID,
		AssignmentID:   input.AssignmentID,
		VersionNumber:  version,
		Status:         StatusDraft,
		Skills:         skills,
	}
	m.plans[id] = row
	m.versions[input.AssignmentID] = version
	return copyPlan(row), nil
}

func (m *Memory) GetPlan(_ context.Context, organisationID, planID string) (Plan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[planID]
	if !ok || row.OrganisationID != organisationID {
		return Plan{}, ErrNotFound
	}
	return copyPlan(row), nil
}

func (m *Memory) ReplacePlan(_ context.Context, input ReplacePlanInput) (Plan, error) {
	input.Skills = trimSkills(input.Skills)
	if !present(input.OrganisationID) || !present(input.PlanID) || !validStatus(input.Status) {
		return Plan{}, ErrInvalid
	}
	if err := validateSkills(input.Skills); err != nil {
		return Plan{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[input.PlanID]
	if !ok || row.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	if row.Status == StatusApproved {
		return Plan{}, ErrImmutable
	}
	skills, err := materialiseSkills(input.Skills)
	if err != nil {
		return Plan{}, err
	}
	row.Status = input.Status
	row.Skills = skills
	m.plans[input.PlanID] = row
	return copyPlan(row), nil
}

func materialiseSkills(inputs []SkillInput) ([]PlanSkill, error) {
	skills := make([]PlanSkill, 0, len(inputs))
	for _, input := range inputs {
		id, err := newID()
		if err != nil {
			return nil, err
		}
		items := make([]AssessmentItem, 0, len(input.AssessmentItems))
		for _, item := range input.AssessmentItems {
			itemID, err := newID()
			if err != nil {
				return nil, err
			}
			items = append(items, AssessmentItem{ID: itemID, Prompt: item.Prompt})
		}
		bands := make([]RubricBand, 0, len(input.Rubric))
		for _, band := range input.Rubric {
			bands = append(bands, RubricBand{Level: band.Level, Descriptor: band.Descriptor})
		}
		skills = append(skills, PlanSkill{
			ID:                id,
			RequirementID:     input.RequirementID,
			CapabilityID:      input.CapabilityID,
			RequiredLevel:     input.RequiredLevel,
			RecommendedMethod: input.RecommendedMethod,
			Rationale:         input.Rationale,
			Instructions:      input.Instructions,
			RequiresSMEReview: input.RequiresSMEReview,
			RubricNeedsReview: input.RubricNeedsReview,
			EditedByPersonID:  input.EditedByPersonID,
			AssessmentItems:   items,
			Rubric:            bands,
		})
	}
	return skills, nil
}

func (m *Memory) EditSkill(_ context.Context, input EditSkillInput) (Plan, error) {
	input = trimSkillEdit(input)
	if err := validateSkillEdit(input); err != nil {
		return Plan{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[input.PlanID]
	if !ok || row.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	if err := ensureSkillEditable(row.Status); err != nil {
		return Plan{}, err
	}
	index := -1
	for i, skill := range row.Skills {
		if skill.ID == input.SkillID {
			index = i
			break
		}
	}
	if index < 0 {
		return Plan{}, ErrNotFound
	}
	items := make([]AssessmentItem, 0, len(input.AssessmentItems))
	for _, item := range input.AssessmentItems {
		itemID, err := newID()
		if err != nil {
			return Plan{}, err
		}
		items = append(items, AssessmentItem{ID: itemID, Prompt: item.Prompt})
	}
	bands := make([]RubricBand, 0, len(input.Rubric))
	for _, band := range input.Rubric {
		bands = append(bands, RubricBand{Level: band.Level, Descriptor: band.Descriptor})
	}
	row.Skills[index].RecommendedMethod = input.RecommendedMethod
	row.Skills[index].Instructions = input.Instructions
	row.Skills[index].EditedByPersonID = input.EditorPersonID
	row.Skills[index].AssessmentItems = items
	row.Skills[index].Rubric = bands
	m.plans[input.PlanID] = row
	return copyPlan(row), nil
}

func (m *Memory) SubmitPlan(_ context.Context, input ReviewInput) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[input.PlanID]
	if !ok || row.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	if err := ensureSubmittable(row.Status); err != nil {
		return Plan{}, err
	}
	row.Status = StatusUnderReview
	row.ChangeReason = ""
	m.plans[input.PlanID] = row
	return copyPlan(row), nil
}

func (m *Memory) RequestChanges(_ context.Context, input ReviewInput) (Plan, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) || !shortText(input.Reason, 280) {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[input.PlanID]
	if !ok || row.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	if err := ensureReviewable(row.Status); err != nil {
		return Plan{}, err
	}
	row.Status = StatusChangesRequested
	row.ChangeReason = input.Reason
	m.plans[input.PlanID] = row
	return copyPlan(row), nil
}

func (m *Memory) ApprovePlan(_ context.Context, input ReviewInput) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.ActorPersonID) || input.At.IsZero() {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[input.PlanID]
	if !ok || row.OrganisationID != input.OrganisationID {
		return Plan{}, ErrNotFound
	}
	if err := ensureApprovable(row.Status); err != nil {
		return Plan{}, err
	}
	row.Status = StatusApproved
	row.ApprovedByPersonID = input.ActorPersonID
	row.ApprovedAt = input.At.UTC()
	m.plans[input.PlanID] = row
	return copyPlan(row), nil
}

func (m *Memory) CopyApprovedPlan(_ context.Context, organisationID, planID string) (Plan, error) {
	if !present(organisationID) || !present(planID) {
		return Plan{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.plans[planID]
	if !ok || row.OrganisationID != organisationID {
		return Plan{}, ErrNotFound
	}
	if row.Status != StatusApproved {
		return Plan{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Plan{}, err
	}
	skills, err := materialiseSkills(skillInputs(row.Skills))
	if err != nil {
		return Plan{}, err
	}
	version := m.versions[row.AssignmentID] + 1
	copy := Plan{
		ID:             id,
		OrganisationID: row.OrganisationID,
		AssignmentID:   row.AssignmentID,
		VersionNumber:  version,
		Status:         StatusDraft,
		Skills:         skills,
	}
	m.plans[id] = copy
	m.versions[row.AssignmentID] = version
	return copyPlan(copy), nil
}

func skillInputs(skills []PlanSkill) []SkillInput {
	inputs := make([]SkillInput, 0, len(skills))
	for _, skill := range skills {
		items := make([]ItemInput, 0, len(skill.AssessmentItems))
		for _, item := range skill.AssessmentItems {
			items = append(items, ItemInput{Prompt: item.Prompt})
		}
		bands := make([]RubricInput, 0, len(skill.Rubric))
		for _, band := range skill.Rubric {
			bands = append(bands, RubricInput{Level: band.Level, Descriptor: band.Descriptor})
		}
		inputs = append(inputs, SkillInput{
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     skill.RequiredLevel,
			RecommendedMethod: skill.RecommendedMethod,
			Rationale:         skill.Rationale,
			Instructions:      skill.Instructions,
			RequiresSMEReview: skill.RequiresSMEReview,
			RubricNeedsReview: skill.RubricNeedsReview,
			EditedByPersonID:  skill.EditedByPersonID,
			AssessmentItems:   items,
			Rubric:            bands,
		})
	}
	return inputs
}

func trimSkillEdit(input EditSkillInput) EditSkillInput {
	input.RecommendedMethod = strings.TrimSpace(input.RecommendedMethod)
	input.Instructions = strings.TrimSpace(input.Instructions)
	for i, item := range input.AssessmentItems {
		input.AssessmentItems[i].Prompt = strings.TrimSpace(item.Prompt)
	}
	for i, band := range input.Rubric {
		input.Rubric[i].Descriptor = strings.TrimSpace(band.Descriptor)
	}
	return input
}

func copyPlan(row Plan) Plan {
	skills := make([]PlanSkill, len(row.Skills))
	for i, skill := range row.Skills {
		skill.AssessmentItems = append([]AssessmentItem(nil), skill.AssessmentItems...)
		skill.Rubric = append([]RubricBand(nil), skill.Rubric...)
		skills[i] = skill
	}
	row.Skills = skills
	return row
}
