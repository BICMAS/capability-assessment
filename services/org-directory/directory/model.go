package directory

import (
	"context"
	"errors"
	"time"
)

const SeedOrganisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"

var ErrNotFound = errors.New("person not found")

type Department struct {
	ID             string
	OrganisationID string
	Name           string
}

type Person struct {
	ID             string
	OrganisationID string
	Name           string
	DepartmentName string
}

type RoleTitle struct {
	ID    string
	Title string
}

type Profile struct {
	Person
	Roles []RoleTitle
}

type RoleAssignment struct {
	ID             string
	OrganisationID string
	PersonID       string
	PersonName     string
	Title          string
	CreatedAt      time.Time
}

type AssignmentChanged struct {
	Type             string `json:"type"`
	OrganisationID   string `json:"organisationId"`
	PersonID         string `json:"personId"`
	RoleAssignmentID string `json:"roleAssignmentId"`
	OccurredAt       string `json:"occurredAt"`
}

type Publisher interface {
	PublishAssignmentChanged(ctx context.Context, event AssignmentChanged) error
}

type Store interface {
	ListDepartments(ctx context.Context, organisationID string) ([]Department, error)
	ListPeople(ctx context.Context, organisationID string) ([]Person, error)
	ListRoleAssignments(ctx context.Context, organisationID string) ([]RoleAssignment, error)
	Profile(ctx context.Context, organisationID, personID string) (Profile, error)
	CreateRoleAssignment(ctx context.Context, organisationID, personID, title string, createdAt time.Time) (RoleAssignment, error)
}
