package roleprofile

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

func (m *Memory) AssignApproval(_ context.Context, organisationID, versionID, actorID, assigneeID string) (ApprovalAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return ApprovalAssignment{}, ErrNotFound
	}
	if version.Status != StatusInReview || version.WorkflowStage == "" {
		return ApprovalAssignment{}, ErrImmutable
	}
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ApprovalAssignment{}, ErrInvalid
	}
	if assignmentForStage(m.assignments, versionID, version.WorkflowStage).ID != "" {
		return ApprovalAssignment{}, ErrDuplicate
	}
	id, err := newID()
	if err != nil {
		return ApprovalAssignment{}, err
	}
	created := ApprovalAssignment{
		ID:                 id,
		OrganisationID:     organisationID,
		ProfileVersionID:   versionID,
		AssigneePersonID:   assigneeID,
		AssignedByPersonID: actorID,
		CreatedAt:          clock(),
		StageKey:           version.WorkflowStage,
	}
	m.assignments = append(m.assignments, created)
	if err := m.audit(organisationID, version.ProfileID, versionID, actorID, ActionApprovalAssigned, created.CreatedAt); err != nil {
		return ApprovalAssignment{}, err
	}
	return created, nil
}

func (m *Memory) ListApprovals(_ context.Context, organisationID, versionID, actorID string, roles []string) (ApprovalRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return ApprovalRecord{}, ErrNotFound
	}
	return approvalView(*version, m.stepsFor(organisationID, versionID), m.assignments, m.decisions, actorID, roles), nil
}

func (m *Memory) DecideApproval(_ context.Context, organisationID, versionID, assignmentID, actorID, decision, comment string, expected int, privileged bool, roles []string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	saved := m.capture()
	version, err := m.decideLocked(organisationID, versionID, assignmentID, actorID, decision, comment, expected, privileged, roles)
	if err != nil {
		m.restore(saved)
		return Version{}, err
	}
	return version, nil
}

func (m *Memory) decideLocked(organisationID, versionID, assignmentID, actorID, decision, comment string, expected int, privileged bool, roles []string) (Version, error) {
	version, err := m.guard(organisationID, versionID, expected, StatusInReview)
	if err != nil {
		return Version{}, err
	}
	steps := m.stepsFor(organisationID, versionID)
	assignment, stage, err := planDecision(*version, steps, m.assignments, m.decisions, assignmentID, actorID, decision, privileged, roles)
	if err != nil {
		return Version{}, err
	}
	if assignment.ID == "" {
		created, err := m.addAssignment(version, actorID, actorID, stage)
		if err != nil {
			return Version{}, err
		}
		assignment = created
	}
	if err := m.recordDecision(version, assignment.ID, actorID, decision, comment, stage); err != nil {
		return Version{}, err
	}
	switch strings.TrimSpace(decision) {
	case DecisionApproved:
		if following, more := nextStep(steps, stage); more {
			version.WorkflowStage = following.StageKey
			version.Revision++
			if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionStageApproved, clock()); err != nil {
				return Version{}, err
			}
			break
		}
		if err := m.publishLocked(organisationID, version, actorID); err != nil {
			return Version{}, err
		}
	case DecisionChanges:
		version.Status = StatusDraft
		version.WorkflowStage = ""
		version.Revision++
		if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionChangesRequested, clock()); err != nil {
			return Version{}, err
		}
	case DecisionRejected:
		version.Revision++
		if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionRejected, clock()); err != nil {
			return Version{}, err
		}
	}
	return cloneVersion(*version), nil
}

func (m *Memory) addAssignment(version *Version, actorID, assigneeID, stage string) (ApprovalAssignment, error) {
	id, err := newID()
	if err != nil {
		return ApprovalAssignment{}, err
	}
	created := ApprovalAssignment{
		ID:                 id,
		OrganisationID:     version.OrganisationID,
		ProfileVersionID:   version.ID,
		AssigneePersonID:   assigneeID,
		AssignedByPersonID: actorID,
		CreatedAt:          clock(),
		StageKey:           stage,
	}
	m.assignments = append(m.assignments, created)
	return created, nil
}

