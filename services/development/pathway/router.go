package pathway

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/development/health"
	"github.com/bicmas-academy/learn/services/token"
)

func NewRouter(check func(context.Context) error, secret, webhookSecret []byte, store Store, publisher Publisher) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	router.Post("/webhooks/completions", CompleteLearning(store, publisher, webhookSecret))
	protected := router.With(token.Middleware(secret))
	protected.Post("/course-mappings", MapCourse(store))
	protected.Post("/pathways", CreatePathway(store))
	protected.Get("/pathways", ReadPathway(store))
	return router
}
