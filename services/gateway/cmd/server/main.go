package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bicmas-academy/learn/services/gateway/auth"
	"github.com/bicmas-academy/learn/services/gateway/events"
	"github.com/bicmas-academy/learn/services/gateway/health"
	"github.com/bicmas-academy/learn/services/gateway/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "service", health.ServiceName, "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	secret := os.Getenv("DEV_JWT_SECRET")
	if secret == "" {
		return errors.New("DEV_JWT_SECRET is required")
	}
	dev, err := auth.NewDevHMAC([]byte(secret), 12*time.Hour)
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
	httpServer := &http.Server{
		Addr:    listenAddr(),
		Handler: server.NewRouter(dev, dev, publisher),
	}
	logger.Info("listening", "service", health.ServiceName, "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return ":" + port
}
