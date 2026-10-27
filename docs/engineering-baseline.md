# BICMAS Learn — Engineering Baseline

Audit date: 2026-10-19  
Scope: repository inspection only. Application code was not modified. Dependencies were not upgraded. No new services were introduced.

**Verification note:** Go and TypeScript test commands listed below were **not** confirmed green in this audit pass. A local attempt hung / failed on Go build-cache permissions and did not complete. Treat pass/fail as unknown until the commands are run in a normal developer environment.

---

## 1. Architecture map

```text
Browser
  └─ apps/web (Next.js App Router :3000)
       ├─ POST /session → gateway POST /dev/login + GET /me  (sets httpOnly session cookie)
       ├─ middleware role gate (admin | employee | manager)
       └─ server-side fetch → domain services (direct URLs, not via gateway proxy)

Infra (docker compose): Postgres 16 · NATS JetStream · MinIO
Domain Go services (local `go run`, chi):
  gateway:8080 · audit:8081 · notify:8082 · org-directory:8083
  capability-framework:8084 · assessment-gap:8085 · development:8086
  assessment-plan:8087
Shared module: services/token (JWT verify middleware)

Events: NATS stream LEARN_EVENTS
  Producers → audit durable consumers (append-only records)
```

The gateway issues and verifies development JWTs and publishes `person.signed_in`. It does **not** reverse-proxy domain HTTP today; the Next app calls each service URL from env. The gateway README still describes future “route organisation-scoped requests”; that routing layer is not present in code.

### Service responsibilities

| Service | Port | Database | Owns | Does not own |
| --- | --- | --- | --- | --- |
| `gateway` | 8080 | none | Dev login, `/me`, JWT issue/verify, `person.signed_in` | Domain records, HTTP proxy |
| `audit` | 8081 | `audit` | Append-only audit rows from JetStream | Producing domain events |
| `notify` | 8082 | `notify` | Inbox list API + `inbox_items` table | No writer/consumer found; UI does not call it |
| `org-directory` | 8083 | `org_directory` | Orgs, departments, people, manager links, directory role titles | Capability roles |
| `capability-framework` | 8084 | `capability_framework` | Framework versions, roles, capabilities, requirements, indicators, CSV import (S3), publish | Person levels |
| `assessment-gap` | 8085 | `assessment_gap` | Attempts, calculated levels/gaps, manager overrides | Plan sessions |
| `development` | 8086 | `development` | Course mappings, pathways, LMS completion webhook | Assessment scoring |
| `assessment-plan` | 8087 | `assessment_plan` | Role assignments, plans, sessions, proposals, confirm → gap attempts | Directory people |
| `token` | — | — | Shared HS256 verify + claims middleware | HTTP server |

### Frontend surface

| Area | Route prefix | Primary actor |
| --- | --- | --- |
| Entrance | `/` | Dev person choice (Ada / Eli / Maya) |
| Admin studio | `/admin` | Frameworks → Assignments → Plans; Organisation; Record |
| Employee | `/employee` | Capability instruments + assessment session answers |
| Manager | `/manager`, `/manager/people/[personId]` | Team bands + manager assessment + proposal confirm |

---

## 2. Authentication flow

### Runtime path (as implemented)

1. Entrance form posts to Next `POST /session` with `person` ∈ `{ada, eli, maya}`.
2. Next calls gateway `POST /dev/login` with JSON `{ "person": "…" }`.
3. Gateway looks up a hard-coded development directory, issues HS256 JWT (`issuer: bicmas-learn-dev`, TTL 12h), publishes `person.signed_in`, returns `{ token }`.
4. Next stores the token in httpOnly cookie `session` (12h, `sameSite=lax`, `secure` in production).
5. Middleware calls gateway `GET /me` with `Authorization: Bearer <cookie>` and enforces path role (`admin` / `employee` / `manager`).
6. Server components/actions forward the same bearer token to domain services. Domain services verify with `services/token` against `DEV_JWT_SECRET` (same shared secret).

### Development-only authentication paths

| Path | Location | Notes |
| --- | --- | --- |
| `POST /dev/login` | `services/gateway` | Hard-coded people; not OIDC |
| Entrance person buttons | `apps/web/app/page.tsx` | Dev personas only |
| `POST /session` | `apps/web/app/session/route.ts` | Thin wrapper over `/dev/login` |
| Shared HMAC secret | `DEV_JWT_SECRET` | Required by every Go service; labelled development in `.env.example` |
| Hard-coded claims | `services/gateway/auth/directory.go` | Person IDs, roles, `managerOf` |

