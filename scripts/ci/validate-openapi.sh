#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

if ! command -v npx >/dev/null 2>&1; then
  echo "npx is required to validate OpenAPI specs" >&2
  exit 1
fi

mkdir -p artifacts
fail=0
echo "== OpenAPI lint (Redocly) ==" | tee artifacts/openapi-lint.log
for spec in contracts/openapi/*.yaml; do
  echo "-- ${spec}" | tee -a artifacts/openapi-lint.log
  if ! npx --yes @redocly/cli@1.34.3 lint "$spec" --config redocly.yaml 2>&1 | tee -a artifacts/openapi-lint.log; then
    fail=1
  fi
done

exit "$fail"
