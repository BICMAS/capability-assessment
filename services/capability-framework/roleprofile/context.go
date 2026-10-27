package roleprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	PromptVersion = "role-skills-profile-v1"

	KindTechnical   = "technical"
	KindBehavioural = "behavioural"
	KindKnowledge   = "knowledge"

	JobQueued    = "queued"
	JobRunning   = "running"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
	JobStale     = "stale"
	JobCancelled = "cancelled"

	ambiguousLimit = 24
)

type ContextSnapshot struct {
	Purpose          string      `json:"purpose"`
	Responsibilities string      `json:"responsibilities"`
	BusinessOutcomes string      `json:"businessOutcomes"`
	DecisionScope    string      `json:"decisionScope"`
	ReportingLevel   string      `json:"reportingLevel"`
	Industry         string      `json:"industry"`
	Region           string      `json:"region"`
	TeamSize         *int        `json:"teamSize"`
	Sources          []SourceRef `json:"sources"`
}

type SourceRef struct {
	Title        string `json:"title"`
	ReferenceURI string `json:"referenceUri"`
}

type ContextRecord struct {
	ProfileVersionID string
	Version          int
	Hash             string
	Snapshot         ContextSnapshot
	CreatedAt        time.Time
}

type Question struct {
	ID               string     `json:"id"`
	ProfileVersionID string     `json:"profileVersionId"`
	ContextVersion   int        `json:"contextVersion"`
	Round            int        `json:"round"`
	Field            string     `json:"field"`
	Question         string     `json:"question"`
	Critical         bool       `json:"critical"`
	Answer           string     `json:"answer"`
	Unresolved       bool       `json:"unresolved"`
	AskedActorID     string     `json:"askedActorId"`
	AnsweredActorID  string     `json:"answeredActorId"`
	AskedAt          time.Time  `json:"askedAt"`
	AnsweredAt       *time.Time `json:"answeredAt"`
}

type Session struct {
	ProfileVersionID string          `json:"profileVersionId"`
	ContextVersion   int             `json:"contextVersion"`
	ContextHash      string          `json:"contextHash"`
	Round            int             `json:"round"`
	Questions        []Question      `json:"questions"`
	Unresolved       []string        `json:"unresolved"`
	Complete         bool            `json:"complete"`
	Snapshot         ContextSnapshot `json:"snapshot"`
}

type AnswerInput struct {
	QuestionID string
	Value      string
}

type UnsupportedSuggestion struct {
	SkillID string `json:"skillId"`
	Kind    string `json:"kind"`
	Reason  string `json:"reason"`
}

type Job struct {
	ID               string                  `json:"id"`
	OrganisationID   string                  `json:"organisationId"`
	ProfileVersionID string                  `json:"profileVersionId"`
	ContextVersion   int                     `json:"contextVersion"`
	ContextHash      string                  `json:"contextHash"`
	IdempotencyKey   string                  `json:"idempotencyKey"`
	Status           string                  `json:"status"`
	ModelID          string                  `json:"modelId"`
	PromptVersion    string                  `json:"promptVersion"`
	InputHash        string                  `json:"inputHash"`
	OutputHash       string                  `json:"outputHash"`
	LatencyMS        int                     `json:"latencyMs"`
	PromptTokens     int                     `json:"promptTokens"`
	CompletionTokens int                     `json:"completionTokens"`
	Attempts         int                     `json:"attempts"`
	LastError        string                  `json:"lastError"`
	Unsupported      []UnsupportedSuggestion `json:"unsupported"`
	Uncertainty      []string                `json:"uncertainty"`
	Result           json.RawMessage         `json:"result,omitempty"`
	CancelRequested  bool                    `json:"cancelRequested"`
	CreatedBy        string                  `json:"createdBy"`
	CreatedAt        time.Time               `json:"createdAt"`
	StartedAt        *time.Time              `json:"startedAt"`
	FinishedAt       *time.Time              `json:"finishedAt"`
}

type GeneratedSkill struct {
	SkillID             string `json:"skillId"`
	Kind                string `json:"kind"`
	Definition          string `json:"definition"`
	RequiredProficiency int    `json:"requiredProficiency"`
	Priority            string `json:"priority"`
	Rationale           string `json:"rationale"`
	Uncertainty         string `json:"uncertainty,omitempty"`
}

type generatedOutput struct {
	Skills      []GeneratedSkill        `json:"skills"`
	Unsupported []UnsupportedSuggestion `json:"unsupported"`
}

func snapshotFrom(version Version) ContextSnapshot {
	sources := make([]SourceRef, 0, len(version.Sources))
	for _, source := range version.Sources {
		sources = append(sources, SourceRef{Title: source.Title, ReferenceURI: source.ReferenceURI})
	}
	return ContextSnapshot{
		Purpose:          strings.TrimSpace(version.Purpose),
		Responsibilities: strings.TrimSpace(version.Responsibilities),
		BusinessOutcomes: strings.TrimSpace(version.BusinessOutcomes),
		DecisionScope:    strings.TrimSpace(version.DecisionScope),
		ReportingLevel:   strings.TrimSpace(version.ReportingLevel),
		Industry:         strings.TrimSpace(version.Industry),
		Region:           strings.TrimSpace(version.Region),
		TeamSize:         copyInt(version.TeamSize),
		Sources:          sources,
	}
}

