#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

mkdir -p artifacts
: >artifacts/go-test.log

echo "== go test (workspace packages) ==" | tee -a artifacts/go-test.log
pkgs="$(go list ./... | grep -v '/node_modules/' || true)"
if [ -n "$pkgs" ]; then
  # shellcheck disable=SC2086
  go test $pkgs 2>&1 | tee -a artifacts/go-test.log
fi

for dir in services/*/; do
  if [ -f "${dir}go.mod" ]; then
    echo "== go test ${dir} ==" | tee -a artifacts/go-test.log
    (cd "$dir" && go test ./...) 2>&1 | tee -a artifacts/go-test.log
  fi
done
