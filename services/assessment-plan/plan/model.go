package plan

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalid          = errors.New("invalid")
	ErrImmutable        = errors.New("approved plan cannot be changed")
	ErrNotPublished     = errors.New("framework version is not published")
	ErrRoleNotOnVersion = errors.New("role is not on this framework version")
	ErrUpstream         = errors.New("upstream unavailable")
	ErrPersonOutsideOrg = errors.New("person is outside the organisation")
	ErrAlreadyGenerated = errors.New("generated plan already exists")
	ErrNotEditable      = errors.New("plan cannot be edited in this status")
	ErrProvider         = errors.New("model provider is not configured")
	ErrNotApproved      = errors.New("plan is not approved")
)

const (
	AssignmentStatus = "assigned"

	StatusDraft            = "draft"
	StatusGenerated        = "generated"
	StatusUnderReview      = "under_review"
	StatusChangesRequested = "changes_requested"
	StatusApproved         = "approved"

	SessionAssigned  = "assigned"
	SessionStarted   = "started"
	SessionSubmitted = "submitted"

	OutcomeAssigned = "assigned"
	OutcomeSkipped  = "skipped"
	OutcomeFailed   = "failed"

	ReasonAssigned  = "assigned"
	ReasonException = "assigned with an exception reason"
	ReasonSkipped   = "already assigned to this version and role"
	ReasonOutside   = "person is outside the organisation"
	ReasonTitle     = "directory role title does not match the framework role"
)

type Assignment struct {
	ID                 string
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	RoleID             string
	AssignedByPersonID string
	AssignedAt         time.Time
	Status             string
	ExceptionReason    string
}

type AssignmentInput struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	RoleID             string
	AssignedByPersonID string
	AssignedAt         time.Time
	ExceptionReason    string
}

type AssessmentItem struct {
	ID     string
	Prompt string
}

type RubricBand struct {
	Level      int
	Descriptor string
}

type PlanSkill struct {
	ID                string
	RequirementID     string
	CapabilityID      string
	RequiredLevel     int
	RecommendedMethod string
	Rationale         string
	Instructions      string
	RequiresSMEReview bool
	RubricNeedsReview bool
	EditedByPersonID  string
	AssessmentItems   []AssessmentItem
	Rubric            []RubricBand
}

type ItemInput struct {
	Prompt string
}

type RubricInput struct {
	Level      int
	Descriptor string
}

type SkillInput struct {
	RequirementID     string
	CapabilityID      string
	RequiredLevel     int
	RecommendedMethod string
	Rationale         string
	Instructions      string
	RequiresSMEReview bool
	RubricNeedsReview bool
	EditedByPersonID  string
	AssessmentItems   []ItemInput
	Rubric            []RubricInput
}

type Plan struct {
	ID                 string
	OrganisationID     string
	AssignmentID       string
	VersionNumber      int
	Status             string
	ChangeReason       string
	ApprovedByPersonID string
	ApprovedAt         time.Time
	PromptVersion      string
	ModelName          string
	Skills             []PlanSkill
}

type EditSkillInput struct {
	OrganisationID    string
	PlanID            string
	SkillID           string
	EditorPersonID    string
	RecommendedMethod string
	Instructions      string
	AssessmentItems   []ItemInput
	Rubric            []RubricInput
}

type ReviewInput struct {
	OrganisationID string
	PlanID         string
	ActorPersonID  string
	Reason         string
	At             time.Time
}

type PlanInput struct {
	OrganisationID string
	AssignmentID   string
	PromptVersion  string
	ModelName      string
	Skills         []SkillInput
}

type ReplacePlanInput struct {
	OrganisationID string
	PlanID         string
	Status         string
	Skills         []SkillInput
}

type FrameworkRole struct {
	ID   string
	Name string
}

type BehaviouralIndicator struct {
	Level     int
	Statement string
}

type FrameworkCapability struct {
	ID         string
	Name       string
	Indicators []BehaviouralIndicator
}

type FrameworkRequirement struct {
	ID            string
	RoleID        string
	CapabilityID  string
	RequiredLevel int
}

