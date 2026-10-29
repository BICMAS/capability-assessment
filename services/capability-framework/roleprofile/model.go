package roleprofile

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft      = "profile_draft"
	StatusInReview   = "profile_in_review"
	StatusPublished  = "profile_published"
	StatusSuperseded = "profile_superseded"

	SourceReference      = "reference"
	SourceJobDescription = "job_description"

	OriginUser        = "user"
	OriginImported    = "imported"
	OriginAISuggested = "ai_suggested"

	QuestionService          = "service"
	QuestionModel            = "model"
	QuestionModelUnavailable = "service_after_model_error"

	DecisionApproved = "approved"
	DecisionRejected = "rejected"
	DecisionChanges  = "changes_requested"

	CategoryTechnical     = "technical"
	CategoryLeadership    = "leadership"
	CategoryDomain        = "domain"
	CategoryCommunication = "communication"
	CategoryCompliance    = "compliance"

	PriorityRequired  = "required"
	PriorityPreferred = "preferred"
	PriorityOptional  = "optional"

	ProvenanceAISuggested = "ai_suggested"
	ProvenanceHRAdjusted  = "hr_adjusted"
	ProvenanceImported    = "imported"
	ProvenanceManual      = "manually_added"

	ReportingIndividual = "individual"
	ReportingTeamLead   = "team_lead"
	ReportingManager    = "manager"
	ReportingDirector   = "director"
	ReportingExecutive  = "executive"

	ActionCreated          = "role_profile_created"
	ActionContext          = "context_updated"
	ActionSkillAdded       = "skill_added"
	ActionSkillUpdated     = "skill_updated"
	ActionSkillRemoved     = "skill_removed"
	ActionReordered        = "skills_reordered"
	ActionSubmitted        = "submitted_for_review"
	ActionPublished        = "published"
	ActionSuperseded       = "version_superseded"
	ActionApprovalAssigned = "approval_assigned"
	ActionStageApproved    = "approval_stage_approved"
	ActionRejected         = "approval_rejected"
	ActionResubmitted      = "resubmitted_for_review"
	ActionChangesRequested = "changes_requested"

	StageHRReview       = "hr_review"
	StageManagerReview  = "manager_review"
	StageLeaderApproval = "leader_approval"

	QueueAwaiting   = "awaiting"
	QueueSubmitted  = "submitted"
	QueueReviewed   = "reviewed"
	ActionComment   = "comment_added"
	ActionNextDraft = "next_draft_created"
	ActionAIDraft   = "ai_draft_attached"
)

var (
	ErrNotFound    = errors.New("role profile not found")
	ErrInvalid     = errors.New("invalid role profile")
	ErrConflict    = errors.New("stale role profile revision")
	ErrImmutable   = errors.New("role profile version cannot be changed")
	ErrDuplicate   = errors.New("duplicate role profile record")
	ErrStale       = errors.New("role context changed")
	ErrUnresolved  = errors.New("critical role context is unresolved")
	ErrBudget      = errors.New("generation budget exceeded")
	ErrCancelled   = errors.New("generation cancelled")
	ErrUnavailable = errors.New("role profile generation is unavailable")
	ErrForbidden   = errors.New("approval is not assigned to this person")
)

type Role struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	Name           string `json:"name"`
}

type Skill struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Definition     string `json:"definition"`
}

type Source struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ReferenceURI string `json:"referenceUri"`
	Kind         string `json:"kind,omitempty"`
	Body         string `json:"body,omitempty"`
	Origin       string `json:"origin,omitempty"`
}

type ApprovalAssignment struct {
	ID                 string    `json:"id"`
	OrganisationID     string    `json:"organisationId"`
	ProfileVersionID   string    `json:"profileVersionId"`
	AssigneePersonID   string    `json:"assigneePersonId"`
	AssignedByPersonID string    `json:"assignedByPersonId"`
	CreatedAt          time.Time `json:"createdAt"`
	StageKey           string    `json:"stageKey,omitempty"`
}

type ApprovalDecision struct {
	ID               string    `json:"id"`
	OrganisationID   string    `json:"organisationId"`
	ProfileVersionID string    `json:"profileVersionId"`
	AssignmentID     string    `json:"assignmentId"`
	ActorPersonID    string    `json:"actorPersonId"`
	Decision         string    `json:"decision"`
	Comment          string    `json:"comment"`
	CreatedAt        time.Time `json:"createdAt"`
	StageKey         string    `json:"stageKey,omitempty"`
	SubmissionCycle  int       `json:"submissionCycle,omitempty"`
}

