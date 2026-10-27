Cross-service events live here as JSON Schema, beside the OpenAPI specifications.

- `person.signed_in` — a person signed in
- `audit.recorded` — an append-only audit row was stored
- `assignment.changed` — a role assignment changed
- `framework.published` — a framework version was published
- `level.overridden` — a manager replaced a calculated level
- `learning.completed` — a pathway learning item was completed
- `assessment.assigned` — an admin assigned a person to a published framework role
- `assessment.plan_approved` — an admin approved an assessment plan
- `assessment.completed` — an authorised user confirmed a submitted assessment session
- `role_profile.published` — an approved role skills profile version was published

Example payloads live in `examples/`. Compatibility baselines live in `baseline/` (required fields and properties must not be removed). Validate with `bash scripts/ci/validate-events.sh`.
