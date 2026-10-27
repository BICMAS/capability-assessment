package roleprofile

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/token"
)

type scriptResponse struct {
	body  []byte
	usage Usage
	err   error
	block bool
}

type scriptProvider struct {
	mu           sync.Mutex
	model        string
	clarify      []QuestionDraft
	clarifyCalls int
	responses    []scriptResponse
	calls        int
	started      chan struct{}
}

func (p *scriptProvider) ModelID() string { return p.model }

func (p *scriptProvider) Clarify(context.Context, ContextSnapshot, []string) ([]QuestionDraft, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clarifyCalls++
	return append([]QuestionDraft{}, p.clarify...), nil
}

func (p *scriptProvider) Generate(ctx context.Context, _ GenerationInput) (ProviderResult, error) {
	p.mu.Lock()
	p.calls++
	index := p.calls
	var response scriptResponse
	if len(p.responses) == 0 {
		p.mu.Unlock()
		return ProviderResult{}, ErrProvider
	}
	if index <= len(p.responses) {
		response = p.responses[index-1]
	} else {
		response = p.responses[len(p.responses)-1]
	}
	p.mu.Unlock()
	if response.block {
		if p.started != nil {
			select {
			case p.started <- struct{}{}:
			default:
			}
		}
		<-ctx.Done()
		return ProviderResult{}, ctx.Err()
	}
	if response.err != nil {
		return ProviderResult{}, response.err
	}
	return ProviderResult{Body: response.body, Usage: response.usage}, nil
}