func hashSnapshot(snapshot ContextSnapshot) (string, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return hashBytes(payload), nil
}

func hashBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

type fieldGap struct {
	field     string
	critical  bool
	missing   bool
	ambiguous bool
}

func contextGaps(snapshot ContextSnapshot) []fieldGap {
	checks := []fieldGap{
		{field: "purpose", critical: true, missing: snapshot.Purpose == "", ambiguous: shortText(snapshot.Purpose)},
		{field: "responsibilities", critical: true, missing: snapshot.Responsibilities == "", ambiguous: shortText(snapshot.Responsibilities)},
		{field: "businessOutcomes", critical: true, missing: snapshot.BusinessOutcomes == "", ambiguous: shortText(snapshot.BusinessOutcomes)},
		{field: "decisionScope", critical: true, missing: snapshot.DecisionScope == "", ambiguous: shortText(snapshot.DecisionScope)},
		{field: "reportingLevel", critical: true, missing: !validReporting(snapshot.ReportingLevel)},
		{field: "industry", missing: snapshot.Industry == ""},
		{field: "region", missing: snapshot.Region == ""},
		{field: "teamSize", missing: snapshot.TeamSize == nil || *snapshot.TeamSize <= 0},
		{field: "sources", missing: len(snapshot.Sources) == 0},
	}
	return checks
}

func shortText(value string) bool {
	return value != "" && len(value) < ambiguousLimit
}

func questionFor(field string) (string, bool) {
	prompts := map[string]string{
		"purpose":          "What is the purpose of this role?",
		"responsibilities": "Which responsibilities does this role own?",
		"businessOutcomes": "Which business outcomes is this role accountable for?",
		"decisionScope":    "Which decisions can this role make without escalation?",
		"reportingLevel":   "Which reporting level applies: individual, team lead, manager, director, or executive?",
		"industry":         "Which industry does this role operate in?",
		"region":           "Which region does this role cover?",
		"teamSize":         "How many people does this role work with or lead?",
		"sources":          "Which approved source document should be used for this role?",
	}
	prompt, ok := prompts[field]
	return prompt, ok
}

func applyAnswer(snapshot ContextSnapshot, field, value string) (ContextSnapshot, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return snapshot, nil
	}
	switch field {
	case "purpose":
		snapshot.Purpose = value
	case "responsibilities":
		snapshot.Responsibilities = value
	case "businessOutcomes":
		snapshot.BusinessOutcomes = value
	case "decisionScope":
		snapshot.DecisionScope = value
	case "reportingLevel":
		if !validReporting(value) {
			return snapshot, ErrInvalid
		}
		snapshot.ReportingLevel = value
	case "industry":
		snapshot.Industry = value
	case "region":
		snapshot.Region = value
	case "teamSize":
		var size int
		for _, digit := range value {
			if digit < '0' || digit > '9' {
				return snapshot, ErrInvalid
			}
			size = size*10 + int(digit-'0')
		}
		if size <= 0 {
			return snapshot, ErrInvalid
		}
		snapshot.TeamSize = &size
	case "sources":
		snapshot.Sources = append(snapshot.Sources, SourceRef{Title: value, ReferenceURI: "answer:" + hashBytes([]byte(value))[:12]})
	default:
		return snapshot, ErrInvalid
	}
	return snapshot, nil
}

func contextFromSnapshot(snapshot ContextSnapshot) Context {
	sources := make([]Source, 0, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		sources = append(sources, Source{Title: source.Title, ReferenceURI: source.ReferenceURI})
	}
	return Context{
		Purpose:          snapshot.Purpose,
		Responsibilities: snapshot.Responsibilities,
		BusinessOutcomes: snapshot.BusinessOutcomes,
		DecisionScope:    snapshot.DecisionScope,
		Industry:         snapshot.Industry,
		Region:           snapshot.Region,
		TeamSize:         copyInt(snapshot.TeamSize),
		ReportingLevel:   snapshot.ReportingLevel,
		Sources:          sources,
	}
}

func criticalUnresolved(snapshot ContextSnapshot, questions []Question) []string {
	answered := map[string]bool{}
	for _, question := range questions {
		if question.AnsweredAt != nil {
			answered[question.Field] = true
		}
	}
	unresolved := make([]string, 0)
	for _, gap := range contextGaps(snapshot) {
		if !gap.critical {
			continue
		}
		if gap.missing || (answered[gap.field] && fieldValue(snapshot, gap.field) == "") {
			unresolved = append(unresolved, gap.field)
		}
	}
	return unresolved
}

func fieldValue(snapshot ContextSnapshot, field string) string {
	switch field {
	case "purpose":
		return snapshot.Purpose
	case "responsibilities":
		return snapshot.Responsibilities
	case "businessOutcomes":
		return snapshot.BusinessOutcomes
	case "decisionScope":
		return snapshot.DecisionScope
	case "reportingLevel":
		return snapshot.ReportingLevel
	default:
		return "set"
	}
}

func validKind(value string) bool {
	switch value {
	case KindTechnical, KindBehavioural, KindKnowledge:
		return true
	default:
		return false
	}
}
