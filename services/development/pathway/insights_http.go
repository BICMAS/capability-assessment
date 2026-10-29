package pathway

import (
	"net/http"
	"strings"

	"github.com/bicmas-academy/learn/services/token"
)

func SaveResource(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if !claims.HasRole("admin") && !claims.HasRole("hr") {
			writeError(w, http.StatusForbidden)
			return
		}
		var body struct {
			Title       string `json:"title"`
			SkillID     string `json:"skillId"`
			TargetLevel int    `json:"targetLevel"`
			Reference   string `json:"reference"`
			Rationale   string `json:"rationale"`
		}
		if !decode(w, r, &body) {
			return
		}
		resource, err := store.SaveResource(r.Context(), ResourceInput{
			OrganisationID: claims.OrganisationID, Title: body.Title, SkillID: body.SkillID,
			TargetLevel: body.TargetLevel, Reference: body.Reference, Rationale: body.Rationale,
		})
		writeResult(w, http.StatusCreated, resourceJSON(resource), err)
	}
}

func ListResources(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		resources, err := store.ListResources(r.Context(), claims.OrganisationID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		payload := make([]map[string]any, 0, len(resources))
		for _, resource := range resources {
			payload = append(payload, resourceJSON(resource))
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func CreateRecommendation(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body struct {
			PersonID     string `json:"personId"`
			CapabilityID string `json:"capabilityId"`
			TargetLevel  int    `json:"targetLevel"`
			State        string `json:"state"`
		}
		if !decode(w, r, &body) {
			return
		}
		if !allowInsightPerson(claims, body.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		recommendation, err := RecommendLearning(r.Context(), store, claims.OrganisationID, GapSignal{
			PersonID: body.PersonID, CapabilityID: body.CapabilityID, TargetLevel: body.TargetLevel, State: body.State,
		})
		writeResult(w, http.StatusCreated, recommendation, err)
	}
}

func ListLearningRecommendations(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		personID := r.URL.Query().Get("personId")
		if personID == "" && !insightPrivileged(claims) {
			writeError(w, http.StatusForbidden)
			return
		}
		if personID != "" && !allowInsightPerson(claims, personID) {
			writeError(w, http.StatusForbidden)
			return
		}
		rows, err := store.ListRecommendations(r.Context(), claims.OrganisationID, personID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if rows == nil {
			rows = []LearningRecommendation{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func CreatePractice(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body struct {
			PersonID    string `json:"personId"`
			ScenarioKey string `json:"scenarioKey"`
			Message     string `json:"message"`
			Record      bool   `json:"record"`
		}
		if !decode(w, r, &body) {
			return
		}
		if !allowInsightPerson(claims, body.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		record, err := Practise(r.Context(), store, PracticeInput{
			OrganisationID: claims.OrganisationID, PersonID: body.PersonID, ScenarioKey: body.ScenarioKey,
			Message: body.Message, Record: body.Record,
		})
		writeResult(w, http.StatusCreated, record, err)
	}
}

func ListPracticeSessions(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		personID := strings.TrimSpace(r.URL.Query().Get("personId"))
		if personID == "" || !allowInsightPerson(claims, personID) {
			writeError(w, http.StatusForbidden)
			return
		}
		rows, err := store.ListPractice(r.Context(), claims.OrganisationID, personID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if rows == nil {
			rows = []PracticeRecord{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func allowInsightPerson(claims token.Claims, personID string) bool {
	if personID == "" {
		return false
	}
	if insightPrivileged(claims) || claims.PersonID == personID {
		return true
	}
	if !claims.HasRole("manager") {
		return false
	}
	for _, id := range claims.ManagerOf {
		if id == personID {
			return true
		}
	}
	return false
}

func insightPrivileged(claims token.Claims) bool {
	return claims.HasRole("admin") || claims.HasRole("hr") || claims.HasRole("leader")
}

func resourceJSON(resource Resource) map[string]any {
	return map[string]any{
		"id": resource.ID, "organisationId": resource.OrganisationID, "title": resource.Title,
		"skillId": resource.SkillID, "targetLevel": resource.TargetLevel, "reference": resource.Reference,
		"rationale": resource.Rationale,
	}
}
