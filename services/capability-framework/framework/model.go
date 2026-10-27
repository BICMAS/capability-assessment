package framework

import (
	"context"
	"errors"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/csvimport"
)

const (
	StatusDraft     = "draft"
	StatusInReview  = "in_review"
	StatusApproved  = "approved"
	StatusPublished = "published"
	StatusRetired   = "retired"
)

var (
	ErrNotFound  = errors.New("framework version not found")
	ErrImmutable = errors.New("published version cannot be changed")
	ErrInvalid   = errors.New("invalid framework")
	ErrPending   = errors.New("pending proposal")
)

const (
	ProposalPending  = "pending"
	ProposalAccepted = "accepted"
	ProposalEdited   = "edited"
	ProposalRejected = "rejected"
)

type Role struct {
	ID   string
	Name string
}

type Indicator struct {
	ID        string
	Statement string
}

type ProficiencyLevel struct {
	ID         string
	Level      int
	Indicators []Indicator
}

type Capability struct {
	ID     string
	Name   string
	Levels []ProficiencyLevel
}

type Requirement struct {
	ID            string
	RoleID        string
	CapabilityID  string
	RequiredLevel int
}

type RowError struct {
	Row     int
	Message string
}

type Proposal struct {
	ID             string
	Kind           string
	Status         string
	CapabilityCode string
	Message        string
}

type Version struct {
	ID             string
	OrganisationID string
	FrameworkID    string
	FrameworkName  string
	VersionNumber  int
	Status         string
	Roles          []Role
	Capabilities   []Capability
	Requirements   []Requirement
	RowErrors      []RowError
	Proposals      []Proposal
}

type ImportResult struct {
	ObjectKey string
	Version   *Version
	RowErrors []RowError
	Proposals []Proposal
}

type Store interface {
	CreateDraft(ctx context.Context, organisationID, name string) (Version, error)
	ListVersions(ctx context.Context, organisationID string) ([]Version, error)
	GetVersion(ctx context.Context, organisationID, versionID string) (Version, error)
	AddRole(ctx context.Context, organisationID, versionID, name string) (Role, error)
	AddCapability(ctx context.Context, organisationID, versionID, name string) (Capability, error)
	AddRequirement(ctx context.Context, organisationID, versionID, roleID, capabilityID string, requiredLevel int) (Requirement, error)
	// AddRoleCapability creates a capability and its role requirement in one operation.
	AddRoleCapability(ctx context.Context, organisationID, versionID, roleID, name string, requiredLevel int) (Capability, Requirement, error)
	AddIndicator(ctx context.Context, organisationID, versionID, capabilityID string, level int, statement string) (Indicator, error)
	PatchRole(ctx context.Context, organisationID, versionID, roleID, name string) (Version, error)
	Publish(ctx context.Context, organisationID, versionID string) (Version, error)
	PublishWithEvent(ctx context.Context, organisationID, versionID string, event OutboxEvent) (Version, error)
	CreateNextDraft(ctx context.Context, organisationID, versionID string) (Version, error)
	ApplyImport(ctx context.Context, organisationID, objectKey string, parsed csvimport.ParseResult) (ImportResult, error)
	SubmitForReview(ctx context.Context, organisationID, versionID string) (Version, error)
	DecideProposal(ctx context.Context, organisationID, versionID, proposalID, status, message string) (Proposal, error)
	Approve(ctx context.Context, organisationID, versionID string) (Version, error)
	UpdateIndicator(ctx context.Context, organisationID, versionID, indicatorID, statement string) (Indicator, error)
}

type FrameworkPublished struct {
	Type               string `json:"type"`
	OrganisationID     string `json:"organisationId"`
	FrameworkVersionID string `json:"frameworkVersionId"`
	ActorPersonID      string `json:"actorPersonId"`
	OccurredAt         string `json:"occurredAt"`
}

const PublicationSchemaVersion = 1

type OutboxEvent struct {
	ID            string
	EventType     string
	SchemaVersion int
	AggregateID   string
	OccurredAt    time.Time
	CorrelationID string
	Payload       []byte
}

func (e OutboxEvent) validate() error {
	if e.ID == "" || e.EventType == "" || e.SchemaVersion < 1 || e.AggregateID == "" || e.CorrelationID == "" || e.OccurredAt.IsZero() || len(e.Payload) == 0 {
		return ErrInvalid
	}
	return nil
}

func importedFrameworkName(department string) string {
	if department == "" {
		return "Imported framework"
	}
	return department + " Framework"
}

func validLevel(level int) bool {
	return level >= 0 && level <= 4
}

func immutable(status string) bool {
	return status == StatusPublished || status == StatusRetired
}
