package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type PersonSignedIn struct {
	Type           string `json:"type"`
	PersonID       string `json:"personId"`
	OrganisationID string `json:"organisationId"`
	OccurredAt     string `json:"occurredAt"`
}

type Publisher interface {
	PublishPersonSignedIn(ctx context.Context, event PersonSignedIn) error
}

func LoginHandler(issuer TokenIssuer, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Person string `json:"person"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		person, ok := developmentPeople()[body.Person]
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		token, err := issuer.Issue(body.Person)
		if errors.Is(err, ErrUnknownPerson) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := publisher.PublishPersonSignedIn(r.Context(), PersonSignedIn{
			Type:           "person.signed_in",
			PersonID:       person.PersonID,
			OrganisationID: person.OrganisationID,
			OccurredAt:     time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	name := ""
	if person, found := personByID(claims.PersonID); found {
		name = person.Name
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(meResponse{
		PersonID:       claims.PersonID,
		OrganisationID: claims.OrganisationID,
		Roles:          claims.Roles,
		ManagerOf:      claims.ManagerOf,
		Name:           name,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type meResponse struct {
	PersonID       string   `json:"personId"`
	OrganisationID string   `json:"organisationId"`
	Roles          []string `json:"roles"`
	ManagerOf      []string `json:"managerOf"`
	Name           string   `json:"name"`
}

func RequireToken(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			claims, err := verifier.Verify(raw)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || !claims.HasRole(role) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.Contains(token, " ") {
		return "", false
	}
	return token, true
}
