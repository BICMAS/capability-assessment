package inbox

import (
	"encoding/json"
	"net/http"

	"github.com/bicmas-academy/learn/services/notify/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func ListHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		rows, err := store.List(r.Context(), claims.OrganisationID, claims.PersonID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		items := make([]openapi.InboxItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, openapi.InboxItem{
				Id:             row.ID,
				OrganisationId: row.OrganisationID,
				PersonId:       row.PersonID,
				Title:          row.Title,
				Body:           row.Body,
				CreatedAt:      row.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(openapi.Inbox{Items: items}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
