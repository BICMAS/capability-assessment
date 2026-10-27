.PHONY: up down tidy test web ci ci-go ci-web ci-contracts ci-security ci-integration fmt

up:
	docker compose --env-file .env.example up -d --wait
	set -a; . ./.env.example; set +a; python3 deploy/s3/create_bucket.py

down:
	docker compose --env-file .env.example down

tidy:
	go mod tidy
	@for dir in services/*; do \
		(cd $$dir && go mod tidy); \
	done
	go work sync

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.gomodcache/*' -not -path './.bin/*')

test:
	bash scripts/ci/go-test.sh

web:
	npm --prefix apps/web run dev

ci-go:
	@unformatted="$$(gofmt -l $$(find . -name '*.go' -not -path './.gomodcache/*' -not -path './.bin/*'))"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	bash scripts/ci/go-vet.sh
	go install honnef.co/go/tools/cmd/staticcheck@latest
	bash scripts/ci/go-staticcheck.sh
	bash scripts/ci/go-test.sh

ci-web:
	npm --prefix apps/web ci
	npm --prefix apps/web run typecheck
	npm --prefix apps/web run lint
	npm --prefix apps/web run test
	npm --prefix apps/web run build

ci-contracts:
	bash scripts/ci/validate-openapi.sh
	bash scripts/ci/validate-events.sh
	bash scripts/ci/check-generated.sh

ci-security:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	bash scripts/ci/govulncheck.sh
	npm --prefix apps/web ci
	npm --prefix apps/web audit --omit=dev --audit-level=high

ci-integration:
	@test -n "$${NATS_URL}" || (echo "set NATS_URL" >&2; exit 1)
	@test -n "$${INTEGRATION_DATABASE_URL}" || (echo "set INTEGRATION_DATABASE_URL" >&2; exit 1)
	bash scripts/ci/go-integration.sh

ci: ci-go ci-web ci-contracts ci-security
