package roleprofile

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

type Limits struct {
	MaxRounds           int
	MaxQuestions        int
	MaxOutputBytes      int
	MaxCompletionTokens int
	TenantDailyJobs     int
	MaxAttempts         int
	AttemptTimeout      time.Duration
	Backoff             time.Duration
	PollInterval        time.Duration
	RunningLease        time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxRounds:           3,
		MaxQuestions:        5,
		MaxOutputBytes:      64 * 1024,
		MaxCompletionTokens: 4000,
		TenantDailyJobs:     20,
		MaxAttempts:         3,
		AttemptTimeout:      30 * time.Second,
		Backoff:             20 * time.Millisecond,
		PollInterval:        time.Second,
		RunningLease:        2 * time.Minute,
	}
}

type profileAI interface {
	GetVersion(ctx context.Context, organisationID, versionID string) (Version, error)
	GetProfile(ctx context.Context, organisationID, profileID string) (Profile, error)
	UpdateContext(ctx context.Context, organisationID, versionID, actorID string, expected int, details Context) (Version, error)
	ListCanonicalSkills(ctx context.Context, organisationID string) ([]Skill, error)
	EnsureContext(ctx context.Context, organisationID, versionID, actorID string) (ContextRecord, error)
	ListQuestions(ctx context.Context, organisationID, versionID string) ([]Question, error)
	SaveQuestions(ctx context.Context, organisationID string, questions []Question) error
	MarkAnswers(ctx context.Context, organisationID, versionID, actorID string, answers []AnswerInput) error
	JobByIdempotency(ctx context.Context, organisationID, key string) (Job, error)
	CreateGenerationJob(ctx context.Context, job Job) (Job, error)
	GetGenerationJob(ctx context.Context, organisationID, jobID string) (Job, error)
	ClaimGenerationJobs(ctx context.Context, limit int, lease time.Duration) ([]Job, error)
	SaveGenerationJob(ctx context.Context, job Job) error
	CountGenerationJobs(ctx context.Context, organisationID string, since time.Time) (int, error)
	ApplyGeneratedDraft(ctx context.Context, organisationID, versionID, contextHash string, draft GeneratedDraft, replace bool) (Version, error)
}

type GeneratedDraft struct {
	Model                string
	PromptVersion        string
	SourceContextVersion string
	InputHash            string
	GeneratedAt          time.Time
	Purpose              string
	Skills               []GeneratedSkill
}

type Workflow struct {
	store    profileAI
	provider Provider
	limits   Limits
	logger   *slog.Logger
}

func NewWorkflow(store profileAI, provider Provider, limits Limits, logger *slog.Logger) *Workflow {
	if limits.MaxRounds == 0 {
		limits = DefaultLimits()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Workflow{store: store, provider: provider, limits: limits, logger: logger}
}

func (w *Workflow) Run(ctx context.Context) {
	ticker := time.NewTicker(w.limits.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessQueued(ctx)
		}
	}
}

func (w *Workflow) Session(ctx context.Context, organisationID, versionID, actorID string) (Session, error) {
	record, err := w.store.EnsureContext(ctx, organisationID, versionID, actorID)
	if err != nil {
		return Session{}, err
	}
	questions, err := w.store.ListQuestions(ctx, organisationID, versionID)
	if err != nil {
		return Session{}, err
	}
	return w.session(record, questions), nil
}

func (w *Workflow) Clarify(ctx context.Context, organisationID, versionID, actorID string) (Session, error) {
	if actorID == "" {
		return Session{}, ErrInvalid
	}
	version, err := w.store.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Session{}, err
	}
	if version.Status != StatusDraft {
		return Session{}, ErrImmutable
	}
	record, err := w.store.EnsureContext(ctx, organisationID, versionID, actorID)
	if err != nil {
		return Session{}, err
	}
	questions, err := w.store.ListQuestions(ctx, organisationID, versionID)
	if err != nil {
		return Session{}, err
	}
	open := openQuestions(questions, record.Snapshot)
	if len(open) > 0 {
		return w.session(record, questions), nil
	}
	round := maxRound(questions)
	gaps := askableGaps(record.Snapshot, questions)
	if len(gaps) == 0 || round >= w.limits.MaxRounds {
		return w.session(record, questions), nil
	}
	if hasCriticalMissing(gaps) {
		return w.addQuestions(ctx, organisationID, actorID, record, questions, round+1, deterministicQuestions(gaps, record.Snapshot))
	}
	drafts := deterministicQuestions(ambiguousGaps(gaps), record.Snapshot)
	if w.provider != nil && len(drafts) > 0 {
		fields := make([]string, 0, len(drafts))
		for _, draft := range drafts {
			fields = append(fields, draft.Field)
		}
		modelQuestions, err := w.provider.Clarify(ctx, record.Snapshot, fields)
		if err != nil {
			for i := range drafts {
				drafts[i].AuthoredBy = QuestionModelUnavailable
			}
		} else {
			drafts = mergeQuestions(drafts, modelQuestions)
		}
	}
	if len(drafts) == 0 {
		return w.session(record, questions), nil
	}
	return w.addQuestions(ctx, organisationID, actorID, record, questions, round+1, drafts)
}

