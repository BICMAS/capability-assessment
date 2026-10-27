package outbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const RoleProfilePublishedType = "role_profile.published"

var roleProfilePublishedFields = map[string]struct{}{
	"type":             {},
	"organisationId":   {},
	"profileId":        {},
	"profileVersionId": {},
	"versionNumber":    {},
	"actorPersonId":    {},
	"occurredAt":       {},
}

// ValidateRoleProfilePublished checks a payload against contracts/events/role_profile.published.schema.json.
func ValidateRoleProfilePublished(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var fields map[string]any
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("role_profile.published payload: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("role_profile.published payload: trailing data")
	}
	if len(fields) != len(roleProfilePublishedFields) {
		return fmt.Errorf("role_profile.published payload: unexpected fields")
	}
	for key := range fields {
		if _, ok := roleProfilePublishedFields[key]; !ok {
			return fmt.Errorf("role_profile.published payload: unknown field %s", key)
		}
	}
	if fields["type"] != RoleProfilePublishedType {
		return fmt.Errorf("role_profile.published payload: type")
	}
	for _, key := range []string{"organisationId", "profileId", "profileVersionId", "actorPersonId", "occurredAt"} {
		value, ok := fields[key].(string)
		if !ok || value == "" {
			return fmt.Errorf("role_profile.published payload: %s", key)
		}
	}
	number, ok := fields["versionNumber"].(float64)
	if !ok || number < 1 || number != float64(int(number)) {
		return fmt.Errorf("role_profile.published payload: versionNumber")
	}
	if _, err := time.Parse(time.RFC3339, fields["occurredAt"].(string)); err != nil {
		return fmt.Errorf("role_profile.published payload: occurredAt: %w", err)
	}
	return nil
}
