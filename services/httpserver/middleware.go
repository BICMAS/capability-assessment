package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// Wrap applies request-id, access logging, body-size limits, and content-type checks.
// It does not log Authorization headers, cookies, or request bodies.
func Wrap(handler http.Handler, service string, logger *slog.Logger, maxBodyBytes int64) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxBodyBytes
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID))

		if r.Body != nil && r.Body != http.NoBody {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}

		if !allowedContentType(r) {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			logAccess(logger, service, requestID, r, http.StatusUnsupportedMediaType, time.Since(start))
			return
		}

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		handler.ServeHTTP(recorder, r)
		logAccess(logger, service, requestID, r, recorder.status, time.Since(start))
	})
}

// RequestID returns the request id from context when Wrap has run.
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func allowedContentType(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return true
	}
	if r.ContentLength == 0 {
		return true
	}
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return false
	}
	media := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	switch media {
	case "application/json", "multipart/form-data", "application/x-www-form-urlencoded":
		return true
	default:
		return false
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logAccess(logger *slog.Logger, service, requestID string, r *http.Request, status int, duration time.Duration) {
	route := r.URL.Path
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		route = rctx.RoutePattern()
	}
	logger.Info("request",
		"service", service,
		"request_id", requestID,
		"method", r.Method,
		"route", route,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	)
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf[:])
}