func (w *Workflow) Answer(ctx context.Context, organisationID, versionID, actorID string, answers []AnswerInput) (Session, error) {
	if actorID == "" || len(answers) == 0 {
		return Session{}, ErrInvalid
	}
	version, err := w.store.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Session{}, err
	}
	if version.Status != StatusDraft {
		return Session{}, ErrImmutable
	}
	questions, err := w.store.ListQuestions(ctx, organisationID, versionID)
	if err != nil {
		return Session{}, err
	}
	snapshot := snapshotFrom(version)
	accepted := make([]AnswerInput, 0, len(answers))
	for _, answer := range answers {
		question, ok := findQuestion(questions, answer.QuestionID)
		if !ok || (question.AnsweredAt != nil && strings.TrimSpace(answer.Value) == "") {
			return Session{}, ErrInvalid
		}
		updated, err := applyAnswer(snapshot, question.Field, answer.Value)
		if err != nil {
			return Session{}, err
		}
		snapshot = updated
		accepted = append(accepted, AnswerInput{QuestionID: question.ID, Value: strings.TrimSpace(answer.Value)})
	}
	before, err := hashSnapshot(snapshotFrom(version))
	if err != nil {
		return Session{}, err
	}
	after, err := hashSnapshot(snapshot)
	if err != nil {
		return Session{}, err
	}
	if before != after {
		if _, err := w.store.UpdateContext(ctx, organisationID, versionID, actorID, version.Revision, contextFromSnapshot(snapshot)); err != nil {
			return Session{}, err
		}
	}
	if err := w.store.MarkAnswers(ctx, organisationID, versionID, actorID, accepted); err != nil {
		return Session{}, err
	}
	return w.Clarify(ctx, organisationID, versionID, actorID)
}

