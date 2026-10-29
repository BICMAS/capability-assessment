package roleprofile

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/token"
)

func Mount(router chi.Router, store Store, workflow *Workflow) {
	if workflow == nil {
		if ai, ok := store.(profileAI); ok {
			workflow = NewWorkflow(ai, nil, DefaultLimits(), nil)
		}
	}
	router.Post("/profile-roles", createRole(store))
	router.Post("/canonical-skills", createSkill(store))
	router.Post("/role-profiles", createDraft(store))
	router.Get("/role-profiles/{profileId}", getProfile(store))
	router.Get("/role-profiles/{profileId}/versions", listVersions(store))
	router.Get("/role-profiles/{profileId}/compare", compare(store))
	router.Get("/role-profiles/{profileId}/history", history(store))
	router.Patch("/role-profile-versions/{versionId}", updateContext(store))
	router.Post("/role-profile-versions/{versionId}/job-description", uploadJobDescription(store))
	router.Get("/role-profile-versions/{versionId}/job-description/{fileId}", downloadJobDescription(store))
	router.Post("/role-profile-versions/{versionId}/skills", addSkill(store))
	router.Patch("/role-profile-versions/{versionId}/skills/{skillAssignmentId}", updateSkill(store))
	router.Delete("/role-profile-versions/{versionId}/skills/{skillAssignmentId}", removeSkill(store))
	router.Put("/role-profile-versions/{versionId}/skill-order", reorderSkills(store))
	router.Post("/role-profile-versions/{versionId}/skill-edits", saveSkillEdits(store))
	router.Post("/role-profile-versions/{versionId}/skill-merges", mergeSkills(store))
	router.Post("/role-profile-versions/{versionId}/submit", submit(store))
	router.Post("/role-profile-versions/{versionId}/resubmit", resubmit(store))
	router.Post("/role-profile-versions/{versionId}/publish", publish(store))
	router.Post("/role-profile-versions/{versionId}/next-draft", nextDraft(store))
	router.Post("/role-profile-versions/{versionId}/comments", comment(store))
	router.Get("/role-profile-workflow", getApprovalPath(store))
	router.Put("/role-profile-workflow", saveApprovalPath(store))
	router.Get("/role-profile-approvals", listApprovalQueue(store))
	router.Post("/role-profile-versions/{versionId}/approval-assignments", assignApproval(store))
	router.Get("/role-profile-versions/{versionId}/approvals", listApprovals(store))
	router.Post("/role-profile-versions/{versionId}/approval-decisions", decideApproval(store))
	if workflow != nil {
		router.Post("/role-profile-versions/{versionId}/clarification", startClarification(workflow))
		router.Get("/role-profile-versions/{versionId}/clarification", readClarification(workflow))
		router.Post("/role-profile-versions/{versionId}/clarification/answers", answerClarification(workflow))
		router.Post("/role-profile-versions/{versionId}/generation", startGeneration(workflow))
		router.Get("/role-profile-jobs/{jobId}", readGeneration(workflow))
		router.Post("/role-profile-jobs/{jobId}/retry", retryGeneration(workflow))
		router.Post("/role-profile-jobs/{jobId}/cancel", cancelGeneration(workflow))
	}
}

func createRole(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &body) {
			return
		}
		role, err := store.CreateRole(r.Context(), claims.OrganisationID, body.Name)
		writeResult(w, http.StatusCreated, role, err)
	}
}

func createSkill(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			Name       string `json:"name"`
			Category   string `json:"category"`
			Definition string `json:"definition"`
		}
		if !decode(w, r, &body) {
			return
		}
		skill, err := store.CreateSkill(r.Context(), claims.OrganisationID, Skill{Name: body.Name, Category: body.Category, Definition: body.Definition})
		writeResult(w, http.StatusCreated, skill, err)
	}
}

func createDraft(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			RoleID             string `json:"roleId"`
			FrameworkVersionID string `json:"frameworkVersionId"`
		}
		if !decode(w, r, &body) {
			return
		}
		profile, err := store.CreateDraft(r.Context(), claims.OrganisationID, body.RoleID, claims.PersonID, body.FrameworkVersionID)
		writeResult(w, http.StatusCreated, profile, err)
	}
}

