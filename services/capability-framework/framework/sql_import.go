package framework

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/csvimport"
	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

func (s SQLStore) ApplyImport(ctx context.Context, organisationID, objectKey string, parsed csvimport.ParseResult) (ImportResult, error) {
	result := ImportResult{
		ObjectKey: objectKey,
		RowErrors: mapRowErrors(parsed.RowErrors),
		Proposals: []Proposal{},
	}
	if len(parsed.Rows) == 0 {
		return result, nil
	}
	version, err := s.CreateDraft(ctx, organisationID, importedFrameworkName(parsed.Rows[0].Department))
	if err != nil {
		return ImportResult{}, err
	}
	importID, err := newID()
	if err != nil {
		return ImportResult{}, err
	}
	createdAt := time.Now().UTC().Truncate(time.Second)
	if err := s.queries.InsertFrameworkImport(ctx, db.InsertFrameworkImportParams{
		ID:                 importID,
		OrganisationID:     organisationID,
		FrameworkVersionID: sql.NullString{String: version.ID, Valid: true},
		ObjectKey:          objectKey,
		CreatedAt:          createdAt,
	}); err != nil {
		return ImportResult{}, err
	}
	for _, row := range parsed.Rows {
		rowID, err := newID()
		if err != nil {
			return ImportResult{}, err
		}
		if err := s.queries.InsertImportRow(ctx, db.InsertImportRowParams{
			ID:                    rowID,
			OrganisationID:        organisationID,
			ImportID:              importID,
			FrameworkVersionID:    version.ID,
			RowNumber:             int32(row.RowNumber),
			FrameworkVersionLabel: row.FrameworkVersion,
			Department:            row.Department,
			RoleCode:              row.RoleCode,
			RoleName:              row.RoleName,
			CapabilityCode:        row.CapabilityCode,
			CapabilityName:        row.CapabilityName,
			CapabilityType:        row.CapabilityType,
			Definition:            row.Definition,
			RequiredLevel:         row.RequiredLevel,
			ProficiencyLevels:     row.ProficiencyLevels,
			BehaviouralIndicators: row.BehaviouralIndicators,
			AssessmentMethods:     row.AssessmentMethods,
			LearningAssetIds:      row.LearningAssetIDs,
			PracticeScenarioIds:   row.PracticeScenarioIDs,
			WorkplaceMission:      row.WorkplaceMission,
			Weight:                row.Weight,
			GovernanceStatus:      row.GovernanceStatus,
		}); err != nil {
			return ImportResult{}, err
		}
		role, err := s.AddRole(ctx, organisationID, version.ID, row.RoleName)
		if err != nil {
			return ImportResult{}, err
		}
		capability, err := s.AddCapability(ctx, organisationID, version.ID, row.CapabilityName)
		if err != nil {
			return ImportResult{}, err
		}
		if row.RequiredLevelValue == nil {
			continue
		}
		if _, err := s.AddRequirement(ctx, organisationID, version.ID, role.ID, capability.ID, *row.RequiredLevelValue); err != nil {
			return ImportResult{}, err
		}
		for _, indicator := range row.Indicators {
			if _, err := s.AddIndicator(ctx, organisationID, version.ID, capability.ID, indicator.Level, indicator.Statement); err != nil {
				return ImportResult{}, err
			}
		}
	}
	for _, rowError := range parsed.RowErrors {
		errorID, err := newID()
		if err != nil {
			return ImportResult{}, err
		}
		if err := s.queries.InsertImportRowError(ctx, db.InsertImportRowErrorParams{
			ID:                 errorID,
			OrganisationID:     organisationID,
			FrameworkVersionID: version.ID,
			RowNumber:          int32(rowError.Row),
			Message:            rowError.Message,
		}); err != nil {
			return ImportResult{}, err
		}
	}
	for _, proposal := range parsed.Proposals {
		proposalID, err := newID()
		if err != nil {
			return ImportResult{}, err
		}
		if err := s.queries.InsertReviewProposal(ctx, db.InsertReviewProposalParams{
			ID:                 proposalID,
			OrganisationID:     organisationID,
			FrameworkVersionID: version.ID,
			Kind:               proposal.Kind,
			Status:             ProposalPending,
			CapabilityCode:     proposal.CapabilityCode,
			Message:            proposal.Message,
			CreatedAt:          createdAt,
		}); err != nil {
			return ImportResult{}, err
		}
	}
	stored, err := s.GetVersion(ctx, organisationID, version.ID)
	if err != nil {
		return ImportResult{}, err
	}
	result.Version = &stored
	result.RowErrors = append([]RowError{}, stored.RowErrors...)
	result.Proposals = append([]Proposal{}, stored.Proposals...)
	return result, nil
}