func (w *Workflow) Enqueue(ctx context.Context, organisationID, versionID, actorID, idempotencyKey string, contextVersion int, replace bool) (Job, error) {
	if w.provider == nil {
		return Job{}, ErrUnavailable
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if actorID == "" || len(idempotencyKey) < 8 || len(idempotencyKey) > 80 {
		return Job{}, ErrInvalid
	}
	version, err := w.store.GetVersion(ctx, organisationID, versionID)
	if err != nil {
		return Job{}, err
	}
	if version.Status != StatusDraft {
		return Job{}, ErrImmutable
	}
	record, err := w.store.EnsureContext(ctx, organisationID, versionID, actorID)
	if err != nil {
		return Job{}, err
	}
	if record.Version != contextVersion {
		return Job{}, ErrStale
	}
	questions, err := w.store.ListQuestions(ctx, organisationID, versionID)
	if err != nil {
		return Job{}, err
	}
	if unresolved := criticalUnresolved(record.Snapshot, questions); len(unresolved) > 0 {
		return Job{}, ErrUnresolved
	}
	if existing, err := w.store.JobByIdempotency(ctx, organisationID, idempotencyKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Job{}, err
	}
	if !replace && draftExists(version) {
		return Job{}, ErrConflict
	}
	since := time.Now().UTC().Truncate(24 * time.Hour)
	count, err := w.store.CountGenerationJobs(ctx, organisationID, since)
	if err != nil {
		return Job{}, err
	}
	if count >= w.limits.TenantDailyJobs {
		return Job{}, ErrBudget
	}
	inputValue, err := w.generationInput(ctx, organisationID, version.ProfileID, record.Snapshot)
	if err != nil {
		return Job{}, err
	}
	input, err := json.Marshal(inputValue)
	if err != nil {
		return Job{}, err
	}
	now := clock()
	job, err := w.store.CreateGenerationJob(ctx, Job{
		OrganisationID:   organisationID,
		ProfileVersionID: versionID,
		ContextVersion:   record.Version,
		ContextHash:      record.Hash,
		IdempotencyKey:   idempotencyKey,
		ReplaceExisting:  replace,
		Status:           JobQueued,
		ModelID:          w.provider.ModelID(),
		PromptVersion:    PromptVersion,
		InputHash:        hashBytes(input),
		Unsupported:      []UnsupportedSuggestion{},
		Uncertainty:      []string{},
		CreatedBy:        actorID,
		CreatedAt:        now,
	})
	if err != nil {
		return Job{}, err
	}
	w.logger.Info("role profile generation queued", "jobId", job.ID, "organisationId", organisationID, "contextVersion", job.ContextVersion, "inputHash", job.InputHash)
	return job, nil
}

func (w *Workflow) GetJob(ctx context.Context, organisationID, jobID string) (Job, error) {
	return w.store.GetGenerationJob(ctx, organisationID, jobID)
}

func (w *Workflow) Retry(ctx context.Context, organisationID, jobID, actorID string) (Job, error) {
	job, err := w.store.GetGenerationJob(ctx, organisationID, jobID)
	if err != nil {
		return Job{}, err
	}
	if job.Status == JobQueued || job.Status == JobRunning || job.Status == JobSucceeded {
		return job, nil
	}
	record, err := w.store.EnsureContext(ctx, organisationID, job.ProfileVersionID, actorID)
	if err != nil {
		return Job{}, err
	}
	if record.Version != job.ContextVersion || record.Hash != job.ContextHash {
		job.Status = JobStale
		job.LastError = "stale_context"
		finished := clock()
		job.FinishedAt = &finished
		if err := w.store.SaveGenerationJob(ctx, job); err != nil {
			return Job{}, err
		}
		return job, ErrStale
	}
	job.Status = JobQueued
	job.LastError = ""
	job.CancelRequested = false
	job.FinishedAt = nil
	if err := w.store.SaveGenerationJob(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (w *Workflow) Cancel(ctx context.Context, organisationID, jobID string) (Job, error) {
	job, err := w.store.GetGenerationJob(ctx, organisationID, jobID)
	if err != nil {
		return Job{}, err
	}
	if job.Status == JobSucceeded {
		return job, ErrImmutable
	}
	job.CancelRequested = true
	if job.Status == JobQueued || job.Status == JobFailed || job.Status == JobStale {
		job.Status = JobCancelled
		finished := clock()
		job.FinishedAt = &finished
	}
	if err := w.store.SaveGenerationJob(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (w *Workflow) ProcessQueued(ctx context.Context) {
	jobs, err := w.store.ClaimGenerationJobs(ctx, 5, w.limits.RunningLease)
	if err != nil {
		w.logger.Error("role profile generation claim failed", "err", err.Error())
		return
	}
	for _, job := range jobs {
		w.process(ctx, job)
	}
}

func (w *Workflow) process(ctx context.Context, job Job) {
	if job.CancelRequested {
		w.finish(ctx, job, JobCancelled, "cancelled", nil)
		return
	}
	record, err := w.store.EnsureContext(ctx, job.OrganisationID, job.ProfileVersionID, job.CreatedBy)
	if err != nil {
		w.finish(ctx, job, JobFailed, "context", nil)
		return
	}
	if record.Hash != job.ContextHash || record.Version != job.ContextVersion {
		w.finish(ctx, job, JobStale, "stale_context", nil)
		return
	}
	version, err := w.store.GetVersion(ctx, job.OrganisationID, job.ProfileVersionID)
	if err != nil {
		w.finish(ctx, job, JobFailed, "context", nil)
		return
	}
	input, err := w.generationInput(ctx, job.OrganisationID, version.ProfileID, record.Snapshot)
	if err != nil {
		w.finish(ctx, job, JobFailed, "profile", nil)
		return
	}
	input.PromptVersion = job.PromptVersion
	var result ProviderResult
	started := time.Now()
	genCtx, stop := context.WithCancel(ctx)
	defer stop()
	go watchCancellation(genCtx, w.store, job, stop)
	calls := 0
	err = callWithRetry(genCtx, w.limits, func(attemptCtx context.Context) error {
		calls++
		current, currentErr := w.store.GetGenerationJob(attemptCtx, job.OrganisationID, job.ID)
		if currentErr == nil && current.CancelRequested {
			return context.Canceled
		}
		generated, callErr := w.provider.Generate(attemptCtx, input)
		if callErr != nil {
			return callErr
		}
		result = generated
		return nil
	})
	job.Attempts += calls
	job.LatencyMS = int(time.Since(started).Milliseconds())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			w.finish(ctx, job, JobCancelled, "cancelled", nil)
			return
		}
		w.finish(ctx, job, JobFailed, providerCode(err), nil)
		return
	}
	if len(result.Body) > w.limits.MaxOutputBytes || result.Usage.CompletionTokens > w.limits.MaxCompletionTokens {
		w.finish(ctx, job, JobFailed, "limit", nil)
		return
	}
	accepted, unsupported, uncertainty, err := validateGenerated(result.Body, input.Taxonomy)
	job.Result = append(json.RawMessage{}, result.Body...)
	job.OutputHash = hashBytes(result.Body)
	job.PromptTokens = result.Usage.PromptTokens
	job.CompletionTokens = result.Usage.CompletionTokens
	job.Unsupported = unsupported
	job.Uncertainty = uncertainty
	if err != nil || len(accepted) == 0 {
		code := "invalid_output"
		if err == nil {
			code = "no_accepted_skills"
		}
		w.finish(ctx, job, JobFailed, code, nil)
		return
	}
	fresh, err := w.store.EnsureContext(ctx, job.OrganisationID, job.ProfileVersionID, job.CreatedBy)
	if err != nil || fresh.Hash != job.ContextHash {
		w.finish(ctx, job, JobStale, "stale_context", nil)
		return
	}
	current, err := w.store.GetGenerationJob(ctx, job.OrganisationID, job.ID)
	if err == nil && current.CancelRequested {
		w.finish(ctx, job, JobCancelled, "cancelled", nil)
		return
	}
	if _, err := w.store.ApplyGeneratedDraft(ctx, job.OrganisationID, job.ProfileVersionID, job.ContextHash, GeneratedDraft{
		Model:                job.ModelID,
		PromptVersion:        job.PromptVersion,
		SourceContextVersion: itoa(job.ContextVersion),
		InputHash:            job.InputHash,
		GeneratedAt:          clock(),
		Purpose:              record.Snapshot.Purpose,
		Skills:               accepted,
	}, job.ReplaceExisting); err != nil {
		w.finish(ctx, job, JobFailed, "attach", nil)
		return
	}
	w.finish(ctx, job, JobSucceeded, "", nil)
	w.logger.Info("role profile generation finished", "jobId", job.ID, "status", JobSucceeded, "modelId", job.ModelID, "promptVersion", job.PromptVersion, "inputHash", job.InputHash, "outputHash", job.OutputHash, "latencyMs", job.LatencyMS)
}

func (w *Workflow) finish(ctx context.Context, job Job, status, code string, _ error) {
	job.Status = status
	job.LastError = code
	finished := clock()
	job.FinishedAt = &finished
	if err := w.store.SaveGenerationJob(ctx, job); err != nil {
		w.logger.Error("role profile generation save failed", "jobId", job.ID, "err", err.Error())
	}
}

func (w *Workflow) addQuestions(ctx context.Context, organisationID, actorID string, record ContextRecord, existing []Question, round int, drafts []QuestionDraft) (Session, error) {
	if len(drafts) > w.limits.MaxQuestions {
		drafts = drafts[:w.limits.MaxQuestions]
	}
	now := clock()
	questions := make([]Question, 0, len(drafts))
	for _, draft := range drafts {
		prompt, critical := questionMeta(draft.Field, draft.Question)
		id, err := newID()
		if err != nil {
			return Session{}, err
		}
		authored := draft.AuthoredBy
		if authored == "" {
			authored = QuestionService
		}
		questions = append(questions, Question{
			ID:               id,
			ProfileVersionID: record.ProfileVersionID,
			ContextVersion:   record.Version,
			Round:            round,
			Field:            draft.Field,
			Question:         prompt,
			Critical:         critical,
			AuthoredBy:       authored,
			AskedActorID:     actorID,
			AskedAt:          now,
		})
	}
	if err := w.store.SaveQuestions(ctx, organisationID, questions); err != nil {
		return Session{}, err
	}
	return w.session(record, append(existing, questions...)), nil
}

func (w *Workflow) session(record ContextRecord, questions []Question) Session {
	unresolved := criticalUnresolved(record.Snapshot, questions)
	for i := range questions {
		if questions[i].Critical && questions[i].AnsweredAt != nil && fieldValue(record.Snapshot, questions[i].Field) == "" {
			questions[i].Unresolved = true
		}
		questions[i].Reason = questionReason(questions[i].Field, record.Snapshot)
		questions[i].Options = questionOptions(questions[i].Field)
	}
	copied := append([]Question{}, questions...)
	if copied == nil {
		copied = []Question{}
	}
	return Session{
		ProfileVersionID: record.ProfileVersionID,
		ContextVersion:   record.Version,
		ContextHash:      record.Hash,
		Round:            maxRound(questions),
		Questions:        copied,
		Unresolved:       unresolved,
		Complete:         len(unresolved) == 0,
		Snapshot:         record.Snapshot,
	}
}

func validateGenerated(raw []byte, taxonomy []Skill) ([]GeneratedSkill, []UnsupportedSuggestion, []string, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var output generatedOutput
	if err := decoder.Decode(&output); err != nil {
		return nil, nil, nil, ErrInvalid
	}
	known := map[string]Skill{}
	for _, skill := range taxonomy {
		known[skill.ID] = skill
	}
	accepted := make([]GeneratedSkill, 0)
	unsupported := make([]UnsupportedSuggestion, 0)
	seen := map[string]struct{}{}
	kinds := map[string]struct{}{}
	for _, skill := range output.Skills {
		reason := skillProblem(skill, known, seen)
		if reason != "" {
			unsupported = append(unsupported, UnsupportedSuggestion{SkillID: skill.SkillID, Kind: skill.Kind, Reason: reason})
			continue
		}
		seen[skill.SkillID] = struct{}{}
		kinds[skill.Kind] = struct{}{}
		skill.Definition = strings.TrimSpace(skill.Definition)
		skill.Rationale = strings.TrimSpace(skill.Rationale)
		skill.Criteria = strings.TrimSpace(skill.Criteria)
		skill.Uncertainty = strings.TrimSpace(skill.Uncertainty)
		accepted = append(accepted, skill)
	}
	unsupported = append(unsupported, output.Unsupported...)
	uncertainty := make([]string, 0)
	for _, kind := range []string{KindTechnical, KindBehavioural, KindKnowledge} {
		if _, ok := kinds[kind]; !ok {
			uncertainty = append(uncertainty, "missing_kind:"+kind)
		}
	}
	for _, skill := range accepted {
		if skill.Uncertainty != "" {
			uncertainty = append(uncertainty, skill.SkillID+":"+skill.Uncertainty)
		}
	}
	return accepted, unsupported, uncertainty, nil
}

func skillProblem(skill GeneratedSkill, known map[string]Skill, seen map[string]struct{}) string {
	if strings.TrimSpace(skill.SkillID) == "" || strings.TrimSpace(skill.Definition) == "" || strings.TrimSpace(skill.Rationale) == "" {
		return "missing_field"
	}
	if !validKind(skill.Kind) {
		return "kind"
	}
	if !validProficiency(skill.RequiredProficiency) {
		return "proficiency"
	}
	if !validPriority(skill.Priority) {
		return "priority"
	}
	canonical, ok := known[skill.SkillID]
	if !ok {
		return "not_in_taxonomy"
	}
	if !kindAgrees(skill.Kind, canonical.Category) {
		return "category"
	}
	if _, ok := seen[skill.SkillID]; ok {
		return "duplicate"
	}
	if utf8.RuneCountInString(strings.TrimSpace(skill.Criteria)) > 2000 {
		return "criteria"
	}
	return ""
}

func (w *Workflow) generationInput(ctx context.Context, organisationID, profileID string, snapshot ContextSnapshot) (GenerationInput, error) {
	profile, err := w.store.GetProfile(ctx, organisationID, profileID)
	if err != nil {
		return GenerationInput{}, err
	}
	taxonomy, err := w.store.ListCanonicalSkills(ctx, organisationID)
	if err != nil {
		return GenerationInput{}, err
	}
	name := strings.TrimSpace(profile.RoleName)
	if name == "" {
		return GenerationInput{}, ErrInvalid
	}
	return GenerationInput{PromptVersion: PromptVersion, RoleName: name, Snapshot: snapshot, Taxonomy: taxonomy}, nil
}

func draftExists(version Version) bool {
	return len(version.Skills) > 0 || len(version.AI.Draft) > 0 || version.AI.GeneratedAt != nil
}

func kindAgrees(kind, category string) bool {
	switch kind {
	case KindTechnical:
		return category == CategoryTechnical
	case KindBehavioural:
		return category == CategoryLeadership || category == CategoryCommunication
	case KindKnowledge:
		return category == CategoryDomain || category == CategoryCompliance
	default:
		return false
	}
}

func deterministicQuestions(gaps []fieldGap, snapshot ContextSnapshot) []QuestionDraft {
	drafts := make([]QuestionDraft, 0)
	for _, gap := range gaps {
		prompt, ok := questionFor(gap.field, snapshot)
		if !ok {
			continue
		}
		drafts = append(drafts, QuestionDraft{Field: gap.field, Question: prompt, AuthoredBy: QuestionService})
	}
	return drafts
}

func ambiguousGaps(gaps []fieldGap) []fieldGap {
	matched := make([]fieldGap, 0)
	for _, gap := range gaps {
		if gap.ambiguous {
			matched = append(matched, gap)
		}
	}
	return matched
}

func hasCriticalMissing(gaps []fieldGap) bool {
	for _, gap := range gaps {
		if gap.critical && gap.missing {
			return true
		}
	}
	return false
}

func askableGaps(snapshot ContextSnapshot, questions []Question) []fieldGap {
	answered := map[string]bool{}
	for _, question := range questions {
		if question.AnsweredAt != nil {
			answered[question.Field] = true
		}
	}
	gaps := make([]fieldGap, 0)
	for _, gap := range contextGaps(snapshot) {
		if answered[gap.field] {
			continue
		}
		if gap.missing || gap.ambiguous {
			gaps = append(gaps, gap)
		}
	}
	return gaps
}

func openQuestions(questions []Question, snapshot ContextSnapshot) []Question {
	gaps := map[string]bool{}
	for _, gap := range contextGaps(snapshot) {
		if gap.missing || gap.ambiguous {
			gaps[gap.field] = true
		}
	}
	open := make([]Question, 0)
	for _, question := range questions {
		if question.AnsweredAt == nil && gaps[question.Field] {
			open = append(open, question)
		}
	}
	return open
}

func watchCancellation(ctx context.Context, store profileAI, job Job, stop context.CancelFunc) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current, err := store.GetGenerationJob(context.Background(), job.OrganisationID, job.ID)
			if err == nil && current.CancelRequested {
				stop()
				return
			}
		}
	}
}

func maxRound(questions []Question) int {
	round := 0
	for _, question := range questions {
		if question.Round > round {
			round = question.Round
		}
	}
	return round
}

func findQuestion(questions []Question, id string) (Question, bool) {
	for _, question := range questions {
		if question.ID == id {
			return question, true
		}
	}
	return Question{}, false
}

func mergeQuestions(base []QuestionDraft, extra []QuestionDraft) []QuestionDraft {
	allowed := map[string]int{}
	for i, draft := range base {
		allowed[draft.Field] = i
	}
	for _, draft := range extra {
		index, ok := allowed[draft.Field]
		cleaned := strings.TrimSpace(draft.Question)
		if !ok || cleaned == "" || len([]rune(cleaned)) > 500 {
			continue
		}
		base[index].Question = cleaned
		base[index].AuthoredBy = QuestionModel
	}
	return base
}

func questionMeta(field, question string) (string, bool) {
	fallback, _ := questionFor(field, ContextSnapshot{})
	critical := false
	for _, gap := range contextGaps(ContextSnapshot{}) {
		if gap.field == field {
			critical = gap.critical
		}
	}
	if strings.TrimSpace(question) == "" {
		question = fallback
	}
	return question, critical
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
