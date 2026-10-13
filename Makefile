.PHONY: up down tidy test web

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

test:
	go test ./...
	@for dir in services/*; do \
		(cd $$dir && go test ./...); \
	done

web:
	npm --prefix apps/web run dev
