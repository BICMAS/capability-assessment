package roleprofile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
	"github.com/bicmas-academy/learn/services/capability-framework/outbox"
)

type SQLStore struct {
	database *sql.DB
	queries  *db.Queries
}

func NewSQLStore(database *sql.DB) SQLStore {
	return SQLStore{database: database, queries: db.New(database)}
}

func (s SQLStore) CreateRole(ctx context.Context, organisationID, name string) (Role, error) {
	name = strings.TrimSpace(name)
	if organisationID == "" || name == "" {
		return Role{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Role{}, err
	}
	err = s.queries.InsertProfileRole(ctx, db.InsertProfileRoleParams{
		ID:             id,
		OrganisationID: organisationID,
		Name:           name,
		CreatedAt:      clock(),
	})
	if err != nil {
		return Role{}, mapWrite(err)
	}
	return Role{ID: id, OrganisationID: organisationID, Name: name}, nil
}

func (s SQLStore) CreateSkill(ctx context.Context, organisationID string, skill Skill) (Skill, error) {
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Definition = strings.TrimSpace(skill.Definition)
	if organisationID == "" || skill.Name == "" || skill.Definition == "" || !validCategory(skill.Category) {
		return Skill{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Skill{}, err
	}
	err = s.queries.InsertCanonicalSkill(ctx, db.InsertCanonicalSkillParams{
		ID:             id,
		OrganisationID: organisationID,
		Name:           skill.Name,
		Category:       skill.Category,
		Definition:     skill.Definition,
		CreatedAt:      clock(),
	})
	if err != nil {
		return Skill{}, mapWrite(err)
	}
	return Skill{ID: id, OrganisationID: organisationID, Name: skill.Name, Category: skill.Category, Definition: skill.Definition}, nil
}

func (s SQLStore) CreateDraft(ctx context.Context, organisationID, roleID, actorID, frameworkVersionID string) (Profile, error) {
	if organisationID == "" || actorID == "" {
		return Profile{}, ErrInvalid
	}
	var profile Profile
	err := s.inTx(ctx, func(q *db.Queries) error {
		role, err := q.GetProfileRole(ctx, db.GetProfileRoleParams{OrganisationID: organisationID, ID: roleID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if frameworkVersionID != "" {
			_, err = q.GetFrameworkVersion(ctx, db.GetFrameworkVersionParams{OrganisationID: organisationID, ID: frameworkVersionID})
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
		}
		profileID, err := newID()
		if err != nil {
			return err
		}
		versionID, err := newID()
		if err != nil {
			return err
		}
		now := clock()
		if err := q.InsertRoleProfile(ctx, db.InsertRoleProfileParams{
			ID:                profileID,
			OrganisationID:    organisationID,
			RoleID:            role.ID,
			CreatedByPersonID: actorID,
			CreatedAt:         now,
		}); err != nil {
			return mapWrite(err)
		}
		if err := q.InsertRoleProfileVersion(ctx, newVersionParams(versionID, organisationID, profileID, frameworkVersionID, 1, now)); err != nil {
			return err
		}
		if err := writeAudit(ctx, q, organisationID, profileID, versionID, actorID, ActionCreated, now); err != nil {
			return err
		}
		profile, err = readProfile(ctx, q, organisationID, profileID)
		return err
	})
	return profile, err
}

func (s SQLStore) GetProfile(ctx context.Context, organisationID, profileID string) (Profile, error) {
	return readProfile(ctx, s.queries, organisationID, profileID)
}

func (s SQLStore) UpdateContext(ctx context.Context, organisationID, versionID, actorID string, expected int, details Context) (Version, error) {
	if err := validContext(details); err != nil {
		return Version{}, err
	}
	if actorID == "" {
		return Version{}, ErrInvalid
	}
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		if _, err := expectRow(ctx, q, organisationID, versionID, expected, StatusDraft); err != nil {
			return err
		}
		updated, err := q.UpdateRoleProfileContext(ctx, db.UpdateRoleProfileContextParams{
			ID:               versionID,
			OrganisationID:   organisationID,
			Purpose:          strings.TrimSpace(details.Purpose),
			Responsibilities: strings.TrimSpace(details.Responsibilities),
			BusinessOutcomes: strings.TrimSpace(details.BusinessOutcomes),
			DecisionScope:    strings.TrimSpace(details.DecisionScope),
			Industry:         strings.TrimSpace(details.Industry),
			Region:           strings.TrimSpace(details.Region),
			TeamSize:         nullInt(details.TeamSize),
			ReportingLevel:   details.ReportingLevel,
			Revision:         int32(expected),
		})
		if err != nil {
			return mapWrite(err)
		}
		if updated == 0 {
			return ErrConflict
		}
		if err := q.DeleteRoleProfileSources(ctx, db.DeleteRoleProfileSourcesParams{OrganisationID: organisationID, ProfileVersionID: versionID}); err != nil {
			return err
		}
		for _, source := range details.Sources {
			id, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertRoleProfileSource(ctx, db.InsertRoleProfileSourceParams{
				ID:               id,
				OrganisationID:   organisationID,
				ProfileVersionID: versionID,
				Title:            strings.TrimSpace(source.Title),
				ReferenceUri:     strings.TrimSpace(source.ReferenceURI),
			}); err != nil {
				return err
			}
		}
		row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
		if err != nil {
			return err
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionContext, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) AddSkill(ctx context.Context, organisationID, versionID, actorID string, expected int, input SkillInput) (Version, error) {
	if err := validSkillInput(input); err != nil {
		return Version{}, err
	}
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := expectRow(ctx, q, organisationID, versionID, expected, StatusDraft)
		if err != nil {
			return err
		}
		skill, err := q.GetCanonicalSkill(ctx, db.GetCanonicalSkillParams{OrganisationID: organisationID, ID: input.SkillID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if skill.Category != input.Category {
			return ErrInvalid
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		existing, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if err := q.InsertRoleProfileSkill(ctx, db.InsertRoleProfileSkillParams{
			ID:                  id,
			OrganisationID:      organisationID,
			ProfileVersionID:    versionID,
			SkillID:             skill.ID,
			Category:            skill.Category,
			Definition:          strings.TrimSpace(input.Definition),
			RequiredProficiency: int32(input.RequiredProficiency),
			Priority:            input.Priority,
			Rationale:           strings.TrimSpace(input.Rationale),
			Provenance:          input.Provenance,
			Position:            int32(len(existing) + 1),
		}); err != nil {
			return mapWrite(err)
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillAdded, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) UpdateSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int, edit SkillEdit) (Version, error) {
	if err := validSkillEdit(edit); err != nil {
		return Version{}, err
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
		var current *db.ListRoleProfileSkillsRow
		for i := range skills {
			if skills[i].ID == assignmentID {
				current = &skills[i]
				break
			}
		}
		if current == nil {
			return ErrNotFound
		}
		provenance := current.Provenance
		if provenance == ProvenanceAISuggested {
			provenance = ProvenanceHRAdjusted
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		if err := q.UpdateRoleProfileSkill(ctx, db.UpdateRoleProfileSkillParams{
			ID:                  assignmentID,
			OrganisationID:      organisationID,
			ProfileVersionID:    versionID,
			Definition:          strings.TrimSpace(edit.Definition),
			RequiredProficiency: int32(edit.RequiredProficiency),
			Priority:            edit.Priority,
			Rationale:           strings.TrimSpace(edit.Rationale),
			Provenance:          provenance,
		}); err != nil {
			return mapWrite(err)
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillUpdated, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) RemoveSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int) (Version, error) {
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := expectRow(ctx, q, organisationID, versionID, expected, StatusDraft)
		if err != nil {
			return err
		}
		deleted, err := q.DeleteRoleProfileSkill(ctx, db.DeleteRoleProfileSkillParams{ID: assignmentID, OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		if deleted == 0 {
			return ErrNotFound
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		skills, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		for i, skill := range skills {
			if err := q.UpdateRoleProfileSkillPosition(ctx, db.UpdateRoleProfileSkillPositionParams{
				ID:               skill.ID,
				OrganisationID:   organisationID,
				ProfileVersionID: versionID,
				Position:         int32(i + 1),
			}); err != nil {
				return err
			}
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionSkillRemoved, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) ReorderSkills(ctx context.Context, organisationID, versionID, actorID string, expected int, assignmentIDs []string) (Version, error) {
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
		current := make([]SkillAssignment, 0, len(skills))
		for _, skill := range skills {
			current = append(current, SkillAssignment{ID: skill.ID})
		}
		if !permutation(current, assignmentIDs) {
			return ErrInvalid
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: int32(expected)})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		for position, id := range assignmentIDs {
			if err := q.UpdateRoleProfileSkillPosition(ctx, db.UpdateRoleProfileSkillPositionParams{
				ID:               id,
				OrganisationID:   organisationID,
				ProfileVersionID: versionID,
				Position:         int32(position + 1),
			}); err != nil {
				return err
			}
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionReordered, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) ListVersions(ctx context.Context, organisationID, profileID string) ([]Version, error) {
	if _, err := s.queries.GetRoleProfile(ctx, db.GetRoleProfileParams{OrganisationID: organisationID, ID: profileID}); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	ids, err := s.queries.ListRoleProfileVersionIDs(ctx, db.ListRoleProfileVersionIDsParams{OrganisationID: organisationID, RoleProfileID: profileID})
	if err != nil {
		return nil, err
	}
	versions := make([]Version, 0, len(ids))
	for _, id := range ids {
		version, err := loadVersion(ctx, s.queries, organisationID, id)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (s SQLStore) Compare(ctx context.Context, organisationID, profileID, fromID, toID string) (Comparison, error) {
	if _, err := readProfile(ctx, s.queries, organisationID, profileID); err != nil {
		return Comparison{}, err
	}
	from, err := loadVersion(ctx, s.queries, organisationID, fromID)
	if err != nil {
		return Comparison{}, err
	}
	to, err := loadVersion(ctx, s.queries, organisationID, toID)
	if err != nil {
		return Comparison{}, err
	}
	if from.ProfileID != profileID || to.ProfileID != profileID {
		return Comparison{}, ErrNotFound
	}
	return compareVersions(from, to), nil
}

func (s SQLStore) Submit(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	return s.transition(ctx, organisationID, versionID, actorID, expected, StatusDraft, StatusInReview, ActionSubmitted)
}

func (s SQLStore) Publish(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	var version Version
	err := s.inTxWith(ctx, func(q *db.Queries, tx *sql.Tx) error {
		if _, err := expectRow(ctx, q, organisationID, versionID, expected, StatusInReview); err != nil {
			return err
		}
		current, err := loadVersion(ctx, q, organisationID, versionID)
		if err != nil {
			return err
		}
		if err := readyForReview(current); err != nil {
			return err
		}
		occurred := clock()
		updated, err := q.TransitionRoleProfileVersion(ctx, db.TransitionRoleProfileVersionParams{
			ID:             versionID,
			OrganisationID: organisationID,
			Revision:       int32(expected),
			Status:         StatusPublished,
			Status_2:       StatusInReview,
		})
		if err != nil {
			return err
		}
		if updated == 0 {
			return ErrConflict
		}
		if err := writeAudit(ctx, q, organisationID, current.ProfileID, versionID, actorID, ActionPublished, occurred); err != nil {
			return err
		}
		payload, err := roleProfilePublishedPayload(organisationID, current.ProfileID, versionID, actorID, current.VersionNumber, occurred)
		if err != nil {
			return err
		}
		eventID, err := newID()
		if err != nil {
			return err
		}
		correlationID, err := newID()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO outbox_events (
				id, event_type, schema_version, aggregate_id, occurred_at, correlation_id,
				payload, status, attempts, next_attempt_at, last_error, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7::jsonb, 'pending', 0, $8, '', $9
			)`, eventID, outbox.RoleProfilePublishedType, publicationSchemaVersion, versionID, occurred, correlationID, string(payload), occurred, occurred); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) transition(ctx context.Context, organisationID, versionID, actorID string, expected int, from, to, action string) (Version, error) {
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		if _, err := expectRow(ctx, q, organisationID, versionID, expected, from); err != nil {
			return err
		}
		current, err := loadVersion(ctx, q, organisationID, versionID)
		if err != nil {
			return err
		}
		if err := readyForReview(current); err != nil {
			return err
		}
		updated, err := q.TransitionRoleProfileVersion(ctx, db.TransitionRoleProfileVersionParams{
			ID:             versionID,
			OrganisationID: organisationID,
			Revision:       int32(expected),
			Status:         to,
			Status_2:       from,
		})
		if err != nil {
			return err
		}
		if updated == 0 {
			return ErrConflict
		}
		if err := writeAudit(ctx, q, organisationID, current.ProfileID, versionID, actorID, action, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) NextDraft(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		source, err := expectRow(ctx, q, organisationID, versionID, expected, StatusPublished)
		if err != nil {
			return err
		}
		maxNumber, err := q.MaxRoleProfileVersionNumber(ctx, db.MaxRoleProfileVersionNumberParams{OrganisationID: organisationID, RoleProfileID: source.RoleProfileID})
		if err != nil {
			return err
		}
		if source.VersionNumber != maxNumber {
			return ErrConflict
		}
		loaded, err := loadVersion(ctx, q, organisationID, versionID)
		if err != nil {
			return err
		}
		nextID, err := newID()
		if err != nil {
			return err
		}
		now := clock()
		generated := sql.NullTime{}
		if loaded.AI.GeneratedAt != nil {
			generated = sql.NullTime{Time: *loaded.AI.GeneratedAt, Valid: true}
		}
		if err := q.InsertRoleProfileVersion(ctx, db.InsertRoleProfileVersionParams{
			ID:                     nextID,
			OrganisationID:         organisationID,
			RoleProfileID:          source.RoleProfileID,
			VersionNumber:          source.VersionNumber + 1,
			Revision:               1,
			Status:                 StatusDraft,
			Purpose:                source.Purpose,
			Responsibilities:       source.Responsibilities,
			BusinessOutcomes:       source.BusinessOutcomes,
			DecisionScope:          source.DecisionScope,
			Industry:               source.Industry,
			Region:                 source.Region,
			TeamSize:               source.TeamSize,
			ReportingLevel:         source.ReportingLevel,
			FrameworkVersionID:     source.FrameworkVersionID,
			AiDraft:                source.AiDraft,
			AiModel:                source.AiModel,
			AiSourceContextVersion: source.AiSourceContextVersion,
			AiGeneratedAt:          generated,
			CreatedAt:              now,
		}); err != nil {
			return err
		}
		for _, sourceDoc := range loaded.Sources {
			id, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertRoleProfileSource(ctx, db.InsertRoleProfileSourceParams{
				ID:               id,
				OrganisationID:   organisationID,
				ProfileVersionID: nextID,
				Title:            sourceDoc.Title,
				ReferenceUri:     sourceDoc.ReferenceURI,
			}); err != nil {
				return err
			}
		}
		for _, skill := range loaded.Skills {
			id, err := newID()
			if err != nil {
				return err
			}
			if err := q.InsertRoleProfileSkill(ctx, db.InsertRoleProfileSkillParams{
				ID:                  id,
				OrganisationID:      organisationID,
				ProfileVersionID:    nextID,
				SkillID:             skill.SkillID,
				Category:            skill.Category,
				Definition:          skill.Definition,
				RequiredProficiency: int32(skill.RequiredProficiency),
				Priority:            skill.Priority,
				Rationale:           skill.Rationale,
				Provenance:          skill.Provenance,
				Position:            int32(skill.Position),
			}); err != nil {
				return err
			}
		}
		if err := writeAudit(ctx, q, organisationID, source.RoleProfileID, nextID, actorID, ActionNextDraft, now); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, nextID)
		return err
	})
	return version, err
}

func (s SQLStore) Comment(ctx context.Context, organisationID, versionID, actorID, body string) (Comment, error) {
	body = strings.TrimSpace(body)
	if actorID == "" || body == "" {
		return Comment{}, ErrInvalid
	}
	var comment Comment
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != StatusInReview {
			return ErrImmutable
		}
		id, err := newID()
		if err != nil {
			return err
		}
		now := clock()
		if err := q.InsertRoleProfileComment(ctx, db.InsertRoleProfileCommentParams{
			ID:               id,
			OrganisationID:   organisationID,
			ProfileVersionID: versionID,
			AuthorPersonID:   actorID,
			Body:             body,
			CreatedAt:        now,
		}); err != nil {
			return err
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionComment, now); err != nil {
			return err
		}
		comment = Comment{ID: id, ProfileVersionID: versionID, AuthorPersonID: actorID, Body: body, CreatedAt: now}
		return nil
	})
	return comment, err
}

func (s SQLStore) History(ctx context.Context, organisationID, profileID string) (History, error) {
	if _, err := s.queries.GetRoleProfile(ctx, db.GetRoleProfileParams{OrganisationID: organisationID, ID: profileID}); errors.Is(err, sql.ErrNoRows) {
		return History{}, ErrNotFound
	} else if err != nil {
		return History{}, err
	}
	events, err := s.queries.ListRoleProfileAudit(ctx, db.ListRoleProfileAuditParams{OrganisationID: organisationID, RoleProfileID: profileID})
	if err != nil {
		return History{}, err
	}
	comments, err := s.queries.ListRoleProfileComments(ctx, db.ListRoleProfileCommentsParams{OrganisationID: organisationID, RoleProfileID: profileID})
	if err != nil {
		return History{}, err
	}
	history := History{ProfileID: profileID, Events: make([]AuditEvent, 0, len(events)), Comments: make([]Comment, 0, len(comments))}
	for _, event := range events {
		history.Events = append(history.Events, AuditEvent{
			ID:               event.ID,
			ProfileVersionID: event.ProfileVersionID,
			ActorPersonID:    event.ActorPersonID,
			Action:           event.Action,
			OccurredAt:       event.OccurredAt,
		})
	}
	for _, comment := range comments {
		history.Comments = append(history.Comments, Comment{
			ID:               comment.ID,
			ProfileVersionID: comment.ProfileVersionID,
			AuthorPersonID:   comment.AuthorPersonID,
			Body:             comment.Body,
			CreatedAt:        comment.CreatedAt,
		})
	}
	return history, nil
}

func (s SQLStore) AttachAIDraft(ctx context.Context, organisationID, versionID, actorID string, draft AIDraft) (Version, error) {
	if err := validAIDraft(draft); err != nil {
		return Version{}, err
	}
	var version Version
	err := s.inTx(ctx, func(q *db.Queries) error {
		row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != StatusDraft || row.Purpose != "" || row.AiDraft != "" {
			return ErrImmutable
		}
		existing, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
		if err != nil {
			return err
		}
		if len(existing) != 0 {
			return ErrImmutable
		}
		touched, err := q.TouchRoleProfileDraft(ctx, db.TouchRoleProfileDraftParams{ID: versionID, OrganisationID: organisationID, Revision: row.Revision})
		if err != nil {
			return err
		}
		if touched == 0 {
			return ErrConflict
		}
		assignments := make([]SkillAssignment, 0, len(draft.Skills))
		seen := map[string]struct{}{}
		for _, suggested := range draft.Skills {
			if _, ok := seen[suggested.SkillID]; ok {
				return ErrDuplicate
			}
			seen[suggested.SkillID] = struct{}{}
			skill, err := q.GetCanonicalSkill(ctx, db.GetCanonicalSkillParams{OrganisationID: organisationID, ID: suggested.SkillID})
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			if strings.TrimSpace(suggested.Definition) == "" || !validProficiency(suggested.RequiredProficiency) || !validPriority(suggested.Priority) {
				return ErrInvalid
			}
			id, err := newID()
			if err != nil {
				return err
			}
			assignment := SkillAssignment{
				ID:                  id,
				SkillID:             skill.ID,
				SkillName:           skill.Name,
				Category:            skill.Category,
				Definition:          strings.TrimSpace(suggested.Definition),
				RequiredProficiency: suggested.RequiredProficiency,
				Priority:            suggested.Priority,
				Rationale:           strings.TrimSpace(suggested.Rationale),
				Provenance:          ProvenanceAISuggested,
				Position:            len(assignments) + 1,
			}
			if err := q.InsertRoleProfileSkill(ctx, db.InsertRoleProfileSkillParams{
				ID:                  assignment.ID,
				OrganisationID:      organisationID,
				ProfileVersionID:    versionID,
				SkillID:             assignment.SkillID,
				Category:            assignment.Category,
				Definition:          assignment.Definition,
				RequiredProficiency: int32(assignment.RequiredProficiency),
				Priority:            assignment.Priority,
				Rationale:           assignment.Rationale,
				Provenance:          assignment.Provenance,
				Position:            int32(assignment.Position),
			}); err != nil {
				return mapWrite(err)
			}
			assignments = append(assignments, assignment)
		}
		raw, err := snapshotDraft(draft, assignments)
		if err != nil {
			return err
		}
		updated, err := q.SetRoleProfileAIDraft(ctx, db.SetRoleProfileAIDraftParams{
			ID:                     versionID,
			OrganisationID:         organisationID,
			Purpose:                strings.TrimSpace(draft.Purpose),
			AiDraft:                string(raw),
			AiModel:                strings.TrimSpace(draft.Model),
			AiSourceContextVersion: strings.TrimSpace(draft.SourceContextVersion),
			AiGeneratedAt:          sql.NullTime{Time: draft.GeneratedAt.UTC().Truncate(time.Second), Valid: true},
		})
		if err != nil {
			return err
		}
		if updated == 0 {
			return ErrImmutable
		}
		if err := writeAudit(ctx, q, organisationID, row.RoleProfileID, versionID, actorID, ActionAIDraft, clock()); err != nil {
			return err
		}
		version, err = loadVersion(ctx, q, organisationID, versionID)
		return err
	})
	return version, err
}

func (s SQLStore) inTx(ctx context.Context, fn func(*db.Queries) error) error {
	return s.inTxWith(ctx, func(q *db.Queries, _ *sql.Tx) error { return fn(q) })
}

func (s SQLStore) inTxWith(ctx context.Context, fn func(*db.Queries, *sql.Tx) error) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.queries.WithTx(tx), tx); err != nil {
		return err
	}
	return tx.Commit()
}

func expectRow(ctx context.Context, q *db.Queries, organisationID, versionID string, expected int, status string) (db.RoleProfileVersion, error) {
	row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
	if errors.Is(err, sql.ErrNoRows) {
		return db.RoleProfileVersion{}, ErrNotFound
	}
	if err != nil {
		return db.RoleProfileVersion{}, err
	}
	if row.Status != status {
		return db.RoleProfileVersion{}, ErrImmutable
	}
	if int(row.Revision) != expected {
		return db.RoleProfileVersion{}, ErrConflict
	}
	return row, nil
}

func readProfile(ctx context.Context, q *db.Queries, organisationID, profileID string) (Profile, error) {
	row, err := q.GetRoleProfile(ctx, db.GetRoleProfileParams{OrganisationID: organisationID, ID: profileID})
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	versionID, err := q.LatestRoleProfileVersionID(ctx, db.LatestRoleProfileVersionIDParams{OrganisationID: organisationID, RoleProfileID: profileID})
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	version, err := loadVersion(ctx, q, organisationID, versionID)
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		ID:                row.ID,
		OrganisationID:    row.OrganisationID,
		RoleID:            row.RoleID,
		RoleName:          row.RoleName,
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt,
		Version:           version,
	}, nil
}

func loadVersion(ctx context.Context, q *db.Queries, organisationID, versionID string) (Version, error) {
	row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrNotFound
	}
	if err != nil {
		return Version{}, err
	}
	skills, err := q.ListRoleProfileSkills(ctx, db.ListRoleProfileSkillsParams{OrganisationID: organisationID, ProfileVersionID: versionID})
	if err != nil {
		return Version{}, err
	}
	sources, err := q.ListRoleProfileSources(ctx, db.ListRoleProfileSourcesParams{OrganisationID: organisationID, ProfileVersionID: versionID})
	if err != nil {
		return Version{}, err
	}
	version := Version{
		ID:                 row.ID,
		ProfileID:          row.RoleProfileID,
		OrganisationID:     row.OrganisationID,
		VersionNumber:      int(row.VersionNumber),
		Revision:           int(row.Revision),
		Status:             row.Status,
		Purpose:            row.Purpose,
		Responsibilities:   row.Responsibilities,
		BusinessOutcomes:   row.BusinessOutcomes,
		DecisionScope:      row.DecisionScope,
		Industry:           row.Industry,
		Region:             row.Region,
		TeamSize:           intPtr(row.TeamSize),
		ReportingLevel:     row.ReportingLevel,
		FrameworkVersionID: row.FrameworkVersionID.String,
		Sources:            make([]Source, 0, len(sources)),
		Skills:             make([]SkillAssignment, 0, len(skills)),
		AI: AIMetadata{
			Model:                row.AiModel,
			SourceContextVersion: row.AiSourceContextVersion,
		},
		CreatedAt: row.CreatedAt,
	}
	if row.AiGeneratedAt.Valid {
		generated := row.AiGeneratedAt.Time.UTC()
		version.AI.GeneratedAt = &generated
	}
	if row.AiDraft != "" {
		version.AI.Draft = []byte(row.AiDraft)
	}
	for _, source := range sources {
		version.Sources = append(version.Sources, Source{ID: source.ID, Title: source.Title, ReferenceURI: source.ReferenceUri})
	}
	for _, skill := range skills {
		version.Skills = append(version.Skills, SkillAssignment{
			ID:                  skill.ID,
			SkillID:             skill.SkillID,
			SkillName:           skill.SkillName,
			Category:            skill.Category,
			Definition:          skill.Definition,
			RequiredProficiency: int(skill.RequiredProficiency),
			Priority:            skill.Priority,
			Rationale:           skill.Rationale,
			Provenance:          skill.Provenance,
			Position:            int(skill.Position),
		})
	}
	return version, nil
}

func newVersionParams(id, organisationID, profileID, frameworkVersionID string, number int, createdAt time.Time) db.InsertRoleProfileVersionParams {
	return db.InsertRoleProfileVersionParams{
		ID:                     id,
		OrganisationID:         organisationID,
		RoleProfileID:          profileID,
		VersionNumber:          int32(number),
		Revision:               1,
		Status:                 StatusDraft,
		ReportingLevel:         "",
		FrameworkVersionID:     nullString(frameworkVersionID),
		AiDraft:                "",
		AiModel:                "",
		AiSourceContextVersion: "",
		CreatedAt:              createdAt,
	}
}

func writeAudit(ctx context.Context, q *db.Queries, organisationID, profileID, versionID, actorID, action string, occurredAt time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	return q.InsertRoleProfileAudit(ctx, db.InsertRoleProfileAuditParams{
		ID:               id,
		OrganisationID:   organisationID,
		RoleProfileID:    profileID,
		ProfileVersionID: versionID,
		ActorPersonID:    actorID,
		Action:           action,
		OccurredAt:       occurredAt,
	})
}

func nullInt(value *int) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*value), Valid: true}
}

func intPtr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	number := int(value.Int32)
	return &number
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func mapWrite(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrDuplicate
		case "23503":
			return ErrNotFound
		case "23514":
			return ErrInvalid
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
