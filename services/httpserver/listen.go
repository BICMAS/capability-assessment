package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// New builds an http.Server with explicit timeouts.
func New(addr string, handler http.Handler, timeouts Timeouts) *http.Server {
	if timeouts.ReadHeaderTimeout <= 0 || timeouts.ReadTimeout <= 0 || timeouts.WriteTimeout <= 0 || timeouts.IdleTimeout <= 0 {
		panic("httpserver: timeouts must be positive")
	}
	if timeouts.ShutdownTimeout <= 0 {
		timeouts.ShutdownTimeout = 15 * time.Second
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		ReadTimeout:       timeouts.ReadTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		IdleTimeout:       timeouts.IdleTimeout,
	}
}

// ListenAndServe runs the server until SIGINT/SIGTERM, then shuts down with a deadline.
// It does not call os.Exit; callers own process exit after deferred cleanup.
func ListenAndServe(server *http.Server, logger *slog.Logger, service string, shutdownTimeout time.Duration) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	return listenAndServe(server, logger, service, shutdownTimeout, signals)
}

func listenAndServe(server *http.Server, logger *slog.Logger, service string, shutdownTimeout time.Duration, signals <-chan os.Signal) error {
	if server == nil {
		return errors.New("httpserver: server is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "service", service, "addr", server.Addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-signals:
		logger.Info("shutdown signal", "service", service, "signal", fmt.Sprint(sig))
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		if err := <-errCh; err != nil {
			return err
		}
		logger.Info("shutdown complete", "service", service)
		return nil
	}
}
