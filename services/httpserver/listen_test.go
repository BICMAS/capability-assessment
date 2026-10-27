package httpserver

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewRejectsNonPositiveTimeouts(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = New(":0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Timeouts{})
}

func TestListenAndServeGracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	server := New(addr, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), StandardTimeouts())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	signals := make(chan os.Signal, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		errCh <- listenAndServe(server, logger, "test", 2*time.Second, signals)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}

	signals <- os.Interrupt
	wg.Wait()
	if err := <-errCh; err != nil {
		t.Fatalf("listenAndServe = %v", err)
	}
}

func TestWrapRejectsBadContentTypeAndOversizedBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("content type", func(t *testing.T) {
		handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), "test", logger, 64)
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader("hello"))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("too large", func(t *testing.T) {
		handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}), "test", logger, 8)
		req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader([]byte(`{"x":"0123456789"}`)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("json ok", func(t *testing.T) {
		handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), "test", logger, 64)
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
		if rec.Header().Get("X-Request-ID") == "" {
			t.Fatal("missing request id")
		}
	})
}
