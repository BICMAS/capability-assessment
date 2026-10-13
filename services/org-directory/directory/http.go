package directory

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/org-directory/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func ListDepartments(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		rows, err := store.ListDepartments(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		response := make([]openapi.Department, 0, len(rows))
		for _, row := range rows {
			response = append(response, openapi.Department{
				Id:             row.ID,
				OrganisationId: row.OrganisationID,
				Name:           row.Name,
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func ListPeople(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized)
			return
		}
		if !claims.HasRole("admin") && !claims.HasRole("manager") {
			writeError(w, http.StatusForbidden)
			return
		}
		rows, err := store.ListPeople(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		if claims.HasRole("manager") && !claims.HasRole("admin") {
			rows = filterManaged(rows, claims.ManagerOf)
		}
		response := make([]openapi.Person, 0, len(rows))
		for _, row := range rows {
			response = append(response, personJSON(row))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func ReadPerson(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized)
			return
		}
		personID := chi.URLParam(r, "personId")
		if !canReadPerson(claims, personID) {
			writeError(w, http.StatusForbidden)
			return
		}
		writeProfile(w, r, store, claims.OrganisationID, personID)
	}
}

func ReadProfile(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized)
			return
		}
		writeProfile(w, r, store, claims.OrganisationID, claims.PersonID)
	}
}

func ListRoleAssignments(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		rows, err := store.ListRoleAssignments(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		response := make([]openapi.RoleAssignment, 0, len(rows))
		for _, row := range rows {
			response = append(response, assignmentJSON(row))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func CreateRoleAssignment(store Store, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateRoleAssignment
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil || body.PersonId == "" || body.Title == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		createdAt := time.Now().UTC().Truncate(time.Second)
		assignment, err := store.CreateRoleAssignment(r.Context(), claims.OrganisationID, body.PersonId, body.Title, createdAt)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusBadRequest)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		if err := PublishAssignment(r.Context(), publisher, assignment); err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, assignmentJSON(assignment))
	}
}

func writeProfile(w http.ResponseWriter, r *http.Request, store Store, organisationID, personID string) {
	profile, err := store.Profile(r.Context(), organisationID, personID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError)
		return
	}
	roles := make([]openapi.RoleTitle, 0, len(profile.Roles))
	for _, role := range profile.Roles {
		roles = append(roles, openapi.RoleTitle{Id: role.ID, Title: role.Title})
	}
	writeJSON(w, http.StatusOK, openapi.Profile{
		Id:             profile.ID,
		OrganisationId: profile.OrganisationID,
		Name:           profile.Name,
		DepartmentName: profile.DepartmentName,
		Roles:          roles,
	})
}

func requireAdmin(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized)
		return token.Claims{}, false
	}
	if !claims.HasRole("admin") {
		writeError(w, http.StatusForbidden)
		return token.Claims{}, false
	}
	return claims, true
}

func canReadPerson(claims token.Claims, personID string) bool {
	if claims.PersonID == personID || claims.HasRole("admin") {
		return true
	}
	if !claims.HasRole("manager") {
		return false
	}
	for _, managed := range claims.ManagerOf {
		if managed == personID {
			return true
		}
	}
	return false
}

func filterManaged(people []Person, managerOf []string) []Person {
	allowed := make(map[string]struct{}, len(managerOf))
	for _, id := range managerOf {
		allowed[id] = struct{}{}
	}
	matched := make([]Person, 0)
	for _, person := range people {
		if _, ok := allowed[person.ID]; ok {
			matched = append(matched, person)
		}
	}
	return matched
}

func personJSON(person Person) openapi.Person {
	return openapi.Person{
		Id:             person.ID,
		OrganisationId: person.OrganisationID,
		Name:           person.Name,
		DepartmentName: person.DepartmentName,
	}
}

func assignmentJSON(assignment RoleAssignment) openapi.RoleAssignment {
	return openapi.RoleAssignment{
		Id:             assignment.ID,
		OrganisationId: assignment.OrganisationID,
		PersonId:       assignment.PersonID,
		PersonName:     assignment.PersonName,
		Title:          assignment.Title,
		CreatedAt:      assignment.CreatedAt,
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
