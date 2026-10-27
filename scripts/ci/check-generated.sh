#!/usr/bin/env bash
# sqlc packages must regenerate bit-for-bit with the pinned sqlc version.
# OpenAPI *.gen.go snapshots are smoke-tested (generator runs + packages build).
# Refreshing *.gen.go to a newer oapi-codegen is a dedicated upgrade change.
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

export GOBIN="${root}/.bin"
export PATH="${GOBIN}:${PATH}"
mkdir -p "$GOBIN" artifacts

SQLC_VERSION=v1.26.0
if ! command -v sqlc >/dev/null 2>&1 || ! sqlc version 2>/dev/null | grep -q 'v1.26.0'; then
  GOMODCACHE="${root}/.gomodcache" go install github.com/sqlc-dev/sqlc/cmd/sqlc@"${SQLC_VERSION}"
fi
if ! command -v oapi-codegen >/dev/null 2>&1; then
  go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
fi

echo "== sqlc $(sqlc version) ==" | tee artifacts/check-generated.log

echo "== sqlc compile ==" | tee -a artifacts/check-generated.log
for dir in services/*/; do
  if [ -f "${dir}sqlc.yaml" ]; then
    echo "-- ${dir}" | tee -a artifacts/check-generated.log
    (cd "$dir" && sqlc compile) 2>&1 | tee -a artifacts/check-generated.log
  fi
done

echo "== sqlc generate must match committed db packages ==" | tee -a artifacts/check-generated.log
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
for dir in services/*/; do
  if [ -f "${dir}sqlc.yaml" ]; then
    service="$(basename "$dir")"
    echo "-- ${service}" | tee -a artifacts/check-generated.log
    mkdir -p "${tmpdir}/${service}"
    cp -R "${dir}." "${tmpdir}/${service}/"
    (cd "${tmpdir}/${service}" && sqlc generate)
    if ! diff -ru "${dir}db" "${tmpdir}/${service}/db" >"${tmpdir}/${service}.diff"; then
      echo "sqlc output drifted for ${service}:" | tee -a artifacts/check-generated.log
      cat "${tmpdir}/${service}.diff" | tee -a artifacts/check-generated.log
      exit 1
    fi
  fi
done

echo "== oapi-codegen smoke generate ==" | tee -a artifacts/check-generated.log
while IFS='|' read -r service spec; do
  [ -z "$service" ] && continue
  out="${tmpdir}/$(basename "$service")-models.gen.go"
  echo "-- ${service}" | tee -a artifacts/check-generated.log
  (
    cd "$service"
    oapi-codegen -generate types -package openapi -o "$out" "$spec"
  ) 2>&1 | tee -a artifacts/check-generated.log
  test -s "$out"
done <<'EOF'
services/audit|../../contracts/openapi/audit.yaml
services/notify|../../contracts/openapi/notify.yaml
services/org-directory|../../contracts/openapi/org-directory.yaml
services/capability-framework|../../contracts/openapi/capability-framework.yaml
EOF

echo "== committed openapi model packages build ==" | tee -a artifacts/check-generated.log
for service in services/audit services/notify services/org-directory services/capability-framework; do
  (cd "$service" && go test ./openapi -count=0) 2>&1 | tee -a artifacts/check-generated.log
done

echo "generated-code checks passed" | tee -a artifacts/check-generated.log
