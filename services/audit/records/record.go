package records

import (
	"context"
	"time"
)

const (
	ActionPersonSignedIn       = "person.signed_in"
	ActionAssignmentChanged    = "assignment.changed"
	ActionFrameworkPublished   = "framework.published"
	ActionLevelOverridden      = "level.overridden"
	ActionLearningCompleted    = "learning.completed"
	EntityTypePerson           = "person"
	EntityTypeRoleAssignment   = "role_assignment"
	EntityTypeFrameworkVersion = "framework_version"
	EntityTypeLevelOverride    = "level_override"
	EntityTypePlanItem         = "plan_item"
)

type Record struct {
	ActorPersonID  string
	OrganisationID string
	Action         string
	EntityType     string
	EntityID       string
	OccurredAt     time.Time
}

type Store interface {
	Append(ctx context.Context, record Record) error
	List(ctx context.Context, organisationID string) ([]Record, error)
}
