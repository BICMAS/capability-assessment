package roleprofile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

func (s SQLStore) GetVersion(ctx context.Context, organisationID, versionID string) (Version, error) {
	return loadVersion(ctx, s.queries, organisationID, versionID)
}

func (s SQLStore) ListCanonicalSkills(ctx context.Context, organisationID string) ([]Skill, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, organisation_id, name, category, definition
		FROM canonical_skills
		WHERE organisation_id = $1
		ORDER BY name ASC
	`, organisationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	skills := make([]Skill, 0)
	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.ID, &skill.OrganisationID, &skill.Name, &skill.Category, &skill.Definition); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, rows.Err()
}

func (s SQLStore) EnsureContext(ctx context.Context, organisationID, versionID, actorID string) (ContextRecord, error) {
	if actorID == "" {
		return ContextRecord{}, ErrInvalid
	}
	version, err := loadVersion(ctx, s.queries, organisationID, versionID)
	if err != nil {
		return ContextRecord{}, err
	}
	snapshot := snapshotFrom(version)
	hash, err := hashSnapshot(snapshot)
	if err != nil {
		return ContextRecord{}, err
	}
	record, err := s.latestContext(ctx, organisationID, versionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ContextRecord{}, err
	}
	if err == nil && record.Hash == hash {
		record.Snapshot = snapshot
		return record, nil
	}
	next := 1
	if err == nil {
		next = record.Version + 1
	}
	id, err := newID()
	if err != nil {
		return ContextRecord{}, err
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return ContextRecord{}, err
	}
	created := clock()
	if _, err := s.database.ExecContext(ctx, `
		INSERT INTO role_profile_contexts (
			id, organisation_id, profile_version_id, version_number, content_hash, snapshot, created_by_person_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, organisationID, versionID, next, hash, string(raw), actorID, created); err != nil {
		return ContextRecord{}, err
	}
	return ContextRecord{ProfileVersionID: versionID, Version: next, Hash: hash, Snapshot: snapshot, CreatedAt: created}, nil
}

func (s SQLStore) ListQuestions(ctx context.Context, organisationID, versionID string) ([]Question, error) {
	if _, err := loadVersion(ctx, s.queries, organisationID, versionID); err != nil {
		return nil, err
	}
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, profile_version_id, context_version, round, field, question, critical, answer,
		       asked_actor_id, answered_actor_id, asked_at, answered_at, authored_by
		FROM role_profile_questions
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY asked_at ASC, id ASC
	`, organisationID, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	questions := make([]Question, 0)
	for rows.Next() {
		question, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}
	return questions, rows.Err()
}

func (s SQLStore) SaveQuestions(ctx context.Context, organisationID string, questions []Question) error {
	for _, question := range questions {
		if _, err := s.database.ExecContext(ctx, `
			INSERT INTO role_profile_questions (
				id, organisation_id, profile_version_id, context_version, round, field, question, critical,
				answer, asked_actor_id, answered_actor_id, asked_at, authored_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '', $9, '', $10, $11)
		`, question.ID, organisationID, question.ProfileVersionID, question.ContextVersion, question.Round, question.Field, question.Question, question.Critical, question.AskedActorID, question.AskedAt, questionAuthor(question.AuthoredBy)); err != nil {
			return err
		}
	}
	return nil
}

func (s SQLStore) MarkAnswers(ctx context.Context, organisationID, versionID, actorID string, answers []AnswerInput) error {
	now := clock()
	for _, answer := range answers {
		updated, err := s.database.ExecContext(ctx, `
			UPDATE role_profile_questions
			SET answer = $4, answered_actor_id = $5, answered_at = $6
			WHERE id = $1 AND organisation_id = $2 AND profile_version_id = $3
		`, answer.QuestionID, organisationID, versionID, answer.Value, actorID, now)
		if err != nil {
			return err
		}
		count, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (s SQLStore) JobByIdempotency(ctx context.Context, organisationID, key string) (Job, error) {
	row := s.database.QueryRowContext(ctx, jobSelect+` WHERE organisation_id = $1 AND idempotency_key = $2`, organisationID, key)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

func (s SQLStore) CreateGenerationJob(ctx context.Context, job Job) (Job, error) {
	id, err := newID()
	if err != nil {
		return Job{}, err
	}
	job.ID = id
	unsupported, err := json.Marshal(nonNilUnsupported(job.Unsupported))
	if err != nil {
		return Job{}, err
	}
	uncertainty, err := json.Marshal(nonNilUncertainty(job.Uncertainty))
	if err != nil {
		return Job{}, err
	}
	_, err = s.database.ExecContext(ctx, `
		INSERT INTO role_profile_generation_jobs (
			id, organisation_id, profile_version_id, context_version, context_hash, idempotency_key, status,
			model_id, prompt_version, input_hash, output_hash, latency_ms, prompt_tokens, completion_tokens,
			attempts, last_error, unsupported, uncertainty, result, cancel_requested, replace_existing, created_by_person_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, '', 0, 0, 0,
			0, '', $11, $12, '', false, $13, $14, $15
		)
	`, job.ID, job.OrganisationID, job.ProfileVersionID, job.ContextVersion, job.ContextHash, job.IdempotencyKey, job.Status, job.ModelID, job.PromptVersion, job.InputHash, string(unsupported), string(uncertainty), job.ReplaceExisting, job.CreatedBy, job.CreatedAt)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return s.JobByIdempotency(ctx, job.OrganisationID, job.IdempotencyKey)
		}
		return Job{}, err
	}
	return s.GetGenerationJob(ctx, job.OrganisationID, job.ID)
}

func (s SQLStore) GetGenerationJob(ctx context.Context, organisationID, jobID string) (Job, error) {
	row := s.database.QueryRowContext(ctx, jobSelect+` WHERE organisation_id = $1 AND id = $2`, organisationID, jobID)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

func (s SQLStore) ClaimGenerationJobs(ctx context.Context, limit int, lease time.Duration) ([]Job, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE role_profile_generation_jobs
		SET status = 'queued'
		WHERE status = 'running' AND started_at < $1
	`, time.Now().UTC().Add(-lease)); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, jobSelect+`
		WHERE status = 'queued'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	claimed := make([]Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := clock()
	for i := range claimed {
		if _, err := tx.ExecContext(ctx, `
			UPDATE role_profile_generation_jobs
			SET status = 'running', started_at = $2
			WHERE id = $1 AND status = 'queued'
		`, claimed[i].ID, now); err != nil {
			return nil, err
		}
		claimed[i].Status = JobRunning
		started := now
		claimed[i].StartedAt = &started
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s SQLStore) SaveGenerationJob(ctx context.Context, job Job) error {
	unsupported, err := json.Marshal(nonNilUnsupported(job.Unsupported))
	if err != nil {
		return err
	}
	uncertainty, err := json.Marshal(nonNilUncertainty(job.Uncertainty))
	if err != nil {
		return err
	}
	updated, err := s.database.ExecContext(ctx, `
		UPDATE role_profile_generation_jobs
		SET status = $3, output_hash = $4, latency_ms = $5, prompt_tokens = $6, completion_tokens = $7,
		    attempts = $8, last_error = $9, unsupported = $10, uncertainty = $11, result = $12, cancel_requested = $13,
		    started_at = $14, finished_at = $15
		WHERE id = $1 AND organisation_id = $2
	`, job.ID, job.OrganisationID, job.Status, job.OutputHash, job.LatencyMS, job.PromptTokens, job.CompletionTokens, job.Attempts, job.LastError, string(unsupported), string(uncertainty), string(job.Result), job.CancelRequested, nullTime(job.StartedAt), nullTime(job.FinishedAt))
	if err != nil {
		return err
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s SQLStore) CountGenerationJobs(ctx context.Context, organisationID string, since time.Time) (int, error) {
	var count int
	err := s.database.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM role_profile_generation_jobs
		WHERE organisation_id = $1 AND created_at >= $2 AND status <> 'cancelled'
	`, organisationID, since).Scan(&count)
	return count, err
}

