package directory

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/org-directory/health"
	"github.com/bicmas-academy/learn/services/token"
)

func NewRouter(check func(context.Context) error, secret []byte, store Store, publisher Publisher) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	protected := router.With(token.Middleware(secret))
	protected.Get("/departments", ListDepartments(store))
	protected.Get("/people", ListPeople(store))
	protected.Get("/people/{personId}", ReadPerson(store))
	protected.Get("/profile", ReadProfile(store))
	protected.Get("/role-assignments", ListRoleAssignments(store))
	protected.Post("/role-assignments", CreateRoleAssignment(store, publisher))
	return router
}
