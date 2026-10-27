package roleprofile

import (
	"encoding/json"
	"time"

	"github.com/bicmas-academy/learn/services/capability-framework/outbox"
)

const publicationSchemaVersion = 1

type publishedEvent struct {
	Type             string `json:"type"`
	OrganisationID   string `json:"organisationId"`
	ProfileID        string `json:"profileId"`
	ProfileVersionID string `json:"profileVersionId"`
	VersionNumber    int    `json:"versionNumber"`
	ActorPersonID    string `json:"actorPersonId"`
	OccurredAt       string `json:"occurredAt"`
}

func roleProfilePublishedPayload(organisationID, profileID, versionID, actorID string, versionNumber int, occurredAt time.Time) ([]byte, error) {
	payload, err := json.Marshal(publishedEvent{
		Type:             outbox.RoleProfilePublishedType,
		OrganisationID:   organisationID,
		ProfileID:        profileID,
		ProfileVersionID: versionID,
		VersionNumber:    versionNumber,
		ActorPersonID:    actorID,
		OccurredAt:       occurredAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	if err := outbox.ValidateRoleProfilePublished(payload); err != nil {
		return nil, ErrInvalid
	}
	return payload, nil
}