func (s SQLStore) Approve(ctx context.Context, organisationID, versionID string) (Version, error) {
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(version.Status) {
		return Version{}, ErrImmutable
	}
	if version.Status != StatusDraft && version.Status != StatusInReview {
		return Version{}, ErrInvalid
	}
	for _, proposal := range version.Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	if err := s.queries.MarkVersionApproved(ctx, db.MarkVersionApprovedParams{
		ID:             versionID,
		OrganisationID: organisationID,
	}); err != nil {
		return Version{}, err
	}
	return s.GetVersion(ctx, organisationID, versionID)
}

func (s SQLStore) UpdateIndicator(ctx context.Context, organisationID, versionID, indicatorID, statement string) (Indicator, error) {
	if statement == "" {
		return Indicator{}, ErrInvalid
	}
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Indicator{}, err
	}
	if immutable(version.Status) {
		return Indicator{}, ErrImmutable
	}
	for _, capability := range version.Capabilities {
		for _, level := range capability.Levels {
			for _, indicator := range level.Indicators {
				if indicator.ID != indicatorID {
					continue
				}
				if err := s.queries.UpdateBehaviouralIndicator(ctx, db.UpdateBehaviouralIndicatorParams{
					Statement:      statement,
					ID:             indicatorID,
					OrganisationID: organisationID,
				}); err != nil {
					return Indicator{}, err
				}
				indicator.Statement = statement
				return indicator, nil
			}
		}
	}
	return Indicator{}, ErrNotFound
}

func (s SQLStore) SubmitForReview(ctx context.Context, organisationID, versionID string) (Version, error) {
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(version.Status) {
		return Version{}, ErrImmutable
	}
	if version.Status != StatusDraft {
		return Version{}, ErrInvalid
	}
	for _, proposal := range version.Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	if err := s.queries.MarkVersionInReview(ctx, db.MarkVersionInReviewParams{
		ID:             versionID,
		OrganisationID: organisationID,
	}); err != nil {
		return Version{}, err
	}
	return s.GetVersion(ctx, organisationID, versionID)
}

func (s SQLStore) DecideProposal(ctx context.Context, organisationID, versionID, proposalID, status, message string) (Proposal, error) {
	if status != ProposalAccepted && status != ProposalEdited && status != ProposalRejected {
		return Proposal{}, ErrInvalid
	}
	version, err := s.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Proposal{}, err
	}
	if immutable(version.Status) {
		return Proposal{}, ErrImmutable
	}
	current, err := s.queries.GetReviewProposal(ctx, db.GetReviewProposalParams{
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		ID:                 proposalID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Proposal{}, ErrNotFound
	}
	if err != nil {
		return Proposal{}, err
	}
	if current.Status != ProposalPending {
		return Proposal{}, ErrInvalid
	}
	if message == "" {
		message = current.Message
	}
	if err := s.queries.DecideReviewProposal(ctx, db.DecideReviewProposalParams{
		Status:             status,
		Message:            message,
		ID:                 proposalID,
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
	}); err != nil {
		return Proposal{}, err
	}
	updated, err := s.queries.GetReviewProposal(ctx, db.GetReviewProposalParams{
		OrganisationID:     organisationID,
		FrameworkVersionID: versionID,
		ID:                 proposalID,
	})
	if err != nil {
		return Proposal{}, err
	}
	return Proposal{
		ID:             updated.ID,
		Kind:           updated.Kind,
		Status:         updated.Status,
		CapabilityCode: updated.CapabilityCode,
		Message:        updated.Message,
	}, nil
}
