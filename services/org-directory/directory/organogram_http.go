package directory

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/token"
)

func ListOrganograms(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramReader(w, r)
		if !ok {
			return
		}
		rows, err := store.ListOrganograms(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []Organogram{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func CreateOrganogram(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		created, err := store.CreateOrganogram(r.Context(), claims.OrganisationID, body.Name, claims.PersonID, time.Now().UTC().Truncate(time.Second))
		if errors.Is(err, ErrInvalid) {
			writeError(w, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrDuplicate) {
			writeError(w, http.StatusConflict)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func ListPositions(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramReader(w, r)
		if !ok {
			return
		}
		rows, err := store.ListPositions(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"))
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		if rows == nil {
			rows = []Position{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func CreatePosition(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		var body struct {
			DepartmentID        string `json:"departmentId"`
			Title               string `json:"title"`
			PersonID            string `json:"personId"`
			ReportsToPositionID string `json:"reportsToPositionId"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		created, err := store.CreatePosition(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"), PositionInput{
			DepartmentID:        body.DepartmentID,
			Title:               body.Title,
			PersonID:            body.PersonID,
			ReportsToPositionID: body.ReportsToPositionID,
		}, time.Now().UTC().Truncate(time.Second))
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrInvalid) {
			writeError(w, http.StatusBadRequest)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func requireOrganogramReader(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	return requireDirectoryAccess(w, r, func(claims token.Claims) bool { return claims.CanReadOrganogram() })
}

func requireOrganogramMaintainer(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	return requireDirectoryAccess(w, r, func(claims token.Claims) bool { return claims.CanMaintainOrganogram() })
}

func requireDirectoryAccess(w http.ResponseWriter, r *http.Request, allowed func(token.Claims) bool) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized)
		return token.Claims{}, false
	}
	if !allowed(claims) {
		writeError(w, http.StatusForbidden)
		return token.Claims{}, false
	}
	return claims, true
}
