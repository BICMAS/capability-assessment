package health

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const ServiceName = "gateway"

func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/health", Handler())
	return router
}

func Handler() http.HandlerFunc {
	return handleHealth
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": ServiceName,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
