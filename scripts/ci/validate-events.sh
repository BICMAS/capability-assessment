#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

mkdir -p artifacts
{
  echo "== event schemas + examples + compatibility =="
  node scripts/ci/validate-events.mjs
} 2>&1 | tee artifacts/events-validate.log
