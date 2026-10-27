# Gateway

The gateway owns no domain records. It authenticates callers and exposes `/health`, `/ready`, and `/me`. Domain HTTP still goes directly to each service URL.

HTTP timeouts and graceful shutdown use `services/httpserver` (`StandardTimeouts`). `GET /health` is liveness; `GET /ready` checks the NATS connection. See `docs/http-lifecycle.md`.

## Development authentication

Local demos use a gated development login:

- `APP_ENV` — use `development` (or leave unset). Values `production` / `prod` mark a production process.
- `DEV_LOGIN_ENABLED` — must be the boolean `true` to register `POST /dev/login`. Unset or `false` leaves the route unregistered (HTTP 404). Invalid values refuse startup.
- `DEV_JWT_SECRET` — required when development login is enabled. Minimum 32 characters. Weak placeholders are rejected. No default secret is invented.

Development tokens are HS256 JWTs with issuer `bicmas-learn-dev`. Claims (`personId`, `organisationId`, `roles`, `managerOf`) come only from the fixed demo directory in the gateway auth package. Clients cannot choose arbitrary administrator claims.

`DevHMAC` is the development issuer/verifier only. It must not be used as a production identity provider.

## Production authentication contract

Production must not enable development login. Startup refuses:

1. `APP_ENV=production` (or `prod`) with `DEV_LOGIN_ENABLED=true`
2. `APP_ENV=production` until a real OIDC verifier is wired

A production verifier must validate:

- issuer
- audience
- expiry
- signing keys (JWKS)
- required claims, then map into the internal `Claims` shape

Roles and manager scope must come from the identity provider or a trusted directory lookup — never from client-supplied JSON on a login form.

`auth.NewOIDCVerifier` documents that contract and currently returns `ErrOIDCNotConfigured`. Do not substitute a stub that accepts unsigned or demo tokens.

Person and organisation identifiers for Ada, Eli, and Maya stay fixed until an explicit migration plan replaces them.
