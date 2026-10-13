package framework

import (
	"context"

	"github.com/bicmas-academy/learn/services/capability-framework/csvimport"
)

func (m *Memory) ApplyImport(ctx context.Context, organisationID, objectKey string, parsed csvimport.ParseResult) (ImportResult, error) {
	result := ImportResult{
		ObjectKey: objectKey,
		RowErrors: mapRowErrors(parsed.RowErrors),
		Proposals: []Proposal{},
	}
	if len(parsed.Rows) == 0 {
		return result, nil
	}
	name := importedFrameworkName(parsed.Rows[0].Department)
	version, err := m.CreateDraft(ctx, organisationID, name)
	if err != nil {
		return ImportResult{}, err
	}
	for _, row := range parsed.Rows {
		role, err := m.AddRole(ctx, organisationID, version.ID, row.RoleName)
		if err != nil {
			return ImportResult{}, err
		}
		capability, err := m.AddCapability(ctx, organisationID, version.ID, row.CapabilityName)
		if err != nil {
			return ImportResult{}, err
		}
		if row.RequiredLevelValue != nil {
			if _, err := m.AddRequirement(ctx, organisationID, version.ID, role.ID, capability.ID, *row.RequiredLevelValue); err != nil {
				return ImportResult{}, err
			}
			for _, indicator := range row.Indicators {
				if _, err := m.AddIndicator(ctx, organisationID, version.ID, capability.ID, indicator.Level, indicator.Statement); err != nil {
					return ImportResult{}, err
				}
			}
		}
	}
	proposals := make([]Proposal, 0, len(parsed.Proposals))
	for _, proposal := range parsed.Proposals {
		id, err := newID()
		if err != nil {
			return ImportResult{}, err
		}
		proposals = append(proposals, Proposal{
			ID:             id,
			Kind:           proposal.Kind,
			Status:         ProposalPending,
			CapabilityCode: proposal.CapabilityCode,
			Message:        proposal.Message,
		})
	}
	m.mu.Lock()
	index, err := m.versionIndex(organisationID, version.ID)
	if err != nil {
		m.mu.Unlock()
		return ImportResult{}, err
	}
	m.versions[index].RowErrors = append([]RowError{}, result.RowErrors...)
	m.versions[index].Proposals = proposals
	stored := cloneVersion(m.versions[index])
	m.mu.Unlock()
	result.Version = &stored
	result.Proposals = append([]Proposal{}, proposals...)
	return result, nil
}

func (m *Memory) Approve(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(m.versions[index].Status) {
		return Version{}, ErrImmutable
	}
	if m.versions[index].Status != StatusDraft && m.versions[index].Status != StatusInReview {
		return Version{}, ErrInvalid
	}
	for _, proposal := range m.versions[index].Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	m.versions[index].Status = StatusApproved
	return cloneVersion(m.versions[index]), nil
}

func (m *Memory) UpdateIndicator(_ context.Context, organisationID, versionID, indicatorID, statement string) (Indicator, error) {
	if statement == "" {
		return Indicator{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Indicator{}, err
	}
	if immutable(m.versions[index].Status) {
		return Indicator{}, ErrImmutable
	}
	for capabilityIndex := range m.versions[index].Capabilities {
		for levelIndex := range m.versions[index].Capabilities[capabilityIndex].Levels {
			for indicatorIndex := range m.versions[index].Capabilities[capabilityIndex].Levels[levelIndex].Indicators {
				indicator := &m.versions[index].Capabilities[capabilityIndex].Levels[levelIndex].Indicators[indicatorIndex]
				if indicator.ID != indicatorID {
					continue
				}
				indicator.Statement = statement
				return *indicator, nil
			}
		}
	}
	return Indicator{}, ErrNotFound
}

func (m *Memory) SubmitForReview(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if immutable(m.versions[index].Status) {
		return Version{}, ErrImmutable
	}
	if m.versions[index].Status != StatusDraft {
		return Version{}, ErrInvalid
	}
	for _, proposal := range m.versions[index].Proposals {
		if proposal.Status == ProposalPending {
			return Version{}, ErrPending
		}
	}
	m.versions[index].Status = StatusInReview
	return cloneVersion(m.versions[index]), nil
}

func (m *Memory) DecideProposal(_ context.Context, organisationID, versionID, proposalID, status, message string) (Proposal, error) {
	if status != ProposalAccepted && status != ProposalEdited && status != ProposalRejected {
		return Proposal{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	index, err := m.versionIndex(organisationID, versionID)
	if err != nil {
		return Proposal{}, err
	}
	if immutable(m.versions[index].Status) {
		return Proposal{}, ErrImmutable
	}
	for i, proposal := range m.versions[index].Proposals {
		if proposal.ID != proposalID {
			continue
		}
		if proposal.Status != ProposalPending {
			return Proposal{}, ErrInvalid
		}
		m.versions[index].Proposals[i].Status = status
		if message != "" {
			m.versions[index].Proposals[i].Message = message
		}
		return m.versions[index].Proposals[i], nil
	}
	return Proposal{}, ErrNotFound
}

func mapRowErrors(errors []csvimport.RowError) []RowError {
	mapped := make([]RowError, 0, len(errors))
	for _, rowError := range errors {
		mapped = append(mapped, RowError{Row: rowError.Row, Message: rowError.Message})
	}
	return mapped
}