Code comments state production intent: an OIDC adapter in front of `Verifier`, mapping into the same claims shape. That adapter is **not** implemented.

### Seeded development people (stable IDs)

| Key | Role claims | Notes |
| --- | --- | --- |
| `ada` | `admin` | Framework admin |
| `eli` | `employee` | Managed by Maya |
| `maya` | `manager` | `managerOf` includes Eli |

---

## 3. Data ownership, migrations, contracts, events

### Databases

Created by `deploy/postgres/init.sql` on first compose up: `audit`, `notify`, `org_directory`, `capability_framework`, `assessment_gap`, `development`, `assessment_plan`.

Each domain service migrates its own DB with goose embed at process start. `DATABASE_URL` overrides the service-specific `DATABASE_URL_*` when set (do not export one URL for all services).

### OpenAPI

| Spec | Service | Frontend client |
| --- | --- | --- |
| *(none)* | gateway | Hand-written `apps/web/lib/gateway.ts` |
| `audit.yaml` | audit | Hand-written `apps/web/lib/audit.ts` |
| `notify.yaml` | notify | `apps/web/lib/generated/notify/inbox.ts` (unused by UI) |
| `org-directory.yaml` | org-directory | Hand-written `apps/web/lib/org-directory.ts` |
| `capability-framework.yaml` | capability-framework | Hand-written `apps/web/lib/capability-framework.ts` |
| `assessment-gap.yaml` | assessment-gap | `apps/web/lib/generated/assessment-gap/levels.ts` |
| `development.yaml` | development | `apps/web/lib/generated/development/pathways.ts` |
| `assessment-plan.yaml` | assessment-plan | Hand-written under `apps/web/lib/generated/assessment-plan/plans.ts` |

Go model packages under `services/*/openapi/` exist (some `*.gen.go`). No checked-in oapi-codegen config or regenerate Makefile target was found; regeneration process is **uncertain**.

### Events (JSON Schema in `contracts/events/`)

| Event | Producer (code) | Consumer (code) |
| --- | --- | --- |
| `person.signed_in` | gateway | audit |
| `assignment.changed` | org-directory | audit |
| `framework.published` | capability-framework | audit |
| `level.overridden` | assessment-gap | audit |
| `learning.completed` | development | audit |
| `assessment.assigned` | assessment-plan | audit |
| `assessment.plan_approved` | assessment-plan | audit |
| `audit.recorded` | **No producer found** | **No consumer found** |

Stream name: `LEARN_EVENTS`. Every publisher Open() also CreateOrUpdateStream with the same subject list (duplicated stream bootstrap).

### Business rules that must be preserved

Do not change these without an explicit product decision:

- Capability framework: published versions immutable; edit → next draft; CSV import proposal gates; publish freezes and emits `framework.published`.
- Assessment method selection (case-insensitive contains on capability name): `knowledge` → `knowledge_test`; else `negotiation` or `communication` → `ai_avatar_roleplay`; else `case_study`. Rationale strings stay in Go.
- Plan statuses: `draft | generated | under_review | changes_requested | approved`. Approved plans immutable. Model draft requires configured provider + valid JSON or no save.
- Session lifecycle: `assigned → started → submitted`; answers max 8000 chars. Scoring → band levels; percentage = round(mean(levels)/4*100). Method → evidence source mapping: knowledge_test → knowledge, case_study → situational_judgement, ai_avatar_roleplay → ai_interview.
- Confirm writes assessment-gap attempts (admin or that person’s manager); second confirm conflicts; employee cannot post `ai_interview`.
- Explain / level math: mean of available scoring sources ÷ 25; self-assessment stored but excluded; status remains provisional; simulation has no writer (`ErrNoWriter`).
- UI: no verified badge; provisional stays provisional when gap says so.

---

## 4. Tests and commands

### Existing Go tests (present in repo)

Rough inventory (~38 `*_test.go` files under `services/`, plus root `health_test.go`):

- gateway: auth, health  
- audit: health, records HTTP, JetStream apply packages  
- notify: health, inbox HTTP  
- org-directory: health, directory HTTP, main  
- capability-framework: health, publish, immutable, import, main  
- assessment-gap: health, levels HTTP, override, regression (incl. 2.8 / 1.2 style expectations), main  
- assessment-plan: health, assign, generate, review, session, plan, drafter, main  
- development: health, pathway, main  
- root module: aggregated `/health` smoke across health routers  