func getProfile(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		profile, err := store.GetProfile(r.Context(), claims.OrganisationID, chi.URLParam(r, "profileId"))
		writeResult(w, http.StatusOK, profile, err)
	}
}

func updateContext(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision      int      `json:"expectedRevision"`
			Purpose               string   `json:"purpose"`
			Responsibilities      string   `json:"responsibilities"`
			BusinessOutcomes      string   `json:"businessOutcomes"`
			DecisionScope         string   `json:"decisionScope"`
			Industry              string   `json:"industry"`
			Region                string   `json:"region"`
			TeamSize              *int     `json:"teamSize"`
			ReportingLevel        string   `json:"reportingLevel"`
			OrganogramID          string   `json:"organogramId"`
			PositionID            string   `json:"positionId"`
			BusinessUnit          string   `json:"businessUnit"`
			SubIndustry           string   `json:"subIndustry"`
			OrganisationalContext string   `json:"organisationalContext"`
			FrameworkVersionID    *string  `json:"frameworkVersionId"`
			Sources               []Source `json:"sources"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.UpdateContext(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.ExpectedRevision, Context{
			Purpose:               body.Purpose,
			Responsibilities:      body.Responsibilities,
			BusinessOutcomes:      body.BusinessOutcomes,
			DecisionScope:         body.DecisionScope,
			Industry:              body.Industry,
			Region:                body.Region,
			TeamSize:              body.TeamSize,
			ReportingLevel:        body.ReportingLevel,
			OrganogramID:          body.OrganogramID,
			PositionID:            body.PositionID,
			BusinessUnit:          body.BusinessUnit,
			SubIndustry:           body.SubIndustry,
			OrganisationalContext: body.OrganisationalContext,
			FrameworkVersionID:    body.FrameworkVersionID,
			Sources:               body.Sources,
		})
		writeResult(w, http.StatusOK, version, err)
	}
}

func addSkill(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision    int    `json:"expectedRevision"`
			SkillID             string `json:"skillId"`
			Category            string `json:"category"`
			Definition          string `json:"definition"`
			RequiredProficiency int    `json:"requiredProficiency"`
			Priority            string `json:"priority"`
			Rationale           string `json:"rationale"`
			Criteria            string `json:"criteria"`
			Provenance          string `json:"provenance"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.AddSkill(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.ExpectedRevision, SkillInput{
			SkillID:             body.SkillID,
			Category:            body.Category,
			Definition:          body.Definition,
			RequiredProficiency: body.RequiredProficiency,
			Priority:            body.Priority,
			Rationale:           body.Rationale,
			Criteria:            body.Criteria,
			Provenance:          body.Provenance,
		})
		writeResult(w, http.StatusCreated, version, err)
	}
}

func updateSkill(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision    int     `json:"expectedRevision"`
			Definition          string  `json:"definition"`
			RequiredProficiency int     `json:"requiredProficiency"`
			Priority            string  `json:"priority"`
			Rationale           string  `json:"rationale"`
			DisplayName         *string `json:"displayName"`
			Category            *string `json:"category"`
			Criteria            *string `json:"criteria"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.UpdateSkill(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), chi.URLParam(r, "skillAssignmentId"), claims.PersonID, body.ExpectedRevision, SkillEdit{
			Definition:          body.Definition,
			RequiredProficiency: body.RequiredProficiency,
			Priority:            body.Priority,
			Rationale:           body.Rationale,
			DisplayName:         body.DisplayName,
			Category:            body.Category,
			Criteria:            body.Criteria,
		})
		writeResult(w, http.StatusOK, version, err)
	}
}

func removeSkill(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		expected, ok := revisionQuery(w, r)
		if !ok {
			return
		}
		version, err := store.RemoveSkill(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), chi.URLParam(r, "skillAssignmentId"), claims.PersonID, expected)
		writeResult(w, http.StatusOK, version, err)
	}
}

func reorderSkills(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision   int      `json:"expectedRevision"`
			SkillAssignmentIDs []string `json:"skillAssignmentIds"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.ReorderSkills(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.ExpectedRevision, body.SkillAssignmentIDs)
		writeResult(w, http.StatusOK, version, err)
	}
}

