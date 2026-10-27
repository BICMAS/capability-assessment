# HTTP server lifecycle

Shared utilities live in `services/httpserver`. Every Go service process uses them for timeouts, request wrapping, and graceful shutdown.

## Timeouts

| Preset | Services | ReadHeader | Read | Write | Idle | Shutdown |
| --- | --- | --- | --- | --- | --- | --- |
| `StandardTimeouts` | gateway, audit, notify, org-directory, capability-framework, assessment-gap, development | 5s | 15s | 30s | 60s | 15s |
| `LongRequestTimeouts` | assessment-plan | 5s | 30s | 120s | 90s | 30s |

`assessment-plan` still calls the model synchronously when generating a plan. `WriteTimeout` is therefore longer than the 60s model HTTP client timeout. Prefer moving draft generation to an asynchronous job; do not shorten WriteTimeout around that call.

## Shutdown

On `SIGINT` or `SIGTERM` the process:

1. Stops accepting new connections (`http.Server.Shutdown` with the preset deadline).
2. Returns from `httpserver.ListenAndServe` so `run` can finish.
3. Runs deferred closes for SQL pools, NATS publishers/subscriptions, and related resources.

`main` only calls `os.Exit` after `run` returns, so deferred cleanup always executes.

## Health

| Path | Meaning |
| --- | --- |
| `GET /health` | Liveness: process is up. No dependency checks. |
| `GET /ready` | Readiness: critical local dependencies only. |

Readiness checks:

- Domain services with Postgres: current database identity query.
- gateway: NATS connection.
- audit: database and NATS connection.
- Upstream HTTP services (capability-framework, org-directory, assessment-gap from assessment-plan) are **not** probed from readiness, to avoid cascading failures.
- S3 is not probed from capability-framework readiness (import-only dependency).

## Request middleware

`httpserver.Wrap` adds:

- `X-Request-ID` (honours inbound header or generates one)
- Structured access log: `service`, `request_id`, `method`, `route`, `status`, `duration_ms`
- Body size limit (`DefaultMaxBodyBytes` 1 MiB; capability-framework uses `UploadMaxBodyBytes` 8 MiB)
- Content-Type checks for POST/PUT/PATCH with a body: `application/json`, `multipart/form-data`, or `application/x-www-form-urlencoded`

Logs do not include Authorization headers, cookies, bearer tokens, or request bodies.

## Local run

```bash
make up
# export vars from .env.example (including APP_ENV / DEV_LOGIN_ENABLED for gateway)
go run ./services/gateway/cmd/server
go run ./services/audit/cmd/server
# …same pattern for each service under services/*/cmd/server

curl -s localhost:8080/health
curl -s localhost:8080/ready
```
