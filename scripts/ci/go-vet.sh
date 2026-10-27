#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

mkdir -p artifacts
echo "== go vet (workspace packages) ==" | tee artifacts/go-vet.log
pkgs="$(go list ./... | grep -v '/node_modules/' || true)"
if [ -n "$pkgs" ]; then
  # shellcheck disable=SC2086
  go vet $pkgs 2>&1 | tee -a artifacts/go-vet.log
fi

for dir in services/*/; do
  if [ -f "${dir}go.mod" ]; then
    echo "== go vet ${dir} ==" | tee -a artifacts/go-vet.log
    (cd "$dir" && go vet ./...) 2>&1 | tee -a artifacts/go-vet.log
  fi
done