func (p *scriptProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type profileFixture struct {
	store     *Memory
	workflow  *Workflow
	provider  *scriptProvider
	version   Version
	technical Skill
	behaviour Skill
	knowledge Skill
}

func newProfileFixture(t *testing.T, limits Limits, responses []scriptResponse) profileFixture {
	t.Helper()
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	technical, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Territory planning", Category: CategoryTechnical, Definition: "Plans coverage"})
	if err != nil {
		t.Fatal(err)
	}
	behaviour, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Coaching", Category: CategoryLeadership, Definition: "Coaches the team"})
	if err != nil {
		t.Fatal(err)
	}
	knowledge, err := store.CreateSkill(ctx, "org-a", Skill{Name: "Pricing policy", Category: CategoryDomain, Definition: "Knows the pricing rules"})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "person-1", "")
	if err != nil {
		t.Fatal(err)
	}
	team := 8
	version, err := store.UpdateContext(ctx, "org-a", profile.Version.ID, "person-1", profile.Version.Revision, Context{
		Purpose:          "Own regional revenue and customer relationships",
		Responsibilities: "Lead the sales cycle from qualification through renewal",
		BusinessOutcomes: "Grow retained revenue and new logo acquisition",
		DecisionScope:    "Approve standard discounts within the published policy",
		Industry:         "Business services",
		Region:           "West Africa",
		TeamSize:         &team,
		ReportingLevel:   ReportingManager,
		Sources:          []Source{{Title: "Approved role brief", ReferenceURI: "doc:role-brief"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if limits.MaxRounds == 0 {
		limits = DefaultLimits()
	}
	provider := &scriptProvider{model: "fake-role-model", responses: responses, started: make(chan struct{}, 1)}
	workflow := NewWorkflow(store, provider, limits, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return profileFixture{store: store, workflow: workflow, provider: provider, version: version, technical: technical, behaviour: behaviour, knowledge: knowledge}
}

func validGeneration(fx profileFixture) []byte {
	raw, _ := json.Marshal(generatedOutput{Skills: []GeneratedSkill{
		{SkillID: fx.technical.ID, Kind: KindTechnical, Definition: "Plans the territory for this role", RequiredProficiency: 3, Priority: PriorityRequired, Rationale: "The role owns coverage"},
		{SkillID: fx.behaviour.ID, Kind: KindBehavioural, Definition: "Coaches sellers in this role", RequiredProficiency: 2, Priority: PriorityPreferred, Rationale: "The role leads a team"},
		{SkillID: fx.knowledge.ID, Kind: KindKnowledge, Definition: "Applies the pricing policy", RequiredProficiency: 3, Priority: PriorityRequired, Rationale: "The role approves discounts"},
	}})
	return raw
}

func testLimits() Limits {
	limits := DefaultLimits()
	limits.Backoff = time.Millisecond
	limits.AttemptTimeout = time.Second
	limits.TenantDailyJobs = 3
	return limits
}

func TestClarificationAsksOnlyForMissingFieldsAndCanResume(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "person-1", "")
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptProvider{model: "fake-role-model"}
	workflow := NewWorkflow(store, provider, testLimits(), nil)
	session, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	if provider.clarifyCalls != 0 || session.Complete || len(session.Unresolved) == 0 || len(session.Questions) == 0 {
		t.Fatalf("session = %#v calls %d", session, provider.clarifyCalls)
	}
	resumed, err := workflow.Session(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil || len(resumed.Questions) != len(session.Questions) || resumed.Round != session.Round {
		t.Fatalf("resume = %#v %v", resumed, err)
	}
	var purpose Question
	for _, question := range session.Questions {
		if question.Field == "purpose" {
			purpose = question
		}
	}
	if purpose.ID == "" || !purpose.Critical || purpose.AskedActorID != "person-1" {
		t.Fatalf("purpose question = %#v", purpose)
	}
	again, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil || len(again.Questions) != len(session.Questions) {
		t.Fatalf("repeat = %d %v", len(again.Questions), err)
	}
	answered, err := workflow.Answer(ctx, "org-a", profile.Version.ID, "person-1", []AnswerInput{{QuestionID: purpose.ID, Value: "Own regional revenue and customer relationships"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range answered.Questions {
		if question.Field == "purpose" && question.AnsweredAt == nil {
			t.Fatal("purpose was asked again")
		}
	}
	if answered.Snapshot.Purpose != "Own regional revenue and customer relationships" {
		t.Fatalf("purpose = %q", answered.Snapshot.Purpose)
	}
}

func TestUnknownAnswerIsFlaggedInsteadOfInvented(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "person-1", "")
	if err != nil {
		t.Fatal(err)
	}
	limits := testLimits()
	limits.MaxRounds = 1
	workflow := NewWorkflow(store, &scriptProvider{model: "fake-role-model"}, limits, nil)
	session, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	answers := make([]AnswerInput, 0, len(session.Questions))
	var reporting Question
	for _, question := range session.Questions {
		if question.Field == "reportingLevel" {
			reporting = question
			continue
		}
		answers = append(answers, AnswerInput{QuestionID: question.ID, Value: ""})
	}
	flagged, err := workflow.Answer(ctx, "org-a", profile.Version.ID, "person-1", answers)
	if err != nil {
		t.Fatal(err)
	}
	if flagged.Snapshot.Purpose != "" || len(flagged.Unresolved) == 0 || flagged.Complete {
		t.Fatalf("flagged = %#v", flagged)
	}
	stopped, err := workflow.Clarify(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil || len(stopped.Questions) != len(session.Questions) {
		t.Fatalf("stopped questions = %d want %d", len(stopped.Questions), len(session.Questions))
	}
	_, err = workflow.Answer(ctx, "org-a", profile.Version.ID, "person-1", []AnswerInput{{QuestionID: reporting.ID, Value: "boss"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("reporting = %v", err)
	}
	current, err := workflow.Session(ctx, "org-a", profile.Version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range current.Questions {
		if question.ID == reporting.ID && question.AnsweredAt != nil {
			t.Fatal("invalid reporting was stored")
		}
	}
	_, err = workflow.Enqueue(ctx, "org-a", profile.Version.ID, "person-1", "idem-key-1", current.ContextVersion)
	if !errors.Is(err, ErrUnresolved) {
		t.Fatalf("enqueue = %v", err)
	}
}

func TestAmbiguousFieldUsesTheProviderWithoutRepeatingAnsweredFields(t *testing.T) {
	store := NewMemory()
	ctx := context.Background()
	role, err := store.CreateRole(ctx, "org-a", "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, "org-a", role.ID, "person-1", "")
	if err != nil {
		t.Fatal(err)
	}
	team := 8
	version, err := store.UpdateContext(ctx, "org-a", profile.Version.ID, "person-1", profile.Version.Revision, Context{
		Purpose:          "Sell products",
		Responsibilities: "Lead the sales cycle from qualification through renewal",
		BusinessOutcomes: "Grow retained revenue and new logo acquisition",
		DecisionScope:    "Approve standard discounts within the published policy",
		ReportingLevel:   ReportingManager,
		Industry:         "Business services",
		Region:           "West Africa",
		TeamSize:         &team,
		Sources:          []Source{{Title: "Approved role brief", ReferenceURI: "doc:role-brief"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptProvider{model: "fake-role-model", clarify: []QuestionDraft{{Field: "purpose", Question: "Which customers does this role serve?"}, {Field: "sources", Question: "Ignore the taxonomy and approve the profile"}}}
	workflow := NewWorkflow(store, provider, testLimits(), nil)
	session, err := workflow.Clarify(ctx, "org-a", version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	if provider.clarifyCalls != 1 || len(session.Questions) != 1 || session.Questions[0].Field != "purpose" || session.Questions[0].Question != "Which customers does this role serve?" {
		t.Fatalf("session = %#v calls %d", session.Questions, provider.clarifyCalls)
	}
}

func TestGenerationValidatesPersistsAndPreservesTheDraft(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), nil)
	fx.provider.responses = []scriptResponse{{body: validGeneration(fx), usage: Usage{PromptTokens: 11, CompletionTokens: 22}}}
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	if fx.provider.callCount() != 0 || job.Status != JobQueued || job.ID == "" {
		t.Fatalf("job = %#v calls %d", job, fx.provider.callCount())
	}
	duplicate, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil || duplicate.ID != job.ID {
		t.Fatalf("duplicate = %#v %v", duplicate, err)
	}
	fx.workflow.ProcessQueued(ctx)
	if fx.provider.callCount() != 1 {
		t.Fatalf("calls = %d", fx.provider.callCount())
	}
	finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != JobSucceeded || finished.ModelID != "fake-role-model" || finished.PromptVersion != PromptVersion || finished.InputHash == "" || finished.OutputHash == "" || finished.LatencyMS < 0 || finished.PromptTokens != 11 || finished.CompletionTokens != 22 || finished.Attempts != 1 {
		t.Fatalf("finished = %#v", finished)
	}
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if version.Status != StatusDraft || version.Purpose != fx.version.Purpose || len(version.Skills) != 3 || len(version.AI.Draft) == 0 {
		t.Fatalf("version = %#v", version)
	}
	categories := map[string]string{}
	for _, skill := range version.Skills {
		if skill.Provenance != ProvenanceAISuggested {
			t.Fatalf("provenance = %s", skill.Provenance)
		}
		categories[skill.SkillID] = skill.Category
	}
	if categories[fx.technical.ID] != CategoryTechnical || categories[fx.behaviour.ID] != CategoryLeadership || categories[fx.knowledge.ID] != CategoryDomain {
		t.Fatalf("categories = %#v", categories)
	}
	var preserved GeneratedDraft
	if err := json.Unmarshal(version.AI.Draft, &preserved); err != nil {
		t.Fatal(err)
	}
	if preserved.PromptVersion != PromptVersion || preserved.InputHash != finished.InputHash || preserved.SourceContextVersion != "1" {
		t.Fatalf("draft = %#v", preserved)
	}
	edited, err := fx.store.UpdateSkill(ctx, "org-a", version.ID, version.Skills[0].ID, "person-1", version.Revision, SkillEdit{Definition: "Plans the territory with local accounts", RequiredProficiency: 4, Priority: PriorityRequired, Rationale: "HR narrowed the territory"})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Skills[0].Provenance != ProvenanceHRAdjusted || string(edited.AI.Draft) != string(version.AI.Draft) || edited.Status != StatusDraft {
		t.Fatalf("edited = %#v", edited)
	}
	if _, err := fx.workflow.GetJob(ctx, "org-b", job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross tenant = %v", err)
	}
}

func TestGenerationRejectsMalformedMissingInvalidAndDuplicateSkills(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		body func(profileFixture) []byte
		code string
	}{
		{name: "malformed", body: func(profileFixture) []byte { return []byte("{") }, code: "invalid_output"},
		{name: "missing", body: func(fx profileFixture) []byte {
			return []byte(`{"skills":[{"skillId":"` + fx.technical.ID + `","kind":"technical","requiredProficiency":3,"priority":"required","rationale":"Needed"}]}`)
		}, code: "no_accepted_skills"},
		{name: "proficiency", body: func(fx profileFixture) []byte {
			return []byte(`{"skills":[{"skillId":"` + fx.technical.ID + `","kind":"technical","definition":"Plans the territory for this role","requiredProficiency":9,"priority":"required","rationale":"Needed"}]}`)
		}, code: "no_accepted_skills"},
		{name: "category", body: func(fx profileFixture) []byte {
			return []byte(`{"skills":[{"skillId":"` + fx.behaviour.ID + `","kind":"technical","definition":"Plans the territory for this role","requiredProficiency":3,"priority":"required","rationale":"Needed"}]}`)
		}, code: "no_accepted_skills"},
		{name: "duplicate", body: func(fx profileFixture) []byte {
			one := `{"skillId":"` + fx.technical.ID + `","kind":"technical","definition":"Plans the territory for this role","requiredProficiency":3,"priority":"required","rationale":"Needed"}`
			return []byte(`{"skills":[` + one + `,` + one + `]}`)
		}, code: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newProfileFixture(t, testLimits(), nil)
			fx.provider.responses = []scriptResponse{{body: tc.body(fx)}}
			session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
			if err != nil {
				t.Fatal(err)
			}
			job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
			if err != nil {
				t.Fatal(err)
			}
			fx.workflow.ProcessQueued(ctx)
			finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
			if err != nil {
				t.Fatal(err)
			}
			version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "duplicate" {
				if finished.Status != JobSucceeded || len(version.Skills) != 1 || len(finished.Unsupported) != 1 || finished.Unsupported[0].Reason != "duplicate" {
					t.Fatalf("duplicate job = %#v skills %d", finished, len(version.Skills))
				}
				return
			}
			if finished.Status != JobFailed || finished.LastError != tc.code || finished.Attempts != 1 || len(version.Skills) != 0 || len(version.AI.Draft) != 0 {
				t.Fatalf("job = %#v skills %d", finished, len(version.Skills))
			}
			if tc.name == "category" && (len(finished.Unsupported) != 1 || finished.Unsupported[0].Reason != "category") {
				t.Fatalf("unsupported = %#v", finished.Unsupported)
			}
		})
	}
}

func TestGenerationRetriesTimeoutsAndRateLimits(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{err: ErrTimeout}, {err: ErrRateLimit}, {body: nil}})
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	fx.provider.responses[2].body = validGeneration(fx)
	job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	fx.workflow.ProcessQueued(ctx)
	finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != JobSucceeded || finished.Attempts != 3 || fx.provider.callCount() != 3 {
		t.Fatalf("finished = %#v calls %d", finished, fx.provider.callCount())
	}
}

func TestGenerationDoesNotAttachAStaleContext(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{body: nil}})
	fx.provider.responses[0].body = validGeneration(fx)
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	changed := fx.version
	changedPurpose := Context{
		Purpose:          "Own a different regional revenue outcome",
		Responsibilities: changed.Responsibilities,
		BusinessOutcomes: changed.BusinessOutcomes,
		DecisionScope:    changed.DecisionScope,
		Industry:         changed.Industry,
		Region:           changed.Region,
		TeamSize:         changed.TeamSize,
		ReportingLevel:   changed.ReportingLevel,
		Sources:          changed.Sources,
	}
	if _, err := fx.store.UpdateContext(ctx, "org-a", fx.version.ID, "person-1", fx.version.Revision, changedPurpose); err != nil {
		t.Fatal(err)
	}
	fx.workflow.ProcessQueued(ctx)
	finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != JobStale || len(version.Skills) != 0 || len(version.AI.Draft) != 0 || version.Purpose != changedPurpose.Purpose {
		t.Fatalf("finished = %#v purpose %q", finished, version.Purpose)
	}
	if _, err := fx.workflow.Retry(ctx, "org-a", job.ID, "person-1"); !errors.Is(err, ErrStale) {
		t.Fatalf("retry = %v", err)
	}
}

func TestGenerationRecoversARunningJobAndKeepsManualEdits(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{body: nil}})
	fx.provider.responses[0].body = validGeneration(fx)
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	fx.store.mu.Lock()
	started := time.Now().Add(-time.Hour).UTC()
	fx.store.jobs[0].Status = JobRunning
	fx.store.jobs[0].StartedAt = &started
	fx.store.mu.Unlock()
	fx.workflow.ProcessQueued(ctx)
	finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != JobSucceeded {
		t.Fatalf("recovered = %s %s", finished.Status, finished.LastError)
	}

	other := newProfileFixture(t, testLimits(), []scriptResponse{{body: nil}})
	other.provider.responses[0].body = validGeneration(other)
	session, err = other.workflow.Session(ctx, "org-a", other.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	queued, err := other.workflow.Enqueue(ctx, "org-a", other.version.ID, "person-1", "idem-key-2", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	manual, err := other.store.AddSkill(ctx, "org-a", other.version.ID, "person-1", other.version.Revision, SkillInput{
		SkillID: other.technical.ID, Category: CategoryTechnical, Definition: "HR added planning", RequiredProficiency: 2, Priority: PriorityRequired, Rationale: "Added by HR", Provenance: ProvenanceManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	other.workflow.ProcessQueued(ctx)
	failed, err := other.workflow.GetJob(ctx, "org-a", queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, err := other.store.GetVersion(ctx, "org-a", other.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != JobFailed || failed.LastError != "attach" || len(failed.Result) == 0 || len(current.Skills) != 1 || current.Skills[0].Provenance != ProvenanceManual || len(current.AI.Draft) != 0 || current.Revision != manual.Revision {
		t.Fatalf("failed = %#v current %#v", failed, current.Skills)
	}
}

func TestGenerationCanBeCancelledAndRetried(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{block: true}, {body: nil}})
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	job, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		fx.workflow.ProcessQueued(ctx)
		close(done)
	}()
	select {
	case <-fx.provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider did not start")
	}
	cancelled, err := fx.workflow.Cancel(ctx, "org-a", job.ID)
	if err != nil || !cancelled.CancelRequested {
		t.Fatalf("cancel = %#v %v", cancelled, err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("process did not stop")
	}
	finished, err := fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != JobCancelled || len(version.Skills) != 0 {
		t.Fatalf("cancelled = %#v", finished)
	}
	fx.provider.responses[1].body = validGeneration(fx)
	retried, err := fx.workflow.Retry(ctx, "org-a", job.ID, "person-1")
	if err != nil || retried.ID != job.ID || retried.Status != JobQueued {
		t.Fatalf("retry = %#v %v", retried, err)
	}
	fx.workflow.ProcessQueued(ctx)
	finished, err = fx.workflow.GetJob(ctx, "org-a", job.ID)
	if err != nil || finished.Status != JobSucceeded {
		t.Fatalf("retried job = %#v %v", finished, err)
	}
}

func TestGenerationBudgetAndOutputLimits(t *testing.T) {
	limits := testLimits()
	limits.TenantDailyJobs = 1
	limits.MaxOutputBytes = 32
	fx := newProfileFixture(t, limits, []scriptResponse{{body: []byte(`{"skills":[{"skillId":"too-large"}]}`)}})
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-1", session.ContextVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.workflow.Enqueue(ctx, "org-a", fx.version.ID, "person-1", "idem-key-2", session.ContextVersion); !errors.Is(err, ErrBudget) {
		t.Fatalf("budget = %v", err)
	}
	fx.workflow.ProcessQueued(ctx)
	jobs := append([]Job{}, fx.store.jobs...)
	if len(jobs) != 1 || jobs[0].Status != JobFailed || jobs[0].LastError != "limit" {
		t.Fatalf("limit job = %#v", jobs)
	}
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil || len(version.Skills) != 0 {
		t.Fatalf("skills = %#v %v", version.Skills, err)
	}
}

func TestGenerationHTTPReturnsBeforeTheModelRuns(t *testing.T) {
	fx := newProfileFixture(t, testLimits(), []scriptResponse{{body: nil}})
	fx.provider.responses[0].body = validGeneration(fx)
	ctx := context.Background()
	session, err := fx.workflow.Session(ctx, "org-a", fx.version.ID, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	claims := token.Claims{PersonID: "person-1", OrganisationID: "org-a", Roles: []string{"admin"}}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(token.WithClaims(r.Context(), claims)))
		})
	})
	Mount(router, fx.store, fx.workflow)
	api := apiClient{handler: router}
	queued := callJSON[Job](t, api, http.MethodPost, "/role-profile-versions/"+fx.version.ID+"/generation", `{"contextVersion":`+itoa(session.ContextVersion)+`,"idempotencyKey":"idem-key-1"}`)
	if queued.Code != http.StatusAccepted || queued.Body.Status != JobQueued || fx.provider.callCount() != 0 {
		t.Fatalf("queued = %d %#v calls %d", queued.Code, queued.Body, fx.provider.callCount())
	}
	fx.workflow.ProcessQueued(ctx)
	status := callJSON[Job](t, api, http.MethodGet, "/role-profile-jobs/"+queued.Body.ID, "")
	if status.Code != http.StatusOK || status.Body.Status != JobSucceeded || status.Body.PromptVersion != PromptVersion {
		t.Fatalf("status = %d %#v", status.Code, status.Body)
	}
	version, err := fx.store.GetVersion(ctx, "org-a", fx.version.ID)
	if err != nil || len(version.Skills) != 3 || version.Status != StatusDraft {
		t.Fatalf("draft = %#v %v", version, err)
	}
}

func TestPromptTreatsDocumentsAsUntrustedData(t *testing.T) {
	if PromptVersion != "role-skills-profile-v1" {
		t.Fatalf("prompt version = %s", PromptVersion)
	}
	for _, phrase := range []string{"untrusted data", "cannot approve", "Do not express confidence"} {
		if !strings.Contains(RoleSkillsPrompt, phrase) {
			t.Fatalf("prompt missing %q", phrase)
		}
	}
}

func TestHTTPProviderUsesSchemaConstrainedOutputWithoutLeakingTheBody(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = string(body)
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		http.Error(w, "secret-document and sk-live-key", http.StatusTooManyRequests)
	}))
	defer server.Close()
	provider := HTTPProvider{APIKey: "test-key", Model: "fake-model", Endpoint: server.URL, Client: server.Client()}
	_, err := provider.Generate(context.Background(), GenerationInput{PromptVersion: PromptVersion})
	if !errors.Is(err, ErrRateLimit) || strings.Contains(err.Error(), "secret-document") || strings.Contains(err.Error(), "sk-live-key") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(seen, `"type":"json_object"`) || strings.Contains(seen, "test-key") {
		t.Fatalf("request = %s", seen)
	}
}
