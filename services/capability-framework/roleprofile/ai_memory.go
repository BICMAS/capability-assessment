package roleprofile

import (
	"context"
	"encoding/json"
	"time"
)

func (m *Memory) GetVersion(_ context.Context, organisationID, versionID string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return Version{}, ErrNotFound
	}
	return cloneVersion(*version), nil
}

func (m *Memory) ListCanonicalSkills(_ context.Context, organisationID string) ([]Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	skills := make([]Skill, 0)
	for _, skill := range m.skills {
		if skill.OrganisationID == organisationID {
			skills = append(skills, skill)
		}
	}
	return skills, nil
}

func (m *Memory) EnsureContext(_ context.Context, organisationID, versionID, actorID string) (ContextRecord, error) {
	if actorID == "" {
		return ContextRecord{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return ContextRecord{}, ErrNotFound
	}
	snapshot := snapshotFrom(*version)
	hash, err := hashSnapshot(snapshot)
	if err != nil {
		return ContextRecord{}, err
	}
	latest := m.latestContext(versionID)
	if latest.Version > 0 && latest.Hash == hash {
		return latest, nil
	}
	record := ContextRecord{
		ProfileVersionID: versionID,
		Version:          latest.Version + 1,
		Hash:             hash,
		Snapshot:         snapshot,
		CreatedAt:        clock(),
	}
	m.contexts = append(m.contexts, record)
	return record, nil
}

func (m *Memory) ListQuestions(_ context.Context, organisationID, versionID string) ([]Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.version(organisationID, versionID); !ok {
		return nil, ErrNotFound
	}
	questions := make([]Question, 0)
	for _, question := range m.questions {
		if question.ProfileVersionID == versionID {
			questions = append(questions, question)
		}
	}
	return questions, nil
}

func (m *Memory) SaveQuestions(_ context.Context, organisationID string, questions []Question) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, question := range questions {
		if _, ok := m.version(organisationID, question.ProfileVersionID); !ok {
			return ErrNotFound
		}
		m.questions = append(m.questions, question)
	}
	return nil
}

func (m *Memory) MarkAnswers(_ context.Context, organisationID, versionID, actorID string, answers []AnswerInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.version(organisationID, versionID); !ok {
		return ErrNotFound
	}
	now := clock()
	for _, answer := range answers {
		found := false
		for i := range m.questions {
			if m.questions[i].ID == answer.QuestionID && m.questions[i].ProfileVersionID == versionID {
				m.questions[i].Answer = answer.Value
				m.questions[i].AnsweredActorID = actorID
				m.questions[i].AnsweredAt = &now
				found = true
			}
		}
		if !found {
			return ErrNotFound
		}
	}
	return nil
}

func (m *Memory) JobByIdempotency(_ context.Context, organisationID, key string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range m.jobs {
		if job.OrganisationID == organisationID && job.IdempotencyKey == key {
			return cloneJob(job), nil
		}
	}
	return Job{}, ErrNotFound
}

func (m *Memory) CreateGenerationJob(_ context.Context, job Job) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := newID()
	if err != nil {
		return Job{}, err
	}
	job.ID = id
	if job.Unsupported == nil {
		job.Unsupported = []UnsupportedSuggestion{}
	}
	if job.Uncertainty == nil {
		job.Uncertainty = []string{}
	}
	m.jobs = append(m.jobs, job)
	return cloneJob(job), nil
}

func (m *Memory) GetGenerationJob(_ context.Context, organisationID, jobID string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range m.jobs {
		if job.ID == jobID && job.OrganisationID == organisationID {
			return cloneJob(job), nil
		}
	}
	return Job{}, ErrNotFound
}

func (m *Memory) ClaimGenerationJobs(_ context.Context, limit int, lease time.Duration) ([]Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	claimed := make([]Job, 0)
	for i := range m.jobs {
		if m.jobs[i].Status == JobRunning && m.jobs[i].StartedAt != nil && m.jobs[i].StartedAt.Before(now.Add(-lease)) {
			m.jobs[i].Status = JobQueued
		}
	}
	for i := range m.jobs {
		if m.jobs[i].Status != JobQueued || len(claimed) >= limit {
			continue
		}
		started := now
		m.jobs[i].Status = JobRunning
		m.jobs[i].StartedAt = &started
		claimed = append(claimed, cloneJob(m.jobs[i]))
	}
	return claimed, nil
}

func (m *Memory) SaveGenerationJob(_ context.Context, job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.jobs {
		if m.jobs[i].ID == job.ID && m.jobs[i].OrganisationID == job.OrganisationID {
			m.jobs[i] = job
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) CountGenerationJobs(_ context.Context, organisationID string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, job := range m.jobs {
		if job.OrganisationID == organisationID && !job.CreatedAt.Before(since) && job.Status != JobCancelled {
			count++
		}
	}
	return count, nil
}

func (m *Memory) ApplyGeneratedDraft(_ context.Context, organisationID, versionID, contextHash string, draft GeneratedDraft) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return Version{}, ErrNotFound
	}
	if version.Status != StatusDraft {
		return Version{}, ErrImmutable
	}
	latest := m.latestContext(versionID)
	if latest.Hash != contextHash {
		return Version{}, ErrStale
	}
	if len(version.Skills) != 0 || len(version.AI.Draft) != 0 {
		return Version{}, ErrConflict
	}
	assignments := make([]SkillAssignment, 0, len(draft.Skills))
	for _, skill := range draft.Skills {
		canonical, ok := m.skill(organisationID, skill.SkillID)
		if !ok {
			return Version{}, ErrNotFound
		}
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		assignments = append(assignments, SkillAssignment{
			ID:                  id,
			SkillID:             canonical.ID,
			SkillName:           canonical.Name,
			Category:            canonical.Category,
			Definition:          skill.Definition,
			RequiredProficiency: skill.RequiredProficiency,
			Priority:            skill.Priority,
			Rationale:           skill.Rationale,
			Provenance:          ProvenanceAISuggested,
			Position:            len(assignments) + 1,
		})
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return Version{}, err
	}
	generated := draft.GeneratedAt.UTC().Truncate(time.Second)
	version.Skills = assignments
	version.AI = AIMetadata{
		Model:                draft.Model,
		SourceContextVersion: draft.SourceContextVersion,
		GeneratedAt:          &generated,
		Draft:                raw,
	}
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, "generation", ActionAIDraft, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) latestContext(versionID string) ContextRecord {
	var latest ContextRecord
	for _, record := range m.contexts {
		if record.ProfileVersionID == versionID && record.Version >= latest.Version {
			latest = record
		}
	}
	return latest
}

func cloneJob(job Job) Job {
	cloned := job
	cloned.Unsupported = append([]UnsupportedSuggestion{}, job.Unsupported...)
	cloned.Uncertainty = append([]string{}, job.Uncertainty...)
	if job.Result != nil {
		cloned.Result = append(json.RawMessage{}, job.Result...)
	}
	if job.StartedAt != nil {
		started := *job.StartedAt
		cloned.StartedAt = &started
	}
	if job.FinishedAt != nil {
		finished := *job.FinishedAt
		cloned.FinishedAt = &finished
	}
	return cloned
}