func (s SQLStore) ApplyGeneratedDraft(ctx context.Context, organisationID, versionID, contextHash string, draft GeneratedDraft, replace bool) (Version, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return Version{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)
	current, err := loadVersion(ctx, q, organisationID, versionID)
	if err != nil {
		return Version{}, err
	}
	if current.Status != StatusDraft {
		return Version{}, ErrImmutable
	}
	if !replace && draftExists(current) {
		return Version{}, ErrConflict
	}
	kept := map[string]struct{}{}
	position := 1
	if replace {
		for _, skill := range current.Skills {
			if skill.Provenance == ProvenanceAISuggested {
				continue
			}
			kept[skill.SkillID] = struct{}{}
			if skill.Position >= position {
				position = skill.Position + 1
			}
		}
		for _, skill := range draft.Skills {
			if _, exists := kept[skill.SkillID]; exists {
				return Version{}, ErrConflict
			}
		}
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM role_profile_skills
			WHERE organisation_id = $1 AND profile_version_id = $2 AND provenance = 'ai_suggested'
		`, organisationID, versionID); err != nil {
			return Version{}, err
		}
	}
	var hash string
	err = tx.QueryRowContext(ctx, `
		SELECT content_hash
		FROM role_profile_contexts
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY version_number DESC
		LIMIT 1
	`, organisationID, versionID).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && hash != contextHash) {
		return Version{}, ErrStale
	}
	if err != nil {
		return Version{}, err
	}
	now := clock()
	for _, skill := range draft.Skills {
		canonical, err := q.GetCanonicalSkill(ctx, db.GetCanonicalSkillParams{OrganisationID: organisationID, ID: skill.SkillID})
		if errors.Is(err, sql.ErrNoRows) {
			return Version{}, ErrNotFound
		}
		if err != nil {
			return Version{}, err
		}
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		if err := q.InsertRoleProfileSkill(ctx, db.InsertRoleProfileSkillParams{
			ID:                  id,
			OrganisationID:      organisationID,
			ProfileVersionID:    versionID,
			SkillID:             canonical.ID,
			Category:            canonical.Category,
			Definition:          skill.Definition,
			RequiredProficiency: int32(skill.RequiredProficiency),
			Priority:            skill.Priority,
			Rationale:           skill.Rationale,
			Criteria:            skill.Criteria,
			Provenance:          ProvenanceAISuggested,
			Position:            int32(position),
			CreatedAt:           now,
			UpdatedAt:           now,
		}); err != nil {
			return Version{}, err
		}
		position++
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return Version{}, err
	}
	statement := `
		UPDATE role_profile_versions
		SET ai_draft = $3, ai_model = $4, ai_source_context_version = $5, ai_generated_at = $6, revision = revision + 1
		WHERE id = $1 AND organisation_id = $2 AND status = 'profile_draft'`
	if !replace {
		statement += ` AND ai_draft = ''`
	}
	updated, err := tx.ExecContext(ctx, statement, versionID, organisationID, string(raw), draft.Model, draft.SourceContextVersion, draft.GeneratedAt)
	if err != nil {
		return Version{}, err
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return Version{}, err
	}
	if count == 0 {
		return Version{}, ErrConflict
	}
	if err := writeAudit(ctx, q, organisationID, current.ProfileID, versionID, "generation", ActionAIDraft, clock()); err != nil {
		return Version{}, err
	}
	if err := tx.Commit(); err != nil {
		return Version{}, err
	}
	return loadVersion(ctx, s.queries, organisationID, versionID)
}

func (s SQLStore) latestContext(ctx context.Context, organisationID, versionID string) (ContextRecord, error) {
	var record ContextRecord
	var raw string
	err := s.database.QueryRowContext(ctx, `
		SELECT profile_version_id, version_number, content_hash, snapshot, created_at
		FROM role_profile_contexts
		WHERE organisation_id = $1 AND profile_version_id = $2
		ORDER BY version_number DESC
		LIMIT 1
	`, organisationID, versionID).Scan(&record.ProfileVersionID, &record.Version, &record.Hash, &raw, &record.CreatedAt)
	if err != nil {
		return ContextRecord{}, err
	}
	if err := json.Unmarshal([]byte(raw), &record.Snapshot); err != nil {
		return ContextRecord{}, err
	}
	return record, nil
}

func questionAuthor(value string) string {
	switch value {
	case QuestionModel, QuestionModelUnavailable:
		return value
	default:
		return QuestionService
	}
}

func scanQuestion(row interface{ Scan(...any) error }) (Question, error) {
	var question Question
	var answered sql.NullTime
	err := row.Scan(&question.ID, &question.ProfileVersionID, &question.ContextVersion, &question.Round, &question.Field, &question.Question, &question.Critical, &question.Answer, &question.AskedActorID, &question.AnsweredActorID, &question.AskedAt, &answered, &question.AuthoredBy)
	if answered.Valid {
		value := answered.Time
		question.AnsweredAt = &value
	}
	return question, err
}

const jobSelect = `
	SELECT id, organisation_id, profile_version_id, context_version, context_hash, idempotency_key, status,
	       model_id, prompt_version, input_hash, output_hash, latency_ms, prompt_tokens, completion_tokens,
	       attempts, last_error, unsupported, uncertainty, result, cancel_requested, replace_existing, created_by_person_id, created_at,
	       started_at, finished_at
	FROM role_profile_generation_jobs`

func scanJob(row interface{ Scan(...any) error }) (Job, error) {
	var job Job
	var unsupported, uncertainty, result string
	var started, finished sql.NullTime
	err := row.Scan(
		&job.ID, &job.OrganisationID, &job.ProfileVersionID, &job.ContextVersion, &job.ContextHash, &job.IdempotencyKey, &job.Status,
		&job.ModelID, &job.PromptVersion, &job.InputHash, &job.OutputHash, &job.LatencyMS, &job.PromptTokens, &job.CompletionTokens,
		&job.Attempts, &job.LastError, &unsupported, &uncertainty, &result, &job.CancelRequested, &job.ReplaceExisting, &job.CreatedBy, &job.CreatedAt,
		&started, &finished,
	)
	if err != nil {
		return Job{}, err
	}
	if unsupported != "" {
		if err := json.Unmarshal([]byte(unsupported), &job.Unsupported); err != nil {
			return Job{}, err
		}
	}
	if uncertainty != "" {
		if err := json.Unmarshal([]byte(uncertainty), &job.Uncertainty); err != nil {
			return Job{}, err
		}
	}
	job.Unsupported = nonNilUnsupported(job.Unsupported)
	job.Uncertainty = nonNilUncertainty(job.Uncertainty)
	if result != "" {
		job.Result = json.RawMessage(result)
	}
	if started.Valid {
		value := started.Time
		job.StartedAt = &value
	}
	if finished.Valid {
		value := finished.Time
		job.FinishedAt = &value
	}
	return job, nil
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nonNilUnsupported(values []UnsupportedSuggestion) []UnsupportedSuggestion {
	if values == nil {
		return []UnsupportedSuggestion{}
	}
	return values
}

func nonNilUncertainty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