type ApprovalPathStage struct {
	StageKey     string `json:"stageKey"`
	Label        string `json:"label"`
	RequiredRole string `json:"requiredRole"`
	Position     int    `json:"position"`
}

type WorkflowStep struct {
	ID               string `json:"id"`
	OrganisationID   string `json:"organisationId"`
	ProfileVersionID string `json:"profileVersionId"`
	StageKey         string `json:"stageKey"`
	Label            string `json:"label"`
	RequiredRole     string `json:"requiredRole"`
	Position         int    `json:"position"`
}

type ApprovalRecord struct {
	Assignments      []ApprovalAssignment `json:"assignments"`
	Decisions        []ApprovalDecision   `json:"decisions"`
	Steps            []WorkflowStep       `json:"steps"`
	CurrentStage     string               `json:"currentStage"`
	StageLabel       string               `json:"stageLabel"`
	AssigneePersonID string               `json:"assigneePersonId"`
	Outstanding      string               `json:"outstanding"`
	CanDecide        bool                 `json:"canDecide"`
	CanResubmit      bool                 `json:"canResubmit"`
}

type ApprovalQueueItem struct {
	ProfileID        string     `json:"profileId"`
	RoleName         string     `json:"roleName"`
	Department       string     `json:"department"`
	VersionID        string     `json:"versionId"`
	VersionNumber    int        `json:"versionNumber"`
	Revision         int        `json:"revision"`
	Status           string     `json:"status"`
	WorkflowStage    string     `json:"workflowStage"`
	StageLabel       string     `json:"stageLabel"`
	SubmittedAt      *time.Time `json:"submittedAt"`
	AssigneePersonID string     `json:"assigneePersonId"`
	Outstanding      string     `json:"outstanding"`
}

