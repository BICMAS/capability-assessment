package httpserver

import "time"

// Timeouts holds explicit http.Server timeout settings.
type Timeouts struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// StandardTimeouts covers typical JSON CRUD endpoints.
func StandardTimeouts() Timeouts {
	return Timeouts{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   15 * time.Second,
	}
}

// LongRequestTimeouts covers assessment-plan endpoints that still call the
// model synchronously when generating a plan draft. Prefer moving that work to
// an asynchronous job; until then WriteTimeout must exceed the model client timeout.
func LongRequestTimeouts() Timeouts {
	return Timeouts{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       90 * time.Second,
		ShutdownTimeout:   30 * time.Second,
	}
}

const (
	// DefaultMaxBodyBytes limits JSON and small multipart request bodies.
	DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB
	// UploadMaxBodyBytes limits CSV import uploads.
	UploadMaxBodyBytes int64 = 8 << 20 // 8 MiB
)
