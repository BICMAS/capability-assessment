package levels

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/assessment-gap/health"
	"github.com/bicmas-academy/learn/services/token"
)

func NewRouter(check func(context.Context) error, secret []byte, store Store, publisher Publisher) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	protected := router.With(token.Middleware(secret))
	protected.Post("/attempts", RecordAttempt(store))
	protected.Get("/levels", ReadLevel(store))
	protected.Post("/levels/{levelId}/overrides", OverrideLevel(store, publisher))
	return router
}