Frontend checks (from `apps/web`):

```bash
cd apps/web
npm run typecheck   # tsc --noEmit
npm run lint        # eslint
npm run test        # vitest unit + component tests
npm run check       # typecheck + lint + test
```

See also `docs/frontend-transactions.md` for mutation boundaries (capability + requirement is atomic in Go).

### Reproducible commands

```bash
# Infrastructure
make up          # postgres, nats, minio + bucket script via .env.example
make down

# Go (from repo root)
make tidy
make test        # go test ./... then each services/*

# Per-service (example)
cd services/assessment-gap && go test ./...
cd services/assessment-plan && go test ./...

# Web
make web         # npm --prefix apps/web run dev
cd apps/web && npm run typecheck   # tsc --noEmit
cd apps/web && npm run build
```

**This audit did not obtain a successful test run.** Re-run `make test` and `npm run typecheck` before treating the suite as green.

---

## 5. Frontend scripts gap

`apps/web/package.json` scripts:

| Script | Present |
| --- | --- |
| `dev` / `build` / `start` | yes |
| `typecheck` | yes |
| `lint` | yes (`eslint`) |
| `test` / `test:watch` | yes (`vitest`) |
| `check` | yes (typecheck + lint + test) |
| `format` / `prettier` | no |
| `test:e2e` | no |

---

## 6. OpenAPI / handler / frontend inconsistencies

Confirmed from route tables and clients:

| Item | Class | Detail |
| --- | --- | --- |
| No gateway OpenAPI | Contract gap | `/dev/login`, `/me`, `/health` undocumented in `contracts/openapi/` |
| `/health` omitted from several specs | Contract gap | Handlers expose `/health` on audit, notify, org-directory, capability-framework; YAML often omits it (assessment-* and development include it) |
| `audit.recorded` schema without runtime | Contract drift | Schema + events README list it; no Go publish/consume found |
| assessment-gap service README stale | Doc defect | README says AI interview has no writers; code allows admin/manager `ai_interview` attempts; only simulation is `columnOnly` |
| Notify client unused | Product / wiring gap | `listInbox` exists; no UI import; no inbox writer path found in notify service |
| Frontend missing `POST /plans/{planId}/session` | Client coverage gap | OpenAPI + Go handler exist; web client has no wrapper. Primary path creates a session inside approve; admin re-create via API is UI-invisible |
| `patchFrameworkRole` (PATCH version) | Client coverage gap | OpenAPI + Go router exist; no frontend caller found |
| Handwritten clients under `lib/generated/` | Naming drift | OpenAPI README says assessment-plan client is handwritten; path still says `generated/` |
| Regenerating clients | Uncertainty | No codegen script found in Makefile / package scripts |

Path lists for assessment-plan, assessment-gap, org-directory core CRUD used by the UI largely align with Go routers. Full schema field parity was not exhaustively diffed; treat deep schema drift as **uncertain** pending a contract test.

---

## 7. Defect / risk register

Priority: **P0** block local/prod correctness or security today · **P1** high friction or likely breakage · **P2** consistency / hygiene · **P3** backlog / product completeness.

### Confirmed code / config defects

| ID | Pri | Finding | Evidence |
| --- | --- | --- | --- |
| D1 | P1 | `AUDIT_URL` used by web (`apps/web/lib/audit.ts`) but absent from `.env.example` | Defaults to `http://localhost:8081`; easy misconfig in non-default layouts |
| D2 | P1 | `ASSESSMENT_GAP_URL` required for session confirmation HTTP calls but not validated at assessment-plan startup | `main.go` requires framework + directory URLs; gap URL is `os.Getenv` only → empty base until confirm |
| D3 | P2 | assessment-gap `README.md` contradicts writable sources | Code: `writable` includes `ai_interview`; README claims no writers |
| D4 | P2 | Gateway README claims request routing; implementation is auth-only | `services/gateway/README.md` vs `server/router.go` |
| D5 | P2 | `audit.recorded` contracted but unused | `contracts/events/` only |

### Potential risks (not proven production incidents)

