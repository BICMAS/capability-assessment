package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bicmas-academy/learn/services/notify/db"
	"github.com/bicmas-academy/learn/services/notify/health"
	"github.com/bicmas-academy/learn/services/notify/inbox"
	"github.com/bicmas-academy/learn/services/notify/migrations"
)

const (
	defaultPort        = "8082"
	serviceDatabaseURL = "DATABASE_URL_NOTIFY"
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

	server := &http.Server{
		Addr: listenAddr(),
		Handler: inbox.NewRouter(func(ctx context.Context) error {
			return checkDatabase(ctx, database, databaseName)
		}, []byte(secret), inbox.NewSQLStore(db.New(database))),
	}
	logger.Info("listening", "service", health.ServiceName, "addr", server.Addr, "database", databaseName)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
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

func checkDatabase(ctx context.Context, db *sql.DB, want string) error {
	var got string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&got); err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("connected to %s", got)
	}
	return nil
}