func (m *Memory) recordDecision(version *Version, assignmentID, actorID, decision, comment, stage string) error {
	decision = strings.TrimSpace(decision)
	comment = strings.TrimSpace(comment)
	if !validDecision(decision) {
		return ErrInvalid
	}
	if decision != DecisionApproved && comment == "" {
		return ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return err
	}
	m.decisions = append(m.decisions, ApprovalDecision{
		ID:               id,
		OrganisationID:   version.OrganisationID,
		ProfileVersionID: version.ID,
		AssignmentID:     assignmentID,
		ActorPersonID:    actorID,
		Decision:         decision,
		Comment:          comment,
		CreatedAt:        clock(),
		StageKey:         stage,
		SubmissionCycle:  version.SubmissionCycle,
	})
	return nil
}

func (m *Memory) assignment(organisationID, versionID, assignmentID string) (ApprovalAssignment, bool) {
	for _, assignment := range m.assignments {
		if assignment.OrganisationID == organisationID && assignment.ProfileVersionID == versionID && assignment.ID == assignmentID {
			return assignment, true
		}
	}
	return ApprovalAssignment{}, false
}

func (m *Memory) stagesFor(organisationID string) []ApprovalPathStage {
	if m.paths != nil {
		if stages, ok := m.paths[organisationID]; ok && len(stages) > 0 {
			return copyPath(stages)
		}
	}
	return defaultApprovalPath()
}

func (m *Memory) stepsFor(organisationID, versionID string) []WorkflowStep {
	matched := make([]WorkflowStep, 0)
	for _, step := range m.steps {
		if step.OrganisationID == organisationID && step.ProfileVersionID == versionID {
			matched = append(matched, step)
		}
	}
	return matched
}

func (m *Memory) beginCycle(version *Version, actorID string) error {
	stages := m.stagesFor(version.OrganisationID)
	built := make([]WorkflowStep, 0, len(stages))
	for _, stage := range stages {
		id, err := newID()
		if err != nil {
			return err
		}
		built = append(built, WorkflowStep{
			ID:               id,
			OrganisationID:   version.OrganisationID,
			ProfileVersionID: version.ID,
			StageKey:         stage.StageKey,
			Label:            stage.Label,
			RequiredRole:     stage.RequiredRole,
			Position:         stage.Position,
		})
	}
	retained := make([]WorkflowStep, 0, len(m.steps))
	for _, step := range m.steps {
		if step.OrganisationID == version.OrganisationID && step.ProfileVersionID == version.ID {
			continue
		}
		retained = append(retained, step)
	}
	m.steps = append(retained, built...)
	now := clock()
	version.WorkflowStage = stages[0].StageKey
	version.SubmittedAt = &now
	version.SubmittedByPersonID = actorID
	version.SubmissionCycle++
	return nil
}

func (m *Memory) GetApprovalPath(_ context.Context, organisationID string) ([]ApprovalPathStage, error) {
	if organisationID == "" {
		return nil, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stagesFor(organisationID), nil
}

func (m *Memory) SaveApprovalPath(_ context.Context, organisationID, actorID string, stages []ApprovalPathStage) error {
	if organisationID == "" || actorID == "" {
		return ErrInvalid
	}
	path, err := normaliseApprovalPath(stages)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paths == nil {
		m.paths = map[string][]ApprovalPathStage{}
	}
	m.paths[organisationID] = path
	return nil
}

func (m *Memory) Resubmit(_ context.Context, organisationID, versionID, actorID string, expected int, roles []string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.guard(organisationID, versionID, expected, StatusInReview)
	if err != nil {
		return Version{}, err
	}
	if !canResubmitVersion(*version, actorID, roles, m.decisions) {
		return Version{}, ErrForbidden
	}
	if err := m.beginCycle(version, actorID); err != nil {
		return Version{}, err
	}
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionResubmitted, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) ListApprovalQueue(_ context.Context, organisationID, personID, view string, roles []string) ([]ApprovalQueueItem, error) {
	if organisationID == "" || personID == "" || !validQueue(view) {
		return nil, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]ApprovalQueueItem, 0)
	for _, version := range m.versions {
		if version.OrganisationID != organisationID {
			continue
		}
		profile, ok := m.identity(organisationID, version.ProfileID)
		if !ok {
			continue
		}
		record := approvalView(version, m.stepsFor(organisationID, version.ID), m.assignments, m.decisions, personID, roles)
		if !queueMatch(view, version, personID, m.decisions, record) {
			continue
		}
		items = append(items, ApprovalQueueItem{
			ProfileID:        version.ProfileID,
			RoleName:         profile.RoleName,
			Department:       version.BusinessUnit,
			VersionID:        version.ID,
			VersionNumber:    version.VersionNumber,
			Revision:         version.Revision,
			Status:           version.Status,
			WorkflowStage:    version.WorkflowStage,
			StageLabel:       record.StageLabel,
			SubmittedAt:      copyTime(version.SubmittedAt),
			AssigneePersonID: record.AssigneePersonID,
			Outstanding:      record.Outstanding,
		})
	}
	return items, nil
}

