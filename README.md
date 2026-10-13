# BICMAS Learn

BICMAS Learn is a Capability Operating System that sits above an existing LMS.

**Product rule:** AI coaches. Managers develop. People perform.

The operating loop is Define → Assess → Diagnose → Develop → Practise → Apply → Validate → Reassess.

## Layout

- `apps/web` — Next.js App Router for the admin, employee, and manager portals.
- `services/gateway` — internal authentication and request routing. Owns no domain records.
- `services/audit` — append-only audit records.
- `services/notify` — in-app inboxes.
- `services/org-directory` — organisations, departments, people, manager links, and role assignments.
- `services/capability-framework` — frameworks, published versions, roles, capabilities, and requirements.
- `services/assessment-gap` — assessment attempts, evidence citations, and calculated levels and gaps.
- `services/development` — capability-to-course mappings, pathways, and learning completions.
- `contracts/openapi` — OpenAPI specifications for the Go services.
- `contracts/events` — cross-service event definitions.