func saveSkillEdits(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision int `json:"expectedRevision"`
			Edits            []struct {
				SkillAssignmentID   string  `json:"skillAssignmentId"`
				Definition          string  `json:"definition"`
				RequiredProficiency int     `json:"requiredProficiency"`
				Priority            string  `json:"priority"`
				Rationale           string  `json:"rationale"`
				DisplayName         *string `json:"displayName"`
				Category            *string `json:"category"`
				Criteria            *string `json:"criteria"`
			} `json:"edits"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		edits := make([]SkillRevision, 0, len(body.Edits))
		for _, edit := range body.Edits {
			edits = append(edits, SkillRevision{
				AssignmentID: edit.SkillAssignmentID,
				Edit: SkillEdit{
					Definition:          edit.Definition,
					RequiredProficiency: edit.RequiredProficiency,
					Priority:            edit.Priority,
					Rationale:           edit.Rationale,
					DisplayName:         edit.DisplayName,
					Category:            edit.Category,
					Criteria:            edit.Criteria,
				},
			})
		}
		version, err := store.SaveEdits(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.ExpectedRevision, edits)
		writeResult(w, http.StatusOK, version, err)
	}
}

func mergeSkills(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			ExpectedRevision int    `json:"expectedRevision"`
			KeepID           string `json:"keepId"`
			RemoveID         string `json:"removeId"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.MergeSkills(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.KeepID, body.RemoveID, body.ExpectedRevision)
		writeResult(w, http.StatusOK, version, err)
	}
}

func listVersions(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		versions, err := store.ListVersions(r.Context(), claims.OrganisationID, chi.URLParam(r, "profileId"))
		writeResult(w, http.StatusOK, versions, err)
	}
}

func compare(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		result, err := store.Compare(r.Context(), claims.OrganisationID, chi.URLParam(r, "profileId"), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
		writeResult(w, http.StatusOK, result, err)
	}
}

func submit(store Store) http.HandlerFunc {
	return revisionAction(func(claims token.Claims) bool { return claims.CanMaintainRoleProfiles() }, func(r *http.Request, claims token.Claims, expected int) (Version, error) {
		return store.Submit(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, expected)
	})
}

func publish(store Store) http.HandlerFunc {
	return revisionAction(func(claims token.Claims) bool { return claims.CanApproveRoleProfiles() }, func(r *http.Request, claims token.Claims, expected int) (Version, error) {
		return store.Publish(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, expected)
	})
}

func nextDraft(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		expected, ok := expectedBody(w, r)
		if !ok {
			return
		}
		version, err := store.NextDraft(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, expected)
		writeResult(w, http.StatusCreated, version, err)
	}
}

func comment(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			Body string `json:"body"`
		}
		if !decode(w, r, &body) {
			return
		}
		saved, err := store.Comment(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.Body)
		writeResult(w, http.StatusCreated, saved, err)
	}
}

func history(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		result, err := store.History(r.Context(), claims.OrganisationID, chi.URLParam(r, "profileId"))
		writeResult(w, http.StatusOK, result, err)
	}
}

func revisionAction(allowed func(token.Claims) bool, run func(*http.Request, token.Claims, int) (Version, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAccess(w, r, allowed)
		if !ok {
			return
		}
		expected, ok := expectedBody(w, r)
		if !ok {
			return
		}
		version, err := run(r, claims, expected)
		writeResult(w, http.StatusOK, version, err)
	}
}

func expectedBody(w http.ResponseWriter, r *http.Request) (int, bool) {
	var body struct {
		ExpectedRevision int `json:"expectedRevision"`
	}
	if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
		return 0, false
	}
	return body.ExpectedRevision, true
}

func revisionQuery(w http.ResponseWriter, r *http.Request) (int, bool) {
	expected, err := strconv.Atoi(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		writeCode(w, http.StatusBadRequest, "invalid")
		return 0, false
	}
	if !positiveRevision(w, expected) {
		return 0, false
	}
	return expected, true
}

func positiveRevision(w http.ResponseWriter, revision int) bool {
	if revision < 1 {
		writeCode(w, http.StatusBadRequest, "invalid")
		return false
	}
	return true
}