func queueMatch(view string, version Version, personID string, decisions []ApprovalDecision, record ApprovalRecord) bool {
	switch view {
	case QueueAwaiting:
		return record.CanDecide
	case QueueSubmitted:
		return version.SubmittedByPersonID == personID
	case QueueReviewed:
		for _, decision := range decisions {
			if decision.ProfileVersionID == version.ID && decision.ActorPersonID == personID {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func validQueue(view string) bool {
	switch view {
	case QueueAwaiting, QueueSubmitted, QueueReviewed:
		return true
	default:
		return false
	}
}

type memoryState struct {
	versions     []Version
	assignments  []ApprovalAssignment
	decisions    []ApprovalDecision
	steps        []WorkflowStep
	events       []AuditEvent
	publications [][]byte
}

func (m *Memory) capture() memoryState {
	versions := make([]Version, len(m.versions))
	for i := range m.versions {
		versions[i] = cloneVersion(m.versions[i])
	}
	publications := make([][]byte, len(m.publications))
	for i, payload := range m.publications {
		publications[i] = append([]byte(nil), payload...)
	}
	return memoryState{
		versions:     versions,
		assignments:  append([]ApprovalAssignment{}, m.assignments...),
		decisions:    append([]ApprovalDecision{}, m.decisions...),
		steps:        append([]WorkflowStep{}, m.steps...),
		events:       append([]AuditEvent{}, m.events...),
		publications: publications,
	}
}

func (m *Memory) restore(saved memoryState) {
	m.versions = saved.versions
	m.assignments = saved.assignments
	m.decisions = saved.decisions
	m.steps = saved.steps
	m.events = saved.events
	m.publications = saved.publications
}

func (s SQLStore) openReview(ctx context.Context, organisationID, versionID, actorID string, expected int, from, action string) (Version, error) {
	var version Version
	err := s.inTxWith(ctx, func(q *db.Queries, tx *sql.Tx) error {
		current, err := expectRow(ctx, q, organisationID, versionID, expected, from)
		if err != nil {
			return err
		}
		loaded, err := loadVersion(ctx, q, organisationID, versionID)
		if err != nil {
			return err
		}
		if err := readyForReview(loaded); err != nil {
			return err
		}
		stages, err := readPath(ctx, tx, organisationID)
		if err != nil {
			return err
		}
		if len(stages) == 0 {
			stages = defaultApprovalPath()
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM role_profile_workflow_steps WHERE organisation_id = $1 AND profile_version_id = $2`, organisationID, versionID); err != nil {
			return err
		}
		for _, stage := range stages {
			id, err := newID()
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO role_profile_workflow_steps (
					id, organisation_id, profile_version_id, stage_key, label, required_role, position
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, id, organisationID, versionID, stage.StageKey, stage.Label, stage.RequiredRole, stage.Position); err != nil {
				return err
			}
		}
		now := clock()
		result, err := tx.ExecContext(ctx, `
			UPDATE role_profile_versions
			SET status = $4, workflow_stage = $5, submitted_at = $6, submitted_by_person_id = $7,
				submission_cycle = submission_cycle + 1, revision = revision + 1
			WHERE organisation_id = $1 AND id = $2 AND revision = $3 AND status = $8
		`, organisationID, versionID, expected, StatusInReview, stages[0].StageKey, now, actorID, from)
		if err != nil {
			return err
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if updated == 0 {
			return ErrConflict
		}
		if err := writeAudit(ctx, q, organisationID, current.RoleProfileID, versionID, actorID, action, now); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) AssignApproval(ctx context.Context, organisationID, versionID, actorID, assigneeID string) (ApprovalAssignment, error) {
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ApprovalAssignment{}, ErrInvalid
	}
	var created ApprovalAssignment
	err := s.inTxWith(ctx, func(q *db.Queries, tx *sql.Tx) error {
		row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
		if err != nil {
			return mapWrite(err)
		}
		if row.Status != StatusInReview {
			return ErrImmutable
		}
		id, err := newID()
		if err != nil {
			return err
		}
		created = ApprovalAssignment{
			ID:                 id,
			OrganisationID:     organisationID,
			ProfileVersionID:   versionID,
			AssigneePersonID:   assigneeID,
			AssignedByPersonID: actorID,
			CreatedAt:          clock(),
			StageKey:           row.WorkflowStage,
		}
		if row.Status != StatusInReview || row.WorkflowStage == "" {
			return ErrImmutable
		}
		var existing string
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM role_profile_approval_assignments
			WHERE organisation_id = $1 AND profile_version_id = $2 AND stage_key = $3
		`, organisationID, versionID, row.WorkflowStage).Scan(&existing)
		if err == nil {
			return ErrDuplicate
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO role_profile_approval_assignments (
				id, organisation_id, profile_version_id, assignee_person_id, assigned_by_person_id, created_at, stage_key
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, created.ID, created.OrganisationID, created.ProfileVersionID, created.AssigneePersonID, created.AssignedByPersonID, created.CreatedAt, created.StageKey); err != nil {
			return mapWrite(err)
		}
		return writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionApprovalAssigned, created.CreatedAt)
	})
	return created, err
}

func (s SQLStore) ListApprovals(ctx context.Context, organisationID, versionID, actorID string, roles []string) (ApprovalRecord, error) {
	loaded, err := loadVersion(ctx, s.queries, organisationID, versionID)
	if err != nil {
		return ApprovalRecord{}, err
	}
	steps, err := readSteps(ctx, s.database, organisationID, versionID)
	if err != nil {
		return ApprovalRecord{}, err
	}
	assignments, err := readAssignments(ctx, s.database, organisationID, versionID)
	if err != nil {
		return ApprovalRecord{}, err
	}
	decisions, err := readDecisions(ctx, s.database, organisationID, versionID)
	if err != nil {
		return ApprovalRecord{}, err
	}
	return approvalView(loaded, steps, assignments, decisions, actorID, roles), nil
}

func (s SQLStore) DecideApproval(ctx context.Context, organisationID, versionID, assignmentID, actorID, decision, comment string, expected int, privileged bool, roles []string) (Version, error) {
	decision = strings.TrimSpace(decision)
	comment = strings.TrimSpace(comment)
	if !validDecision(decision) || (decision != DecisionApproved && comment == "") {
		return Version{}, ErrInvalid
	}
	var version Version
	err := s.inTxWith(ctx, func(q *db.Queries, tx *sql.Tx) error {
		if _, err := expectRow(ctx, q, organisationID, versionID, expected, StatusInReview); err != nil {
			return err
		}
		loaded, err := loadVersion(ctx, q, organisationID, versionID)
		if err != nil {
			return err
		}
		steps, err := readSteps(ctx, tx, organisationID, versionID)
		if err != nil {
			return err
		}
		assignments, err := readAssignments(ctx, tx, organisationID, versionID)
		if err != nil {
			return err
		}
		decisions, err := readDecisions(ctx, tx, organisationID, versionID)
		if err != nil {
			return err
		}
		assignment, stage, err := planDecision(loaded, steps, assignments, decisions, assignmentID, actorID, decision, privileged, roles)
		if err != nil {
			return err
		}
		if assignment.ID == "" {
			id, err := newID()
			if err != nil {
				return err
			}
			assignment = ApprovalAssignment{
				ID: id, OrganisationID: organisationID, ProfileVersionID: versionID,
				AssigneePersonID: actorID, AssignedByPersonID: actorID, CreatedAt: clock(), StageKey: stage,
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO role_profile_approval_assignments (
					id, organisation_id, profile_version_id, assignee_person_id, assigned_by_person_id, created_at, stage_key
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, assignment.ID, organisationID, versionID, actorID, actorID, assignment.CreatedAt, stage); err != nil {
				return mapWrite(err)
			}
		}
		id, err := newID()
		if err != nil {
			return err
		}
		recordedAt := clock()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO role_profile_approval_decisions (
				id, organisation_id, profile_version_id, assignment_id, actor_person_id, decision, comment, created_at, stage_key, submission_cycle
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, id, organisationID, versionID, assignment.ID, actorID, decision, comment, recordedAt, stage, loaded.SubmissionCycle); err != nil {
			return err
		}
		switch decision {
		case DecisionApproved:
			if following, more := nextStep(steps, stage); more {
				if err := advanceStage(ctx, q, tx, organisationID, versionID, actorID, expected, following.StageKey); err != nil {
					return err
				}
				break
			}
			if err := publishVersion(ctx, q, tx, organisationID, versionID, actorID, expected); err != nil {
				return err
			}
		case DecisionChanges:
			if err := returnToDraft(ctx, q, tx, organisationID, versionID, actorID, expected); err != nil {
				return err
			}
		default:
			if err := retargetVersion(ctx, q, tx, organisationID, versionID, actorID, expected, StatusInReview, ActionRejected); err != nil {
				return err
			}
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) GetApprovalPath(ctx context.Context, organisationID string) ([]ApprovalPathStage, error) {
	if organisationID == "" {
		return nil, ErrInvalid
	}
	stages, err := readPath(ctx, s.database, organisationID)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return defaultApprovalPath(), nil
	}
	return stages, nil
}

func (s SQLStore) SaveApprovalPath(ctx context.Context, organisationID, actorID string, stages []ApprovalPathStage) error {
	if organisationID == "" || actorID == "" {
		return ErrInvalid
	}
	path, err := normaliseApprovalPath(stages)
	if err != nil {
		return err
	}
	return s.inTxWith(ctx, func(_ *db.Queries, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM role_profile_approval_paths WHERE organisation_id = $1`, organisationID); err != nil {
			return err
		}
		for _, stage := range path {
			id, err := newID()
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO role_profile_approval_paths (id, organisation_id, stage_key, label, required_role, position)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, id, organisationID, stage.StageKey, stage.Label, stage.RequiredRole, stage.Position); err != nil {
				return mapWrite(err)
			}
		}
		return nil
	})
}

func (s SQLStore) Resubmit(ctx context.Context, organisationID, versionID, actorID string, expected int, roles []string) (Version, error) {
	loaded, err := loadVersion(ctx, s.queries, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	decisions, err := readDecisions(ctx, s.database, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if !canResubmitVersion(loaded, actorID, roles, decisions) {
		if loaded.Status != StatusInReview {
			return Version{}, ErrImmutable
		}
		return Version{}, ErrForbidden
	}
	return s.openReview(ctx, organisationID, versionID, actorID, expected, StatusInReview, ActionResubmitted)
}

func (s SQLStore) ListApprovalQueue(ctx context.Context, organisationID, personID, view string, roles []string) ([]ApprovalQueueItem, error) {
	if organisationID == "" || personID == "" || !validQueue(view) {
		return nil, ErrInvalid
	}
	rows, err := s.database.QueryContext(ctx, `
		SELECT v.id
		FROM role_profile_versions v
		WHERE v.organisation_id = $1
		ORDER BY v.created_at ASC, v.id ASC
	`, organisationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]ApprovalQueueItem, 0)
	for _, id := range ids {
		loaded, err := loadVersion(ctx, s.queries, organisationID, id)
		if err != nil {
			return nil, err
		}
		profile, err := readProfile(ctx, s.queries, organisationID, loaded.ProfileID)
		if err != nil {
			return nil, err
		}
		steps, err := readSteps(ctx, s.database, organisationID, id)
		if err != nil {
			return nil, err
		}
		assignments, err := readAssignments(ctx, s.database, organisationID, id)
		if err != nil {
			return nil, err
		}
		decisions, err := readDecisions(ctx, s.database, organisationID, id)
		if err != nil {
			return nil, err
		}
		record := approvalView(loaded, steps, assignments, decisions, personID, roles)
		if !queueMatch(view, loaded, personID, decisions, record) {
			continue
		}
		items = append(items, ApprovalQueueItem{
			ProfileID: loaded.ProfileID, RoleName: profile.RoleName, Department: loaded.BusinessUnit,
			VersionID: loaded.ID, VersionNumber: loaded.VersionNumber, Revision: loaded.Revision,
			Status: loaded.Status, WorkflowStage: loaded.WorkflowStage, StageLabel: record.StageLabel,
			SubmittedAt: copyTime(loaded.SubmittedAt), AssigneePersonID: record.AssigneePersonID, Outstanding: record.Outstanding,
		})
	}
	return items, nil
}

func advanceStage(ctx context.Context, q *db.Queries, tx *sql.Tx, organisationID, versionID, actorID string, expected int, next string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE role_profile_versions
		SET workflow_stage = $4, revision = revision + 1
		WHERE organisation_id = $1 AND id = $2 AND revision = $3 AND status = $5
	`, organisationID, versionID, expected, next, StatusInReview)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return ErrConflict
	}
	current, err := loadVersion(ctx, q, organisationID, versionID)
	if err != nil {
		return err
	}
	return writeAudit(ctx, q, organisationID, current.ProfileID, versionID, actorID, ActionStageApproved, clock())
}

func returnToDraft(ctx context.Context, q *db.Queries, tx *sql.Tx, organisationID, versionID, actorID string, expected int) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE role_profile_versions
		SET status = $4, workflow_stage = '', revision = revision + 1
		WHERE organisation_id = $1 AND id = $2 AND revision = $3 AND status = $5
	`, organisationID, versionID, expected, StatusDraft, StatusInReview)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return ErrConflict
	}
	current, err := loadVersion(ctx, q, organisationID, versionID)
	if err != nil {
		return err
	}
	return writeAudit(ctx, q, organisationID, current.ProfileID, versionID, actorID, ActionChangesRequested, clock())
}

type rowQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func readPath(ctx context.Context, query rowQuery, organisationID string) ([]ApprovalPathStage, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT stage_key, label, required_role, position
		FROM role_profile_approval_paths
		WHERE organisation_id = $1
		ORDER BY position ASC
	`, organisationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stages := make([]ApprovalPathStage, 0)
	for rows.Next() {
		var stage ApprovalPathStage
		if err := rows.Scan(&stage.StageKey, &stage.Label, &stage.RequiredRole, &stage.Position); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}

func readSteps(ctx context.Context, query rowQuery, organisationID, versionID string) ([]WorkflowStep, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT id, organisation_id, profile_version_id, stage_key, label, required_role, position
		FROM role_profile_workflow_steps
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY position ASC, id ASC
	`, organisationID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	steps := make([]WorkflowStep, 0)
	for rows.Next() {
		var step WorkflowStep
		if err := rows.Scan(&step.ID, &step.OrganisationID, &step.ProfileVersionID, &step.StageKey, &step.Label, &step.RequiredRole, &step.Position); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func readAssignments(ctx context.Context, query rowQuery, organisationID, versionID string) ([]ApprovalAssignment, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT id, organisation_id, profile_version_id, assignee_person_id, assigned_by_person_id, created_at, stage_key
		FROM role_profile_approval_assignments
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY created_at ASC, id ASC
	`, organisationID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matched := make([]ApprovalAssignment, 0)
	for rows.Next() {
		var row ApprovalAssignment
		if err := rows.Scan(&row.ID, &row.OrganisationID, &row.ProfileVersionID, &row.AssigneePersonID, &row.AssignedByPersonID, &row.CreatedAt, &row.StageKey); err != nil {
			return nil, err
		}
		matched = append(matched, row)
	}
	return matched, rows.Err()
}

func readDecisions(ctx context.Context, query rowQuery, organisationID, versionID string) ([]ApprovalDecision, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT id, organisation_id, profile_version_id, assignment_id, actor_person_id, decision, comment, created_at, stage_key, submission_cycle
		FROM role_profile_approval_decisions
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY created_at ASC, id ASC
	`, organisationID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matched := make([]ApprovalDecision, 0)
	for rows.Next() {
		var row ApprovalDecision
		if err := rows.Scan(&row.ID, &row.OrganisationID, &row.ProfileVersionID, &row.AssignmentID, &row.ActorPersonID, &row.Decision, &row.Comment, &row.CreatedAt, &row.StageKey, &row.SubmissionCycle); err != nil {
			return nil, err
		}
		matched = append(matched, row)
	}
	return matched, rows.Err()
}
