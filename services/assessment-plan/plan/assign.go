package plan

import (
	"context"
	"errors"
	"strings"
	"time"
)

func Assign(ctx context.Context, store Store, frameworks FrameworkReader, directory DirectoryReader, publisher Publisher, input AssignInput) ([]PersonResult, error) {
	if !present(input.OrganisationID) || !present(input.ActorPersonID) || !present(input.FrameworkVersionID) || !present(input.RoleID) || len(input.People) == 0 || input.AssignedAt.IsZero() {
		return nil, ErrInvalid
	}
	version, err := frameworks.ReadVersion(ctx, input.Authorization, input.FrameworkVersionID)
	if err != nil {
		return nil, err
	}
	if version.Status != "published" {
		return nil, ErrNotPublished
	}
	roleName, ok := roleName(version.Roles, input.RoleID)
	if !ok {
		return nil, ErrRoleNotOnVersion
	}

	results := make([]PersonResult, 0, len(input.People))
	for _, person := range input.People {
		result, err := assignPerson(ctx, store, directory, publisher, input, roleName, person)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func assignPerson(ctx context.Context, store Store, directory DirectoryReader, publisher Publisher, input AssignInput, roleName string, person AssignPerson) (PersonResult, error) {
	personID := strings.TrimSpace(person.PersonID)
	if !present(personID) {
		return PersonResult{PersonID: person.PersonID, Outcome: OutcomeFailed, Reason: ReasonOutside}, nil
	}
	existing, err := store.FindAssignment(ctx, input.OrganisationID, personID, input.FrameworkVersionID, input.RoleID)
	if err == nil {
		return PersonResult{PersonID: personID, Outcome: OutcomeSkipped, Reason: ReasonSkipped, AssignmentID: existing.ID}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return PersonResult{}, err
	}

	profile, err := directory.ReadPerson(ctx, input.Authorization, personID)
	if errors.Is(err, ErrPersonOutsideOrg) {
		return PersonResult{PersonID: personID, Outcome: OutcomeFailed, Reason: ReasonOutside}, nil
	}
	if err != nil {
		return PersonResult{}, err
	}

	exception := strings.TrimSpace(person.ExceptionReason)
	matches := titleMatches(profile.RoleTitles, roleName)
	if !matches && exception == "" {
		return PersonResult{PersonID: personID, Outcome: OutcomeFailed, Reason: ReasonTitle}, nil
	}
	storedReason := ""
	resultReason := ReasonAssigned
	if !matches {
		storedReason = exception
		resultReason = ReasonException
	}
	created, err := store.CreateAssignment(ctx, AssignmentInput{
		OrganisationID:     input.OrganisationID,
		PersonID:           personID,
		FrameworkVersionID: input.FrameworkVersionID,
		RoleID:             input.RoleID,
		AssignedByPersonID: input.ActorPersonID,
		AssignedAt:         input.AssignedAt.UTC(),
		ExceptionReason:    storedReason,
	})
	if errors.Is(err, ErrInvalid) {
		return PersonResult{PersonID: personID, Outcome: OutcomeFailed, Reason: "exception reason is invalid"}, nil
	}
	if err != nil {
		return PersonResult{}, err
	}
	occurredAt := input.AssignedAt.UTC().Format(time.RFC3339)
	if err := publisher.PublishAssessmentAssigned(ctx, AssessmentAssigned{
		Type:               "assessment.assigned",
		OrganisationID:     input.OrganisationID,
		PersonID:           personID,
		AssignmentID:       created.ID,
		ActorPersonID:      input.ActorPersonID,
		FrameworkVersionID: input.FrameworkVersionID,
		RoleID:             input.RoleID,
		OccurredAt:         occurredAt,
	}); err != nil {
		return PersonResult{}, err
	}
	return PersonResult{PersonID: personID, Outcome: OutcomeAssigned, Reason: resultReason, AssignmentID: created.ID}, nil
}

func roleName(roles []FrameworkRole, roleID string) (string, bool) {
	for _, role := range roles {
		if role.ID == roleID {
			return role.Name, true
		}
	}
	return "", false
}

func titleMatches(titles []string, roleName string) bool {
	for _, title := range titles {
		if strings.TrimSpace(title) == roleName {
			return true
		}
	}
	return false
}
