#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

if ! command -v staticcheck >/dev/null 2>&1; then
  echo "staticcheck is required" >&2
  exit 1
fi

mkdir -p artifacts .cache/staticcheck
export STATICCHECK_CACHE="${root}/.cache/staticcheck"
checks='all,-ST1000,-ST1003,-ST1016,-ST1020,-ST1021,-ST1022,-S1016'

echo "== staticcheck (workspace packages) ==" | tee artifacts/go-staticcheck.log
pkgs="$(go list ./... | grep -v '/node_modules/' || true)"
if [ -n "$pkgs" ]; then
  # shellcheck disable=SC2086
  staticcheck -checks="$checks" $pkgs 2>&1 | tee -a artifacts/go-staticcheck.log
fi

for dir in services/*/; do
  if [ -f "${dir}go.mod" ]; then
    echo "== staticcheck ${dir} ==" | tee -a artifacts/go-staticcheck.log
    (cd "$dir" && staticcheck -checks="$checks" ./...) 2>&1 | tee -a artifacts/go-staticcheck.log
  fi
done
