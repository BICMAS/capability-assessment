package plan

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ProfileMapPrompt = "role-profile-snapshot-1"
	ProfileMapModel  = "role-profile-map"

	GapMissing = "missing"
	GapBelow   = "below"
	GapMet     = "met"

	// EmploymentDecision is recorded on insights and recommendations.
	// A gap or pathway is not an employment decision.
	EmploymentDecision = "not_an_employment_decision"
)

var (
	ErrProfileUnpublished = errors.New("role profile version is not published")
	ErrRolePeriod         = errors.New("role assignment does not cover the assessment period")
	ErrForbidden          = errors.New("forbidden")
)

type ProfileSkillSnapshot struct {
	AssignmentID        string
	SkillID             string
	Name                string
	Definition          string
	RequiredProficiency int
	Priority            string
	Rationale           string
}

type ProfileSource struct {
	ProfileID      string
	VersionID      string
	VersionNumber  int
	OrganisationID string
	RoleID         string
	RoleName       string
	Status         string
	Skills         []ProfileSkillSnapshot
}

type ProfileBaseline struct {
	ID               string
	OrganisationID   string
	PlanID           string
	AssignmentID     string
	PersonID         string
	ProfileID        string
	ProfileVersionID string
	VersionNumber    int
	RoleID           string
	RoleName         string
	Skills           []ProfileSkillSnapshot
}

type RoleTenure struct {
	PersonID       string
	Title          string
	DepartmentName string
	StartedAt      time.Time
	EndedAt        time.Time
}

type ProfilePlanInput struct {
	OrganisationID string
	PersonID       string
	ActorPersonID  string
	AssignedAt     time.Time
	Tenures        []RoleTenure
	Profile        ProfileSource
}

type ProfilePlanStore interface {
	Store
	SaveProfileBaseline(ctx context.Context, baseline ProfileBaseline) (ProfileBaseline, error)
	ProfileBaseline(ctx context.Context, organisationID, planID string) (ProfileBaseline, error)
	LatestPlan(ctx context.Context, organisationID, assignmentID string) (Plan, error)
	ListProfileBaselines(ctx context.Context, organisationID string) ([]ProfileBaseline, error)
}

// MapProfileRequirements copies required skills from one published role-profile
// version into assessment-plan requirements. Proficiency stays on the 0–4 scale.
// Preferred and optional skills stay on the snapshot and are not requirements.
func MapProfileRequirements(profile ProfileSource) ([]SkillInput, error) {
	if profile.Status != "profile_published" {
		return nil, ErrProfileUnpublished
	}
	if !present(profile.ProfileID) || !present(profile.VersionID) || !present(profile.RoleID) || profile.VersionNumber < 1 {
		return nil, ErrInvalid
	}
	skills := make([]SkillInput, 0)
	seen := map[string]struct{}{}
	for _, skill := range profile.Skills {
		if skill.Priority != "required" {
			continue
		}
		if !present(skill.AssignmentID) || !present(skill.SkillID) || !present(skill.Name) {
			return nil, ErrInvalid
		}
		if skill.RequiredProficiency < 0 || skill.RequiredProficiency > 4 {
			return nil, ErrInvalid
		}
		if _, ok := seen[skill.SkillID]; ok {
			return nil, ErrInvalid
		}
		seen[skill.SkillID] = struct{}{}
		method, _ := chooseMethod(skill.Name)
		definition := strings.TrimSpace(skill.Definition)
		if definition == "" {
			definition = strings.TrimSpace(skill.Name)
		}
		rationale := strings.TrimSpace(skill.Rationale)
		if rationale == "" {
			rationale = "Required by the approved role profile."
		}
		if utf8.RuneCountInString(rationale) > 280 {
			rationale = string([]rune(rationale)[:280])
		}
		skills = append(skills, SkillInput{
			RequirementID:     skill.AssignmentID,
			CapabilityID:      skill.SkillID,
			RequiredLevel:     skill.RequiredProficiency,
			RecommendedMethod: method,
			Rationale:         rationale,
			Instructions:      definition,
			AssessmentItems:   []ItemInput{{Prompt: "Demonstrate " + strings.TrimSpace(skill.Name)}},
			Rubric:            []RubricInput{{Level: skill.RequiredProficiency, Descriptor: definition}},
		})
	}
	if len(skills) == 0 {
		return nil, ErrInvalid
	}
	return skills, nil
}

