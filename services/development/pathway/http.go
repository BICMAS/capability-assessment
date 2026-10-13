package pathway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/bicmas-academy/learn/services/development/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func MapCourse(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if !claims.HasRole("admin") {
			writeError(w, http.StatusForbidden)
			return
		}
		var body openapi.CourseMappingInput
		if !decode(w, r, &body) {
			return
		}
		mapping, err := store.MapCourse(r.Context(), MappingInput{
			OrganisationID:   claims.OrganisationID,
			ExternalCourseID: body.ExternalCourseID,
			CapabilityID:     body.CapabilityID,
			TargetLevel:      body.TargetLevel,
			Title:            body.Title,
		})
		writeResult(w, http.StatusCreated, mappingJSON(mapping), err)
	}
}

func CreatePathway(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.CreatePathway
		if !decode(w, r, &body) {
			return
		}
		if !allowPathway(claims, body.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		pathway, err := store.CreatePathway(r.Context(), PathwayInput{
			OrganisationID: claims.OrganisationID,
			PersonID:       body.PersonID,
			CapabilityID:   body.CapabilityID,
			TargetLevel:    body.TargetLevel,
		})
		writeResult(w, http.StatusCreated, pathwayJSON(pathway), err)
	}
}

func ReadPathway(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		personID := r.URL.Query().Get("personId")
		capabilityID := r.URL.Query().Get("capabilityId")
		if personID == "" || capabilityID == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		if !allowPathway(claims, personID) {
			writeError(w, http.StatusForbidden)
			return
		}
		pathway, err := store.GetPathway(r.Context(), claims.OrganisationID, personID, capabilityID)
		writeResult(w, http.StatusOK, pathwayJSON(pathway), err)
	}
}

func CompleteLearning(store Store, publisher Publisher, secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || !signed(secret, body, r.Header.Get("X-Learning-Signature")) {
			writeError(w, http.StatusUnauthorized)
			return
		}
		var payload openapi.CompletionWebhook
		if err := json.Unmarshal(body, &payload); err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		completedAt, err := time.Parse(time.RFC3339, payload.CompletedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		pathway, created, err := store.Complete(r.Context(), CompletionInput{
			OrganisationID: payload.OrganisationID,
			PersonID:       payload.PersonID,
			PlanItemID:     payload.PlanItemID,
			CompletedAt:    completedAt,
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if created {
			if err := publisher.PublishLearningCompleted(r.Context(), LearningCompleted{
				Type:           "learning.completed",
				OrganisationID: payload.OrganisationID,
				PersonID:       payload.PersonID,
				PlanItemID:     payload.PlanItemID,
				OccurredAt:     completedAt.UTC().Format(time.RFC3339),
			}); err != nil {
				writeError(w, http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, http.StatusOK, pathwayJSON(pathway))
	}
}

func signed(secret, body []byte, presented string) bool {
	if len(secret) == 0 || presented == "" {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(presented))
}

func allowPathway(claims token.Claims, personID string) bool {
	if claims.HasRole("admin") || claims.PersonID == personID {
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

func mappingJSON(mapping Mapping) openapi.CourseMapping {
	return openapi.CourseMapping{
		ID:               mapping.ID,
		OrganisationID:   mapping.OrganisationID,
		ExternalCourseID: mapping.ExternalCourseID,
		CapabilityID:     mapping.CapabilityID,
		TargetLevel:      mapping.TargetLevel,
		Title:            mapping.Title,
	}
}

func pathwayJSON(pathway Pathway) openapi.Pathway {
	items := make([]openapi.PlanItem, 0, len(pathway.Items))
	for _, item := range pathway.Items {
		encoded := openapi.PlanItem{
			ID:     item.ID,
			Stage:  item.Stage,
			Title:  item.Title,
			Status: item.Status,
		}
		if item.DeepLink != "" {
			link := item.DeepLink
			encoded.DeepLink = &link
		}
		items = append(items, encoded)
	}
	return openapi.Pathway{
		ID:             pathway.ID,
		OrganisationID: pathway.OrganisationID,
		PersonID:       pathway.PersonID,
		CapabilityID:   pathway.CapabilityID,
		TargetLevel:    pathway.TargetLevel,
		Items:          items,
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
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest)
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