type FrameworkVersion struct {
	ID           string
	Status       string
	Roles        []FrameworkRole
	Capabilities []FrameworkCapability
	Requirements []FrameworkRequirement
}

type FrameworkReader interface {
	ReadVersion(ctx context.Context, authorization, versionID string) (FrameworkVersion, error)
}

type PersonProfile struct {
	ID         string
	RoleTitles []string
}

type DirectoryReader interface {
	ReadPerson(ctx context.Context, authorization, personID string) (PersonProfile, error)
}

type AssessmentAssigned struct {
	Type               string
	OrganisationID     string
	PersonID           string
	AssignmentID       string
	ActorPersonID      string
	FrameworkVersionID string
	RoleID             string
	OccurredAt         string
}

type PlanApproved struct {
	Type           string
	OrganisationID string
	PersonID       string
	AssignmentID   string
	PlanID         string
	ActorPersonID  string
	VersionNumber  int
	OccurredAt     string
}

type Publisher interface {
	PublishAssessmentAssigned(ctx context.Context, event AssessmentAssigned) error
	PublishPlanApproved(ctx context.Context, event PlanApproved) error
}

type AssignPerson struct {
	PersonID        string
	ExceptionReason string
}

type AssignInput struct {
	Authorization      string
	OrganisationID     string
	ActorPersonID      string
	FrameworkVersionID string
	RoleID             string
	People             []AssignPerson
	AssignedAt         time.Time
}

type PersonResult struct {
	PersonID     string
	Outcome      string
	Reason       string
	AssignmentID string
}

type SessionItem struct {
	ID     string
	Prompt string
	Answer string
}

type SessionSkill struct {
	ID                string
	RequirementID     string
	CapabilityID      string
	RequiredLevel     int
	RecommendedMethod string
	Instructions      string
	Items             []SessionItem
}

type Proposal struct {
	PlanSkillID       string
	RequirementID     string
	CapabilityID      string
	RequiredLevel     int
	RecommendedMethod string
	EvidenceSource    string
	BandLevels        []int
	Percentage        int
}

type Session struct {
	ID             string
	OrganisationID string
	PlanID         string
	PersonID       string
	Status         string
	StartedAt      time.Time
	SubmittedAt    time.Time
	ConfirmedAt    time.Time
	Proposals      []Proposal
	Skills         []SessionSkill
}

type AnswerInput struct {
	ItemID string
	Answer string
}

type Store interface {
	CreateAssignment(ctx context.Context, input AssignmentInput) (Assignment, error)
	FindAssignment(ctx context.Context, organisationID, personID, frameworkVersionID, roleID string) (Assignment, error)
	GetAssignment(ctx context.Context, organisationID, assignmentID string) (Assignment, error)
	CreatePlan(ctx context.Context, input PlanInput) (Plan, error)
	CreateGeneratedPlan(ctx context.Context, input PlanInput) (Plan, error)
	GetPlan(ctx context.Context, organisationID, planID string) (Plan, error)
	ReplacePlan(ctx context.Context, input ReplacePlanInput) (Plan, error)
	EditSkill(ctx context.Context, input EditSkillInput) (Plan, error)
	SubmitPlan(ctx context.Context, input ReviewInput) (Plan, error)
	RequestChanges(ctx context.Context, input ReviewInput) (Plan, error)
	ApprovePlan(ctx context.Context, input ReviewInput) (Plan, error)
	CopyApprovedPlan(ctx context.Context, organisationID, planID string) (Plan, error)
	CreateSession(ctx context.Context, organisationID, planID string) (Session, error)
	ListSessions(ctx context.Context, organisationID, personID string) ([]Session, error)
	GetSession(ctx context.Context, organisationID, sessionID string) (Session, error)
	StartSession(ctx context.Context, organisationID, sessionID, personID string, at time.Time) (Session, error)
	SaveAnswers(ctx context.Context, organisationID, sessionID, personID string, answers []AnswerInput) (Session, error)
	SubmitSession(ctx context.Context, organisationID, sessionID, personID string, at time.Time, proposals []Proposal) (Session, error)
	MarkSessionConfirmed(ctx context.Context, organisationID, sessionID string, at time.Time) (Session, error)
}