// RoleCovers reports whether a directory role assignment is valid at the assessment time.
// An open tenure has a zero EndedAt. A later role change sets EndedAt on the previous tenure.
func RoleCovers(tenures []RoleTenure, personID, roleName string, at time.Time) bool {
	roleName = strings.TrimSpace(roleName)
	if personID == "" || roleName == "" || at.IsZero() {
		return false
	}
	for _, tenure := range tenures {
		if tenure.PersonID != personID || strings.TrimSpace(tenure.Title) != roleName {
			continue
		}
		if tenure.StartedAt.After(at) {
			continue
		}
		if !tenure.EndedAt.IsZero() && !tenure.EndedAt.After(at) {
			continue
		}
		return true
	}
	return false
}

func CreateFromProfile(ctx context.Context, store ProfilePlanStore, input ProfilePlanInput) (Plan, ProfileBaseline, error) {
	if !present(input.OrganisationID) || !present(input.PersonID) || !present(input.ActorPersonID) || input.AssignedAt.IsZero() {
		return Plan{}, ProfileBaseline{}, ErrInvalid
	}
	if input.Profile.OrganisationID != "" && input.Profile.OrganisationID != input.OrganisationID {
		return Plan{}, ProfileBaseline{}, ErrNotFound
	}
	if !RoleCovers(input.Tenures, input.PersonID, input.Profile.RoleName, input.AssignedAt) {
		return Plan{}, ProfileBaseline{}, ErrRolePeriod
	}
	skills, err := MapProfileRequirements(input.Profile)
	if err != nil {
		return Plan{}, ProfileBaseline{}, err
	}
	assignment, err := store.CreateAssignment(ctx, AssignmentInput{
		OrganisationID:     input.OrganisationID,
		PersonID:           input.PersonID,
		FrameworkVersionID: input.Profile.VersionID,
		RoleID:             input.Profile.RoleID,
		AssignedByPersonID: input.ActorPersonID,
		AssignedAt:         input.AssignedAt,
	})
	if err != nil {
		return Plan{}, ProfileBaseline{}, err
	}
	existing, err := store.LatestPlan(ctx, input.OrganisationID, assignment.ID)
	if err == nil {
		baseline, baselineErr := store.ProfileBaseline(ctx, input.OrganisationID, existing.ID)
		return existing, baseline, baselineErr
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Plan{}, ProfileBaseline{}, err
	}
	created, err := store.CreateGeneratedPlan(ctx, PlanInput{
		OrganisationID: input.OrganisationID,
		AssignmentID:   assignment.ID,
		PromptVersion:  ProfileMapPrompt,
		ModelName:      ProfileMapModel,
		Skills:         skills,
	})
	if err != nil {
		return Plan{}, ProfileBaseline{}, err
	}
	baseline, err := store.SaveProfileBaseline(ctx, ProfileBaseline{
		OrganisationID:   input.OrganisationID,
		PlanID:           created.ID,
		AssignmentID:     assignment.ID,
		PersonID:         input.PersonID,
		ProfileID:        input.Profile.ProfileID,
		ProfileVersionID: input.Profile.VersionID,
		VersionNumber:    input.Profile.VersionNumber,
		RoleID:           input.Profile.RoleID,
		RoleName:         input.Profile.RoleName,
		Skills:           copySkillSnapshots(input.Profile.Skills),
	})
	if err != nil {
		return Plan{}, ProfileBaseline{}, err
	}
	return created, baseline, nil
}

func copySkillSnapshots(skills []ProfileSkillSnapshot) []ProfileSkillSnapshot {
	copied := make([]ProfileSkillSnapshot, len(skills))
	copy(copied, skills)
	return copied
}

// ClassifyGap distinguishes a missing assessment from a calculated gap.
// current and gap must be the strings already produced by assessment-gap.
// This function does not calculate a level.
func ClassifyGap(skillID string, requiredProficiency int, found bool, current, gap string) (ClassifiedGap, error) {
	if !present(skillID) || requiredProficiency < 0 || requiredProficiency > 4 {
		return ClassifiedGap{}, ErrInvalid
	}
	if !found {
		return ClassifiedGap{SkillID: skillID, RequiredProficiency: requiredProficiency, State: GapMissing}, nil
	}
	if strings.TrimSpace(current) == "" || strings.TrimSpace(gap) == "" {
		return ClassifiedGap{}, ErrInvalid
	}
	gapTenths, err := parseTenths(gap)
	if err != nil {
		return ClassifiedGap{}, err
	}
	state := GapMet
	if gapTenths > 0 {
		state = GapBelow
	}
	return ClassifiedGap{
		SkillID:             skillID,
		RequiredProficiency: requiredProficiency,
		State:               state,
		CurrentLevel:        current,
		Gap:                 gap,
	}, nil
}

