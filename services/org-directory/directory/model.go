package directory

import (
	"context"
	"errors"
	"time"
)

const SeedOrganisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"

var (
	ErrNotFound   = errors.New("directory record not found")
	ErrInvalid    = errors.New("invalid directory record")
	ErrDuplicate  = errors.New("directory record already exists")
	ErrCycle      = errors.New("reporting relationship is circular")
	ErrHasReports = errors.New("position still has people reporting to it")
	ErrTooLarge   = errors.New("organogram file is too large")
)

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

type Organogram struct {
	ID                string    `json:"id"`
	OrganisationID    string    `json:"organisationId"`
	Name              string    `json:"name"`
	Status            string    `json:"status"`
	CreatedByPersonID string    `json:"createdByPersonId"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Position struct {
	ID                  string    `json:"id"`
	OrganisationID      string    `json:"organisationId"`
	OrganogramID        string    `json:"organogramId"`
	DepartmentID        string    `json:"departmentId"`
	Title               string    `json:"title"`
	PersonID            string    `json:"personId"`
	ReportsToPositionID string    `json:"reportsToPositionId"`
	CreatedAt           time.Time `json:"createdAt"`
}

type PositionInput struct {
	DepartmentID        string
	Title               string
	PersonID            string
	ReportsToPositionID string
}

const (
	OrganogramDraft     = "draft"
	OrganogramPublished = "published"
	FileImported        = "imported"
	FileRejected        = "rejected"
	FileNeedsReview     = "needs_review"
	MaxOrganogramBytes  = 5 << 20
)

type OrganogramFile struct {
	ID                string    `json:"id"`
	OrganisationID    string    `json:"organisationId"`
	OrganogramID      string    `json:"organogramId"`
	FileName          string    `json:"fileName"`
	MediaType         string    `json:"mediaType"`
	ByteSize          int       `json:"byteSize"`
	Status            string    `json:"status"`
	Detail            string    `json:"detail"`
	CreatedByPersonID string    `json:"createdByPersonId"`
	CreatedAt         time.Time `json:"createdAt"`
}

type StoredOrganogramFile struct {
	OrganogramFile
	Body []byte
}

type ImportRejection struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type ImportResult struct {
	File     OrganogramFile    `json:"file"`
	Accepted int               `json:"accepted"`
	Rejected []ImportRejection `json:"rejected"`
	Headers  []string          `json:"headers"`
}

type SheetMapping struct {
	Title      string
	ReportsTo  string
	Department string
}

type Store interface {
	ListDepartments(ctx context.Context, organisationID string) ([]Department, error)
	ListPeople(ctx context.Context, organisationID string) ([]Person, error)
	ListRoleAssignments(ctx context.Context, organisationID string) ([]RoleAssignment, error)
	Profile(ctx context.Context, organisationID, personID string) (Profile, error)
	CreateRoleAssignment(ctx context.Context, organisationID, personID, title string, createdAt time.Time) (RoleAssignment, error)
	CreateOrganogram(ctx context.Context, organisationID, name, actorID string, createdAt time.Time) (Organogram, error)
	ListOrganograms(ctx context.Context, organisationID string) ([]Organogram, error)
	PublishOrganogram(ctx context.Context, organisationID, organogramID string) (Organogram, error)
	CreatePosition(ctx context.Context, organisationID, organogramID string, input PositionInput, createdAt time.Time) (Position, error)
	UpdatePosition(ctx context.Context, organisationID, organogramID, positionID string, input PositionInput) (Position, error)
	DeletePosition(ctx context.Context, organisationID, organogramID, positionID string) error
	ListPositions(ctx context.Context, organisationID, organogramID string) ([]Position, error)
	ImportOrganogramFile(ctx context.Context, organisationID, organogramID, actorID, fileName string, body []byte, mapping SheetMapping, createdAt time.Time) (ImportResult, error)
	ListOrganogramFiles(ctx context.Context, organisationID, organogramID string) ([]OrganogramFile, error)
	ReadOrganogramFile(ctx context.Context, organisationID, organogramID, fileID string) (StoredOrganogramFile, error)
}
