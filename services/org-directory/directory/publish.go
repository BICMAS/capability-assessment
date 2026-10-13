package directory

import (
	"context"
	"time"
)

func PublishAssignment(ctx context.Context, publisher Publisher, assignment RoleAssignment) error {
	return publisher.PublishAssignmentChanged(ctx, AssignmentChanged{
		Type:             "assignment.changed",
		OrganisationID:   assignment.OrganisationID,
		PersonID:         assignment.PersonID,
		RoleAssignmentID: assignment.ID,
		OccurredAt:       assignment.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func PublishStoredAssignments(ctx context.Context, store Store, publisher Publisher, organisationID string) error {
	assignments, err := store.ListRoleAssignments(ctx, organisationID)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		if err := PublishAssignment(ctx, publisher, assignment); err != nil {
			return err
		}
	}
	return nil
}
