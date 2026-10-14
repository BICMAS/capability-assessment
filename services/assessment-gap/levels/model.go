package levels

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrInvalid   = errors.New("invalid")
	ErrNoWriter  = errors.New("no writer")
	ErrForbidden = errors.New("forbidden")
)

// Explanation is the persisted level calculation for one person, framework
// version, and capability.
type Explanation struct {
	ID                   string
	OrganisationID       string
	PersonID             string
	FrameworkVersionID   string
	CapabilityCode       string
	Knowledge            *int
	SituationalJudgement *int
	AIInterview          *int
	Simulation           *int
	ManagerAssessment    *int
	SelfAssessment       *int
	Mean                 Tenths
	Divisor              int
	RequiredLevel        Tenths
	CurrentLevel         Tenths
	Gap                  Tenths
	Status               string
}

// Calculation records writable attempts and persists the explanation.
// Simulation has no writer. AI interview is recorded by an admin or the person's manager.
type Calculation struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
	Sources            Sources
	RequiredLevel      Tenths
	Evidence           map[string][]string
}

// AttemptInput records one writable attempt and recalculates the level.
type AttemptInput struct {
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
	Source             string
	Percentage         int
	EvidenceIDs        []string
	RequiredLevel      Tenths
}

type Attempt struct {
	ID                 string
	OrganisationID     string
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
	Source             string
	Percentage         int
	EvidenceIDs        []string
	RequiredLevel      Tenths
	CreatedAt          time.Time
}

// Override is the manager's replacement for the calculated level.
// CurrentLevel on the explanation stays the calculated value.
type Override struct {
	ID             string
	OrganisationID string
	ExplanationID  string
	ActorPersonID  string
	Reason         string
	PreviousValue  Tenths
	NewValue       Tenths
	CreatedAt      time.Time
}

type OverrideInput struct {
	OrganisationID string
	ExplanationID  string
	ActorPersonID  string
	Reason         string
	NewValue       Tenths
}

// LevelOverridden is the audit event for a manager override.
type LevelOverridden struct {
	Type           string
	OrganisationID string
	OverrideID     string
	ActorPersonID  string
	OccurredAt     string
}

type Publisher interface {
	PublishLevelOverridden(ctx context.Context, event LevelOverridden) error
}

type Store interface {
	Calculate(ctx context.Context, input Calculation) (Explanation, error)
	Record(ctx context.Context, input AttemptInput) (Explanation, error)
	Get(ctx context.Context, organisationID, personID, frameworkVersionID, capabilityCode string) (Explanation, error)
	GetByID(ctx context.Context, organisationID, explanationID string) (Explanation, error)
	Override(ctx context.Context, input OverrideInput) (Override, error)
}
