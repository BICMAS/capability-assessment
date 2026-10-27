package plan

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/assessment-plan/health"
	"github.com/bicmas-academy/learn/services/token"
)

type ProfileBridge struct {
	Profiles ProfileReader
	Tenures  TenureReader
	Levels   LevelReader
	People   PeopleReader
}

func NewRouter(check func(context.Context) error, secret []byte, store Store, frameworks FrameworkReader, directory DirectoryReader, publisher Publisher, drafter Drafter, matcher BandMatcher, gaps GapRecorder, bridge ProfileBridge) http.Handler {
	router := chi.NewRouter()
	healthHandler := health.NewRouter(check)
	router.Get("/health", healthHandler.ServeHTTP)
	protected := router.With(token.Middleware(secret))
	protected.Post("/assignments", AssignPeople(store, frameworks, directory, publisher))
	protected.Get("/assignments/{assignmentId}", ReadAssignment(store))
	protected.Post("/assessment-plans", GeneratePlan(store, frameworks, drafter))
	if profileStore, ok := store.(ProfilePlanStore); ok {
		protected.Post("/profile-assessment-plans", CreateProfilePlan(profileStore, bridge.Profiles, bridge.Tenures))
		protected.Get("/profile-insights", ProfileInsights(profileStore, bridge.Levels, bridge.Tenures, bridge.People))
	}
	protected.Post("/plans", CreatePlan(store))
	protected.Get("/plans/{planId}", ReadPlan(store))
	protected.Put("/plans/{planId}", ReplacePlan(store))
	protected.Patch("/plans/{planId}/skills/{skillId}", EditSkill(store))
	protected.Post("/plans/{planId}/submission", SubmitPlan(store))
	protected.Post("/plans/{planId}/change-requests", RequestChanges(store))
	protected.Post("/plans/{planId}/approval", ApprovePlan(store, publisher))
	protected.Post("/plans/{planId}/versions", CopyApprovedPlan(store))
	protected.Post("/plans/{planId}/session", CreateSession(store))
	protected.Get("/assessment-sessions", ListSessions(store))
	protected.Get("/assessment-sessions/{sessionId}", ReadSession(store))
	protected.Post("/assessment-sessions/{sessionId}/start", StartSession(store))
	protected.Put("/assessment-sessions/{sessionId}/answers", SaveSessionAnswers(store))
	protected.Post("/assessment-sessions/{sessionId}/submission", SubmitSession(store, matcher))
	protected.Post("/assessment-sessions/{sessionId}/confirmation", ConfirmSession(store, gaps, publisher))
	return router
}
