package inbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/token"
)

func TestInboxIsEmptyForPeopleWithoutNotices(t *testing.T) {
	store := &Memory{items: []Item{{
		ID:             "notice-1",
		OrganisationID: "org-1",
		PersonID:       "maya",
		Title:          "Hello",
		Body:           "A notice",
		CreatedAt:      time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC),
	}}}
	handler := ListHandler(store)
	for _, personID := range []string{"ada", "eli"} {
		request := httptest.NewRequest(http.MethodGet, "/inbox", nil)
		request = request.WithContext(token.WithClaims(request.Context(), token.Claims{
			PersonID:       personID,
			OrganisationID: "org-1",
			Roles:          []string{"employee"},
		}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", personID, response.Code)
		}
		var payload struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Items) != 0 {
			t.Fatalf("%s inbox = %#v", personID, payload.Items)
		}
	}
}
