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
	protected.Get("/organograms", ListOrganograms(store))
	protected.Post("/organograms", CreateOrganogram(store))
	protected.Post("/organograms/{organogramId}/publish", PublishOrganogram(store))
	protected.Get("/organograms/{organogramId}/positions", ListPositions(store))
	protected.Post("/organograms/{organogramId}/positions", CreatePosition(store))
	protected.Patch("/organograms/{organogramId}/positions/{positionId}", UpdatePosition(store))
	protected.Delete("/organograms/{organogramId}/positions/{positionId}", DeletePosition(store))
	protected.Get("/organograms/{organogramId}/files", ListOrganogramFiles(store))
	protected.Post("/organograms/{organogramId}/files", UploadOrganogramFile(store))
	protected.Get("/organograms/{organogramId}/files/{fileId}", ReadOrganogramFile(store))
	protected.Get("/organogram-template", DownloadOrganogramTemplate())
	return router
}
