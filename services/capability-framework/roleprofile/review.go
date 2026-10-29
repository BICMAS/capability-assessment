package roleprofile

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

func applySkillEdit(current SkillAssignment, edit SkillEdit) (SkillAssignment, []string, error) {
	if err := validSkillEdit(edit); err != nil {
		return SkillAssignment{}, nil, err
	}
	if tooLong(strings.TrimSpace(edit.Definition), 4000) || tooLong(strings.TrimSpace(edit.Rationale), 4000) {
		return SkillAssignment{}, nil, ErrInvalid
	}
	next := current
	fields := make([]string, 0)
	definition := strings.TrimSpace(edit.Definition)
	rationale := strings.TrimSpace(edit.Rationale)
	if definition != current.Definition {
		next.Definition = definition
		fields = append(fields, "definition")
	}
	if edit.RequiredProficiency != current.RequiredProficiency {
		next.RequiredProficiency = edit.RequiredProficiency
		fields = append(fields, "requiredProficiency")
	}
	if edit.Priority != current.Priority {
		next.Priority = edit.Priority
		fields = append(fields, "priority")
	}
	if rationale != current.Rationale {
		next.Rationale = rationale
		fields = append(fields, "rationale")
	}
	if edit.Category != nil {
		category := strings.TrimSpace(*edit.Category)
		if !validCategory(category) {
			return SkillAssignment{}, nil, ErrInvalid
		}
		if category != current.Category {
			next.Category = category
			fields = append(fields, "category")
		}
	}
	if edit.DisplayName != nil {
		name := strings.TrimSpace(*edit.DisplayName)
		if tooLong(name, 160) {
			return SkillAssignment{}, nil, ErrInvalid
		}
		if name == strings.TrimSpace(current.SkillName) {
			name = ""
		}
		if name != current.DisplayName {
			next.DisplayName = name
			fields = append(fields, "displayName")
		}
	}
	if edit.Criteria != nil {
		criteria := strings.TrimSpace(*edit.Criteria)
		if tooLong(criteria, 2000) {
			return SkillAssignment{}, nil, ErrInvalid
		}
		if criteria != current.Criteria {
			next.Criteria = criteria
			fields = append(fields, "criteria")
		}
	}
	if len(fields) > 0 && next.Provenance == ProvenanceAISuggested {
		next.Provenance = ProvenanceHRAdjusted
		fields = append(fields, "provenance")
	}
	return next, fields, nil
}

func mergeSkillContent(keep, remove SkillAssignment) (SkillAssignment, []string) {
	next := keep
	fields := make([]string, 0)
	if strings.TrimSpace(next.Criteria) == "" && strings.TrimSpace(remove.Criteria) != "" {
		next.Criteria = strings.TrimSpace(remove.Criteria)
		fields = append(fields, "criteria")
	}
	if strings.TrimSpace(next.Rationale) == "" && strings.TrimSpace(remove.Rationale) != "" {
		next.Rationale = strings.TrimSpace(remove.Rationale)
		fields = append(fields, "rationale")
	}
	if len(fields) > 0 && next.Provenance == ProvenanceAISuggested {
		next.Provenance = ProvenanceHRAdjusted
		fields = append(fields, "provenance")
	}
	return next, fields
}

func (m *Memory) UpdateSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int, edit SkillEdit) (Version, error) {
	return m.SaveEdits(ctx, organisationID, versionID, actorID, expected, []SkillRevision{{AssignmentID: assignmentID, Edit: edit}})
}

