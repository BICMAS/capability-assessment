package inbox

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/notify/health"
	"github.com/bicmas-academy/learn/services/token"
)

func NewRouter(check func(context.Context) error, secret []byte, store Store) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	router.With(token.Middleware(secret)).Get("/inbox", ListHandler(store))
	return router
}