type SkillAssignment struct {
	ID                  string    `json:"id"`
	SkillID             string    `json:"skillId"`
	SkillName           string    `json:"skillName"`
	Category            string    `json:"category"`
	Definition          string    `json:"definition"`
	RequiredProficiency int       `json:"requiredProficiency"`
	Priority            string    `json:"priority"`
	Rationale           string    `json:"rationale"`
	Criteria            string    `json:"criteria,omitempty"`
	DisplayName         string    `json:"displayName,omitempty"`
	Provenance          string    `json:"provenance"`
	Position            int       `json:"position"`
	CreatedAt           time.Time `json:"createdAt,omitempty"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

type AIMetadata struct {
	Model                string          `json:"model"`
	SourceContextVersion string          `json:"sourceContextVersion"`
	GeneratedAt          *time.Time      `json:"generatedAt"`
	Draft                json.RawMessage `json:"draft"`
}

type Version struct {
	ID                    string            `json:"id"`
	ProfileID             string            `json:"profileId"`
	OrganisationID        string            `json:"organisationId"`
	VersionNumber         int               `json:"versionNumber"`
	Revision              int               `json:"revision"`
	Status                string            `json:"status"`
	Purpose               string            `json:"purpose"`
	Responsibilities      string            `json:"responsibilities"`
	BusinessOutcomes      string            `json:"businessOutcomes"`
	DecisionScope         string            `json:"decisionScope"`
	Industry              string            `json:"industry"`
	Region                string            `json:"region"`
	TeamSize              *int              `json:"teamSize"`
	ReportingLevel        string            `json:"reportingLevel"`
	OrganogramID          string            `json:"organogramId,omitempty"`
	PositionID            string            `json:"positionId,omitempty"`
	BusinessUnit          string            `json:"businessUnit,omitempty"`
	SubIndustry           string            `json:"subIndustry,omitempty"`
	OrganisationalContext string            `json:"organisationalContext,omitempty"`
	FrameworkVersionID    string            `json:"frameworkVersionId"`
	Sources               []Source          `json:"sources"`
	Skills                []SkillAssignment `json:"skills"`
	AI                    AIMetadata        `json:"ai"`
	CreatedAt             time.Time         `json:"createdAt"`
	SubmittedAt           *time.Time        `json:"submittedAt,omitempty"`
	SubmittedByPersonID   string            `json:"submittedByPersonId,omitempty"`
	SubmissionCycle       int               `json:"submissionCycle,omitempty"`
	WorkflowStage         string            `json:"workflowStage,omitempty"`
	ApprovedAt            *time.Time        `json:"approvedAt,omitempty"`
	ApprovedByPersonID    string            `json:"approvedByPersonId,omitempty"`
}

type Profile struct {
	ID                string    `json:"id"`
	OrganisationID    string    `json:"organisationId"`
	RoleID            string    `json:"roleId"`
	RoleName          string    `json:"roleName"`
	CreatedByPersonID string    `json:"createdByPersonId"`
	CreatedAt         time.Time `json:"createdAt"`
	Version           Version   `json:"version"`
}

type Context struct {
	Purpose               string
	Responsibilities      string
	BusinessOutcomes      string
	DecisionScope         string
	Industry              string
	Region                string
	TeamSize              *int
	ReportingLevel        string
	OrganogramID          string
	PositionID            string
	BusinessUnit          string
	SubIndustry           string
	OrganisationalContext string
	FrameworkVersionID    *string
	Sources               []Source
}

type SkillInput struct {
	SkillID             string
	Category            string
	Definition          string
	RequiredProficiency int
	Priority            string
	Rationale           string
	Criteria            string
	Provenance          string
}

type SkillEdit struct {
	Definition          string
	RequiredProficiency int
	Priority            string
	Rationale           string
	DisplayName         *string
	Category            *string
	Criteria            *string
}

type SkillRevision struct {
	AssignmentID string
	Edit         SkillEdit
}

type SuggestedSkill struct {
	SkillID             string
	Definition          string
	RequiredProficiency int
	Priority            string
	Rationale           string
}

type AIDraft struct {
	Model                string
	SourceContextVersion string
	GeneratedAt          time.Time
	Purpose              string
	Skills               []SuggestedSkill
}

type Comment struct {
	ID               string    `json:"id"`
	ProfileVersionID string    `json:"profileVersionId"`
	AuthorPersonID   string    `json:"authorPersonId"`
	Body             string    `json:"body"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AuditEvent struct {
	ID               string    `json:"id"`
	ProfileVersionID string    `json:"profileVersionId"`
	ActorPersonID    string    `json:"actorPersonId"`
	Action           string    `json:"action"`
	OccurredAt       time.Time `json:"occurredAt"`
}

type ReviewChange struct {
	ID                string    `json:"id"`
	ProfileVersionID  string    `json:"profileVersionId"`
	SkillAssignmentID string    `json:"skillAssignmentId"`
	ActorPersonID     string    `json:"actorPersonId"`
	Fields            []string  `json:"fields"`
	Rationale         string    `json:"rationale"`
	OccurredAt        time.Time `json:"occurredAt"`
}

type History struct {
	ProfileID string         `json:"profileId"`
	Events    []AuditEvent   `json:"events"`
	Comments  []Comment      `json:"comments"`
	Changes   []ReviewChange `json:"changes"`
}

type SkillChange struct {
	SkillID         string `json:"skillId"`
	FromProficiency int    `json:"fromProficiency"`
	ToProficiency   int    `json:"toProficiency"`
	FromPriority    string `json:"fromPriority"`
	ToPriority      string `json:"toPriority"`
}

type Comparison struct {
	From    Version       `json:"from"`
	To      Version       `json:"to"`
	Added   []string      `json:"added"`
	Removed []string      `json:"removed"`
	Changed []SkillChange `json:"changed"`
}

type Store interface {
	CreateRole(ctx context.Context, organisationID, name string) (Role, error)
	CreateSkill(ctx context.Context, organisationID string, skill Skill) (Skill, error)
	CreateDraft(ctx context.Context, organisationID, roleID, actorID, frameworkVersionID string) (Profile, error)
	GetProfile(ctx context.Context, organisationID, profileID string) (Profile, error)
	UpdateContext(ctx context.Context, organisationID, versionID, actorID string, expected int, details Context) (Version, error)
	AddSkill(ctx context.Context, organisationID, versionID, actorID string, expected int, input SkillInput) (Version, error)
	UpdateSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int, edit SkillEdit) (Version, error)
	SaveEdits(ctx context.Context, organisationID, versionID, actorID string, expected int, edits []SkillRevision) (Version, error)
	MergeSkills(ctx context.Context, organisationID, versionID, actorID, keepID, removeID string, expected int) (Version, error)
	RemoveSkill(ctx context.Context, organisationID, versionID, assignmentID, actorID string, expected int) (Version, error)
	ReorderSkills(ctx context.Context, organisationID, versionID, actorID string, expected int, assignmentIDs []string) (Version, error)
	ListVersions(ctx context.Context, organisationID, profileID string) ([]Version, error)
	Compare(ctx context.Context, organisationID, profileID, fromID, toID string) (Comparison, error)
	Submit(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error)
	Publish(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error)
	NextDraft(ctx context.Context, organisationID, versionID, actorID string, expected int) (Version, error)
	Comment(ctx context.Context, organisationID, versionID, actorID, body string) (Comment, error)
	History(ctx context.Context, organisationID, profileID string) (History, error)
	AssignApproval(ctx context.Context, organisationID, versionID, actorID, assigneeID string) (ApprovalAssignment, error)
	ListApprovals(ctx context.Context, organisationID, versionID, actorID string, roles []string) (ApprovalRecord, error)
	DecideApproval(ctx context.Context, organisationID, versionID, assignmentID, actorID, decision, comment string, expected int, privileged bool, roles []string) (Version, error)
	GetApprovalPath(ctx context.Context, organisationID string) ([]ApprovalPathStage, error)
	SaveApprovalPath(ctx context.Context, organisationID, actorID string, stages []ApprovalPathStage) error
	ListApprovalQueue(ctx context.Context, organisationID, personID, view string, roles []string) ([]ApprovalQueueItem, error)
	Resubmit(ctx context.Context, organisationID, versionID, actorID string, expected int, roles []string) (Version, error)
	AttachAIDraft(ctx context.Context, organisationID, versionID, actorID string, draft AIDraft) (Version, error)
}

