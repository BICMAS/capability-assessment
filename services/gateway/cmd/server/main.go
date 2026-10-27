package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/bicmas-academy/learn/services/gateway/auth"
	"github.com/bicmas-academy/learn/services/gateway/config"
	"github.com/bicmas-academy/learn/services/gateway/events"
	"github.com/bicmas-academy/learn/services/gateway/health"
	"github.com/bicmas-academy/learn/services/gateway/server"
	"github.com/bicmas-academy/learn/services/httpserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "service", health.ServiceName, "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	materials, err := auth.NewMaterials(cfg)
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

	timeouts := httpserver.StandardTimeouts()
	handler := httpserver.Wrap(server.NewRouter(server.Options{
		Verifier:  materials.Verifier,
		Publisher: publisher,
		DevLogin:  materials.Issuer,
		Ready: func(context.Context) error {
			return publisher.Ready()
		},
	}), health.ServiceName, logger, httpserver.DefaultMaxBodyBytes)
	httpServer := httpserver.New(listenAddr(), handler, timeouts)
	logger.Info("configured",
		"service", health.ServiceName,
		"addr", httpServer.Addr,
		"app_env", cfg.AppEnv,
		"dev_login_enabled", cfg.DevLoginEnabled,
		"read_header_timeout", timeouts.ReadHeaderTimeout.String(),
		"write_timeout", timeouts.WriteTimeout.String(),
	)
	return httpserver.ListenAndServe(httpServer, logger, health.ServiceName, timeouts.ShutdownTimeout)
}

func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return ":" + port
}