func (m *Memory) SaveEdits(_ context.Context, organisationID, versionID, actorID string, expected int, edits []SkillRevision) (Version, error) {
	for _, edit := range edits {
		if err := validSkillEdit(edit.Edit); err != nil {
			return Version{}, err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.applyEdits(organisationID, versionID, actorID, expected, edits)
}

func (m *Memory) applyEdits(organisationID, versionID, actorID string, expected int, edits []SkillRevision) (Version, error) {
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	seen := map[string]struct{}{}
	type planned struct {
		index  int
		next   SkillAssignment
		fields []string
	}
	changes := make([]planned, 0)
	for _, edit := range edits {
		if strings.TrimSpace(edit.AssignmentID) == "" {
			return Version{}, ErrInvalid
		}
		if _, ok := seen[edit.AssignmentID]; ok {
			return Version{}, ErrInvalid
		}
		seen[edit.AssignmentID] = struct{}{}
		index := skillIndex(version.Skills, edit.AssignmentID)
		if index < 0 {
			return Version{}, ErrNotFound
		}
		next, fields, err := applySkillEdit(version.Skills[index], edit.Edit)
		if err != nil {
			return Version{}, err
		}
		if len(fields) == 0 {
			continue
		}
		changes = append(changes, planned{index: index, next: next, fields: fields})
	}
	if len(changes) == 0 {
		return cloneVersion(*version), nil
	}
	now := clock()
	for _, change := range changes {
		change.next.UpdatedAt = now
		version.Skills[change.index] = change.next
		if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillUpdated, now); err != nil {
			return Version{}, err
		}
		if err := m.recordChange(version.ProfileID, version.ID, change.next.ID, actorID, change.fields, change.next.Rationale, now); err != nil {
			return Version{}, err
		}
	}
	version.Revision++
	return cloneVersion(*version), nil
}

func (m *Memory) MergeSkills(_ context.Context, organisationID, versionID, actorID, keepID, removeID string, expected int) (Version, error) {
	if strings.TrimSpace(keepID) == "" || strings.TrimSpace(removeID) == "" || keepID == removeID {
		return Version{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	keepIndex := skillIndex(version.Skills, keepID)
	removeIndex := skillIndex(version.Skills, removeID)
	if keepIndex < 0 || removeIndex < 0 {
		return Version{}, ErrNotFound
	}
	next, fields := mergeSkillContent(version.Skills[keepIndex], version.Skills[removeIndex])
	now := clock()
	if len(fields) > 0 {
		next.UpdatedAt = now
	}
	kept := make([]SkillAssignment, 0, len(version.Skills)-1)
	for _, skill := range version.Skills {
		if skill.ID == removeID {
			continue
		}
		if skill.ID == keepID {
			skill = next
		}
		skill.Position = len(kept) + 1
		kept = append(kept, skill)
	}
	version.Skills = kept
	version.Revision++
	if len(fields) > 0 {
		if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillUpdated, now); err != nil {
			return Version{}, err
		}
		if err := m.recordChange(version.ProfileID, version.ID, keepID, actorID, fields, next.Rationale, now); err != nil {
			return Version{}, err
		}
	}
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillRemoved, now); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) recordChange(profileID, versionID, assignmentID, actorID string, fields []string, rationale string, occurred time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	m.changes = append(m.changes, ReviewChange{
		ID:                id,
		ProfileVersionID:  versionID,
		SkillAssignmentID: assignmentID,
		ActorPersonID:     actorID,
		Fields:            append([]string{}, fields...),
		Rationale:         rationale,
		OccurredAt:        occurred,
	})
	_ = profileID
	return nil
}

func (s SQLStore) UpdateSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int, edit SkillEdit) (Version, error) {
	return s.SaveEdits(ctx, organisationID, versionID, actorID, expected, []SkillRevision{{AssignmentID: assignmentID, Edit: edit}})
}