func validCategory(value string) bool {
	switch value {
	case CategoryTechnical, CategoryLeadership, CategoryDomain, CategoryCommunication, CategoryCompliance:
		return true
	default:
		return false
	}
}

func validPriority(value string) bool {
	switch value {
	case PriorityRequired, PriorityPreferred, PriorityOptional:
		return true
	default:
		return false
	}
}

func validProficiency(value int) bool {
	return value >= 0 && value <= 4
}

func ProficiencyLabel(level int) string {
	switch level {
	case 1:
		return "Basic"
	case 2:
		return "Intermediate"
	case 3:
		return "Advanced"
	case 4:
		return "Expert"
	default:
		return "Not set"
	}
}

func sourceKind(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SourceReference:
		return SourceReference, nil
	case SourceJobDescription:
		return SourceJobDescription, nil
	default:
		return "", ErrInvalid
	}
}

func validDecision(value string) bool {
	switch value {
	case DecisionApproved, DecisionRejected, DecisionChanges:
		return true
	default:
		return false
	}
}

func validReporting(value string) bool {
	switch value {
	case ReportingIndividual, ReportingTeamLead, ReportingManager, ReportingDirector, ReportingExecutive:
		return true
	default:
		return false
	}
}

func clientProvenance(value string) bool {
	return value == ProvenanceManual || value == ProvenanceImported
}

func validContext(input Context) error {
	if input.TeamSize != nil && *input.TeamSize <= 0 {
		return ErrInvalid
	}
	if input.ReportingLevel != "" && !validReporting(input.ReportingLevel) {
		return ErrInvalid
	}
	if err := validReferencePair(input.OrganogramID, input.PositionID); err != nil {
		return err
	}
	if tooLong(input.Purpose, 4000) || tooLong(input.Responsibilities, 4000) || tooLong(input.BusinessOutcomes, 4000) || tooLong(input.DecisionScope, 4000) || tooLong(input.BusinessUnit, 2000) || tooLong(input.SubIndustry, 200) || tooLong(input.OrganisationalContext, 4000) || tooLong(input.Industry, 200) || tooLong(input.Region, 200) {
		return ErrInvalid
	}
	for _, source := range input.Sources {
		if strings.TrimSpace(source.Title) == "" || strings.TrimSpace(source.ReferenceURI) == "" || tooLong(source.Body, 20000) {
			return ErrInvalid
		}
		if _, err := sourceKind(source.Kind); err != nil {
			return err
		}
		if _, err := sourceOrigin(source.Origin); err != nil {
			return err
		}
	}
	return nil
}

func validReferencePair(organogramID, positionID string) error {
	organogramID = strings.TrimSpace(organogramID)
	positionID = strings.TrimSpace(positionID)
	if (organogramID == "") != (positionID == "") || tooLong(organogramID, 80) || tooLong(positionID, 80) {
		return ErrInvalid
	}
	if organogramID != "" && (!referenceToken(organogramID) || !referenceToken(positionID)) {
		return ErrInvalid
	}
	return nil
}

