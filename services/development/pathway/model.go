package pathway

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid")
)

const (
	StageLearn        = "learn"
	StagePractise     = "practise"
	StageApply        = "apply"
	StageValidate     = "validate"
	StatusReady       = "ready"
	StatusComplete    = "complete"
	StatusUpcoming    = "upcoming"
	StatusRecommended = "recommended"
	VoiceUnconfigured = "unconfigured"
)

type Mapping struct {
	ID               string
	OrganisationID   string
	ExternalCourseID string
	CapabilityID     string
	TargetLevel      int
	Title            string
}

type MappingInput struct {
	OrganisationID   string
	ExternalCourseID string
	CapabilityID     string
	TargetLevel      int
	Title            string
}

type Pathway struct {
	ID             string
	OrganisationID string
	PersonID       string
	CapabilityID   string
	TargetLevel    int
	Items          []PlanItem
}

type PlanItem struct {
	ID       string
	Stage    string
	Title    string
	Status   string
	DeepLink string
}

type PathwayInput struct {
	OrganisationID string
	PersonID       string
	CapabilityID   string
	TargetLevel    int
}

type CompletionInput struct {
	OrganisationID string
	PersonID       string
	PlanItemID     string
	CompletedAt    time.Time
}

type LearningCompleted struct {
	Type           string
	OrganisationID string
	PersonID       string
	PlanItemID     string
	OccurredAt     string
}

type Publisher interface {
	PublishLearningCompleted(ctx context.Context, event LearningCompleted) error
}

type Store interface {
	MapCourse(ctx context.Context, input MappingInput) (Mapping, error)
	FindCourse(ctx context.Context, organisationID, capabilityID string, targetLevel int) (Mapping, error)
	CreatePathway(ctx context.Context, input PathwayInput) (Pathway, error)
	GetPathway(ctx context.Context, organisationID, personID, capabilityID string) (Pathway, error)
	Complete(ctx context.Context, input CompletionInput) (Pathway, bool, error)
	SaveResource(ctx context.Context, input ResourceInput) (Resource, error)
	ListResources(ctx context.Context, organisationID string) ([]Resource, error)
	SaveRecommendation(ctx context.Context, input LearningRecommendation) (LearningRecommendation, error)
	ListRecommendations(ctx context.Context, organisationID, personID string) ([]LearningRecommendation, error)
	SavePractice(ctx context.Context, input PracticeInput) (PracticeRecord, error)
	ListPractice(ctx context.Context, organisationID, personID string) ([]PracticeRecord, error)
}