func (s SQLStore) SaveEdits(ctx context.Context, organisationID, versionID, actorID string, expected int, edits []SkillRevision) (Version, error) {
	for _, edit := range edits {
		if err := validSkillEdit(edit.Edit); err != nil {
			return Version{}, err
		}
	}
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := expectRow(ctx, q, organisationID, versionID, expected, StatusDraft)
		if err != nil {
			return err
		}
		skills, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		current := map[string]SkillAssignment{}
		for _, skill := range skills {
			current[skill.ID] = assignmentFromRow(skill)
		}
		seen := map[string]struct{}{}
		type planned struct {
			next   SkillAssignment
			fields []string
		}
		changes := make([]planned, 0)
		for _, edit := range edits {
			if strings.TrimSpace(edit.AssignmentID) == "" {
				return ErrInvalid
			}
			if _, ok := seen[edit.AssignmentID]; ok {
				return ErrInvalid
			}
			seen[edit.AssignmentID] = struct{}{}
			skill, ok := current[edit.AssignmentID]
			if !ok {
				return ErrNotFound
			}
			next, fields, err := applySkillEdit(skill, edit.Edit)
			if err != nil {
				return err
			}
			if len(fields) == 0 {
				continue
			}
			changes = append(changes, planned{next: next, fields: fields})
		}
		if len(changes) == 0 {
			version, err = loadVersion(ctx, q, organisationID, versionID)
			return err
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		now := clock()
		for _, change := range changes {
			if err := q.UpdateRoleProfileSkill(ctx, db.UpdateRoleProfileSkillParams{
				ID:                  change.next.ID,
				OrganisationID:      organisationID,
				ProfileVersionID:    versionID,
				Category:            change.next.Category,
				Definition:          change.next.Definition,
				RequiredProficiency: int32(change.next.RequiredProficiency),
				Priority:            change.next.Priority,
				Rationale:           change.next.Rationale,
				Criteria:            change.next.Criteria,
				DisplayName:         change.next.DisplayName,
				Provenance:          change.next.Provenance,
				UpdatedAt:           now,
			}); err != nil {
				return mapWrite(err)
			}
			if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillUpdated, now); err != nil {
				return err
			}
			if err := writeReviewChange(ctx, q, organisationID, row.RoleProfileID, versionID, change.next.ID, actorID, change.fields, change.next.Rationale, now); err != nil {
				return err
			}
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) MergeSkills(ctx context.Context, organisationID, versionID, actorID, keepID, removeID string, expected int) (Version, error) {
	if strings.TrimSpace(keepID) == "" || strings.TrimSpace(removeID) == "" || keepID == removeID {
		return Version{}, ErrInvalid
	}
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := expectRow(ctx, q, organisationID, versionID, expected, StatusDraft)
		if err != nil {
			return err
		}
		skills, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		var keep, remove *SkillAssignment
		for _, skill := range skills {
			assignment := assignmentFromRow(skill)
			if assignment.ID == keepID {
				keep = &assignment
			}
			if assignment.ID == removeID {
				remove = &assignment
			}
		}
		if keep == nil || remove == nil {
			return ErrNotFound
		}
		next, fields := mergeSkillContent(*keep, *remove)
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		now := clock()
		if len(fields) > 0 {
			if err := q.UpdateRoleProfileSkill(ctx, db.UpdateRoleProfileSkillParams{
				ID:                  next.ID,
				OrganisationID:      organisationID,
				ProfileVersionID:    versionID,
				Category:            next.Category,
				Definition:          next.Definition,
				RequiredProficiency: int32(next.RequiredProficiency),
				Priority:            next.Priority,
				Rationale:           next.Rationale,
				Criteria:            next.Criteria,
				DisplayName:         next.DisplayName,
				Provenance:          next.Provenance,
				UpdatedAt:           now,
			}); err != nil {
				return mapWrite(err)
			}
			if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillUpdated, now); err != nil {
				return err
			}
			if err := writeReviewChange(ctx, q, organisationID, row.RoleProfileID, versionID, next.ID, actorID, fields, next.Rationale, now); err != nil {
				return err
			}
		}
		deleted, err := q.DeleteRoleProfileSkill(ctx, db.DeleteRoleProfileSkillParams{ID: removeID, OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		if deleted == 0 {
			return ErrNotFound
		}
		remaining, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		for i, skill := range remaining {
			if err := q.UpdateRoleProfileSkillPosition(ctx, db.UpdateRoleProfileSkillPositionParams{
				ID:               skill.ID,
				OrganisationID:   organisationID,
				ProfileVersionID: versionID,
				Position:         int32(i + 1),
				UpdatedAt:        now,
			}); err != nil {
				return err
			}
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillRemoved, now); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func assignmentFromRow(row db.ListRoleProfileSkillsRow) SkillAssignment {
	return SkillAssignment{
		ID:                  row.ID,
		SkillID:             row.SkillID,
		SkillName:           row.SkillName,
		Category:            row.Category,
		Definition:          row.Definition,
		RequiredProficiency: int(row.RequiredProficiency),
		Priority:            row.Priority,
		Rationale:           row.Rationale,
		Criteria:            row.Criteria,
		DisplayName:         row.DisplayName,
		Provenance:          row.Provenance,
		Position:            int(row.Position),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func writeReviewChange(ctx context.Context, q *db.Queries, organisationID, profileID, versionID, assignmentID, actorID string, fields []string, rationale string, occurred time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return q.InsertRoleProfileReviewChange(ctx, db.InsertRoleProfileReviewChangeParams{
		ID:                id,
		OrganisationID:    organisationID,
		RoleProfileID:     profileID,
		ProfileVersionID:  versionID,
		SkillAssignmentID: assignmentID,
		ActorPersonID:     actorID,
		Fields:            string(encoded),
		Rationale:         rationale,
		OccurredAt:        occurred,
	})
}

func reviewChanges(rows []db.ListRoleProfileReviewChangesRow) ([]ReviewChange, error) {
	changes := make([]ReviewChange, 0, len(rows))
	for _, row := range rows {
		var fields []string
		if err := json.Unmarshal([]byte(row.Fields), &fields); err != nil {
			return nil, err
		}
		if fields == nil {
			fields = []string{}
		}
		changes = append(changes, ReviewChange{
			ID:                row.ID,
			ProfileVersionID:  row.ProfileVersionID,
			SkillAssignmentID: row.SkillAssignmentID,
			ActorPersonID:     row.ActorPersonID,
			Fields:            fields,
			Rationale:         row.Rationale,
			OccurredAt:        row.OccurredAt,
		})
	}
	return changes, nil
}
