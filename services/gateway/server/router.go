package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/gateway/auth"
	"github.com/bicmas-academy/learn/services/gateway/health"
)

func NewRouter(issuer auth.TokenIssuer, verifier auth.Verifier, publisher auth.Publisher) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", health.Handler())
	router.Post("/dev/login", auth.LoginHandler(issuer, publisher))
	router.Group(func(protected chi.Router) {
		protected.Use(auth.RequireToken(verifier))
		protected.Get("/me", auth.MeHandler)
	})
	return router
}
