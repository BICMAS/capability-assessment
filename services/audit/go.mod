module github.com/bicmas-academy/learn/services/audit

go 1.27

require (
	github.com/bicmas-academy/learn/services/httpserver v0.0.0
	github.com/bicmas-academy/learn/services/token v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/jackc/pgx/v5 v5.7.4
	github.com/nats-io/nats.go v1.39.1
	github.com/pressly/goose/v3 v3.24.1
)

replace github.com/bicmas-academy/learn/services/token => ../token

require (
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/bicmas-academy/learn/services/httpserver => ../httpserver
