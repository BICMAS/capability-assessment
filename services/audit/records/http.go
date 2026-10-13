package records

import (
	"encoding/json"
	"net/http"

	"github.com/bicmas-academy/learn/services/audit/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func ListHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok || !claims.HasRole("admin") {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		rows, err := store.List(r.Context(), claims.OrganisationID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		response := make([]openapi.Record, 0, len(rows))
		for _, row := range rows {
			response = append(response, openapi.Record{
				ActorPersonId:  row.ActorPersonID,
				OrganisationId: row.OrganisationID,
				Action:         row.Action,
				EntityType:     row.EntityType,
				EntityId:       row.EntityID,
				OccurredAt:     row.OccurredAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