func requireMaintainer(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	return requireAccess(w, r, func(claims token.Claims) bool { return claims.CanMaintainRoleProfiles() })
}

func requireApprover(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	return requireAccess(w, r, func(claims token.Claims) bool { return claims.CanApproveRoleProfiles() })
}

func requireReader(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	return requireAccess(w, r, func(claims token.Claims) bool { return claims.CanReadRoleProfiles() })
}

func requireAccess(w http.ResponseWriter, r *http.Request, allowed func(token.Claims) bool) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthorized")
		return token.Claims{}, false
	}
	if !allowed(claims) {
		writeCode(w, http.StatusForbidden, "forbidden")
		return token.Claims{}, false
	}
	return claims, true
}

func assignApproval(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			AssigneePersonID string `json:"assigneePersonId"`
		}
		if !decode(w, r, &body) {
			return
		}
		assignment, err := store.AssignApproval(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.AssigneePersonID)
		writeResult(w, http.StatusCreated, assignment, err)
	}
}

func listApprovals(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		record, err := store.ListApprovals(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, claims.Roles)
		writeResult(w, http.StatusOK, record, err)
	}
}

func decideApproval(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		var body struct {
			AssignmentID     string `json:"assignmentId"`
			Decision         string `json:"decision"`
			Comment          string `json:"comment"`
			ExpectedRevision int    `json:"expectedRevision"`
		}
		if !decode(w, r, &body) || !positiveRevision(w, body.ExpectedRevision) {
			return
		}
		version, err := store.DecideApproval(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.AssignmentID, claims.PersonID, body.Decision, body.Comment, body.ExpectedRevision, claims.HasRole(token.RoleAdministrator), claims.Roles)
		writeResult(w, http.StatusOK, version, err)
	}
}

func resubmit(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		expected, ok := expectedBody(w, r)
		if !ok {
			return
		}
		version, err := store.Resubmit(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, expected, claims.Roles)
		writeResult(w, http.StatusOK, version, err)
	}
}

func getApprovalPath(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		stages, err := store.GetApprovalPath(r.Context(), claims.OrganisationID)
		writeResult(w, http.StatusOK, stages, err)
	}
}

func saveApprovalPath(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			Stages []ApprovalPathStage `json:"stages"`
		}
		if !decode(w, r, &body) {
			return
		}
		if err := store.SaveApprovalPath(r.Context(), claims.OrganisationID, claims.PersonID, body.Stages); err != nil {
			writeStoreError(w, err)
			return
		}
		stages, err := store.GetApprovalPath(r.Context(), claims.OrganisationID)
		writeResult(w, http.StatusOK, stages, err)
	}
}

func listApprovalQueue(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireReader(w, r)
		if !ok {
			return
		}
		view := r.URL.Query().Get("view")
		if view == "" {
			view = QueueAwaiting
		}
		items, err := store.ListApprovalQueue(r.Context(), claims.OrganisationID, claims.PersonID, view, claims.Roles)
		writeResult(w, http.StatusOK, items, err)
	}
}

func decode(w http.ResponseWriter, r *http.Request, body any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeCode(w, http.StatusBadRequest, "invalid")
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, status int, body any, err error) {
	if err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	case errors.Is(err, ErrInvalid):
		writeCode(w, http.StatusBadRequest, "invalid")
	case errors.Is(err, ErrConflict):
		writeCode(w, http.StatusConflict, "conflict")
	case errors.Is(err, ErrDuplicate):
		writeCode(w, http.StatusConflict, "duplicate")
	case errors.Is(err, ErrImmutable):
		writeCode(w, http.StatusConflict, "immutable")
	case errors.Is(err, ErrStale):
		writeCode(w, http.StatusConflict, "stale")
	case errors.Is(err, ErrUnresolved):
		writeCode(w, http.StatusConflict, "unresolved")
	case errors.Is(err, ErrBudget):
		writeCode(w, http.StatusTooManyRequests, "budget")
	case errors.Is(err, ErrCancelled):
		writeCode(w, http.StatusConflict, "cancelled")
	case errors.Is(err, ErrUnavailable):
		writeCode(w, http.StatusServiceUnavailable, "unavailable")
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	default:
		writeCode(w, http.StatusInternalServerError, "unexpected")
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