type ClassifiedGap struct {
	SkillID             string `json:"skillId"`
	RequiredProficiency int    `json:"requiredProficiency"`
	State               string `json:"state"`
	CurrentLevel        string `json:"currentLevel,omitempty"`
	Gap                 string `json:"gap,omitempty"`
}

func parseTenths(value string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 || len(parts[1]) != 1 {
		return 0, ErrInvalid
	}
	sign := 1
	whole := parts[0]
	if strings.HasPrefix(whole, "-") {
		sign = -1
		whole = strings.TrimPrefix(whole, "-")
	}
	if whole == "" || parts[1] < "0" || parts[1] > "9" {
		return 0, ErrInvalid
	}
	for _, digit := range whole {
		if digit < '0' || digit > '9' {
			return 0, ErrInvalid
		}
	}
	number := 0
	for _, digit := range whole {
		number = number*10 + int(digit-'0')
	}
	return sign * (number*10 + int(parts[1][0]-'0')), nil
}

type InsightActor struct {
	PersonID  string
	Roles     []string
	ManagerOf []string
}

type InsightRequest struct {
	PersonID         string
	Department       string
	ProfileVersionID string
}

type InsightSkill struct {
	SkillID             string
	RequiredProficiency int
	Found               bool
	CurrentLevel        string
	Gap                 string
}

type InsightSubject struct {
	PersonID         string
	Department       string
	ProfileVersionID string
	RoleName         string
	AssessedAt       time.Time
	Tenures          []RoleTenure
	Skills           []InsightSkill
}

type Insight struct {
	ProfileVersionID   string          `json:"profileVersionId"`
	Population         []string        `json:"population"`
	People             []InsightPerson `json:"people"`
	EmploymentDecision string          `json:"employmentDecision"`
}

type InsightPerson struct {
	PersonID string          `json:"personId"`
	Skills   []ClassifiedGap `json:"skills"`
}

func BuildInsight(actor InsightActor, request InsightRequest, subjects []InsightSubject) (Insight, error) {
	if !present(actor.PersonID) || !present(request.ProfileVersionID) {
		return Insight{}, ErrInvalid
	}
	if err := authoriseInsight(actor, request); err != nil {
		return Insight{}, err
	}
	insight := Insight{ProfileVersionID: request.ProfileVersionID, EmploymentDecision: EmploymentDecision}
	for _, subject := range subjects {
		if subject.ProfileVersionID != request.ProfileVersionID {
			continue
		}
		if request.PersonID != "" && subject.PersonID != request.PersonID {
			continue
		}
		if request.Department != "" && subject.Department != request.Department {
			continue
		}
		if !insightVisible(actor, subject.PersonID) {
			continue
		}
		if !RoleCovers(subject.Tenures, subject.PersonID, subject.RoleName, subject.AssessedAt) {
			continue
		}
		person := InsightPerson{PersonID: subject.PersonID}
		for _, skill := range subject.Skills {
			classified, err := ClassifyGap(skill.SkillID, skill.RequiredProficiency, skill.Found, skill.CurrentLevel, skill.Gap)
			if err != nil {
				return Insight{}, err
			}
			person.Skills = append(person.Skills, classified)
		}
		insight.People = append(insight.People, person)
		insight.Population = append(insight.Population, subject.PersonID)
	}
	return insight, nil
}

func authoriseInsight(actor InsightActor, request InsightRequest) error {
	if hasRole(actor, "admin") {
		return nil
	}
	if request.Department != "" {
		return ErrForbidden
	}
	if hasRole(actor, "manager") {
		if request.PersonID == "" || request.PersonID == actor.PersonID || managesPerson(actor, request.PersonID) {
			return nil
		}
		return ErrForbidden
	}
	if request.PersonID != "" && request.PersonID != actor.PersonID {
		return ErrForbidden
	}
	return nil
}

func insightVisible(actor InsightActor, personID string) bool {
	if hasRole(actor, "admin") {
		return true
	}
	if personID == actor.PersonID {
		return true
	}
	return hasRole(actor, "manager") && managesPerson(actor, personID)
}

func hasRole(actor InsightActor, role string) bool {
	for _, candidate := range actor.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func managesPerson(actor InsightActor, personID string) bool {
	for _, id := range actor.ManagerOf {
		if id == personID {
			return true
		}
	}
	return false
}
