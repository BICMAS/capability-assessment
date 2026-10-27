package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/gateway/auth"
	"github.com/bicmas-academy/learn/services/gateway/health"
)

// Options configures the gateway HTTP surface.
// DevLogin is the development token issuer. When nil, POST /dev/login is not registered.
type Options struct {
	Verifier  auth.Verifier
	Publisher auth.Publisher
	DevLogin  auth.TokenIssuer
	Ready     func(context.Context) error
}

// NewRouter builds the gateway routes. Development login is registered only when
// DevLogin is non-nil; callers must not leave that issuer set in production.
func NewRouter(opts Options) http.Handler {
	if opts.Verifier == nil {
		opts.Verifier = auth.RejectingVerifier{}
	}
	router := chi.NewRouter()
	router.Get("/health", health.Live())
	router.Get("/ready", health.Ready(opts.Ready))
	if opts.DevLogin != nil {
		router.Post("/dev/login", auth.LoginHandler(opts.DevLogin, opts.Publisher))
	}
	router.Group(func(protected chi.Router) {
		protected.Use(auth.RequireToken(opts.Verifier))
		protected.Get("/me", auth.MeHandler)
	})
	return router
}
