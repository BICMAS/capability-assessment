package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bicmas-academy/learn/services/capability-framework/blob"
	"github.com/bicmas-academy/learn/services/capability-framework/events"
	"github.com/bicmas-academy/learn/services/capability-framework/framework"
	"github.com/bicmas-academy/learn/services/capability-framework/health"
	"github.com/bicmas-academy/learn/services/capability-framework/migrations"
	"github.com/bicmas-academy/learn/services/capability-framework/outbox"
	"github.com/bicmas-academy/learn/services/capability-framework/roleprofile"
	"github.com/bicmas-academy/learn/services/httpserver"
)

const (
	defaultPort        = "8084"
	serviceDatabaseURL = "DATABASE_URL_CAPABILITY_FRAMEWORK"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "service", health.ServiceName, "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	databaseURL, err := loadDatabaseURL()
	if err != nil {
		return err
	}
	databaseName, err := databaseName(databaseURL)
	if err != nil {
		return err
	}

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := migrate(database); err != nil {
		return err
	}
	secret := os.Getenv("DEV_JWT_SECRET")
	if secret == "" {
		return errors.New("DEV_JWT_SECRET is required")
	}
	bucket, err := loadBucket()
	if err != nil {
		return err
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		return errors.New("NATS_URL is required")
	}
	publisher, err := events.Open(context.Background(), natsURL)
	if err != nil {
		return err
	}
	defer publisher.Close()
	store := framework.NewSQLStore(database)
	profiles := roleprofile.NewSQLStore(database)
	var provider roleprofile.Provider
	if key := strings.TrimSpace(os.Getenv("ROLE_PROFILE_MODEL_API_KEY")); key != "" {
		provider = roleprofile.HTTPProvider{APIKey: key, Model: os.Getenv("ROLE_PROFILE_MODEL_NAME"), Endpoint: os.Getenv("ROLE_PROFILE_MODEL_URL")}
	}
	profileWorkflow := roleprofile.NewWorkflow(profiles, provider, roleprofile.DefaultLimits(), logger)
	relayCtx, stopRelay := context.WithCancel(context.Background())
	defer stopRelay()
	go outbox.NewRelay(outbox.NewSQLStore(database), publisher, &outbox.Metrics{}, logger).Run(relayCtx)
	go profileWorkflow.Run(relayCtx)

	timeouts := httpserver.StandardTimeouts()
	handler := httpserver.Wrap(framework.NewRouter(func(ctx context.Context) error {
		return checkDatabase(ctx, database, databaseName)
	}, []byte(secret), store, bucket, profiles, profileWorkflow), health.ServiceName, logger, httpserver.UploadMaxBodyBytes)
	server := httpserver.New(listenAddr(), handler, timeouts)
	logger.Info("configured", "service", health.ServiceName, "addr", server.Addr, "database", databaseName)
	return httpserver.ListenAndServe(server, logger, health.ServiceName, timeouts.ShutdownTimeout)
}

func loadBucket() (blob.Store, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil, errors.New("S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, and S3_SECRET_KEY are required")
	}
	return blob.NewS3(endpoint, bucket, accessKey, secretKey), nil
}

func loadDatabaseURL() (string, error) {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value, nil
	}
	if value := os.Getenv(serviceDatabaseURL); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("DATABASE_URL or %s is required", serviceDatabaseURL)
}

func databaseName(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if name == "" || strings.Contains(name, "/") {
		return "", errors.New("DATABASE_URL is missing a database name")
	}
	return name, nil
}

func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return ":" + port
}

func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func checkDatabase(ctx context.Context, database *sql.DB, want string) error {
	var got string
	if err := database.QueryRowContext(ctx, "SELECT current_database()").Scan(&got); err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("connected to %s", got)
	}
	return nil
}
