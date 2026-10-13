package framework

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

type SQLStore struct {
	queries *db.Queries
}

func NewSQLStore(queries *db.Queries) SQLStore {
	return SQLStore{queries: queries}
}

func (s SQLStore) CreateDraft(ctx context.Context, organisationID, name string) (Version, error) {
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
	createdAt := time.Now().UTC().Truncate(time.Second)
	if err := s.queries.InsertFramework(ctx, db.InsertFrameworkParams{
		ID:             frameworkID,
		OrganisationID: organisationID,
		Name:           name,
		CreatedAt:      createdAt,
	}); err != nil {
		return Version{}, err
	}
	if err := s.queries.InsertFrameworkVersion(ctx, db.InsertFrameworkVersionParams{
		ID:             versionID,
		OrganisationID: organisationID,
		FrameworkID:    frameworkID,
		VersionNumber:  1,
		Status:         StatusDraft,
		CreatedAt:      createdAt,
	}); err != nil {
		return Version{}, err
	}
	return s.GetVersion(ctx, organisationID, versionID)
}

func (s SQLStore) ListVersions(ctx context.Context, organisationID string) ([]Version, error) {
	rows, err := s.queries.ListFrameworkVersions(ctx, organisationID)
	if err != nil {
		return nil, err
	}
	versions := make([]Version, 0, len(rows))
	for _, row := range rows {
		version, err := s.GetVersion(ctx, organisationID, row.ID)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (s SQLStore) GetVersion(ctx context.Context, organisationID, versionID string) (Version, error) {
	row, err := s.queries.GetFrameworkVersion(ctx, db.GetFrameworkVersionParams{
		OrganisationID: organisationID,
		ID:             versionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrNotFound
	}
	if err != nil {
		return Version{}, err
	}
	return s.hydrate(ctx, row)
}

func (s SQLStore) AddRole(ctx context.Context, organisationID, versionID, name string) (Role, error) {
	if name == "" {
		return Role{}, ErrInvalid
	}
	version, err := s.editable(ctx, organisationID, versionID)
	if err != nil {
		return Role{}, err
	}
	for _, role := range version.Roles {
		if role.Name == name {
			return role, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Role{}, err
	}
	if err := s.queries.InsertRole(ctx, db.InsertRoleParams{
		ID:                 id,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		Name:               name,
	}); err != nil {
		return Role{}, err
	}
	return Role{ID: id, Name: name}, nil
}

func (s SQLStore) AddCapability(ctx context.Context, organisationID, versionID, name string) (Capability, error) {
	if name == "" {
		return Capability{}, ErrInvalid
	}
	version, err := s.editable(ctx, organisationID, versionID)
	if err != nil {
		return Capability{}, err
	}
	for _, capability := range version.Capabilities {
		if capability.Name == name {
			return capability, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Capability{}, err
	}
	if err := s.queries.InsertCapability(ctx, db.InsertCapabilityParams{
		ID:                 id,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		Name:               name,
	}); err != nil {
		return Capability{}, err
	}
	return Capability{ID: id, Name: name, Levels: []ProficiencyLevel{}}, nil
}

func (s SQLStore) AddRequirement(ctx context.Context, organisationID, versionID, roleID, capabilityID string, requiredLevel int) (Requirement, error) {
	if !validLevel(requiredLevel) {
		return Requirement{}, ErrInvalid
	}
	version, err := s.editable(ctx, organisationID, versionID)
	if err != nil {
		return Requirement{}, err
	}
	if !containsRole(version, roleID) || !containsCapability(version, capabilityID) {
		return Requirement{}, ErrNotFound
	}
	for _, requirement := range version.Requirements {
		if requirement.RoleID == roleID && requirement.CapabilityID == capabilityID {
			return requirement, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Requirement{}, err
	}
	if err := s.queries.InsertRoleCapabilityRequirement(ctx, db.InsertRoleCapabilityRequirementParams{
		ID:                 id,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		RoleID:             roleID,
		CapabilityID:       capabilityID,
		RequiredLevel:      int32(requiredLevel),
	}); err != nil {
		return Requirement{}, err
	}
	return Requirement{ID: id, RoleID: roleID, CapabilityID: capabilityID, RequiredLevel: requiredLevel}, nil
}

func (s SQLStore) AddIndicator(ctx context.Context, organisationID, versionID, capabilityID string, level int, statement string) (Indicator, error) {
	if statement == "" || !validLevel(level) {
		return Indicator{}, ErrInvalid
	}
	version, err := s.editable(ctx, organisationID, versionID)
	if err != nil {
		return Indicator{}, err
	}
	var capability *Capability
	for i := range version.Capabilities {
		if version.Capabilities[i].ID == capabilityID {
			capability = &version.Capabilities[i]
			break
		}
	}
	if capability == nil {
		return Indicator{}, ErrNotFound
	}
	var levelID string
	for _, proficiency := range capability.Levels {
		if proficiency.Level == level {
			levelID = proficiency.ID
			for _, indicator := range proficiency.Indicators {
				if indicator.Statement == statement {
					return indicator, nil
				}
			}
			break
		}
	}
	if levelID == "" {
		levelID, err = s.insertLevel(ctx, organisationID, capabilityID, level)
		if err != nil {
			return Indicator{}, err
		}
	}
	id, err := newID()
	if err != nil {
		return Indicator{}, err
	}
	if err := s.queries.InsertBehaviouralIndicator(ctx, db.InsertBehaviouralIndicatorParams{
		ID:                 id,
		OrganisationID:     organisationID,
		ProficiencyLevelID: levelID,
		Statement:          statement,
	}); err != nil {
		return Indicator{}, err
	}
	return Indicator{ID: id, Statement: statement}, nil
}

func (s SQLStore) PatchRole(ctx context.Context, organisationID, versionID, roleID, name string) (Version, error) {
	if name == "" {
		return Version{}, ErrInvalid
	}
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(version.Status) {
		return Version{}, ErrImmutable
	}
	found := false
	for _, role := range version.Roles {
		if role.ID == roleID {
			found = true
			break
		}
	}
	if !found {
		return Version{}, ErrNotFound
	}
	if err := s.queries.RenameRole(ctx, db.RenameRoleParams{
		Name:               name,
		ID:                 roleID,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
	}); err != nil {
		return Version{}, err
	}
	return s.GetVersion(ctx, organisationID, versionID)
}

func (s SQLStore) Publish(ctx context.Context, organisationID, versionID string) (Version, error) {
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(version.Status) {
		return Version{}, ErrImmutable
	}
	if version.Status != StatusDraft && version.Status != StatusInReview && version.Status != StatusApproved {
		return Version{}, ErrInvalid
	}
	for _, proposal := range version.Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	if err := s.queries.PublishVersion(ctx, db.PublishVersionParams{
		ID:             versionID,
		OrganisationID: organisationID,
	}); err != nil {
		return Version{}, err
	}
	return s.GetVersion(ctx, organisationID, versionID)
}

func (s SQLStore) CreateNextDraft(ctx context.Context, organisationID, versionID string) (Version, error) {
	source, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	current, err := s.queries.MaxVersionNumber(ctx, db.MaxVersionNumberParams{
		OrganisationID: organisationID,
		FrameworkID:    source.FrameworkID,
	})
	if err != nil {
		return Version{}, err
	}
	nextID, err := newID()
	if err != nil {
		return Version{}, err
	}
	if err := s.queries.InsertFrameworkVersion(ctx, db.InsertFrameworkVersionParams{
		ID:             nextID,
		OrganisationID: organisationID,
		FrameworkID:    source.FrameworkID,
		VersionNumber:  current + 1,
		Status:         StatusDraft,
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
	}); err != nil {
		return Version{}, err
	}
	roleIDs := map[string]string{}
	for _, role := range source.Roles {
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		roleIDs[role.ID] = id
		if err := s.queries.InsertRole(ctx, db.InsertRoleParams{
			ID:                 id,
			OrganisationID:     organisationID,
			FrameworkVersionID: nextID,
			Name:               role.Name,
		}); err != nil {
			return Version{}, err
		}
	}
	capabilityIDs := map[string]string{}
	for _, capability := range source.Capabilities {
		capabilityID, err := newID()
		if err != nil {
			return Version{}, err
		}
		capabilityIDs[capability.ID] = capabilityID
		if err := s.queries.InsertCapability(ctx, db.InsertCapabilityParams{
			ID:                 capabilityID,
			OrganisationID:     organisationID,
			FrameworkVersionID: nextID,
			Name:               capability.Name,
		}); err != nil {
			return Version{}, err
		}
		for _, level := range capability.Levels {
			levelID, err := newID()
			if err != nil {
				return Version{}, err
			}
			if err := s.queries.InsertProficiencyLevel(ctx, db.InsertProficiencyLevelParams{
				ID:             levelID,
				OrganisationID: organisationID,
				CapabilityID:   capabilityID,
				Level:          int32(level.Level),
			}); err != nil {
				return Version{}, err
			}
			for _, indicator := range level.Indicators {
				indicatorID, err := newID()
				if err != nil {
					return Version{}, err
				}
				if err := s.queries.InsertBehaviouralIndicator(ctx, db.InsertBehaviouralIndicatorParams{
					ID:                 indicatorID,
					OrganisationID:     organisationID,
					ProficiencyLevelID: levelID,
					Statement:          indicator.Statement,
				}); err != nil {
					return Version{}, err
				}
			}
		}
	}
	for _, requirement := range source.Requirements {
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		if err := s.queries.InsertRoleCapabilityRequirement(ctx, db.InsertRoleCapabilityRequirementParams{
			ID:                 id,
			OrganisationID:     organisationID,
			FrameworkVersionID: nextID,
			RoleID:             roleIDs[requirement.RoleID],
			CapabilityID:       capabilityIDs[requirement.CapabilityID],
			RequiredLevel:      int32(requirement.RequiredLevel),
		}); err != nil {
			return Version{}, err
		}
	}
	return s.GetVersion(ctx, organisationID, nextID)
}

func (s SQLStore) editable(ctx context.Context, organisationID, versionID string) (Version, error) {
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(version.Status) {
		return Version{}, ErrImmutable
	}
	return version, nil
}

func (s SQLStore) insertLevel(ctx context.Context, organisationID, capabilityID string, level int) (string, error) {
	existing, err := s.queries.GetProficiencyLevel(ctx, db.GetProficiencyLevelParams{
		OrganisationID: organisationID,
		CapabilityID:   capabilityID,
		Level:          int32(level),
	})
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	if err := s.queries.InsertProficiencyLevel(ctx, db.InsertProficiencyLevelParams{
		ID:             id,
		OrganisationID: organisationID,
		CapabilityID:   capabilityID,
		Level:          int32(level),
	}); err != nil {
		return "", err
	}
	return id, nil
}

func (s SQLStore) hydrate(ctx context.Context, row db.GetFrameworkVersionRow) (Version, error) {
	params := db.ListRolesForVersionParams{OrganisationID: row.OrganisationID, FrameworkVersionID: row.ID}
	roleRows, err := s.queries.ListRolesForVersion(ctx, params)
	if err != nil {
		return Version{}, err
	}
	capabilityRows, err := s.queries.ListCapabilitiesForVersion(ctx, db.ListCapabilitiesForVersionParams(params))
	if err != nil {
		return Version{}, err
	}
	levelRows, err := s.queries.ListProficiencyLevelsForVersion(ctx, db.ListProficiencyLevelsForVersionParams(params))
	if err != nil {
		return Version{}, err
	}
	indicatorRows, err := s.queries.ListBehaviouralIndicatorsForVersion(ctx, db.ListBehaviouralIndicatorsForVersionParams(params))
	if err != nil {
		return Version{}, err
	}
	requirementRows, err := s.queries.ListRequirementsForVersion(ctx, db.ListRequirementsForVersionParams(params))
	if err != nil {
		return Version{}, err
	}

	roles := make([]Role, 0, len(roleRows))
	for _, role := range roleRows {
		roles = append(roles, Role{ID: role.ID, Name: role.Name})
	}
	capabilities := make([]Capability, 0, len(capabilityRows))
	capabilityIndex := map[string]int{}
	for _, capability := range capabilityRows {
		capabilityIndex[capability.ID] = len(capabilities)
		capabilities = append(capabilities, Capability{ID: capability.ID, Name: capability.Name, Levels: []ProficiencyLevel{}})
	}
	levelIndex := map[string][2]int{}
	for _, level := range levelRows {
		index, ok := capabilityIndex[level.CapabilityID]
		if !ok {
			continue
		}
		capabilities[index].Levels = append(capabilities[index].Levels, ProficiencyLevel{
			ID:         level.ID,
			Level:      int(level.Level),
			Indicators: []Indicator{},
		})
		levelIndex[level.ID] = [2]int{index, len(capabilities[index].Levels) - 1}
	}
	for _, indicator := range indicatorRows {
		location, ok := levelIndex[indicator.ProficiencyLevelID]
		if !ok {
			continue
		}
		capabilities[location[0]].Levels[location[1]].Indicators = append(
			capabilities[location[0]].Levels[location[1]].Indicators,
			Indicator{ID: indicator.ID, Statement: indicator.Statement},
		)
	}
	requirements := make([]Requirement, 0, len(requirementRows))
	for _, requirement := range requirementRows {
		requirements = append(requirements, Requirement{
			ID:            requirement.ID,
			RoleID:        requirement.RoleID,
			CapabilityID:  requirement.CapabilityID,
			RequiredLevel: int(requirement.RequiredLevel),
		})
	}
	errorRows, err := s.queries.ListImportRowErrorsForVersion(ctx, db.ListImportRowErrorsForVersionParams(params))
	if err != nil {
		return Version{}, err
	}
	rowErrors := make([]RowError, 0, len(errorRows))
	for _, rowError := range errorRows {
		rowErrors = append(rowErrors, RowError{Row: int(rowError.RowNumber), Message: rowError.Message})
	}
	proposalRows, err := s.queries.ListReviewProposalsForVersion(ctx, db.ListReviewProposalsForVersionParams(params))
	if err != nil {
		return Version{}, err
	}
	proposals := make([]Proposal, 0, len(proposalRows))
	for _, proposal := range proposalRows {
		proposals = append(proposals, Proposal{
			ID:             proposal.ID,
			Kind:           proposal.Kind,
			Status:         proposal.Status,
			CapabilityCode: proposal.CapabilityCode,
			Message:        proposal.Message,
		})
	}
	return Version{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		FrameworkID:    row.FrameworkID,
		FrameworkName:  row.FrameworkName,
		VersionNumber:  int(row.VersionNumber),
		Status:         row.Status,
		Roles:          roles,
		Capabilities:   capabilities,
		Requirements:   requirements,
		RowErrors:      rowErrors,
		Proposals:      proposals,
	}, nil
}

func containsRole(version Version, roleID string) bool {
	for _, role := range version.Roles {
		if role.ID == roleID {
			return true
		}
	}
	return false
}

func containsCapability(version Version, capabilityID string) bool {
	for _, capability := range version.Capabilities {
		if capability.ID == capabilityID {
			return true
		}
	}
	return false
}