| ID | Pri | Finding | Notes |
| --- | --- | --- | --- |
| R1 | P0 | Development login and shared HMAC secret are the only auth | Acceptable for local demo; unsafe if exposed beyond trusted network without OIDC cutover |
| R2 | P0 | Browser talks directly to every service origin | CORS, network exposure, and secret surface differ from a true BFF/gateway; Next server-side fetch mitigates some browser exposure for RSC/actions, but env URLs must stay private |
| R3 | P1 | Plan generation depends on unset `ASSESSMENT_MODEL_API_KEY` | Documented; generate fails closed (no save). End-to-end assign→session→confirm cannot complete without a key |
| R4 | P1 | No CI workflows in repo | `.github/` absent; regressions rely on local `make test` |
| R5 | P1 | ~~No frontend lint/test~~ | Addressed: `apps/web` has `lint`, `test`, and `check` scripts |
| R6 | P2 | Duplicate JetStream stream CreateOrUpdate from every publisher | Works if configs stay identical; drift risk across copies |
| R7 | P2 | Cookie host mismatch (localhost vs 127.0.0.1) | Operational footgun for local sign-in |
| R8 | P3 | Simulation source has no writer | By design today (`ErrNoWriter`) |

### Missing features (not defects unless promised)

| ID | Pri | Finding |
| --- | --- | --- |
| M1 | P2 | Notify inbox not populated / not shown in UI |
| M2 | P2 | OIDC / production auth adapter |
| M3 | P3 | Gateway as reverse proxy / organisation router |
| M4 | P3 | Frontend wrappers for every OpenAPI operation |
| M5 | P3 | Verified level status (code has `StatusVerified` constant; Explain always returns provisional) |

---

## 8. External infrastructure and environment

### Compose services

- Postgres 16 (`POSTGRES_*`)
- NATS with JetStream (`NATS_URL`, monitor port)
- MinIO (`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`) — used by capability-framework CSV import

### Environment variables (names only; do not commit real secrets)

| Variable | Used by |
| --- | --- |
| `DEV_JWT_SECRET` | All Go services + token verify |
| `NATS_URL` | gateway, audit, and event publishers |
| `DATABASE_URL` or `DATABASE_URL_*` | Each domain service |
| `PORT` / `*_PORT` | Listen addresses |
| `GATEWAY_URL` | Next → gateway |
| `ORG_DIRECTORY_URL`, `CAPABILITY_FRAMEWORK_URL`, `ASSESSMENT_GAP_URL`, `ASSESSMENT_PLAN_URL`, `DEVELOPMENT_URL` | Next and/or assessment-plan upstreams |
| `AUDIT_URL` | Next audit client (not in `.env.example`) |
| `S3_*` | capability-framework |
| `ASSESSMENT_MODEL_URL`, `ASSESSMENT_MODEL_NAME`, `ASSESSMENT_MODEL_API_KEY` | assessment-plan drafter (key not in example; set locally) |
| `LEARNING_WEBHOOK_SECRET`, `LMS_BASE_URL` | development webhook |

`.env.example` documents local placeholders only. Do not copy production secrets into the repository.

---

## 9. Recommended implementation order

1. **Stabilise the improvement baseline** — Run `make test` and `apps/web` `typecheck`/`build` in a normal environment; record results. Add `AUDIT_URL` to `.env.example`. Validate `ASSESSMENT_GAP_URL` at assessment-plan boot (P1 D1/D2).
2. **Lock contracts** — Add gateway OpenAPI (or explicitly mark gateway out of contract suite). Align `/health` in YAML. Fix or remove `audit.recorded`. Refresh assessment-gap README. Add a contract smoke checklist or codegen target (P2).
3. **Frontend quality gate** — Add lint + format scripts; decide on Vitest and/or Playwright for session and role gates (P1 R5).
4. **Auth production path** — Design OIDC → claims adapter without changing domain authZ rules; keep `/dev/login` behind an explicit non-production flag (P0 R1).
5. **Complete notify or defer** — Either wire producers + UI, or document notify as scaffold-only (M1).
6. **Gateway routing (optional)** — Only if product requires a single public origin; do not invent new domain services (M3).

---

## 10. Explicit uncertainties

- Whether generated OpenAPI Go/TS artefacts are still regenerated from YAML, and with which tool version.
- Full request/response schema parity between every OpenAPI property and Go JSON tags / frontend type guards (spot-checked routes only).
- Whether notify was intended to consume LEARN_EVENTS (no consumer code found).
- Production deployment topology (compose covers infra only; service Dockerfiles exist but are outside this audit’s runtime verification).
- Exact CI expectations for the organisation (none in-repo).

---

## 11. Out of scope for this baseline

- Application code changes  
- Dependency version changes  
- New microservices  
- Committing secrets or API keys  
- Claiming green tests without a successful local run  
