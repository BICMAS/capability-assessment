package framework

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/capability-framework/blob"
	"github.com/bicmas-academy/learn/services/capability-framework/health"
	"github.com/bicmas-academy/learn/services/capability-framework/roleprofile"
	"github.com/bicmas-academy/learn/services/token"
)

func NewRouter(check func(context.Context) error, secret []byte, store Store, blobs blob.Store, profiles roleprofile.Store, profilesAI *roleprofile.Workflow) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	protected := router.With(token.Middleware(secret))
	protected.Get("/framework-versions", ListVersions(store))
	protected.Get("/published-frameworks", ListPublished(store))
	protected.Post("/frameworks", CreateDraft(store))
	protected.Get("/framework-versions/{versionId}", ReadVersion(store))
	protected.Patch("/framework-versions/{versionId}", PatchRole(store))
	protected.Post("/framework-versions/{versionId}/publish", Publish(store))
	protected.Post("/framework-versions/{versionId}/approve", Approve(store))
	protected.Patch("/framework-versions/{versionId}/indicators/{indicatorId}", UpdateIndicator(store))
	protected.Post("/framework-versions/{versionId}/next-draft", CreateNextDraft(store))
	protected.Post("/framework-versions/{versionId}/roles", AddRole(store))
	protected.Post("/framework-versions/{versionId}/capabilities", AddCapability(store))
	protected.Post("/framework-versions/{versionId}/requirements", AddRequirement(store))
	protected.Post("/framework-versions/{versionId}/role-capabilities", AddRoleCapability(store))
	protected.Post("/framework-versions/{versionId}/indicators", AddIndicator(store))
	protected.Post("/imports", ImportCSV(store, blobs))
	protected.Post("/framework-versions/{versionId}/in-review", SubmitForReview(store))
	protected.Post("/framework-versions/{versionId}/proposals/{proposalId}", DecideProposal(store))
	if profiles != nil {
		roleprofile.Mount(protected, profiles, profilesAI)
	}
	return router
}
