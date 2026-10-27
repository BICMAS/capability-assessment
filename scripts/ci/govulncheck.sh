#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck is required" >&2
  exit 1
fi

mkdir -p artifacts
echo "== govulncheck (workspace packages) ==" | tee artifacts/govulncheck.log
pkgs="$(go list ./... | grep -v '/node_modules/' || true)"
if [ -n "$pkgs" ]; then
  # shellcheck disable=SC2086
  govulncheck $pkgs 2>&1 | tee -a artifacts/govulncheck.log
fi

for dir in services/*/; do
  if [ -f "${dir}go.mod" ]; then
    echo "== govulncheck ${dir} ==" | tee -a artifacts/govulncheck.log
    (cd "$dir" && govulncheck ./...) 2>&1 | tee -a artifacts/govulncheck.log
  fi
done
