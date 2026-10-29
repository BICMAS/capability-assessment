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
	Purpose               string      `json:"purpose"`
	Responsibilities      string      `json:"responsibilities"`
	BusinessOutcomes      string      `json:"businessOutcomes"`
	DecisionScope         string      `json:"decisionScope"`
	ReportingLevel        string      `json:"reportingLevel"`
	Industry              string      `json:"industry"`
	Region                string      `json:"region"`
	TeamSize              *int        `json:"teamSize"`
	OrganogramID          string      `json:"organogramId,omitempty"`
	PositionID            string      `json:"positionId,omitempty"`
	BusinessUnit          string      `json:"businessUnit,omitempty"`
	SubIndustry           string      `json:"subIndustry,omitempty"`
	OrganisationalContext string      `json:"organisationalContext,omitempty"`
	FrameworkVersionID    string      `json:"frameworkVersionId,omitempty"`
	Sources               []SourceRef `json:"sources"`
}

type SourceRef struct {
	Title        string `json:"title"`
	ReferenceURI string `json:"referenceUri"`
	Kind         string `json:"kind,omitempty"`
	Body         string `json:"body,omitempty"`
	Origin       string `json:"origin,omitempty"`
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
	AuthoredBy       string     `json:"authoredBy,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	Options          []string   `json:"options,omitempty"`
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
	ReplaceExisting  bool                    `json:"replaceExisting,omitempty"`
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
	Criteria            string `json:"criteria,omitempty"`
	Uncertainty         string `json:"uncertainty,omitempty"`
}

type generatedOutput struct {
	Skills      []GeneratedSkill        `json:"skills"`
	Unsupported []UnsupportedSuggestion `json:"unsupported"`
}

func snapshotFrom(version Version) ContextSnapshot {
	sources := make([]SourceRef, 0, len(version.Sources))
	for _, source := range version.Sources {
		sources = append(sources, sourceRef(source))
	}
	return ContextSnapshot{
		Purpose:               strings.TrimSpace(version.Purpose),
		Responsibilities:      strings.TrimSpace(version.Responsibilities),
		BusinessOutcomes:      strings.TrimSpace(version.BusinessOutcomes),
		DecisionScope:         strings.TrimSpace(version.DecisionScope),
		ReportingLevel:        strings.TrimSpace(version.ReportingLevel),
		Industry:              strings.TrimSpace(version.Industry),
		Region:                strings.TrimSpace(version.Region),
		TeamSize:              copyInt(version.TeamSize),
		OrganogramID:          strings.TrimSpace(version.OrganogramID),
		PositionID:            strings.TrimSpace(version.PositionID),
		BusinessUnit:          strings.TrimSpace(version.BusinessUnit),
		SubIndustry:           strings.TrimSpace(version.SubIndustry),
		OrganisationalContext: strings.TrimSpace(version.OrganisationalContext),
		FrameworkVersionID:    strings.TrimSpace(version.FrameworkVersionID),
		Sources:               sources,
	}
}

func sourceRef(source Source) SourceRef {
	kind := source.Kind
	if kind == SourceReference {
		kind = ""
	}
	origin := source.Origin
	if origin == "" || origin == OriginUser {
		origin = ""
	}
	return SourceRef{
		Title:        source.Title,
		ReferenceURI: source.ReferenceURI,
		Kind:         kind,
		Body:         strings.TrimSpace(source.Body),
		Origin:       origin,
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

func questionFor(field string, snapshot ContextSnapshot) (string, bool) {
	subject := roleSubject(snapshot)
	prompts := map[string]string{
		"purpose":          "What is the purpose of " + subject + "?",
		"responsibilities": "Which duties must " + subject + " perform, and which technical, behavioural, and knowledge requirements are especially important?",
		"businessOutcomes": "What are the three to five most important business outcomes of " + subject + "?",
		"decisionScope":    "Which decisions must " + subject + " make independently?",
		"reportingLevel":   "Which level of responsibility applies to " + subject + ": individual, team lead, manager, director, or executive?",
		"industry":         "Which industry, sub-industry, and operating conditions apply to " + subject + "?",
		"region":           "Which geographic or operating scope applies to " + subject + "?",
		"teamSize":         "How many people does " + subject + " supervise or work with?",
		"sources":          "Which job description or approved source should be used for " + subject + "?",
	}
	prompt, ok := prompts[field]
	return prompt, ok
}

func roleSubject(snapshot ContextSnapshot) string {
	purpose := strings.TrimSpace(snapshot.Purpose)
	runes := []rune(purpose)
	if len(runes) < 12 {
		return "this role"
	}
	if len(runes) > 80 {
		purpose = string(runes[:80])
	}
	return "this role (" + purpose + ")"
}

func questionReason(field string, snapshot ContextSnapshot) string {
	if fieldMissing(field, snapshot) {
		return "This was not supplied. The skills profile should not guess it."
	}
	return "This is too brief to tell which conditions apply to the role. A more specific answer keeps the skills profile tied to the recorded purpose."
}

func fieldMissing(field string, snapshot ContextSnapshot) bool {
	switch field {
	case "reportingLevel":
		return !validReporting(snapshot.ReportingLevel)
	case "teamSize":
		return snapshot.TeamSize == nil || *snapshot.TeamSize <= 0
	case "sources":
		return len(snapshot.Sources) == 0
	case "industry":
		return strings.TrimSpace(snapshot.Industry) == ""
	case "region":
		return strings.TrimSpace(snapshot.Region) == ""
	default:
		return strings.TrimSpace(fieldValue(snapshot, field)) == ""
	}
}

func questionOptions(field string) []string {
	if field != "reportingLevel" {
		return nil
	}
	return []string{ReportingIndividual, ReportingTeamLead, ReportingManager, ReportingDirector, ReportingExecutive}
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
		kind := source.Kind
		if kind == "" {
			kind = SourceReference
		}
		origin := source.Origin
		if origin == "" {
			origin = OriginUser
		}
		sources = append(sources, Source{Title: source.Title, ReferenceURI: source.ReferenceURI, Kind: kind, Body: source.Body, Origin: origin})
	}
	details := Context{
		Purpose:               snapshot.Purpose,
		Responsibilities:      snapshot.Responsibilities,
		BusinessOutcomes:      snapshot.BusinessOutcomes,
		DecisionScope:         snapshot.DecisionScope,
		Industry:              snapshot.Industry,
		Region:                snapshot.Region,
		TeamSize:              copyInt(snapshot.TeamSize),
		ReportingLevel:        snapshot.ReportingLevel,
		OrganogramID:          snapshot.OrganogramID,
		PositionID:            snapshot.PositionID,
		BusinessUnit:          snapshot.BusinessUnit,
		SubIndustry:           snapshot.SubIndustry,
		OrganisationalContext: snapshot.OrganisationalContext,
		Sources:               sources,
	}
	if snapshot.FrameworkVersionID != "" {
		framework := snapshot.FrameworkVersionID
		details.FrameworkVersionID = &framework
	}
	return details
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
