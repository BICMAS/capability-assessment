package outbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const FrameworkPublishedType = "framework.published"

var frameworkPublishedFields = map[string]struct{}{
	"type":               {},
	"organisationId":     {},
	"frameworkVersionId": {},
	"actorPersonId":      {},
	"occurredAt":         {},
}

// ValidateFrameworkPublished checks a payload against contracts/events/framework.published.schema.json.
// The schema rejects additional properties, so envelope metadata stays on the outbox row and broker headers.
func ValidateFrameworkPublished(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var fields map[string]any
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("framework.published payload: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("framework.published payload: trailing data")
	}
	if len(fields) != len(frameworkPublishedFields) {
		return fmt.Errorf("framework.published payload: unexpected fields")
	}
	for key := range fields {
		if _, ok := frameworkPublishedFields[key]; !ok {
			return fmt.Errorf("framework.published payload: unknown field %s", key)
		}
	}
	if fields["type"] != FrameworkPublishedType {
		return fmt.Errorf("framework.published payload: type")
	}
	for _, key := range []string{"organisationId", "frameworkVersionId", "actorPersonId", "occurredAt"} {
		value, ok := fields[key].(string)
		if !ok || value == "" {
			return fmt.Errorf("framework.published payload: %s", key)
		}
	}
	if _, err := time.Parse(time.RFC3339, fields["occurredAt"].(string)); err != nil {
		return fmt.Errorf("framework.published payload: occurredAt: %w", err)
	}
	return nil
}
