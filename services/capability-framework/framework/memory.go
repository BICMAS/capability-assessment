package framework

import (
	"context"
	"sync"
)

type Memory struct {
	mu       sync.Mutex
	versions []Version
	outbox   []OutboxEvent
}

func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) CreateDraft(_ context.Context, organisationID, name string) (Version, error) {
	if name == "" {
		return Version{}, ErrInvalid
	}
	frameworkID, err := newID()
	if err != nil {
		return Version{}, err
	}
	versionID, err := newID()
	if err != nil {
		return Version{}, err
	}
	version := Version{
		ID:             versionID,
		OrganisationID: organisationID,
		FrameworkID:    frameworkID,
		FrameworkName:  name,
		VersionNumber:  1,
		Status:         StatusDraft,
		Roles:          []Role{},
		Capabilities:   []Capability{},
		Requirements:   []Requirement{},
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions = append(m.versions, version)
	return cloneVersion(version), nil
}

func (m *Memory) ListVersions(_ context.Context, organisationID string) ([]Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matched := make([]Version, 0)
	for _, version := range m.versions {
		if version.OrganisationID == organisationID {
			matched = append(matched, cloneVersion(version))
		}
	}
	return matched, nil
}

func (m *Memory) GetVersion(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	return cloneVersion(m.versions[index]), nil
}

func (m *Memory) AddRole(_ context.Context, organisationID, versionID, name string) (Role, error) {
	if name == "" {
		return Role{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.editable(organisationID, versionID)
	if err != nil {
		return Role{}, err
	}
	for _, role := range m.versions[index].Roles {
		if role.Name == name {
			return role, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Role{}, err
	}
	role := Role{ID: id, Name: name}
	m.versions[index].Roles = append(m.versions[index].Roles, role)
	return role, nil
}

func (m *Memory) AddCapability(_ context.Context, organisationID, versionID, name string) (Capability, error) {
	if name == "" {
		return Capability{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.editable(organisationID, versionID)
	if err != nil {
		return Capability{}, err
	}
	for _, capability := range m.versions[index].Capabilities {
		if capability.Name == name {
			return cloneCapability(capability), nil
		}
	}
	id, err := newID()
	if err != nil {
		return Capability{}, err
	}
	capability := Capability{ID: id, Name: name, Levels: []ProficiencyLevel{}}
	m.versions[index].Capabilities = append(m.versions[index].Capabilities, capability)
	return cloneCapability(capability), nil
}

func (m *Memory) AddRoleCapability(_ context.Context, organisationID, versionID, roleID, name string, requiredLevel int) (Capability, Requirement, error) {
	if name == "" || !validLevel(requiredLevel) {
		return Capability{}, Requirement{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.editable(organisationID, versionID)
	if err != nil {
		return Capability{}, Requirement{}, err
	}
	if !m.hasRole(index, roleID) {
		return Capability{}, Requirement{}, ErrNotFound
	}
	capability := Capability{}
	for _, existing := range m.versions[index].Capabilities {
		if existing.Name == name {
			capability = cloneCapability(existing)
			break
		}
	}
	if capability.ID == "" {
		id, err := newID()
		if err != nil {
			return Capability{}, Requirement{}, err
		}
		capability = Capability{ID: id, Name: name, Levels: []ProficiencyLevel{}}
		m.versions[index].Capabilities = append(m.versions[index].Capabilities, capability)
		capability = cloneCapability(capability)
	}
	for _, requirement := range m.versions[index].Requirements {
		if requirement.RoleID == roleID && requirement.CapabilityID == capability.ID {
			return capability, requirement, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Capability{}, Requirement{}, err
	}
	requirement := Requirement{ID: id, RoleID: roleID, CapabilityID: capability.ID, RequiredLevel: requiredLevel}
	m.versions[index].Requirements = append(m.versions[index].Requirements, requirement)
	return capability, requirement, nil
}

func (m *Memory) AddRequirement(_ context.Context, organisationID, versionID, roleID, capabilityID string, requiredLevel int) (Requirement, error) {
	if !validLevel(requiredLevel) {
		return Requirement{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.editable(organisationID, versionID)
	if err != nil {
		return Requirement{}, err
	}
	if !m.hasRole(index, roleID) || !m.hasCapability(index, capabilityID) {
		return Requirement{}, ErrNotFound
	}
	for _, requirement := range m.versions[index].Requirements {
		if requirement.RoleID == roleID && requirement.CapabilityID == capabilityID {
			return requirement, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Requirement{}, err
	}
	requirement := Requirement{
		ID:            id,
		RoleID:        roleID,
		CapabilityID:  capabilityID,
		RequiredLevel: requiredLevel,
	}
	m.versions[index].Requirements = append(m.versions[index].Requirements, requirement)
	return requirement, nil
}

func (m *Memory) AddIndicator(_ context.Context, organisationID, versionID, capabilityID string, level int, statement string) (Indicator, error) {
	if statement == "" || !validLevel(level) {
		return Indicator{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.editable(organisationID, versionID)
	if err != nil {
		return Indicator{}, err
	}
	capabilityIndex := -1
	for i, capability := range m.versions[index].Capabilities {
		if capability.ID == capabilityID {
			capabilityIndex = i
			break
		}
	}
	if capabilityIndex < 0 {
		return Indicator{}, ErrNotFound
	}
	levelIndex := -1
	for i, proficiency := range m.versions[index].Capabilities[capabilityIndex].Levels {
		if proficiency.Level == level {
			levelIndex = i
			break
		}
	}
	if levelIndex < 0 {
		levelID, err := newID()
		if err != nil {
			return Indicator{}, err
		}
		m.versions[index].Capabilities[capabilityIndex].Levels = append(
			m.versions[index].Capabilities[capabilityIndex].Levels,
			ProficiencyLevel{ID: levelID, Level: level, Indicators: []Indicator{}},
		)
		levelIndex = len(m.versions[index].Capabilities[capabilityIndex].Levels) - 1
	}
	for _, indicator := range m.versions[index].Capabilities[capabilityIndex].Levels[levelIndex].Indicators {
		if indicator.Statement == statement {
			return indicator, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Indicator{}, err
	}
	indicator := Indicator{ID: id, Statement: statement}
	m.versions[index].Capabilities[capabilityIndex].Levels[levelIndex].Indicators = append(
		m.versions[index].Capabilities[capabilityIndex].Levels[levelIndex].Indicators,
		indicator,
	)
	return indicator, nil
}

func (m *Memory) PatchRole(_ context.Context, organisationID, versionID, roleID, name string) (Version, error) {
	if name == "" {
		return Version{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(m.versions[index].Status) {
		return Version{}, ErrImmutable
	}
	for i, role := range m.versions[index].Roles {
		if role.ID == roleID {
			m.versions[index].Roles[i].Name = name
			return cloneVersion(m.versions[index]), nil
		}
	}
	return Version{}, ErrNotFound
}

func (m *Memory) PublishWithEvent(ctx context.Context, organisationID, versionID string, event OutboxEvent) (Version, error) {
	if err := acceptedPublication(event, organisationID, versionID); err != nil {
		return Version{}, err
	}
	version, err := m.Publish(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	m.mu.Lock()
	m.outbox = append(m.outbox, event)
	m.mu.Unlock()
	return version, nil
}

func (m *Memory) Outbox() []OutboxEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]OutboxEvent, len(m.outbox))
	copy(copied, m.outbox)
	return copied
}

func (m *Memory) Publish(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(m.versions[index].Status) {
		return Version{}, ErrImmutable
	}
	if m.versions[index].Status != StatusDraft && m.versions[index].Status != StatusInReview && m.versions[index].Status != StatusApproved {
		return Version{}, ErrInvalid
	}
	for _, proposal := range m.versions[index].Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	m.versions[index].Status = StatusPublished
	return cloneVersion(m.versions[index]), nil
}

func (m *Memory) CreateNextDraft(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	source := m.versions[index]
	nextID, err := newID()
	if err != nil {
		return Version{}, err
	}
	nextNumber := 1
	for _, version := range m.versions {
		if version.FrameworkID == source.FrameworkID && version.VersionNumber >= nextNumber {
			nextNumber = version.VersionNumber + 1
		}
	}
	next := Version{
		ID:             nextID,
		OrganisationID: source.OrganisationID,
		FrameworkID:    source.FrameworkID,
		FrameworkName:  source.FrameworkName,
		VersionNumber:  nextNumber,
		Status:         StatusDraft,
		Roles:          make([]Role, 0, len(source.Roles)),
		Capabilities:   make([]Capability, 0, len(source.Capabilities)),
		Requirements:   make([]Requirement, 0, len(source.Requirements)),
	}
	roleIDs := map[string]string{}
	for _, role := range source.Roles {
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		roleIDs[role.ID] = id
		next.Roles = append(next.Roles, Role{ID: id, Name: role.Name})
	}
	capabilityIDs := map[string]string{}
	for _, capability := range source.Capabilities {
		capabilityID, err := newID()
		if err != nil {
			return Version{}, err
		}
		capabilityIDs[capability.ID] = capabilityID
		copied := Capability{ID: capabilityID, Name: capability.Name, Levels: make([]ProficiencyLevel, 0, len(capability.Levels))}
		for _, level := range capability.Levels {
			levelID, err := newID()
			if err != nil {
				return Version{}, err
			}
			copiedLevel := ProficiencyLevel{ID: levelID, Level: level.Level, Indicators: make([]Indicator, 0, len(level.Indicators))}
			for _, indicator := range level.Indicators {
				indicatorID, err := newID()
				if err != nil {
					return Version{}, err
				}
				copiedLevel.Indicators = append(copiedLevel.Indicators, Indicator{ID: indicatorID, Statement: indicator.Statement})
			}
			copied.Levels = append(copied.Levels, copiedLevel)
		}
		next.Capabilities = append(next.Capabilities, copied)
	}
	for _, requirement := range source.Requirements {
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		next.Requirements = append(next.Requirements, Requirement{
			ID:            id,
			RoleID:        roleIDs[requirement.RoleID],
			CapabilityID:  capabilityIDs[requirement.CapabilityID],
			RequiredLevel: requirement.RequiredLevel,
		})
	}
	m.versions = append(m.versions, next)
	return cloneVersion(next), nil
}

func (m *Memory) versionIndex(organisationID, versionID string) (int, error) {
	for i, version := range m.versions {
		if version.ID == versionID && version.OrganisationID == organisationID {
			return i, nil
		}
	}
	return 0, ErrNotFound
}

func (m *Memory) editable(organisationID, versionID string) (int, error) {
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return 0, err
	}
	if immutable(m.versions[index].Status) {
		return 0, ErrImmutable
	}
	return index, nil
}

func (m *Memory) hasRole(index int, roleID string) bool {
	for _, role := range m.versions[index].Roles {
		if role.ID == roleID {
			return true
		}
	}
	return false
}

func (m *Memory) hasCapability(index int, capabilityID string) bool {
	for _, capability := range m.versions[index].Capabilities {
		if capability.ID == capabilityID {
			return true
		}
	}
	return false
}

func cloneVersion(version Version) Version {
	cloned := version
	cloned.Roles = make([]Role, len(version.Roles))
	copy(cloned.Roles, version.Roles)
	cloned.Capabilities = make([]Capability, len(version.Capabilities))
	for i, capability := range version.Capabilities {
		cloned.Capabilities[i] = cloneCapability(capability)
	}
	cloned.Requirements = make([]Requirement, len(version.Requirements))
	copy(cloned.Requirements, version.Requirements)
	cloned.RowErrors = make([]RowError, len(version.RowErrors))
	copy(cloned.RowErrors, version.RowErrors)
	cloned.Proposals = make([]Proposal, len(version.Proposals))
	copy(cloned.Proposals, version.Proposals)
	return cloned
}

func cloneCapability(capability Capability) Capability {
	cloned := capability
	cloned.Levels = make([]ProficiencyLevel, len(capability.Levels))
	for i, level := range capability.Levels {
		cloned.Levels[i] = level
		cloned.Levels[i].Indicators = make([]Indicator, len(level.Indicators))
		copy(cloned.Levels[i].Indicators, level.Indicators)
	}
	return cloned
}
