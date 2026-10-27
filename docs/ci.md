# CI quality gates

GitHub Actions workflow: `.github/workflows/ci.yml`.

Required jobs (configure these as required status checks on the default branch):

| Check name | What it gates |
| --- | --- |
| `Go` | `gofmt`, `go vet`, `staticcheck`, unit tests |
| `Web` | `npm ci`, typecheck, lint, Vitest, production build |
| `Contracts` | OpenAPI lint, event schema validation, generated-code drift |
| `Security` | Gitleaks, `govulncheck`, production `npm audit` |
| `Integration` | Postgres + NATS persistence and event publish tests |

Failures fail the workflow. Scripts never use `|| true` to hide errors.

## Local equivalents

From the repository root (Go **1.27** as declared in `go.work`; Node **20** as in `apps/web/.nvmrc`):

```bash
# Go format / vet / static analysis / unit tests
gofmt -l $(find . -name '*.go' -not -path './.gomodcache/*')
bash scripts/ci/go-vet.sh
go install honnef.co/go/tools/cmd/staticcheck@latest
bash scripts/ci/go-staticcheck.sh
bash scripts/ci/go-test.sh

# Or the Makefile wrappers:
make ci-go

# Frontend
cd apps/web
npm ci
npm run typecheck
npm run lint
npm run test
npm run build
# Or:
make ci-web

# Contracts + codegen drift
bash scripts/ci/validate-openapi.sh
bash scripts/ci/validate-events.sh
bash scripts/ci/check-generated.sh
make ci-contracts

# Security
gitleaks detect --source . --no-git
go install golang.org/x/vuln/cmd/govulncheck@latest
bash scripts/ci/govulncheck.sh
(cd apps/web && npm audit --omit=dev --audit-level=high)
make ci-security

# Integration (requires ephemeral Postgres + NATS; make up is enough locally)
export NATS_URL=nats://localhost:4222
export INTEGRATION_DATABASE_URL=postgresql://bicmas:bicmas@localhost:5432/capability_framework?sslmode=disable
bash scripts/ci/go-integration.sh
make ci-integration
```

`make ci` runs the non-integration gates that do not need Docker services. Publication events use a transactional outbox; see `docs/transactional-outbox.md`.

Configure GitHub branch protection so pull requests require the workflow job names **Go**, **Web**, **Contracts**, **Security**, and **Integration**. Without that setting, CI can fail while merges remain possible.

## Fake AI in ordinary tests

Assessment-plan unit tests never call a paid model provider. They inject deterministic fakes:

- `scriptDrafter` in `services/assessment-plan/plan/generate_test.go` returns fixed JSON.
- `HTTPDrafter` tests in `drafter_test.go` point at `httptest` servers.

Do not set `ASSESSMENT_MODEL_API_KEY` for unit or CI unit jobs.

## Generated code

- **sqlc** packages under `services/*/db` must regenerate identically with **sqlc v1.26.0** (`scripts/ci/check-generated.sh`).
- **oapi-codegen** is smoke-tested against `contracts/openapi/*.yaml`; committed `*.gen.go` files must still build. Refreshing those snapshots to a newer oapi-codegen is a dedicated upgrade change (newer generators emit typed enums and may pull `oapi-codegen/runtime`).

## Artifacts

On failure or completion, CI uploads:

- `go-test-log`
- `go-integration-log`
- `web-build` (Next.js build traces when present)
