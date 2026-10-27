#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

: "${NATS_URL:?NATS_URL is required}"
: "${INTEGRATION_DATABASE_URL:?INTEGRATION_DATABASE_URL is required}"

mkdir -p artifacts
echo "== integration tests (capability-framework) ==" | tee artifacts/go-integration.log
(
  cd services/capability-framework
    go test -tags=integration -count=1 ./framework ./events ./outbox ./roleprofile
) 2>&1 | tee -a artifacts/go-integration.log
