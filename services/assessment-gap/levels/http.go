package levels

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/assessment-gap/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func RecordAttempt(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.RecordAttempt
		if !decode(w, r, &body) {
			return
		}
		if columnOnly(body.Source) {
			writeError(w, http.StatusBadRequest)
			return
		}
		if !allowRecord(claims, body.PersonID, body.Source) {
			writeError(w, http.StatusForbidden)
			return
		}
		required, err := TenthsFromFloat(body.RequiredLevel, 40)
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		explanation, err := store.Record(r.Context(), AttemptInput{
			OrganisationID:     claims.OrganisationID,
			PersonID:           body.PersonID,
			FrameworkVersionID: body.FrameworkVersionID,
			CapabilityCode:     body.CapabilityCode,
			Source:             body.Source,
			Percentage:         body.Percentage,
			EvidenceIDs:        body.EvidenceIDs,
			RequiredLevel:      required,
		})
		writeResult(w, http.StatusCreated, explanationJSON(explanation), err)
	}
}

func ReadLevel(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		personID := r.URL.Query().Get("personId")
		frameworkVersionID := r.URL.Query().Get("frameworkVersionId")
		capabilityCode := r.URL.Query().Get("capabilityCode")
		if personID == "" || frameworkVersionID == "" || capabilityCode == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		if !allowRead(claims, personID) {
			writeError(w, http.StatusForbidden)
			return
		}
		explanation, err := store.Get(r.Context(), claims.OrganisationID, personID, frameworkVersionID, capabilityCode)
		writeResult(w, http.StatusOK, explanationJSON(explanation), err)
	}
}

func OverrideLevel(store Store, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.CreateOverride
		if !decode(w, r, &body) {
			return
		}
		explanation, err := store.GetByID(r.Context(), claims.OrganisationID, chi.URLParam(r, "levelId"))
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		if !claims.HasRole("manager") || !manages(claims, explanation.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		newValue, err := TenthsFromFloat(body.NewValue, 40)
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		row, err := store.Override(r.Context(), OverrideInput{
			OrganisationID: claims.OrganisationID,
			ExplanationID:  explanation.ID,
			ActorPersonID:  claims.PersonID,
			Reason:         body.Reason,
			NewValue:       newValue,
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := publisher.PublishLevelOverridden(r.Context(), LevelOverridden{
			Type:           "level.overridden",
			OrganisationID: row.OrganisationID,
			OverrideID:     row.ID,
			ActorPersonID:  row.ActorPersonID,
			OccurredAt:     row.CreatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, overrideJSON(row))
	}
}

func allowRecord(claims token.Claims, personID, source string) bool {
	switch source {
	case SourceKnowledge, SourceSituationalJudgement, SourceSelfAssessment:
		return claims.PersonID == personID || claims.HasRole("admin")
	case SourceManagerAssessment:
		return claims.HasRole("admin") || (claims.HasRole("manager") && manages(claims, personID))
	default:
		return false
	}
}

func allowRead(claims token.Claims, personID string) bool {
	if claims.HasRole("admin") || claims.PersonID == personID {
		return true
	}
	return claims.HasRole("manager") && manages(claims, personID)
}

func manages(claims token.Claims, personID string) bool {
	for _, id := range claims.ManagerOf {
		if id == personID {
			return true
		}
	}
	return false
}

func explanationJSON(explanation Explanation) openapi.Explanation {
	return openapi.Explanation{
		ID:                             explanation.ID,
		OrganisationID:                 explanation.OrganisationID,
		PersonID:                       explanation.PersonID,
		FrameworkVersionID:             explanation.FrameworkVersionID,
		CapabilityCode:                 explanation.CapabilityCode,
		KnowledgePercentage:            explanation.Knowledge,
		SituationalJudgementPercentage: explanation.SituationalJudgement,
		AIInterviewPercentage:          explanation.AIInterview,
		SimulationPercentage:           explanation.Simulation,
		ManagerAssessmentPercentage:    explanation.ManagerAssessment,
		SelfAssessmentPercentage:       explanation.SelfAssessment,
		Mean:                           explanation.Mean.String(),
		Divisor:                        explanation.Divisor,
		RequiredLevel:                  explanation.RequiredLevel.String(),
		CurrentLevel:                   explanation.CurrentLevel.String(),
		Gap:                            explanation.Gap.String(),
		Status:                         explanation.Status,
	}
}

func overrideJSON(row Override) openapi.Override {
	return openapi.Override{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		ExplanationID:  row.ExplanationID,
		ActorPersonID:  row.ActorPersonID,
		Reason:         row.Reason,
		PreviousValue:  row.PreviousValue.String(),
		NewValue:       row.NewValue.String(),
	}
}

func claimsFrom(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized)
		return token.Claims{}, false
	}
	return claims, true
}

func decode(w http.ResponseWriter, r *http.Request, body any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeError(w, http.StatusBadRequest)
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, status int, body any, err error) {
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, status, body)
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound)
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrNoWriter):
		writeError(w, http.StatusBadRequest)
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden)
	default:
		writeError(w, http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
