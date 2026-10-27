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
- `services/assessment-plan` — role assignments, assessment plans, and submitted-session proposals.
- `services/httpserver` — shared HTTP timeouts, request wrapping, and graceful shutdown.
- `contracts/openapi` — OpenAPI specifications for the Go services.
- `contracts/events` — cross-service event definitions.

## Local run

```bash
make up          # Postgres, NATS JetStream, MinIO
make web         # Next.js on :3000
# In separate shells, with .env.example exported:
go run ./services/gateway/cmd/server
go run ./services/audit/cmd/server
go run ./services/notify/cmd/server
go run ./services/org-directory/cmd/server
go run ./services/capability-framework/cmd/server
go run ./services/assessment-gap/cmd/server
go run ./services/development/cmd/server
go run ./services/assessment-plan/cmd/server
```

Each Go service exposes `GET /health` (liveness) and `GET /ready` (readiness). Timeout and shutdown behaviour is documented in `docs/http-lifecycle.md`.

## Quality gates

CI runs on every pull request (`.github/workflows/ci.yml`). Configure the `Go`, `Web`, `Contracts`, `Security`, and `Integration` jobs as required checks so a failing gate blocks merge.

Local equivalents:

```bash
make ci              # Go + web + contracts + security (no Docker services)
make up              # Postgres, NATS, MinIO for integration
export NATS_URL=nats://localhost:4222
export INTEGRATION_DATABASE_URL=postgresql://bicmas:bicmas@localhost:5432/capability_framework?sslmode=disable
make ci-integration
```

Details: `docs/ci.md`. Migration and rollout safety: `docs/release-safety.md`. Publication events: `docs/transactional-outbox.md`.

Toolchain pins: Go version from `go.work` (currently **1.27**), Node from `apps/web/.nvmrc` (**20**). Do not silently downgrade either.