func referenceToken(value string) bool {
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func tooLong(value string, limit int) bool {
	return len([]rune(value)) > limit
}

func sourceOrigin(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", OriginUser:
		return OriginUser, nil
	case OriginImported, OriginAISuggested:
		return strings.TrimSpace(value), nil
	default:
		return "", ErrInvalid
	}
}

func validSkillInput(input SkillInput) error {
	if strings.TrimSpace(input.SkillID) == "" || strings.TrimSpace(input.Definition) == "" || !validCategory(input.Category) || !validProficiency(input.RequiredProficiency) || !validPriority(input.Priority) || !clientProvenance(input.Provenance) || tooLong(strings.TrimSpace(input.Criteria), 2000) {
		return ErrInvalid
	}
	return nil
}

func validSkillEdit(edit SkillEdit) error {
	if strings.TrimSpace(edit.Definition) == "" || !validProficiency(edit.RequiredProficiency) || !validPriority(edit.Priority) {
		return ErrInvalid
	}
	return nil
}

func readyForReview(version Version) error {
	if strings.TrimSpace(version.Purpose) == "" || strings.TrimSpace(version.Responsibilities) == "" || strings.TrimSpace(version.BusinessOutcomes) == "" || strings.TrimSpace(version.DecisionScope) == "" || strings.TrimSpace(version.Industry) == "" || strings.TrimSpace(version.Region) == "" {
		return ErrInvalid
	}
	if !validReporting(version.ReportingLevel) || version.TeamSize == nil || *version.TeamSize <= 0 || len(version.Skills) == 0 {
		return ErrInvalid
	}
	return nil
}

func permutation(skills []SkillAssignment, ids []string) bool {
	if len(skills) == 0 || len(skills) != len(ids) {
		return false
	}
	seen := make(map[string]struct{}, len(ids))
	index := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		index[skill.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return false
		}
		if _, ok := index[id]; !ok {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func compareVersions(from, to Version) Comparison {
	fromSkills := make(map[string]SkillAssignment, len(from.Skills))
	for _, skill := range from.Skills {
		fromSkills[skill.SkillID] = skill
	}
	toSkills := make(map[string]SkillAssignment, len(to.Skills))
	for _, skill := range to.Skills {
		toSkills[skill.SkillID] = skill
	}
	added := make([]string, 0)
	removed := make([]string, 0)
	changed := make([]SkillChange, 0)
	for id, skill := range toSkills {
		previous, ok := fromSkills[id]
		if !ok {
			added = append(added, id)
			continue
		}
		if previous.RequiredProficiency != skill.RequiredProficiency || previous.Priority != skill.Priority || previous.Definition != skill.Definition || previous.Rationale != skill.Rationale {
			changed = append(changed, SkillChange{
				SkillID:         id,
				FromProficiency: previous.RequiredProficiency,
				ToProficiency:   skill.RequiredProficiency,
				FromPriority:    previous.Priority,
				ToPriority:      skill.Priority,
			})
		}
	}
	for id := range fromSkills {
		if _, ok := toSkills[id]; !ok {
			removed = append(removed, id)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Slice(changed, func(i, j int) bool { return changed[i].SkillID < changed[j].SkillID })
	from.Skills = nonNilSkills(from.Skills)
	to.Skills = nonNilSkills(to.Skills)
	from.Sources = nonNilSources(from.Sources)
	to.Sources = nonNilSources(to.Sources)
	return Comparison{From: from, To: to, Added: added, Removed: removed, Changed: changed}
}

func nonNilSkills(skills []SkillAssignment) []SkillAssignment {
	if skills == nil {
		return []SkillAssignment{}
	}
	return skills
}

func nonNilSources(sources []Source) []Source {
	if sources == nil {
		return []Source{}
	}
	return sources
}

func blankVersion(id, profileID, organisationID, frameworkVersionID string, number int, createdAt time.Time) Version {
	return Version{
		ID:                 id,
		ProfileID:          profileID,
		OrganisationID:     organisationID,
		VersionNumber:      number,
		Revision:           1,
		Status:             StatusDraft,
		FrameworkVersionID: frameworkVersionID,
		Sources:            []Source{},
		Skills:             []SkillAssignment{},
		CreatedAt:          createdAt,
	}
}
