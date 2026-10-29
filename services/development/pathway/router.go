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
	protected.Post("/learning-resources", SaveResource(store))
	protected.Get("/learning-resources", ListResources(store))
	protected.Post("/learning-recommendations", CreateRecommendation(store))
	protected.Get("/learning-recommendations", ListLearningRecommendations(store))
	protected.Post("/practice-sessions", CreatePractice(store))
	protected.Get("/practice-sessions", ListPracticeSessions(store))
	return router
}
